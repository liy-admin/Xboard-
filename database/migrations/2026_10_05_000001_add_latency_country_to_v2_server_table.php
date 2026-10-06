<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('v2_server', function (Blueprint $table) {
            // 面板主动测速/GeoIP 的结果，仅由 check:server-latency 命令写入。
            // country_code：ISO alpha-2（大写），GeoIP 解析得到，用于「每国只下发一个节点」。
            $table->char('country_code', 2)->nullable()->after('tags');
            // latency_ms：最近一次面板探测到的延迟（毫秒）；null 表示探测失败/未探测。
            // 不决定节点是否可用（可用性以 agent 上报的 is_online 为准），只用于同国家内排序选最快。
            $table->unsignedInteger('latency_ms')->nullable()->after('country_code');
            // last_latency_at：最近一次探测时间戳（Unix 秒），用于判断 latency_ms 是否已过期。
            $table->unsignedInteger('last_latency_at')->nullable()->after('latency_ms');
        });
    }

    public function down(): void
    {
        Schema::table('v2_server', function (Blueprint $table) {
            $table->dropColumn(['country_code', 'latency_ms', 'last_latency_at']);
        });
    }
};
