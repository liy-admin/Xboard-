<?php

namespace App\Console;

use App\Services\Plugin\PluginManager;
use App\Utils\CacheKey;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Foundation\Console\Kernel as ConsoleKernel;
use Illuminate\Support\Facades\Cache;

class Kernel extends ConsoleKernel
{
    /**
     * The Artisan commands provided by your application.
     *
     * @var array
     */
    protected $commands = [
        //
    ];

    /**
     * Define the application's command schedule.
     *
     * @param \Illuminate\Console\Scheduling\Schedule $schedule
     * @return void
     */
    protected function schedule(Schedule $schedule): void
    {
        Cache::put(CacheKey::get('SCHEDULE_LAST_CHECK_AT', null), time());
        // v2board
        $schedule->command('xboard:statistics')->dailyAt('0:10')->onOneServer();
        // check
        $schedule->command('check:order')->everyMinute()->onOneServer()->withoutOverlapping(5);
        $schedule->command('check:commission')->everyMinute()->onOneServer()->withoutOverlapping(5);
        $schedule->command('check:ticket')->everyMinute()->onOneServer()->withoutOverlapping(5);
        $schedule->command('check:traffic-exceeded')->everyMinute()->onOneServer()->withoutOverlapping(10)->runInBackground();
        // reset
        $schedule->command('reset:traffic')->everyMinute()->onOneServer()->withoutOverlapping(10);
        $schedule->command('reset:log')->daily()->onOneServer();
        // send
        $schedule->command('send:remindMail', ['--force'])->dailyAt('11:30')->onOneServer();
        // horizon metrics
        $schedule->command('horizon:snapshot')->everyFiveMinutes()->onOneServer();
        // cleanup stale online_count (GC for Redis TTL expiration)
        $schedule->command('cleanup:online-status')->everyFiveMinutes()->onOneServer();
        // 节点测速 + 国家解析；间隔可在后台配置（server_latency_probe_interval，分钟），
        // 命令内部受 server_latency_probe_enable 控制
        $probeInterval = (int) admin_setting('server_latency_probe_interval', 3);
        $probeInterval = max(1, min($probeInterval, 1440));
        $probeCron = $probeInterval >= 60
            ? ($probeInterval % 60 === 0 ? '0 */' . intdiv($probeInterval, 60) . ' * * *' : '0 * * * *')
            : '*/' . $probeInterval . ' * * * *';
        $schedule->command('check:server-latency')
            ->cron($probeCron)
            ->onOneServer()
            ->withoutOverlapping(max(10, $probeInterval)) // 长间隔/慢探测时锁别过早过期
            ->runInBackground();
        // backup Timing
        // if (env('ENABLE_AUTO_BACKUP_AND_UPDATE', false)) {
        //     $schedule->command('backup:database', ['true'])->daily()->onOneServer();
        // }
        app(PluginManager::class)->registerPluginSchedules($schedule);

    }

    /**
     * Register the commands for the application.
     *
     * @return void
     */
    protected function commands()
    {
        $this->load(__DIR__ . '/Commands');

        try {
            app(PluginManager::class)->initializeEnabledPlugins();
        } catch (\Exception $e) {
        }
        require base_path('routes/console.php');
    }
}
