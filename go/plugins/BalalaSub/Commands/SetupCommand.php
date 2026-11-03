<?php

namespace Plugin\BalalaSub\Commands;

use App\Models\Plugin;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Process;

class SetupCommand extends Command
{
    protected $signature = 'balala-sub:setup
        {--firefox : 同时下载 ruyiPage Firefox 运行时（较大，glibc 环境才可用）}
        {--python= : 覆盖使用的 Python 可执行文件}';

    protected $description = '安装 BalalaSub 抓取环境：pip 安装 ruyiPage（可选下载 Firefox 运行时）';

    public function handle(): int
    {
        $python = $this->resolvePython();
        $this->info("使用 Python: {$python}");

        $ok = true;
        $report = [];

        // pip 安装 ruyiPage：先常规，失败(多为 PEP668 externally-managed)再加 --break-system-packages 重试
        $this->line('▶ pip 安装 ruyiPage …');
        [$pok, $pmsg] = $this->pipInstallRuyipage($python);
        $report['pip 安装 ruyiPage'] = $pok ? 'ok' : $pmsg;
        if ($pok) {
            $this->info('  ✓ 完成');
        } else {
            $ok = false;
            $this->error('  ✗ ' . $pmsg);
        }

        if ($this->option('firefox')) {
            $this->line('▶ 下载 Firefox 运行时 …');
            try {
                $res = Process::timeout(1800)->run([$python, '-m', 'ruyipage', 'install']);
                if ($res->successful()) {
                    $report['下载 Firefox 运行时'] = 'ok';
                    $this->info('  ✓ 完成');
                } else {
                    $ok = false;
                    $report['下载 Firefox 运行时'] = trim($res->errorOutput() ?: $res->output());
                    $this->error('  ✗ ' . $report['下载 Firefox 运行时']);
                }
            } catch (\Throwable $e) {
                $ok = false;
                $report['下载 Firefox 运行时'] = '启动失败: ' . $e->getMessage();
                $this->error('  ✗ ' . $report['下载 Firefox 运行时']);
            }
        }

        try {
            Cache::put('plugin:balala_sub:setup_status', [
                'ok'     => $ok,
                'python' => $python,
                'report' => $report,
                'at'     => now()->toIso8601String(),
            ], 86400 * 7);
        } catch (\Throwable $e) {
            // 缓存不可用不影响安装结果
        }

        if ($ok) {
            $this->info('✅ 环境安装完成');
            if (!$this->option('firefox')) {
                $this->line('提示：Firefox 内核可在后台插件配置页「上传内核」，或加 --firefox 下载。');
            }
            return self::SUCCESS;
        }
        $this->error('❌ 环境安装存在失败项，请检查上面的日志');
        return self::FAILURE;
    }

    /**
     * pip 安装 ruyiPage，自动处理 PEP668（externally-managed-environment）。
     * @return array{0:bool,1:string}
     */
    protected function pipInstallRuyipage(string $python): array
    {
        $base = [$python, '-m', 'pip', 'install', '--upgrade', 'ruyiPage'];
        try {
            $res = Process::timeout(1800)->run($base);
        } catch (\Throwable $e) {
            return [false, '启动失败: ' . $e->getMessage()];
        }
        if ($res->successful()) {
            return [true, 'ok'];
        }
        $err = trim($res->errorOutput() ?: $res->output());
        // PEP668：系统 python 受保护，加 --break-system-packages 重试
        if (stripos($err, 'externally-managed') !== false || stripos($err, 'break-system-packages') !== false) {
            $this->line('  检测到 externally-managed，改用 --break-system-packages 重试…');
            try {
                $res2 = Process::timeout(1800)->run(array_merge($base, ['--break-system-packages']));
            } catch (\Throwable $e) {
                return [false, '重试启动失败: ' . $e->getMessage()];
            }
            if ($res2->successful()) {
                return [true, 'ok'];
            }
            return [false, trim($res2->errorOutput() ?: $res2->output())];
        }
        return [false, $err];
    }

    protected function resolvePython(): string
    {
        if ($opt = $this->option('python')) {
            return (string) $opt;
        }
        $plugin = Plugin::where('code', 'balala_sub')->first();
        $cfg = ($plugin && $plugin->config) ? (json_decode($plugin->config, true) ?: []) : [];
        $configured = trim((string) ($cfg['python_bin'] ?? ''));
        if ($configured !== '' && $configured !== 'python') {
            return $configured;
        }
        // 未配置或为默认 python 时，Linux 优先 python3
        return stripos(PHP_OS, 'WIN') === 0 ? 'python' : 'python3';
    }
}
