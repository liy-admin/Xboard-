<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('v2_user', function (Blueprint $table) {
            // 客户端最近一次上报「已同步到的节点集版本号」（sha1，40 字符），
            // 以及上报时间。用于统计一次节点变更覆盖到了多少用户，见
            // ServerController::synced。
            $table->string('node_revision', 40)->nullable()->after('enc_key');
            $table->unsignedInteger('node_revision_at')->nullable()->after('node_revision');
            // 统计查询就是按版本号做 COUNT/GROUP BY，加索引避免全表扫描。
            $table->index('node_revision');
        });
    }

    public function down(): void
    {
        Schema::table('v2_user', function (Blueprint $table) {
            $table->dropIndex(['node_revision']);
            $table->dropColumn(['node_revision', 'node_revision_at']);
        });
    }
};
