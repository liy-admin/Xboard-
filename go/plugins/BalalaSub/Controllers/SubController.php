<?php

namespace Plugin\BalalaSub\Controllers;

use App\Http\Controllers\PluginController;
use App\Models\Plugin;
use App\Services\Plugin\PluginConfigService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\File;
use Plugin\BalalaSub\Plugin as BalalaPlugin;
use Plugin\BalalaSub\Support\EnvDoctor;

class SubController extends PluginController
{
    /**
     * 读取最近一次抓取到的订阅链接（来自缓存）。
     * GET /api/v1/balala-sub/latest
     */
    public function latest(Request $request)
    {
        if ($error = $this->beforePluginAction()) {
            return $error[1];
        }

        $data = Cache::get(BalalaPlugin::CACHE_KEY);
        if (!$data) {
            return $this->fail([404, '暂无缓存，请先执行 php artisan balala-sub:fetch']);
        }

        return $this->success($data);
    }

    /**
     * 触发一次抓取并返回结果。
     * POST /api/v1/balala-sub/refresh
     */
    public function refresh(Request $request)
    {
        if ($error = $this->beforePluginAction()) {
            return $error[1];
        }

        $code = Artisan::call('balala-sub:fetch', ['--json' => true]);
        $output = trim(Artisan::output());
        $result = json_decode($output, true);

        if ($code !== 0 || !is_array($result) || empty($result['success'])) {
            return $this->fail([500, $result['error'] ?? '抓取失败，详见服务端日志']);
        }

        return $this->success($result);
    }

    /**
     * 环境自检：检查 Python / ruyiPage / Firefox / 凭据。
     * GET /api/v1/balala-sub/doctor?full=1  （full=1 会额外真实抓取一次，较慢）
     */
    public function doctor(Request $request)
    {
        if ($error = $this->beforePluginAction()) {
            return $error[1];
        }
        $full = (bool) $request->input('full', false);
        return $this->success(EnvDoctor::run($full));
    }

    /**
     * 上传 Firefox 内核 zip，解压并定位可执行文件，写入插件配置 chrome_path。
     * POST /api/v1/balala-sub/upload-firefox   (multipart: file)
     */
    public function uploadFirefox(Request $request)
    {
        if ($error = $this->beforePluginAction()) {
            return $error[1];
        }

        $request->validate([
            // Firefox 运行时 zip 一般 70~120MB，放宽到 512MB
            'file' => 'required|file|max:524288',
        ]);

        $file = $request->file('file');
        $name = strtolower($file->getClientOriginalName());
        $isZip = str_ends_with($name, '.zip');
        $isTar = (bool) preg_match('/\.(tar\.(xz|gz|bz2)|tgz|tar)$/', $name);
        if (!$isZip && !$isTar) {
            return $this->fail([400, '请上传 .zip 或 .tar.xz/.tar.gz 格式的 Firefox 内核包']);
        }
        if ($isZip && !class_exists(\ZipArchive::class)) {
            return $this->fail([500, '服务器未启用 PHP zip 扩展，无法解压']);
        }

        $targetDir = storage_path('app/balala_sub/firefox');

        try {
            // 清空旧的，避免残留
            if (File::exists($targetDir)) {
                File::deleteDirectory($targetDir);
            }
            File::ensureDirectoryExists($targetDir);

            if ($isZip) {
                $zip = new \ZipArchive();
                if ($zip->open($file->getRealPath()) !== true) {
                    return $this->fail([400, '压缩包无法打开，可能已损坏']);
                }
                $zip->extractTo($targetDir);
                $zip->close();
            } else {
                // tar.xz / tar.gz 等交给系统 tar（Linux 容器自带；xz 需 xz-utils）
                $result = \Illuminate\Support\Facades\Process::timeout(300)
                    ->run(['tar', '-xf', $file->getRealPath(), '-C', $targetDir]);
                if (!$result->successful()) {
                    return $this->fail([500, 'tar 解压失败：' . trim($result->errorOutput() ?: $result->output())]);
                }
            }
        } catch (\Throwable $e) {
            return $this->fail([500, '解压失败：' . $e->getMessage()]);
        }

        // 定位可执行文件：优先 firefox.exe（Windows），其次 firefox（Linux）
        $exe = $this->locateFirefox($targetDir);
        if (!$exe) {
            return $this->fail([400, '解压后未找到 firefox.exe / firefox 可执行文件']);
        }

        // 写入插件配置 chrome_path（读取扁平配置→合并→回写）
        try {
            $plugin = Plugin::where('code', 'balala_sub')->first();
            $current = ($plugin && $plugin->config) ? (json_decode($plugin->config, true) ?: []) : [];
            $current['chrome_path'] = $exe;
            app(PluginConfigService::class)->updateConfig('balala_sub', $current);
            Cache::forget('plugin_config_balala_sub');
        } catch (\Throwable $e) {
            return $this->fail([500, '保存配置失败：' . $e->getMessage()]);
        }

        return $this->success([
            'path'    => $exe,
            'message' => 'Firefox 内核已上传并设为抓取使用',
        ]);
    }

    /**
     * 在目录下递归查找 firefox 可执行文件。
     */
    private function locateFirefox(string $dir): ?string
    {
        $prefer = ['firefox.exe', 'firefox'];
        foreach ($prefer as $name) {
            $found = $this->findFileByName($dir, $name);
            if ($found) {
                return $found;
            }
        }
        return null;
    }

    private function findFileByName(string $dir, string $name): ?string
    {
        $it = new \RecursiveIteratorIterator(
            new \RecursiveDirectoryIterator($dir, \FilesystemIterator::SKIP_DOTS)
        );
        foreach ($it as $f) {
            if ($f->isFile() && strtolower($f->getFilename()) === strtolower($name)) {
                return $f->getRealPath();
            }
        }
        return null;
    }
}
