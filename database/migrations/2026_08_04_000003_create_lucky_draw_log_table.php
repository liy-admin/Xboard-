<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::create('v2_lucky_draw_log', function (Blueprint $table) {
            $table->integer('id', true);
            $table->integer('user_id');
            $table->boolean('is_win')->default(false);
            $table->integer('hours_credited')->default(0);
            $table->integer('created_at');
            $table->integer('updated_at');
            $table->index('user_id');
            $table->index('created_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('v2_lucky_draw_log');
    }
};
