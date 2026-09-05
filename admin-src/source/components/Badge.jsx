import React from "react";
import { classNames } from "../utils/classNames.js";

const VARIANTS = {
    default: "bg-primary text-primary-foreground hover:bg-primary/80",
    secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
    success: "border-transparent bg-emerald-500 text-white hover:bg-emerald-500/80",
    destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/80",
    outline: "text-foreground",
};

export function Badge({ variant = "default", className, ...props }) {
    return (
        <span
            className={classNames(
                "inline-flex items-center rounded-md border border-transparent px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
                VARIANTS[variant] || VARIANTS.default,
                className,
            )}
            {...props}
        />
    );
}
