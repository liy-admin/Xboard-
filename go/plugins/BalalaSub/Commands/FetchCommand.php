<?php

namespace Plugin\BalalaSub\Commands;

use App\Models\Plugin;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Process;
use Plugin\BalalaSub\Plugin as BalalaPlugin;

class FetchCommand extends Command
{
    protected $signature = 'balala-sub:fetch
        {--email= : 覆盖配置中的登录邮箱}
        {--password= : 覆盖配置中的登录密码}
        {--python= : 覆盖配置中的 Python 可执行文件路径}
        {--firefox= : 覆盖配置中的 Firefox 可执行文件路径}
        {--headless= : 覆盖无头模式，1=无头 0=有界面}
        {--timeout= : 覆盖超时秒数（Firefox 冷启动较慢，建议 >=120）}
        {--no-cache : 不写入缓存，仅打印结果}
        {--json : 仅输出一行 JSON（便于被其它程序调用）}';

    protected $description = '使用 DrissionPage 自动登录 balala.io 并抓取订阅链接 URL';

    public function handle(): int
    {
        $config = $this->pluginConfig();

        $siteUrl  = rtrim((string) ($config['site_url'] ?? 'https://balala.io'), '/');
        $email    = (string) ($this->option('email') ?: ($config['email'] ?? ''));
        $password = (string) ($this->option('password') ?: ($config['password'] ?? ''));
        $cookie   = (string) ($config['cookie'] ?? '');
        $pythonBin = (string) ($this->option('python') ?: ($config['python_bin'] ?? 'python'));
        $chromePath = (string) ($this->option('firefox') ?: ($config['chrome_path'] ?? ''));
        // 解析 Firefox 路径：
        //  - 填了目录（如 /www/wwwroot/firefox/）→ 在目录内递归找 firefox 可执行文件
        //  - 为空 → 用后台上传解压到 storage 的 Firefox
        // 避免把目录或空值丢给 ruyiPage 而回退到它自带内核（某些环境一启动就崩，报“连不上端口”）。
        if ($chromePath !== '' && is_dir($chromePath)) {
            $chromePath = $this->findFirefoxIn($chromePath) ?? $chromePath;
        }
        if ($chromePath === '') {
            $chromePath = $this->findFirefoxIn(storage_path('app/balala_sub/firefox')) ?? '';
        }
        $headless = $this->option('headless') !== null
            ? ((string) $this->option('headless') === '1')
            : (bool) ($config['headless'] ?? true);
        $timeout  = (int) ($this->option('timeout') ?: ($config['timeout'] ?? 120));
        $cacheTtl = (int) ($config['cache_ttl'] ?? 86400);

        if ($cookie === '' && ($email === '' || $password === '')) {
            return $this->emitFail('缺少登录凭据：请填写「会话 Cookie」，或「登录邮箱 + 密码」(也可用 --email/--password 传入)。');
        }

        $script = __DIR__ . '/../python/fetch_sub.py';
        if (!is_file($script)) {
            $this->error("找不到抓取脚本：{$script}");
            return self::FAILURE;
        }

        // 密码等敏感信息通过环境变量传递，避免出现在进程命令行 / 日志里。
        // 只传这几个变量即可：Symfony Process 会自动把它们合并进父进程的完整环境
        // （PATH / APPDATA 等照常继承），不要手动合并 getenv()，否则会使 PATH 重复错乱。
        // 结果文件：避免在无控制台 / Octane-Swoole 环境下抓不到子进程 stdout
        $outDir = storage_path('app/balala_sub');
        if (!is_dir($outDir)) {
            @mkdir($outDir, 0775, true);
        }
        $outFile = $outDir . '/fetch_result_' . uniqid() . '.json';

        $env = [
            'BALALA_URL'         => $siteUrl,
            'BALALA_EMAIL'       => $email,
            'BALALA_PASSWORD'    => $password,
            'BALALA_COOKIE'      => $cookie,
            'BALALA_HEADLESS'    => $headless ? '1' : '0',
            'BALALA_TIMEOUT'     => (string) $timeout,
            'BALALA_CHROME_PATH' => $chromePath,
            'BALALA_OUTFILE'     => $outFile,
            'PYTHONIOENCODING'   => 'utf-8',
        ];

        if (!$this->option('json')) {
            $this->line("正在登录 <info>{$siteUrl}</info> 抓取订阅链接 …");
        }

        try {
            $result = Process::timeout($timeout + 30)
                ->env($env)
                ->run([$pythonBin, realpath($script) ?: $script]);
        } catch (\Throwable $e) {
            Log::error('[balala_sub] process error', ['error' => $e->getMessage()]);
            return $this->emitFail('调用 Python 失败：' . $e->getMessage());
        }

        $stdout = trim($result->output());
        $stderr = trim($result->errorOutput());

        // 优先读结果文件（可靠），再回退到 stdout。
        $fileContent = is_file($outFile) ? trim((string) @file_get_contents($outFile)) : '';
        @unlink($outFile);

        // 脚本最后只打印一行结果 JSON，但 headless 下 DrissionPage/Chrome 可能向 stdout
        // 输出路径提示、GPU 警告等噪声，因此从后往前取最后一行合法 JSON。
        $data = $this->extractJson($fileContent !== '' ? $fileContent : $stdout);
        if (!is_array($data)) {
            Log::error('[balala_sub] invalid output', ['stdout' => $stdout, 'stderr' => $stderr]);
            $reason = $stderr !== '' ? mb_substr($stderr, -300) : ($stdout !== '' ? mb_substr($stdout, -300) : '脚本无输出');
            return $this->emitFail('脚本未返回合法 JSON：' . $reason);
        }

        if (empty($data['success'])) {
            $message = $data['error'] ?? '未知错误';
            if ($this->option('json')) {
                $this->line(json_encode($data, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE));
            } else {
                $this->error('抓取失败：' . $message);
                if ($stderr !== '') {
                    $this->line("<comment>诊断：</comment>\n" . $stderr);
                }
            }
            return self::FAILURE;
        }

        $payload = [
            'subscribe_url' => $data['subscribe_url'] ?? null,
            'subscriptions' => $data['subscriptions'] ?? [],
            'source'        => $data['source'] ?? null,
            'fetched_at'    => now()->toIso8601String(),
        ];

        if (!$this->option('no-cache')) {
            Cache::put(BalalaPlugin::CACHE_KEY, $payload, $cacheTtl);
        }

        if ($this->option('json')) {
            $this->line(json_encode(['success' => true] + $payload, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE));
            return self::SUCCESS;
        }

        $this->info('✅ 抓取成功');
        $this->line('主订阅链接: <info>' . ($payload['subscribe_url'] ?? '(空)') . '</info>');
        if (!empty($payload['subscriptions'])) {
            $rows = array_map(static fn ($s) => [
                $s['client'] ?? '',
                $s['label'] ?? '',
                $s['url'] ?? '',
            ], $payload['subscriptions']);
            $this->table(['客户端', '名称', 'URL'], $rows);
        }
        $this->line('来源: ' . ($payload['source'] ?? '-'));
        if (!$this->option('no-cache')) {
            $this->line('已写入缓存键: ' . BalalaPlugin::CACHE_KEY);
        }

        return self::SUCCESS;
    }

