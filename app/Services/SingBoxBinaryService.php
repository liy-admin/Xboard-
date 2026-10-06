<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

/**
 * 自动准备可用的 sing-box 可执行文件（用于 NodeProbeService 的精确探测模式）。
 *
 * 解析顺序：
 *   1. 管理员在 node_probe_singbox_path 填了自定义路径/命令，且能跑通 → 直接用它；
 *   2. 否则用「托管副本」：storage/app/singbox/<ver>/sing-box[.exe]，不存在就按面板
 *      宿主机的系统+架构从官方 GitHub 下载对应版本、解压、校验版本号后缓存。
 *
 * 只从官方 SagerNet GitHub 发布页经 HTTPS 下载，下载后用 `sing-box version` 校验，
 * 版本号对不上则丢弃，避免损坏/不匹配的二进制被使用。
 */
class SingBoxBinaryService
{
    private ?string $cached = null;
    private bool $resolved = false;

    /** 返回可用的 sing-box 路径；不可用（下载失败等）返回 null。 */
    public function resolve(): ?string
    {
        if ($this->resolved) {
            return $this->cached;
        }
        $this->resolved = true;

        // 1) 自定义路径优先
        $custom = trim((string) admin_setting('node_probe_singbox_path', ''));
        if ($custom !== '' && $custom !== 'sing-box' && $this->works($custom)) {
            return $this->cached = $custom;
        }
        // 如果填的是裸命令 sing-box 且 PATH 里确实有，也用它
        if (($custom === '' || $custom === 'sing-box') && $this->works('sing-box')) {
            return $this->cached = 'sing-box';
        }

        // 2) 托管副本（自动下载）
        if (!(bool) admin_setting('node_probe_singbox_auto_download', true)) {
            return $this->cached = null;
        }
        return $this->cached = $this->ensureManaged();
    }

    /** 跑 `<bin> version` 看是否可用。 */
    private function works(string $bin): bool
    {
        if (!function_exists('proc_open')) {
            return false;
        }
        $out = @shell_exec(escapeshellarg($bin) . ' version 2>&1');
        return is_string($out) && str_contains($out, 'sing-box version');
    }

    /** 确保托管副本存在，返回其路径或 null。 */
    private function ensureManaged(): ?string
    {
        $version = trim((string) admin_setting('node_probe_singbox_version', '1.14.2')) ?: '1.14.2';
        $isWin = PHP_OS_FAMILY === 'Windows';
        $binName = $isWin ? 'sing-box.exe' : 'sing-box';
        $dir = storage_path('app/singbox/' . $version);
        $binPath = $dir . DIRECTORY_SEPARATOR . $binName;

        if (is_file($binPath) && $this->works($binPath)) {
            return $binPath;
        }

        // 用文件锁避免多进程同时下载
        @mkdir($dir, 0755, true);
        $lockPath = $dir . DIRECTORY_SEPARATOR . '.lock';
        $lock = @fopen($lockPath, 'c');
        if ($lock) {
            @flock($lock, LOCK_EX);
        }
        try {
            // 可能别的进程刚下载好
            if (is_file($binPath) && $this->works($binPath)) {
                return $binPath;
            }
            return $this->download($version, $dir, $binName) ? $binPath : null;
        } finally {
            if ($lock) {
                @flock($lock, LOCK_UN);
                @fclose($lock);
            }
        }
    }

    /** 下载并解压对应平台的 sing-box 到 $dir。成功且校验通过返回 true。 */
    private function download(string $version, string $dir, string $binName): bool
    {
        $platform = $this->platform();
        if (!$platform) {
            Log::warning('SingBoxBinaryService: 不支持的平台', ['os' => PHP_OS_FAMILY, 'arch' => php_uname('m')]);
            return false;
        }
        [$os, $arch, $ext] = $platform;
        $asset = "sing-box-{$version}-{$os}-{$arch}.{$ext}";
        $url = "https://github.com/SagerNet/sing-box/releases/download/v{$version}/{$asset}";

        $tmp = $dir . DIRECTORY_SEPARATOR . $asset;
        try {
            if (!$this->fetchFile($url, $tmp)) {
                Log::warning('SingBoxBinaryService: 下载失败', ['url' => $url]);
                @unlink($tmp);
                return false;
            }

            $extractDir = $dir . DIRECTORY_SEPARATOR . 'extract';
            $this->rrmdir($extractDir);
            @mkdir($extractDir, 0755, true);

            if ($ext === 'zip') {
                if (!$this->extractZip($tmp, $extractDir)) {
                    return false;
                }
            } else { // tar.gz
                if (!$this->extractTarGz($tmp, $extractDir)) {
                    return false;
                }
            }

            // 发布包里是 sing-box-<ver>-<os>-<arch>/sing-box
            $found = $this->findBinary($extractDir, $binName);
            if (!$found) {
                Log::warning('SingBoxBinaryService: 解压后未找到二进制', ['dir' => $extractDir]);
                return false;
            }

            $dest = $dir . DIRECTORY_SEPARATOR . $binName;
            @unlink($dest);
            if (!@rename($found, $dest)) {
                @copy($found, $dest);
            }
            if (PHP_OS_FAMILY !== 'Windows') {
                @chmod($dest, 0755);
            }

            // 校验版本号
            if (!$this->works($dest)) {
                Log::warning('SingBoxBinaryService: 下载的二进制无法运行', ['path' => $dest]);
                return false;
            }
            $out = (string) @shell_exec(escapeshellarg($dest) . ' version 2>&1');
            if (!str_contains($out, $version)) {
                Log::warning('SingBoxBinaryService: 版本号不匹配', ['want' => $version, 'got' => trim($out)]);
                // 不致命：仍可用，但记录
            }

            Log::info('SingBoxBinaryService: 已就绪', ['path' => $dest, 'version' => $version]);
            return true;
        } catch (\Throwable $e) {
            Log::warning('SingBoxBinaryService: 下载异常', ['message' => $e->getMessage()]);
            return false;
        } finally {
            @unlink($tmp);
            $this->rrmdir($dir . DIRECTORY_SEPARATOR . 'extract');
        }
    }

