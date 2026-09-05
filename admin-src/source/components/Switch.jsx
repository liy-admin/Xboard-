import React from "react";
import { classNames } from "../utils/classNames.js";

export const Switch = React.forwardRef(function Switch({ className, checked, onCheckedChange, ...props }, ref) {
    return (
        <button
            ref={ref}
            type="button"
            role="switch"
            aria-checked={checked}
            data-state={checked ? "checked" : "unchecked"}
            onClick={() => onCheckedChange?.(!checked)}
            className={classNames(
                "peer settings-switch inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
                checked ? "bg-primary" : "bg-input",
                className,
            )}
            {...props}
        >
            <span
                data-state={checked ? "checked" : "unchecked"}
                className={classNames(
                    "settings-switch-thumb pointer-events-none block h-4 w-4 rounded-full bg-background shadow-lg ring-0 transition-transform",
                    checked ? "translate-x-4" : "translate-x-0",
                )}
            />
        </button>
    );
});
