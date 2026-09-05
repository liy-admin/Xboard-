<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::table('v2_order', function (Blueprint $table) {
            // 这笔订单如果触发了推荐人时长奖励，记录下来的具体秒数（订单周期 ×
            // 奖励比例，跟 commission_balance 是同一时机算出来的姊妹字段，只是
            // 单位是时长不是金额）。
            $table->unsignedBigInteger('referral_duration_seconds')->nullable()->after('commission_balance');
        });

        Schema::create('v2_referral_duration_log', function (Blueprint $table) {
            $table->integer('id', true);
            $table->integer('inviter_id');
            $table->integer('invitee_id');
            $table->char('trade_no', 36);
            $table->unsignedBigInteger('seconds_credited');
            $table->integer('created_at');
            $table->integer('updated_at');
            $table->index('inviter_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('v2_referral_duration_log');
        Schema::table('v2_order', function (Blueprint $table) {
            $table->dropColumn('referral_duration_seconds');
        });
    }
};
