import React from "react";
import { classNames } from "../utils/classNames.js";

export function Card({ className, ...props }) {
    return <div className={classNames("rounded-lg border bg-card text-card-foreground shadow-sm", className)} {...props} />;
}

export function CardHeader({ className, ...props }) {
    return <div className={classNames("flex flex-col space-y-1.5 p-6", className)} {...props} />;
}

export function CardTitle({ className, ...props }) {
    return <h3 className={classNames("font-semibold leading-none tracking-tight", className)} {...props} />;
}

export function CardDescription({ className, ...props }) {
    return <p className={classNames("text-sm text-muted-foreground", className)} {...props} />;
}

export function CardContent({ className, ...props }) {
    return <div className={classNames("p-6 pt-0", className)} {...props} />;
}

export function CardFooter({ className, ...props }) {
    return <div className={classNames("flex items-center p-6 pt-0", className)} {...props} />;
}
