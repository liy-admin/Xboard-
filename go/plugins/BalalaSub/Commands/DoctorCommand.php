<?php

namespace Plugin\BalalaSub\Commands;

use Illuminate\Console\Command;
use Plugin\BalalaSub\Support\EnvDoctor;

class DoctorCommand extends Command
{
    protected $signature = 'balala-sub:doctor {--full : 额外跑一次真实抓取，验证是否达到预期结果}';

    protected $description = 'BalalaSub 环境自检：检查 Python / ruyiPage / Firefox / 凭据，可选真实抓取';

    public function handle(): int
    {
        $this->info('BalalaSub 环境自检' . ($this->option('full') ? '（含真实抓取）' : ''));

        $result = EnvDoctor::run((bool) $this->option('full'));

        $rows = array_map(static function ($c) {
            return [
                $c['ok'] ? '✓' : '✗',
                $c['name'],
                $c['detail'],
            ];
        }, $result['checks']);
        $this->table(['', '检查项', '说明'], $rows);

        if ($result['ok']) {
            $this->info('✅ 环境正常，可以抓取');
            return self::SUCCESS;
        }
        $this->error('❌ 环境存在问题，请按上面「说明」处理');
        return self::FAILURE;
    }
}
