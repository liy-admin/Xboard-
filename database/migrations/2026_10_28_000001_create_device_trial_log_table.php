<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        // 试用领取记录：每次给设备发放试用套餐（注册发放 / 设备体验账号）记一行，
        // 供后台「设备管理」查看领取历史。Go 版在 go/internal/schema/schema.sql 中同步建表。
        Schema::create('v2_device_trial_log', function (Blueprint $table) {
            $table->integer('id', true);
            $table->string('device_id', 191)->index();
            $table->integer('user_id')->index();
            $table->string('email', 64)->default('')->comment('领取时的账号邮箱（账号注销后会被匿名化，这里保留快照）');
            $table->string('source', 16)->comment('register：注册发放；device_trial：设备体验账号');
            $table->integer('plan_id')->nullable();
            $table->integer('expired_at')->nullable();
            $table->string('ip', 64)->nullable();
            $table->integer('created_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('v2_device_trial_log');
    }
};
