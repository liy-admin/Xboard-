<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        // `code` can legitimately repeat across nodes now (e.g. a shared proxy
        // account UUID extracted from imported links applies to several nodes),
        // so it can no longer be enforced unique per type.
        Schema::table('v2_server', function (Blueprint $table) {
            $table->dropUnique(['type', 'code']);
            $table->index(['type', 'code']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('v2_server', function (Blueprint $table) {
            $table->dropIndex(['type', 'code']);
            $table->unique(['type', 'code']);
        });
    }
};
