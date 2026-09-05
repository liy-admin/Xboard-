import React, { createContext, useContext } from "react";
import { classNames } from "../utils/classNames.js";

const TabsContext = createContext({ value: "", onValueChange: () => {} });

export function Tabs({ value, onValueChange, className, children }) {
    return (
        <TabsContext.Provider value={{ value, onValueChange }}>
            <div className={className}>{children}</div>
        </TabsContext.Provider>
    );
}

export function TabsList({ className, children }) {
    return (
        <div className={classNames("inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground", className)}>
            {children}
        </div>
    );
}

export function TabsTrigger({ value, className, children }) {
    const ctx = useContext(TabsContext);
    const active = ctx.value === value;

    return (
        <button
            type="button"
            onClick={() => ctx.onValueChange(value)}
            className={classNames(
                "inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium transition-all",
                active && "bg-background text-foreground shadow",
                className,
            )}
        >
            {children}
        </button>
    );
}

export function TabsContent({ value, className, children }) {
    const ctx = useContext(TabsContext);
    if (ctx.value !== value) return null;
    return <div className={className}>{children}</div>;
}
