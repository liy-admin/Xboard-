<?php

use Illuminate\Support\Facades\Route;
use Plugin\BalalaSub\Controllers\SubController;

// 抓取动作会驱动浏览器登录，属于敏感操作，这里用 admin 中间件保护。
Route::group([
    'prefix' => 'api/v1/balala-sub',
    'middleware' => ['admin'],
], function () {
    Route::get('/latest', [SubController::class, 'latest']);
    Route::post('/refresh', [SubController::class, 'refresh']);
    Route::get('/doctor', [SubController::class, 'doctor']);
    Route::post('/upload-firefox', [SubController::class, 'uploadFirefox']);
});
