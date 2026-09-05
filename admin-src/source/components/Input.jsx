import React from "react";
import { classNames } from "../utils/classNames.js";

export const Input = React.forwardRef(function Input({ className, type = "text", ...props }, ref) {
    return (
        <input
            ref={ref}
            type={type}
            className={classNames(
                "flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50",
                className,
            )}
            {...props}
        />
    );
});
