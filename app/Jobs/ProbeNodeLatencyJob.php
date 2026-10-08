<?php

namespace App\Jobs;

use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Artisan;

/**
 * 后台异步跑一次节点测速（check:server-latency）。
 *
 * 管理端「测速全部」触发 —— sing-box 精确模式下逐节点实跑较慢，放队列里跑，
 * 避免卡住 HTTP 请求。完成后结果写进 v2_server，管理端刷新即可看到。
 */
class ProbeNodeLatencyJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $timeout = 1800; // 最长 30 分钟
    public int $tries = 1;

    /** @param int[]|null $ids 只测指定节点；null 表示全部 */
    public function __construct(private ?array $ids = null)
    {
    }

    public function handle(): void
    {
        $params = [];
        if (!empty($this->ids)) {
            $params['--id'] = $this->ids;
        }
        Artisan::call('check:server-latency', $params);
    }
}
