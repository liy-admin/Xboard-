<?php

namespace App\Http\Controllers\V1\User;

use App\Http\Controllers\Controller;
use App\Models\ReferralDurationLog;
use App\Models\User;
use Illuminate\Http\Request;

class ReferralController extends Controller
{
    /**
     * "推荐有奖"页需要的一切：我的推荐码、分享链接、统计数字、文案。
     *
     * 文案（headline/subtext）走 admin_setting，方便不改客户端代码就调整
     * 措辞；奖励比例同理（[App\Services\OrderService::setInvite] 用的是
     * 同一个 key，展示和实际结算天然保持一致）。
     */
    public function info(Request $request)
    {
        $user = $request->user();
        $code = $user->referral_code ?: $this->assignReferralCode($user);

        $inviteCount = ReferralDurationLog::where('inviter_id', $user->id)
            ->distinct('invitee_id')
            ->count('invitee_id');
        $totalSeconds = (int) ReferralDurationLog::where('inviter_id', $user->id)->sum('seconds_credited');

        $shareLink = $this->buildShareLink($code);

        return $this->success([
            'referral_code' => $code,
            'share_link' => $shareLink,
            'invite_count' => $inviteCount,
            'total_hours' => round($totalSeconds / 3600, 1),
            'reward_percent' => (float) admin_setting('referral_duration_percent', 20),
            'headline' => admin_setting('referral_headline', '推荐朋友，领永久会员'),
            'subtext' => admin_setting(
                'referral_subtext',
                '好友安装快连后，填写您的推荐码(:code)即算推荐成功；当其每次获得会员，您均可薅取 :percent% 的时长！永久有效！'
            ),
        ]);
    }

    /**
     * 分享链接优先用后台自定义的 referral_share_url（支持 :code 占位符，
     * 方便直接指向官网/落地页/App 下载页而非固定的网页注册页）；没配置时
     * 回退到旧逻辑：app_url + 网页版注册页。
     */
    private function buildShareLink(string $code): ?string
    {
        $template = admin_setting('referral_share_url');
        if ($template) {
            $scheme = strtolower(parse_url($template, PHP_URL_SCHEME) ?? '');
            if (!in_array($scheme, ['http', 'https'], true)) {
                $template = null;
            }
        }
        if ($template) {
            return str_contains($template, ':code')
                ? str_replace(':code', $code, $template)
                : $template;
        }

        $appUrl = admin_setting('app_url');
        return $appUrl
            ? rtrim($appUrl, '/') . '/#/register?referral_code=' . $code
            : null;
    }

    /**
     * 首次访问时才按需生成 6 位数字推荐码（000000-999999），冲突就重抽——
     * 用户量在这个量级（百万级以内）冲突概率很低，重抽几次足够。
     */
    private function assignReferralCode(User $user): string
    {
        do {
            $code = str_pad((string) random_int(0, 999999), 6, '0', STR_PAD_LEFT);
        } while (User::where('referral_code', $code)->exists());

        $user->referral_code = $code;
        $user->save();

        return $code;
    }
}
