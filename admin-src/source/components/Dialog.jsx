import React, { useEffect } from "react";
import { createPortal } from "react-dom";
import { classNames } from "../utils/classNames.js";

export function Dialog({ open, onOpenChange, className, children }) {
    useEffect(() => {
        if (!open) {
            return undefined;
        }

        const handleEscape = (event) => {
            if (event.key === "Escape") {
                onOpenChange?.(false);
            }
        };

        window.addEventListener("keydown", handleEscape);
        return () => window.removeEventListener("keydown", handleEscape);
    }, [open, onOpenChange]);

    if (!open || typeof document === "undefined") {
        return null;
    }

    const portalRoot = document.body || document.documentElement;

    return createPortal(
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm"
            onMouseDown={(event) => {
                if (event.target === event.currentTarget) {
                    onOpenChange?.(false);
                }
            }}
        >
            <div
                role="dialog"
                aria-modal="true"
                className={classNames("w-full rounded-lg border bg-background shadow-lg", className)}
                onMouseDown={(event) => event.stopPropagation()}
            >
                {children}
            </div>
        </div>,
        portalRoot,
    );
}
