import React from "react";
import { Input } from "./Input.jsx";
import { Button } from "./Button.jsx";

export function SearchFilterToolbar({
    search,
    onSearchChange,
    searchPlaceholder,
    resetLabel = "重置",
    onReset,
    filters,
    actions,
}) {
    return (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex flex-1 flex-col gap-2 sm:flex-row sm:items-center">
                <Input
                    value={search ?? ""}
                    onChange={(e) => onSearchChange?.(e.target.value)}
                    placeholder={searchPlaceholder}
                    className="sm:max-w-xs"
                />
                {filters}
                {onReset && (
                    <Button variant="outline" size="sm" onClick={onReset} type="button">
                        {resetLabel}
                    </Button>
                )}
            </div>
            {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
    );
}
