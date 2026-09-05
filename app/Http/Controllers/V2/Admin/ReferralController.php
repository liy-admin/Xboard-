<?php

namespace App\Http\Controllers\V2\Admin;

use App\Http\Controllers\Controller;
use App\Models\ReferralDurationLog;
use App\Models\User;
use Illuminate\Http\Request;

/**
 * "推荐有奖"（时长奖励）的管理端查看入口。跟老的邀请码+金额佣金系统
 * （后台已有的"用户管理"里能看 invite_user_id / commission_balance）是
 * 独立的两条数据，所以单独给一个入口，而不是塞进现有页面。
 */
class ReferralController extends Controller
{
    /**
     * 按推荐人汇总：谁推荐了几个人、一共薅到多少小时，推荐人数最多的排前面。
     */
    public function summary(Request $request)
    {
        $rows = ReferralDurationLog::selectRaw(
            'inviter_id, COUNT(DISTINCT invitee_id) as invite_count, SUM(seconds_credited) as total_seconds, MAX(created_at) as last_at'
        )
            ->groupBy('inviter_id')
            ->orderByDesc('invite_count')
            ->limit(200)
            ->get();

        $inviters = User::whereIn('id', $rows->pluck('inviter_id'))
            ->get(['id', 'email'])
            ->keyBy('id');

        $data = $rows->map(function ($row) use ($inviters) {
            return [
                'inviter_id' => $row->inviter_id,
                'inviter_email' => $inviters[$row->inviter_id]->email ?? null,
                'invite_count' => (int) $row->invite_count,
                'total_hours' => round($row->total_seconds / 3600, 1),
                'last_at' => $row->last_at,
            ];
        });

        return $this->success($data);
    }

    /**
     * 明细流水：具体哪个订单触发了谁给谁加了多少时长，按时间倒序，供核对
     * 用（比如用户投诉"说好的奖励没到账"时用来查证）。
     */
    public function records(Request $request)
    {
        $query = ReferralDurationLog::query()->orderByDesc('id');

        if ($inviterId = $request->input('inviter_id')) {
            $query->where('inviter_id', $inviterId);
        }

        $perPage = min(100, (int) $request->input('page_size', 20));
        $logs = $query->paginate($perPage);

        $userIds = collect($logs->items())
            ->flatMap(fn($log) => [$log->inviter_id, $log->invitee_id])
            ->unique();
        $users = User::whereIn('id', $userIds)->get(['id', 'email'])->keyBy('id');

        $data = collect($logs->items())->map(function ($log) use ($users) {
            return [
                'id' => $log->id,
                'inviter_id' => $log->inviter_id,
                'inviter_email' => $users[$log->inviter_id]->email ?? null,
                'invitee_id' => $log->invitee_id,
                'invitee_email' => $users[$log->invitee_id]->email ?? null,
                'trade_no' => $log->trade_no,
                'hours_credited' => round($log->seconds_credited / 3600, 1),
                'created_at' => $log->created_at,
            ];
        });

        return $this->success([
            'data' => $data,
            'total' => $logs->total(),
            'current_page' => $logs->currentPage(),
            'last_page' => $logs->lastPage(),
        ]);
    }
}
