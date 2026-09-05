<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * 设备码绑定记录。
 *
 * - claimed_trial：该设备是否已领取过试用套餐（一旦为 true 永不重置）
 * - user_id：当前绑定的用户（试用用户或真实用户，退出后可能被重置）
 */
class DeviceBinding extends Model
{
    protected $table = 'v2_device_binding';
    protected $dateFormat = 'U';
    protected $guarded = ['id'];
    protected $casts = [
        'created_at' => 'timestamp',
        'updated_at' => 'timestamp',
        'claimed_trial' => 'boolean',
    ];
}
