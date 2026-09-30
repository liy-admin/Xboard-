<?php

namespace App\Console\Commands;

use App\Models\CommissionLog;
use App\Models\ReferralDurationLog;
use Illuminate\Console\Command;
use App\Models\Order;
use App\Models\User;
use Illuminate\Support\Facades\DB;

class CheckCommission extends Command
{
    /**
     * The name and signature of the console command.
     *
     * @var string
     */
    protected $signature = 'check:commission';

    /**
     * The console command description.
     *
     * @var string
     */
    protected $description = '返佣服务';

    /**
     * Create a new command instance.
     *
     * @return void
     */
    public function __construct()
    {
        parent::__construct();
    }

    /**
     * Execute the console command.
     *
     * @return mixed
     */
    public function handle()
    {
        $this->autoCheck();
        $this->autoPayCommission();
    }

    public function autoCheck()
    {
        if ((int)admin_setting('commission_auto_check_enable', 1)) {
            Order::where('commission_status', 0)
                ->where('invite_user_id', '!=', NULL)
                ->where('status', 3)
                ->where('updated_at', '<=', strtotime('-3 day', time()))
                ->update([
                    'commission_status' => 1
                ]);
        }
    }

    public function autoPayCommission()
    {
        $orders = Order::where('commission_status', 1)
            ->where('invite_user_id', '!=', NULL)
            ->get();
        foreach ($orders as $order) {
            try{
                DB::beginTransaction();
                if (!$this->payHandle($order->invite_user_id, $order)) {
                    DB::rollBack();
                    continue;
                }
                $order->commission_status = 2;
                if (!$order->save()) {
                    DB::rollBack();
                    continue;
                }
                DB::commit();
            } catch (\Exception $e){
                DB::rollBack();
                throw $e;
            }
        }
    }

    public function payHandle($inviteUserId, Order $order)
    {
        $level = 3;
        if ((int)admin_setting('commission_distribution_enable', 0)) {
            $commissionShareLevels = [
                0 => (int)admin_setting('commission_distribution_l1'),
                1 => (int)admin_setting('commission_distribution_l2'),
                2 => (int)admin_setting('commission_distribution_l3')
            ];
        } else {
            $commissionShareLevels = [
                0 => 100
            ];
        }
        for ($l = 0; $l < $level; $l++) {
            $inviter = User::find($inviteUserId);
            if (!$inviter) continue;
            if (!isset($commissionShareLevels[$l])) continue;
            $commissionBalance = $order->commission_balance * ($commissionShareLevels[$l] / 100);
            if ($commissionBalance) {
                if ((int)admin_setting('withdraw_close_enable', 0)) {
                    $inviter->increment('balance', $commissionBalance);
                } else {
                    $inviter->increment('commission_balance', $commissionBalance);
                }
                if (!$inviter->save()) {
                    DB::rollBack();
                    return false;
                }
                CommissionLog::create([
                    'invite_user_id' => $inviteUserId,
                    'user_id' => $order->user_id,
                    'trade_no' => $order->trade_no,
                    'order_amount' => $order->total_amount,
                    'get_amount' => $commissionBalance
                ]);
                // update order actual commission balance
                $order->actual_commission_balance = $order->actual_commission_balance + $commissionBalance;
            }

            // 推荐时长奖励：只发给直接推荐人（l === 0），不像金额佣金那样按
            // L1/L2/L3 逐级往上分——"薅时长"这个玩法本来就是奖励第一手推荐，
            // 不需要多级分销那么复杂。
            if ($l === 0 && $order->referral_duration_seconds) {
                // 永久会员（有套餐且 expired_at 为 NULL，代表永不过期）不做覆盖：
                // 若按 `expired_at ?? time()` 累加，会把无限期会员降级成一个几小时/
                // 几天后就到期的有限时间。此时保持永久有效，跳过发放与记账。
                $inviterIsPermanent = $inviter->plan_id !== null && $inviter->expired_at === null;
                if (!$inviterIsPermanent) {
                    $inviter->expired_at = max($inviter->expired_at ?? time(), time()) + $order->referral_duration_seconds;
                    if (!$inviter->save()) {
                        DB::rollBack();
                        return false;
                    }
                    ReferralDurationLog::create([
                        'inviter_id' => $inviteUserId,
                        'invitee_id' => $order->user_id,
                        'trade_no' => $order->trade_no,
                        'seconds_credited' => $order->referral_duration_seconds,
                    ]);
                }
            }

            $inviteUserId = $inviter->invite_user_id;
        }
        return true;
    }

}
