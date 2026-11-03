import React from "react";
import { classNames } from "../utils/classNames.js";

export function Table({ className, ...props }) {
    return (
        <div className="relative w-full overflow-auto rounded-md border">
            <table className={classNames("w-full caption-bottom text-sm", className)} {...props} />
        </div>
    );
}

export function TableHeader({ className, ...props }) {
    return <thead className={classNames("[&_tr]:border-b bg-muted/40", className)} {...props} />;
}

export function TableBody({ className, ...props }) {
    return <tbody className={classNames("[&_tr:last-child]:border-0", className)} {...props} />;
}

export function TableRow({ className, ...props }) {
    return (
        <tr
            className={classNames(
                "border-b border-border/60 transition-colors hover:bg-muted/40 data-[state=selected]:bg-muted/60",
                className,
            )}
            {...props}
        />
    );
}

export function TableHead({ className, ...props }) {
    return (
        <th
            className={classNames(
                "h-11 px-4 text-left align-middle font-medium text-muted-foreground whitespace-nowrap [&:has([role=checkbox])]:pr-0",
                className,
            )}
            {...props}
        />
    );
}

export function TableCell({ className, ...props }) {
    return (
        <td
            className={classNames("px-4 py-3 align-middle whitespace-nowrap [&:has([role=checkbox])]:pr-0", className)}
            {...props}
        />
    );
}

export function TableEmpty({ colSpan, children }) {
    return (
        <TableRow>
            <TableCell colSpan={colSpan} className="h-24 text-center text-muted-foreground">
                {children}
            </TableCell>
        </TableRow>
    );
}