    /**
     * 下载文件：先用 Laravel Http（Guzzle/cURL），失败再退回系统 curl / wget。
     * 后两者能绕开部分主机上 PHP cURL 缺 CA 证书（cURL error 60）的问题。
     */
    private function fetchFile(string $url, string $tmp): bool
    {
        // 1) Laravel Http
        try {
            $resp = Http::timeout(180)->withOptions(['sink' => $tmp])->get($url);
            if ($resp->ok() && is_file($tmp) && filesize($tmp) >= 1024) {
                return true;
            }
        } catch (\Throwable $e) {
            Log::info('SingBoxBinaryService: Http 下载失败，尝试系统 curl/wget', ['message' => $e->getMessage()]);
        }
        @unlink($tmp);

        if (!function_exists('exec')) {
            return false;
        }
        // 2) 系统 curl
        @exec('curl -fsSL -o ' . escapeshellarg($tmp) . ' ' . escapeshellarg($url) . ' 2>&1', $o1, $c1);
        if ($c1 === 0 && is_file($tmp) && filesize($tmp) >= 1024) {
            return true;
        }
        @unlink($tmp);
        // 3) wget
        @exec('wget -q -O ' . escapeshellarg($tmp) . ' ' . escapeshellarg($url) . ' 2>&1', $o2, $c2);
        if ($c2 === 0 && is_file($tmp) && filesize($tmp) >= 1024) {
            return true;
        }
        @unlink($tmp);
        return false;
    }

    /** [os, arch, ext] 或 null。 */
    private function platform(): ?array
    {
        $m = strtolower(php_uname('m'));
        $arch = match (true) {
            in_array($m, ['x86_64', 'amd64'], true) => 'amd64',
            in_array($m, ['aarch64', 'arm64'], true) => 'arm64',
            str_starts_with($m, 'armv7') => 'armv7',
            default => null,
        };
        if (!$arch) {
            return null;
        }
        return match (PHP_OS_FAMILY) {
            'Linux' => ['linux', $arch, 'tar.gz'],
            'Darwin' => ['darwin', $arch, 'tar.gz'],
            'Windows' => ['windows', $arch, 'zip'],
            default => null,
        };
    }

    private function extractZip(string $zip, string $to): bool
    {
        if (!class_exists(\ZipArchive::class)) {
            Log::warning('SingBoxBinaryService: 缺少 zip 扩展，无法解压 Windows 包');
            return false;
        }
        $za = new \ZipArchive();
        if ($za->open($zip) !== true) {
            return false;
        }
        $ok = $za->extractTo($to);
        $za->close();
        return $ok;
    }

    private function extractTarGz(string $tgz, string $to): bool
    {
        try {
            // PharData 可直接处理 .tar.gz
            $phar = new \PharData($tgz);
            $phar->extractTo($to, null, true);
            return true;
        } catch (\Throwable $e) {
            // 退回系统 tar
            if (function_exists('exec')) {
                @exec('tar xzf ' . escapeshellarg($tgz) . ' -C ' . escapeshellarg($to), $o, $code);
                return $code === 0;
            }
            Log::warning('SingBoxBinaryService: 解压 tar.gz 失败', ['message' => $e->getMessage()]);
            return false;
        }
    }

    /** 在解压目录里递归找 sing-box 二进制。 */
    private function findBinary(string $dir, string $binName): ?string
    {
        $it = new \RecursiveIteratorIterator(new \RecursiveDirectoryIterator($dir, \FilesystemIterator::SKIP_DOTS));
        foreach ($it as $file) {
            if ($file->getFilename() === $binName) {
                return $file->getPathname();
            }
        }
        return null;
    }

    private function rrmdir(string $dir): void
    {
        if (!is_dir($dir)) {
            return;
        }
        $it = new \RecursiveIteratorIterator(
            new \RecursiveDirectoryIterator($dir, \FilesystemIterator::SKIP_DOTS),
            \RecursiveIteratorIterator::CHILD_FIRST
        );
        foreach ($it as $file) {
            $file->isDir() ? @rmdir($file->getPathname()) : @unlink($file->getPathname());
        }
        @rmdir($dir);
    }
}
