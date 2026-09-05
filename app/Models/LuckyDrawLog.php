<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * "免费领会员"抽奖流水：每次尝试（无论中奖与否）都会记一条，
 * 用于两件事——(1) 驱动该用户下一次可尝试的冷却时间（取该用户最近一条
 * 记录的 created_at），(2) 统计"今日中奖人数"（is_win=1 且 created_at
 * 落在今天）与"累计发放时长"（sum(hours_credited)）。
 */
class LuckyDrawLog extends Model
{
    protected $table = 'v2_lucky_draw_log';
    protected $dateFormat = 'U';
    protected $guarded = ['id'];
    protected $casts = [
        'is_win' => 'boolean',
        'created_at' => 'timestamp',
        'updated_at' => 'timestamp',
    ];
}
