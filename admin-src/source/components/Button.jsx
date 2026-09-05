import React from "react";
import { classNames } from "../utils/classNames.js";

const VARIANTS = {
    default: "bg-primary text-primary-foreground shadow hover:bg-primary/90",
    destructive: "bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90",
    outline: "border border-input bg-background shadow-sm hover:bg-accent hover:text-accent-foreground",
    ghost: "hover:bg-accent hover:text-accent-foreground",
    link: "text-primary underline-offset-4 hover:underline",
};

const SIZES = {
    default: "h-10 px-4 py-2",
    sm: "h-9 rounded-md px-3",
    lg: "h-11 rounded-md px-8",
    icon: "h-10 w-10",
};

export const Button = React.forwardRef(function Button(
    { as: Component = "button", variant = "default", size = "default", loading = false, className, children, type, ...props },
    ref,
) {
    return (
        <Component
            ref={ref}
            type={type || (Component === "button" ? "button" : undefined)}
            className={classNames(
                "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50",
                VARIANTS[variant] || VARIANTS.default,
                SIZES[size] || SIZES.default,
                loading && "cursor-wait",
                className,
            )}
            aria-busy={loading || undefined}
            disabled={loading || props.disabled || undefined}
            {...props}
        >
            {loading ? (
                <span className="inline-flex items-center gap-2">
                    <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" />
                    {children}
                </span>
            ) : (
                children
            )}
        </Component>
    );
});
