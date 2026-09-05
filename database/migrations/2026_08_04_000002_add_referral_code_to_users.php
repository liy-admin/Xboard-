<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::table('v2_user', function (Blueprint $table) {
            // 6 位随机数字推荐码，跟账号自增 id 脱钩——不暴露"你是第几个注册的"，
            // 也比纯自增 id 更适合当成公开分享的标识。首次访问
            // /referral/info 时才按需生成，见 ReferralController::info()。
            $table->char('referral_code', 6)->nullable()->unique()->after('invite_user_id');
        });
    }

    public function down(): void
    {
        Schema::table('v2_user', function (Blueprint $table) {
            $table->dropColumn('referral_code');
        });
    }
};
