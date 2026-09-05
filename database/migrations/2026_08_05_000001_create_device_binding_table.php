<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::create('v2_device_binding', function (Blueprint $table) {
            $table->integer('id', true);
            // 设备稳定标识（Android ID / iOS Keychain UUID 等，见 Flutter 端
            // resolveDeviceId()），跟登录用的 device_id 是同一个值。
            $table->string('device_id', 191)->unique();
            $table->integer('user_id');
            // trial：设备码首次出现时自动创建的体验影子账号；
            // real：设备曾用于一次真实邮箱登录/注册，绑定后这台设备不再发新体验时长。
            $table->string('type', 16)->default('trial');
            $table->integer('created_at');
            $table->integer('updated_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('v2_device_binding');
    }
};
