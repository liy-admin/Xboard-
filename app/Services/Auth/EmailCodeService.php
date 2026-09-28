<?php

namespace App\Services\Auth;

use App\Utils\CacheKey;
use Illuminate\Support\Facades\Cache;

class EmailCodeService
{
    private const CODE_TTL = 300;
    private const MAX_ATTEMPTS = 5;

    public static function normalize(string $email): string
    {
        return strtolower(trim($email));
    }

    public function issue(string $email): int
    {
        $email = self::normalize($email);
        $code = random_int(100000, 999999);
        Cache::put(CacheKey::get('EMAIL_VERIFY_CODE', $email), $code, self::CODE_TTL);
        return $code;
    }

    public function verify(string $email, string $code): bool
    {
        $email = self::normalize($email);
        $codeKey = CacheKey::get('EMAIL_VERIFY_CODE', $email);
        $failKey = CacheKey::get('EMAIL_VERIFY_ATTEMPTS', $email);

        $cached = Cache::get($codeKey);
        if ($cached === null) {
            return false;
        }

        Cache::add($failKey, 0, self::CODE_TTL);
        $attempts = Cache::increment($failKey);
        if ($attempts > self::MAX_ATTEMPTS) {
            Cache::forget($codeKey);
            return false;
        }

        return hash_equals((string) $cached, $code);
    }

    public function forget(string $email): void
    {
        $email = self::normalize($email);
        Cache::forget(CacheKey::get('EMAIL_VERIFY_CODE', $email));
        Cache::forget(CacheKey::get('EMAIL_VERIFY_ATTEMPTS', $email));
    }
}
