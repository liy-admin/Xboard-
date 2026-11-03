<?php

namespace Plugin\BalalaSub;

use App\Models\Plugin as PluginModel;
use App\Services\Plugin\AbstractPlugin;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Facades\Log;
use Plugin\BalalaSub\Commands\ImportCommand;

class Plugin extends AbstractPlugin
{
    /**
     * 缓存键：最近一次抓取到的订阅链接结果
     */
    public const CACHE_KEY = 'plugin:balala_sub:latest';

    /** 环境安装日志文件 */
    public const SETUP_LOG = 'logs/balala_sub_setup.log';

    /** 导入请求信号键（测速无可用时写入，导入命令消费） */
    public const REQUEST_KEY = 'plugin:balala_sub:import_requested';

    public function boot(): void
    {
        // 闭环：测速发现 0 个可用节点时，若开启「无可用自动重导」，写入导入信号。
        // 真正的导入由定时任务/cron 在 CLI 环境执行（可靠），不在此处同步跑浏览器。
        $this->listen('server.latency.no_available', function () {
            if (!$this->autoReimportOnEmpty()) {
                return;
            }
            try {
                Cache::put(self::REQUEST_KEY, now()->toIso8601String(), 3600);
                Log::info('[balala_sub] 测速无可用节点，已写入导入信号，待定时任务执行');
            } catch (\Throwable $e) {
                Log::warning('[balala_sub] 写入导入信号失败: ' . $e->getMessage());
            }
        });
    }

    /** 从 DB 读「无可用自动重导」开关（在任意进程上下文都可靠） */
    protected function autoReimportOnEmpty(): bool
    {
        try {
            $p = PluginModel::where('code', 'balala_sub')->first();
            $cfg = ($p && $p->config) ? (json_decode($p->config, true) ?: []) : [];
            return !empty($cfg['auto_reimport_on_empty']);
        } catch (\Throwable $e) {
            return false;
        }
    }

    /**
     * 定时任务：每小时触发一次导入命令，命令内部按配置的开关 + 间隔自行判断是否真正执行。
     * 依赖主程序的 schedule:run（宝塔/容器需有每分钟的 cron）。
     */
    public function schedule(Schedule $schedule): void
    {
        // 每分钟触发：命令内部按「导入信号 / 开关+间隔」自行判断是否真正执行。
        // 每分钟是为了让「测速无可用」写入的导入信号能在约 1 分钟内被消费。
        $schedule->command('balala-sub:import')
            ->everyMinute()
            ->onOneServer()
            ->withoutOverlapping(30);
    }

    /**
     * 插件安装时：后台自动安装抓取环境（pip 安装 ruyiPage）。
     *
     * 注意：install() 运行在 Web 请求的数据库事务内，必须快速返回，
     * 因此这里只「启动」一个后台进程去装，绝不同步等待（pip 下载较慢，
     * 同步会阻塞请求甚至导致安装事务回滚）。失败也不影响插件安装。
     */
    public function install(): void
    {
        $this->bootstrapEnv();
    }

    /**
     * 后台拉起环境安装：pip install ruyiPage。
     * Linux 优先 python3，回退 python；Windows 用 python。
     */
    public function bootstrapEnv(): void
    {
        try {
            $log = storage_path(self::SETUP_LOG);
            File::ensureDirectoryExists(dirname($log));

            if (stripos(PHP_OS, 'WIN') === 0) {
                $inner = 'python -m pip install --upgrade ruyiPage';
                @pclose(@popen('start /B cmd /c ' . escapeshellarg($inner . ' >> "' . $log . '" 2>&1'), 'r'));
            } else {
                // 先 python3，失败再试 python；全程后台、脱离当前请求
                $inner = '(python3 -m pip install --upgrade ruyiPage || python -m pip install --upgrade ruyiPage)';
                @shell_exec('nohup sh -c ' . escapeshellarg($inner) . ' >> ' . escapeshellarg($log) . ' 2>&1 &');
            }

            Log::info('[balala_sub] 已在后台启动环境安装 (pip install ruyiPage)，日志: ' . $log);
        } catch (\Throwable $e) {
            Log::warning('[balala_sub] 启动环境安装失败: ' . $e->getMessage());
        }
    }
}
