<?php

namespace App\Http\Controllers\V1\User;

use App\Http\Controllers\Controller;
use App\Models\LuckyDrawLog;
use App\Models\Plan;
use App\Models\User;
use App\Services\UserService;
use Carbon\Carbon;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;

/**
 * "免费领会员"抽奖：每个冷却周期可尝试一次，中奖概率随注册用户总数动态
 * 反推（目标：无论用户规模多大，每天大约固定数量的人能中奖），全部由
 * admin_setting 驱动，客户端不参与胜负判定。
 */
class LuckyDrawController extends Controller
{
    public function info(Request $request)
    {
        $user = $request->user();

        return $this->success([
            'enabled' => (bool) admin_setting('lucky_draw_enabled', 1),
            'reward_hours' => (int) admin_setting('lucky_draw_reward_hours', 1),
            'cooldown_remaining_seconds' => $this->cooldownRemainingSeconds($user->id),
            'today_winner_count' => $this->todayWinnerCount(),
            'total_hours_claimed' => (int) LuckyDrawLog::where('is_win', true)->sum('hours_credited'),
        ]);
    }

    public function claim(Request $request)
    {
        $userId = $request->user()->id;

        $lock = \Illuminate\Support\Facades\Cache::lock('lucky_draw:' . $userId, 10);
        if (!$lock->get()) {
            return $this->fail([429, __('Please try again later')]);
        }

        try {
            $result = DB::transaction(function () use ($userId) {
                if (!(bool) admin_setting('lucky_draw_enabled', 1)) {
                    throw new \Exception(__('This feature is not available'));
                }

                $cooldownRemaining = $this->cooldownRemainingSeconds($userId);
                if ($cooldownRemaining > 0) {
                    throw new \Exception(__('Please try again later'));
                }

                $user = User::lockForUpdate()->find($userId);
                if (!$user) {
                    throw new \Exception(__('The user does not exist'));
                }

                $rewardHours = (int) admin_setting('lucky_draw_reward_hours', 1);
                $isWin = $this->rollWin();
                $hoursCredited = 0;

                if ($isWin) {
                    $hoursCredited = $rewardHours;

                    // 从未开通过任何套餐（plan_id 为空）时，光延长 expired_at
                    // 没有意义——App 和节点鉴权都以 plan_id 是否存在来判断"是否
                    // 有会员"。这种情况下顺带发放试用套餐（复用设备体验账号
                    // 用的同一个 try_out_plan_id 配置），让奖励真正可见/可用。
                    if (!$user->plan_id) {
                        $trialPlanId = (int) admin_setting('try_out_plan_id', 0);
                        $trialPlan = $trialPlanId ? Plan::find($trialPlanId) : null;
                        if ($trialPlan) {
                            $user->plan_id = $trialPlan->id;
                            $user->group_id = $trialPlan->group_id;
                            $user->transfer_enable = $trialPlan->transfer_enable * 1073741824;
                            $user->speed_limit = $trialPlan->speed_limit;
                            $user->device_limit = $trialPlan->device_limit;
                        }
                    }

                    $userService = app(UserService::class);
                    $user = $userService->extendSubscriptionSeconds($user, $hoursCredited * 3600);
                    $user->save();
                }

                LuckyDrawLog::create([
                    'user_id' => $userId,
                    'is_win' => $isWin,
                    'hours_credited' => $hoursCredited,
                ]);

                return [
                    'win' => $isWin,
                    'hours_credited' => $hoursCredited,
                    'cooldown_seconds' => (int) admin_setting('lucky_draw_cooldown_seconds', 3600),
                ];
            });
        } catch (\Exception $e) {
            return $this->fail([400, $e->getMessage()]);
        } finally {
            $lock->release();
        }

        return $this->success($result);
    }

    /**
     * 概率 = 当日目标中奖人数 / 当前注册用户总数，夹在管理员配置的
     * [min%, max%] 区间内——用户越多单人概率越低，从而把每天中奖人数
     * 稳定在目标附近，避免小站被薅穿或大站中奖率过低。
     */
    private function rollWin(): bool
    {
        $totalUsers = max(1, $this->totalUserCount());
        $target = max(0, (int) admin_setting('lucky_draw_daily_target_winners', 500));
        $minPercent = (float) admin_setting('lucky_draw_min_probability_percent', 1);
        $maxPercent = (float) admin_setting('lucky_draw_max_probability_percent', 50);

        $percent = ($target / $totalUsers) * 100;
        $percent = max($minPercent, min($maxPercent, $percent));

        return random_int(1, 1000000) <= (int) round($percent * 10000);
    }

    private function totalUserCount(): int
    {
        return (int) Cache::remember('lucky_draw:total_user_count', 300, function () {
            return User::count();
        });
    }

    private function cooldownRemainingSeconds(int $userId): int
    {
        $cooldownSeconds = (int) admin_setting('lucky_draw_cooldown_seconds', 3600);
        $lastAttempt = LuckyDrawLog::where('user_id', $userId)
            ->orderByDesc('created_at')
            ->first();
        if (!$lastAttempt) {
            return 0;
        }

        $elapsed = time() - (int) $lastAttempt->created_at;
        return max(0, $cooldownSeconds - $elapsed);
    }

    private function todayWinnerCount(): int
    {
        $startAt = Carbon::today()->startOfDay()->timestamp;
        $endAt = Carbon::today()->endOfDay()->timestamp;

        return LuckyDrawLog::where('is_win', true)
            ->whereBetween('created_at', [$startAt, $endAt])
            ->count();
    }
}
