<?php

namespace App\Http\Controllers\V1\Passport;

use App\Helpers\ResponseEnum;
use App\Http\Controllers\Controller;
use App\Http\Requests\Passport\AuthForget;
use App\Http\Requests\Passport\AuthLogin;
use App\Http\Requests\Passport\AuthRegister;
use App\Services\Auth\LoginService;
use App\Services\Auth\MailLinkService;
use App\Services\Auth\RegisterService;
use App\Services\AuthService;
use App\Services\DeviceBindingService;
use App\Models\User;
use Illuminate\Http\Request;

class AuthController extends Controller
{
    protected MailLinkService $mailLinkService;
    protected RegisterService $registerService;
    protected LoginService $loginService;

    public function __construct(
        MailLinkService $mailLinkService,
        RegisterService $registerService,
        LoginService $loginService
    ) {
        $this->mailLinkService = $mailLinkService;
        $this->registerService = $registerService;
        $this->loginService = $loginService;
    }

    /**
     * 判断这次「网页端登录」是否应被拦截。
     *
     * 业务背景：面板的管理后台（React，走 secure_path）、用户网页端（umi）和
     * 移动/桌面 App 打的是同一个 passport 登录接口。运营要求「普通用户只能用
     * App，不允许在网页登录查看账户」，但不能误伤：① 超级管理员登录后台，
     * ② 普通用户在 App 内登录。
     *
     * 区分网页 / App 沿用本项目既有约定（见 AuthService 注释）：App 客户端会带
     * device_id（设备数限制就是靠它去重统计），而网页端（含管理后台与用户端）
     * 不带 device_id。因此仅当：开关关闭网页登录 且 目标账户不是管理员 且 本次
     * 是网页请求（无 device_id）时才拦截；管理员恒放行、App 登录恒放行。
     *
     * 这是真正的服务端边界——即使有人绕过前端直接打接口，普通用户也拿不到
     * 网页会话；前端是否还显示登录页只是观感问题。
     */
    private function isWebLoginBlocked(Request $request, User $user): bool
    {
        if ((int) admin_setting('user_web_login_enable', 1)) {
            return false; // 允许网页登录
        }
        if ($user->is_admin) {
            return false; // 管理员不受限
        }
        $deviceId = trim((string) $request->input('device_id', ''));
        return $deviceId === ''; // 无 device_id → 视为网页端，拦截
    }

    /**
     * 通过邮件链接登录
     */
    public function loginWithMailLink(Request $request)
    {
        $params = $request->validate([
            'email' => 'required|email:strict',
            'redirect' => 'nullable'
        ]);

        // 邮件登录链接最终落地到网页端，关闭网页登录时普通用户不再下发链接。
        if (!(int) admin_setting('user_web_login_enable', 1)) {
            $user = User::byEmail(\App\Services\Auth\EmailCodeService::normalize($params['email']))->first();
            if ($user && !$user->is_admin) {
                return $this->fail([403, __('Web login is not allowed for this account, please sign in from the app')]);
            }
        }

        [$success, $result] = $this->mailLinkService->handleMailLink(
            $params['email'],
            $request->input('redirect')
        );

        if (!$success) {
            return $this->fail($result);
        }

        return $this->success($result);
    }

    /**
     * 用户注册
     */
    public function register(AuthRegister $request)
    {
        [$success, $result] = $this->registerService->register($request);

        if (!$success) {
            return $this->fail($result);
        }

        // 新注册用户必不是管理员；关闭网页登录时，网页端注册完不发放会话。
        if ($this->isWebLoginBlocked($request, $result)) {
            return $this->fail([403, __('Web login is not allowed for this account, please sign in from the app')]);
        }

        $deviceId = $request->input('device_id');
        $authService = new AuthService($result);
        if ($limitError = $authService->checkDeviceLimit($deviceId)) {
            return $this->fail([403, $limitError]);
        }
        DeviceBindingService::bind($deviceId, $result);
        if ($deviceId) {
            $result->device_id = $deviceId;
            $result->save();
        }
        return $this->success($authService->generateAuthData($request->input('device_name'), $deviceId));
    }

    /**
     * 用户登录
     */
    public function login(AuthLogin $request)
    {
        $email = $request->input('email');
        $password = $request->input('password');

        [$success, $result] = $this->loginService->login($email, $password);

        if (!$success) {
            return $this->fail($result);
        }

        if ($this->isWebLoginBlocked($request, $result)) {
            return $this->fail([403, __('Web login is not allowed for this account, please sign in from the app')]);
        }

        $deviceId = $request->input('device_id');
        $authService = new AuthService($result);
        if ($limitError = $authService->checkDeviceLimit($deviceId)) {
            return $this->fail([403, $limitError]);
        }
        DeviceBindingService::bind($deviceId, $result);
        if ($deviceId) {
            $result->device_id = $deviceId;
            $result->save();
        }
        return $this->success($authService->generateAuthData($request->input('device_name'), $deviceId));
    }

    /**
     * 通过token登录
     */
    public function token2Login(Request $request)
    {
        // 处理直接通过token重定向
        if ($token = $request->input('token')) {
            $redirect = '/#/login?verify=' . rawurlencode((string) $token)
                . '&redirect=' . rawurlencode((string) $request->input('redirect', 'dashboard'));

            return redirect()->to(
                admin_setting('app_url')
                ? admin_setting('app_url') . $redirect
                : url($redirect)
            );
        }

        // 处理通过验证码登录
        if ($verify = $request->input('verify')) {
            $userId = $this->mailLinkService->handleTokenLogin($verify);

            if (!$userId) {
                return response()->json([
                    'message' => __('Token error')
                ], 400);
            }

            $user = User::find($userId);

            if (!$user) {
                return response()->json([
                    'message' => __('User not found')
                ], 400);
            }

            // token2Login 是邮件验证码 / 快捷登录在网页端的落地环节，始终视为网页登录。
            if ($this->isWebLoginBlocked($request, $user)) {
                return response()->json([
                    'message' => __('Web login is not allowed for this account, please sign in from the app')
                ], 403);
            }

            $authService = new AuthService($user);

            return response()->json([
                'data' => $authService->generateAuthData()
            ]);
        }

        return response()->json([
            'message' => __('Invalid request')
        ], 400);
    }

    /**
     * 获取快速登录URL
     */
    public function getQuickLoginUrl(Request $request)
    {
        $authorization = $request->input('auth_data') ?? $request->header('authorization');

        if (!$authorization) {
            return response()->json([
                'message' => ResponseEnum::CLIENT_HTTP_UNAUTHORIZED
            ], 401);
        }

        $user = AuthService::findUserByBearerToken($authorization);

        if (!$user) {
            return response()->json([
                'message' => ResponseEnum::CLIENT_HTTP_UNAUTHORIZED_EXPIRED
            ], 401);
        }

        // 快捷登录 URL 指向网页端 /#/login；关闭网页登录时普通用户不再下发。
        if ($this->isWebLoginBlocked($request, $user)) {
            return $this->fail([403, __('Web login is not allowed for this account, please sign in from the app')]);
        }

        $url = $this->loginService->generateQuickLoginUrl($user, $request->input('redirect'));
        return $this->success($url);
    }

    /**
     * 忘记密码处理
     */
    public function forget(AuthForget $request)
    {
        [$success, $result] = $this->loginService->resetPassword(
            $request->input('email'),
            $request->input('email_code'),
            $request->input('password')
        );

        if (!$success) {
            return $this->fail($result);
        }

        return $this->success(true);
    }
}
