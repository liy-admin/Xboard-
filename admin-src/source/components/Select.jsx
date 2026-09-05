import React, { useEffect, useRef, useState } from "react";
import { classNames } from "../utils/classNames.js";

export function Select({ value, onValueChange, children, placeholder, className }) {
    const [open, setOpen] = useState(false);
    const rootRef = useRef(null);

    useEffect(() => {
        if (!open) return;
        const handleDown = (e) => {
            if (!rootRef.current?.contains(e.target)) setOpen(false);
        };
        const handleKey = (e) => {
            if (e.key === "Escape") setOpen(false);
        };
        document.addEventListener("pointerdown", handleDown);
        document.addEventListener("keydown", handleKey);
        return () => {
            document.removeEventListener("pointerdown", handleDown);
            document.removeEventListener("keydown", handleKey);
        };
    }, [open]);

    const trigger = children?.find?.((c) => c?.type === SelectTrigger) || children?.[0];
    const list = children?.find?.((c) => c?.type === SelectContent) || children?.[1];

    const selectedItem =
        value !== undefined && value !== null && value !== ""
            ? React.Children.toArray(list?.props?.children || []).find(
                  (child) => child && String(child.props.value) === String(value),
              )
            : null;

    return (
        <div ref={rootRef} className={classNames("relative", className)}>
            <button
                type="button"
                className="inline-flex h-9 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm font-normal shadow-sm hover:bg-accent hover:text-accent-foreground"
                onClick={() => setOpen((v) => !v)}
            >
                {selectedItem ? (
                    <span>{selectedItem.props.children}</span>
                ) : (
                    trigger?.props?.children || <span className="text-muted-foreground">{placeholder}</span>
                )}
            </button>
            {open && (
                <div className="absolute left-0 top-10 z-50 w-full min-w-[130px] rounded-md border bg-popover p-1 text-popover-foreground shadow-md">
                    {React.Children.map(list?.props?.children || [], (child) => {
                        if (!child) return null;
                        const selected = String(child.props.value) === String(value);
                        return (
                            <button
                                type="button"
                                className={classNames(
                                    "flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent",
                                    selected && "bg-secondary",
                                )}
                                onClick={() => {
                                    onValueChange?.(child.props.value);
                                    setOpen(false);
                                }}
                            >
                                {child.props.children}
                                {selected && " ✓"}
                            </button>
                        );
                    })}
                </div>
            )}
        </div>
    );
}

export function SelectTrigger({ children }) {
    return <>{children}</>;
}

export function SelectContent({ children }) {
    return <>{children}</>;
}

export function SelectItem({ value, children }) {
    return <>{children}</>;
}

export function SelectValue({ placeholder }) {
    return <span className="text-muted-foreground">{placeholder}</span>;
}
