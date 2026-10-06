<?php

namespace Plugin\Bepusdt;

use App\Services\Plugin\AbstractPlugin;
use App\Contracts\PaymentInterface;
use App\Exceptions\ApiException;
use Illuminate\Support\Facades\Log;

class Plugin extends AbstractPlugin implements PaymentInterface
{
    // BEpusdt order status: https://github.com/v03413/BEpusdt/blob/main/docs/api/api.md
    private const STATUS_SUCCESS = 2;

    public function boot(): void
    {
        $this->filter('available_payment_methods', function ($methods) {
            if ($this->getConfig('enabled', true)) {
                $methods['Bepusdt'] = [
                    'name' => $this->getConfig('display_name', 'USDT'),
                    'icon' => $this->getConfig('icon', '💰'),
                    'plugin_code' => $this->getPluginCode(),
                    'type' => 'plugin'
                ];
            }
            return $methods;
        });
    }

    public function form(): array
    {
        return [
            'api_url' => [
                'label' => 'API接口地址',
                'type' => 'string',
                'required' => true,
                'description' => 'BEpusdt部署地址，例如：https://your-bepusdt.com'
            ],
            'token' => [
                'label' => 'API Token',
                'type' => 'string',
                'required' => true,
                'description' => 'BEpusdt后台配置的API Token，用于签名和验签'
            ],
            'trade_type' => [
                'label' => '支付方式',
                'type' => 'string',
                'description' => '例如：usdt.trc20、usdt.erc20、tron.trx、usdc.polygon，留空默认 usdt.trc20'
            ],
            'fiat' => [
                'label' => '结算货币',
                'type' => 'string',
                'description' => 'CNY、USD、EUR、GBP、JPY，留空默认 CNY'
            ],
            'timeout' => [
                'label' => '订单超时时间(秒)',
                'type' => 'string',
                'description' => '最小120秒，留空默认600秒'
            ],
        ];
    }

    public function pay($order): array
    {
        $params = [
            'order_id' => $order['trade_no'],
            'notify_url' => $order['notify_url'],
            'redirect_url' => $order['return_url'],
            'amount' => (float) sprintf('%.2f', $order['total_amount'] / 100),
        ];

        if ($tradeType = $this->getConfig('trade_type')) {
            $params['trade_type'] = $tradeType;
        }
        if ($fiat = $this->getConfig('fiat')) {
            $params['fiat'] = $fiat;
        }
        if ($timeout = $this->getConfig('timeout')) {
            $params['timeout'] = (int) $timeout;
        }

        $params['signature'] = $this->sign($params);

        $ret_raw = $this->curlPost($this->getApiUrl() . '/api/v1/order/create-transaction', json_encode($params));
        $ret = @json_decode($ret_raw, true);

        if (empty($ret['data']['payment_url'])) {
            \Log::error('BEpusdt pay failed', [
                'api_url' => $this->getApiUrl(),
                'request' => $params,
                'raw_response' => $ret_raw,
            ]);
            throw new ApiException($ret['message'] ?? 'BEpusdt create order failed');
        }

        // Return BEpusdt's full order data (trade_id, token address, actual_amount,
        // expiration_time, payment_url, ...) so the client can either redirect to
        // payment_url or render its own inline checkout (address/QR/countdown).
        return [
            'type' => 1,
            'data' => $ret['data'],
        ];
    }

    public function notify($params): array|bool
    {
        // BEpusdt POSTs a JSON body; prefer the raw body so the signed values
        // match byte-for-byte what BEpusdt signed (Laravel's input() is fine too,
        // but decoding the raw body avoids any middleware mutation).
        $raw = trim((string) request()->getContent());
        if ($raw !== '') {
            $decoded = json_decode($raw, true);
            if (is_array($decoded)) {
                $params = $decoded;
            }
        }

        $signature = $params['signature'] ?? '';
        unset($params['signature']);

        $expected = $this->sign($params);
        if (empty($signature) || !$this->hashEqual($signature, $expected)) {
            Log::warning('[BEpusdt] notify signature mismatch', [
                'received' => $signature,
                'expected' => $expected,
                'params' => $params,
            ]);
            return false;
        }

        if ((int) ($params['status'] ?? 0) !== self::STATUS_SUCCESS) {
            // Pending (1) / timeout (3): not an error, just not paid yet.
            Log::info('[BEpusdt] notify ignored, status not success', [
                'order_id' => $params['order_id'] ?? null,
                'status' => $params['status'] ?? null,
            ]);
            return false;
        }

        Log::info('[BEpusdt] notify verified, marking paid', [
            'order_id' => $params['order_id'] ?? null,
            'trade_id' => $params['trade_id'] ?? null,
        ]);

        return [
            'trade_no' => $params['order_id'],
            'callback_no' => $params['trade_id'],
        ];
    }

    private function getApiUrl(): string
    {
        return rtrim((string) $this->getConfig('api_url'), '/');
    }

    private function sign(array $params): string
    {
        $filtered = array_filter($params, fn ($value) => $value !== null && $value !== '');
        ksort($filtered);

        $pairs = [];
        foreach ($filtered as $key => $value) {
            $pairs[] = $key . '=' . $value;
        }

        return md5(implode('&', $pairs) . $this->getConfig('token'));
    }

    private function curlPost(string $url, string $body): string|false
    {
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_HEADER, false);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, 30);
        curl_setopt($ch, CURLOPT_POST, true);
        curl_setopt($ch, CURLOPT_POSTFIELDS, $body);
        curl_setopt($ch, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);
        $result = curl_exec($ch);
        if ($result === false) {
            \Log::error('BEpusdt curl error', [
                'url' => $url,
                'curl_errno' => curl_errno($ch),
                'curl_error' => curl_error($ch),
            ]);
        }
        curl_close($ch);
        return $result;
    }

    private function hashEqual(string $str1, string $str2): bool
    {
        return hash_equals($str2, $str1);
    }
}
