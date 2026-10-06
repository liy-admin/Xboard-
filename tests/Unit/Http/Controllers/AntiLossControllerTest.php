<?php

namespace Tests\Unit\Http\Controllers;

use App\Http\Controllers\V1\Guest\AntiLossController;
use PHPUnit\Framework\TestCase;
use ReflectionMethod;

/**
 * 只测 AntiLossController 里唯一有分支的逻辑：把后台存的（可能被手改/带历史
 * 脏数据的）socials JSON 清洗成客户端预期的形状。纯函数、不碰 DB，所以直接
 * 用 PHPUnit\TestCase 而不是 Laravel 的 TestCase。
 */
class AntiLossControllerTest extends TestCase
{
    private function normalize($input): array
    {
        $method = new ReflectionMethod(AntiLossController::class, 'normalizeSocials');
        $method->setAccessible(true);
        return $method->invoke(new AntiLossController(), $input);
    }

    public function test_keeps_valid_entries_and_fills_missing_keys(): void
    {
        $result = $this->normalize([
            ['name' => 'telegram', 'url' => 'https://t.me/demo'],
            ['name' => 'x', 'url' => 'https://x.com/demo', 'icon_url' => 'https://cdn/x.png'],
        ]);

        $this->assertSame([
            ['name' => 'telegram', 'url' => 'https://t.me/demo', 'icon_url' => ''],
            ['name' => 'x', 'url' => 'https://x.com/demo', 'icon_url' => 'https://cdn/x.png'],
        ], $result);
    }

    public function test_drops_entries_without_a_url(): void
    {
        $result = $this->normalize([
            ['name' => 'telegram', 'url' => ''],
            ['name' => 'facebook'],
            ['name' => 'x', 'url' => 'https://x.com/demo'],
        ]);

        $this->assertCount(1, $result);
        $this->assertSame('x', $result[0]['name']);
    }

    public function test_trims_whitespace(): void
    {
        $result = $this->normalize([
            ['name' => '  telegram  ', 'url' => '  https://t.me/demo  ', 'icon_url' => '  '],
        ]);

        $this->assertSame([
            ['name' => 'telegram', 'url' => 'https://t.me/demo', 'icon_url' => ''],
        ], $result);
    }

    public function test_survives_non_array_and_non_object_items(): void
    {
        $this->assertSame([], $this->normalize('not-an-array'));
        $this->assertSame([], $this->normalize(null));
        $this->assertSame([], $this->normalize(['https://t.me/demo', 42]));
    }
}
