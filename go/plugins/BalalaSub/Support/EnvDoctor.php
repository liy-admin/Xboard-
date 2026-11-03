<?php

namespace Plugin\BalalaSub\Support;

use App\Models\Plugin;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Process;

/**
 * BalalaSub 环境自检：逐项检查抓取所需环境，并可选做一次真实抓取验证是否达到预期结果。
 */
class EnvDoctor
{
    /**
     * @return array{ok:bool, checks:array<int,array{name:string,ok:bool,detail:string}>}
     */
    public static function run(bool $full = false): array
    {
        $cfg = self::config();
        $checks = [];

        // 1) Python
        $python = self::resolvePython($cfg);
        [$pyOk, $pyDetail] = self::probe([$python, '--version']);
        $checks[] = ['name' => 'Python', 'ok' => $pyOk, 'detail' => $pyOk ? trim($pyDetail) . " ({$python})" : "无法运行 {$python}：{$pyDetail}"];

        // 2) ruyiPage
        if ($pyOk) {
            [$rpOk, $rpDetail] = self::probe([$python, '-c', 'import ruyipage,sys; sys.stdout.write(getattr(ruyipage,"__version__","?"))']);
            $checks[] = ['name' => 'ruyiPage', 'ok' => $rpOk, 'detail' => $rpOk ? ('已安装 ' . trim($rpDetail)) : ('未安装，请运行 php artisan balala-sub:setup')];
        } else {
            $checks[] = ['name' => 'ruyiPage', 'ok' => false, 'detail' => 'Python 不可用，跳过'];
        }

        // 3) Firefox 路径存在
        $fx = self::resolveFirefox($cfg);
        $fxExists = $fx !== '' && is_file($fx);
        $checks[] = ['name' => 'Firefox 路径', 'ok' => $fxExists, 'detail' => $fxExists ? $fx : '未找到 Firefox，可在配置上传内核或填写路径'];

        // 4) Firefox 可启动（能跑出版本号，否则多半缺运行库）
        //    full 模式下跳过：真实抓取会启动 Firefox，提前再启动一次会抢占端口/配置导致冲突
        if (!$full) {
            if ($fxExists) {
                [$fxOk, $fxDetail] = self::probe([$fx, '--version']);
                $checks[] = ['name' => 'Firefox 可启动', 'ok' => $fxOk, 'detail' => $fxOk ? trim($fxDetail) : ('启动失败（多半缺运行库）：' . trim($fxDetail))];
            } else {
                $checks[] = ['name' => 'Firefox 可启动', 'ok' => false, 'detail' => '无 Firefox，跳过'];
            }
        }

        // 5) 登录凭据
        $hasCookie = trim((string) ($cfg['cookie'] ?? '')) !== '';
        $hasPwd = trim((string) ($cfg['email'] ?? '')) !== '' && trim((string) ($cfg['password'] ?? '')) !== '';
        $credOk = $hasCookie || $hasPwd;
        $checks[] = ['name' => '登录凭据', 'ok' => $credOk, 'detail' => $credOk ? ($hasCookie ? '已配置 Cookie' : '已配置邮箱+密码') : '缺少邮箱+密码或 Cookie'];

        // 6) 真实抓取（可选，较慢）
        if ($full) {
            try {
                Artisan::call('balala-sub:fetch', ['--json' => true, '--no-cache' => true]);
                $raw = trim(Artisan::output());
                $out = json_decode($raw, true);
                if (is_array($out) && !empty($out['success'])) {
                    $n = count($out['subscriptions'] ?? []);
                    $checks[] = ['name' => '真实抓取', 'ok' => true, 'detail' => "成功，拿到 {$n} 条订阅链接"];
                } else {
                    // 尽量带出真实原因：JSON 里的 error，否则截取原始输出末尾
                    $msg = is_array($out)
                        ? ($out['error'] ?? '未知错误')
                        : ($raw !== '' ? mb_substr($raw, -300) : '脚本无输出');
                    $checks[] = ['name' => '真实抓取', 'ok' => false, 'detail' => '失败：' . $msg];
                }
            } catch (\Throwable $e) {
                $checks[] = ['name' => '真实抓取', 'ok' => false, 'detail' => '异常：' . $e->getMessage()];
            }
        }

        $ok = true;
        foreach ($checks as $c) {
            if (!$c['ok']) {
                $ok = false;
            }
        }

        return ['ok' => $ok, 'checks' => $checks];
    }

    protected static function config(): array
    {
        $plugin = Plugin::where('code', 'balala_sub')->first();
        return ($plugin && $plugin->config) ? (json_decode($plugin->config, true) ?: []) : [];
    }

    protected static function resolvePython(array $cfg): string
    {
        $p = trim((string) ($cfg['python_bin'] ?? ''));
        if ($p !== '' && $p !== 'python') {
            return $p;
        }
        return stripos(PHP_OS, 'WIN') === 0 ? 'python' : 'python3';
    }

    protected static function resolveFirefox(array $cfg): string
    {
        $fx = trim((string) ($cfg['chrome_path'] ?? ''));
        if ($fx !== '' && is_dir($fx)) {
            $fx = self::findFirefoxIn($fx) ?? $fx;
        }
        if ($fx === '') {
            $fx = self::findFirefoxIn(storage_path('app/balala_sub/firefox')) ?? '';
        }
        return $fx;
    }

    protected static function findFirefoxIn(string $base): ?string
    {
        if (!is_dir($base)) {
            return null;
        }
        foreach (['firefox.exe', 'firefox'] as $name) {
            try {
                $it = new \RecursiveIteratorIterator(
                    new \RecursiveDirectoryIterator($base, \FilesystemIterator::SKIP_DOTS)
                );
                foreach ($it as $f) {
                    if ($f->isFile() && strtolower($f->getFilename()) === $name) {
                        return $f->getRealPath();
                    }
                }
            } catch (\Throwable $e) {
                // ignore
            }
        }
        return null;
    }

    /**
     * 运行一个命令，返回 [成功?, 输出(stdout或stderr)]。
     *
     * @param array<int,string> $cmd
     * @return array{0:bool,1:string}
     */
    protected static function probe(array $cmd): array
    {
        try {
            $res = Process::timeout(30)->run($cmd);
            $text = trim($res->output()) !== '' ? trim($res->output()) : trim($res->errorOutput());
            return [$res->successful(), $text];
        } catch (\Throwable $e) {
            return [false, $e->getMessage()];
        }
    }
}
