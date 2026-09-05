<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * 推荐时长奖励流水：好友订单触发返佣的同一时刻，如果该笔订单也算出了
 * referral_duration_seconds，就在这里记一条，用于"成功推荐人数"（distinct
 * invitee_id）和"累计获得时长"（sum(seconds_credited)）这两个统计。
 */
class ReferralDurationLog extends Model
{
    protected $table = 'v2_referral_duration_log';
    protected $dateFormat = 'U';
    protected $guarded = ['id'];
    protected $casts = [
        'created_at' => 'timestamp',
        'updated_at' => 'timestamp',
    ];
}