    /**
     * 在给定目录内递归查找 firefox 可执行文件（优先 firefox.exe，其次 firefox）。
     */
    protected function findFirefoxIn(string $base): ?string
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
     * 统一失败出口：--json 模式输出 {success:false,error}，否则打印可读错误。
     * 保证被 Artisan::call 嵌套调用时也能拿到可解析的 JSON。
     */
    protected function emitFail(string $message): int
    {
        if ($this->option('json')) {
            $this->line(json_encode(['success' => false, 'error' => $message], JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE));
        } else {
            $this->error($message);
        }
        return self::FAILURE;
    }

    /**
     * 从可能含噪声的 stdout 中提取最后一行合法 JSON 对象。
     */
    protected function extractJson(string $stdout): ?array
    {
        // 先尝试整体解析（干净输出的情况）
        $whole = json_decode($stdout, true);
        if (is_array($whole)) {
            return $whole;
        }
        // 从后往前逐行找第一行能解析成对象的 JSON
        $lines = preg_split('/\r\n|\r|\n/', $stdout) ?: [];
        for ($i = count($lines) - 1; $i >= 0; $i--) {
            $line = trim($lines[$i]);
            if ($line === '' || $line[0] !== '{') {
                continue;
            }
            $decoded = json_decode($line, true);
            if (is_array($decoded) && array_key_exists('success', $decoded)) {
                return $decoded;
            }
        }
        return null;
    }

    /**
     * 直接从数据库读取插件配置，Octane 环境下稳妥。
     */
    protected function pluginConfig(): array
    {
        $plugin = Plugin::where('code', 'balala_sub')->first();
        if (!$plugin || !$plugin->config) {
            return [];
        }

        return json_decode($plugin->config, true) ?: [];
    }
}
