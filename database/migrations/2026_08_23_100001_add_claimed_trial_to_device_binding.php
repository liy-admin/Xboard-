<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('v2_device_binding', function (Blueprint $table) {
            $table->boolean('claimed_trial')->default(false)->after('type');
        });

        // 将现有 type=trial 的记录标记为 claimed_trial=true
        DB::table('v2_device_binding')
            ->where('type', 'trial')
            ->update(['claimed_trial' => true]);

        // type=real 但对应设备有过试用用户的也标记
        DB::table('v2_device_binding as b')
            ->where('b.type', 'real')
            ->whereExists(function ($query) {
                $query->select(DB::raw(1))
                    ->from('v2_user as u')
                    ->whereColumn('u.device_id', 'b.device_id')
                    ->where('u.email', 'like', 'device_%@device.local');
            })
            ->update(['claimed_trial' => true]);
    }

    public function down(): void
    {
        Schema::table('v2_device_binding', function (Blueprint $table) {
            $table->dropColumn('claimed_trial');
        });
    }
};
