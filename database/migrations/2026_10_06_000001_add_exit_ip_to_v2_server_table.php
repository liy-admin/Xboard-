<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('v2_server', function (Blueprint $table) {
            // sing-box 精确探测模式下，通过该节点实际请求 IP 回显得到的「真实出口 IP」。
            // 用于后台节点管理展示（中转节点入口≠出口时尤其有用）；tcp 模式为空。
            $table->string('exit_ip', 45)->nullable()->after('latency_ms');
        });
    }

    public function down(): void
    {
        Schema::table('v2_server', function (Blueprint $table) {
            $table->dropColumn('exit_ip');
        });
    }
};
