import React, { useEffect, useRef, useState } from "react";
import { classNames } from "../utils/classNames.js";
import { useTranslation } from "../i18n/index.js";
import { Button } from "./Button.jsx";

export function LanguageSwitcher() {
    const { language, languages, changeLanguage } = useTranslation();
    const [open, setOpen] = useState(false);
    const containerRef = useRef(null);
    const current = languages.find((item) => item.code === language) || languages[0];

    useEffect(() => {
        const handlePointerDown = (event) => {
            if (!containerRef.current) {
                return;
            }

            if (!containerRef.current.contains(event.target)) {
                setOpen(false);
            }
        };

        document.addEventListener("pointerdown", handlePointerDown);
        return () => document.removeEventListener("pointerdown", handlePointerDown);
    }, []);

    return (
        <div ref={containerRef} className="relative">
            <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8 gap-1 px-2"
                onClick={() => setOpen((currentOpen) => !currentOpen)}
                aria-label="Change language"
            >
                <span className="text-sm font-medium">{current.shortName}</span>
                <span className="text-[10px] opacity-60">▾</span>
            </Button>
            {open ? (
                <div className="absolute right-0 z-50 mt-2 w-36 rounded-md border bg-popover p-1 text-popover-foreground shadow-lg">
                    {languages.map((item) => {
                        const active = item.code === language;

                        return (
                            <button
                                key={item.code}
                                type="button"
                                onClick={() => {
                                    changeLanguage(item.code);
                                    setOpen(false);
                                }}
                                className={classNames(
                                    "flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground",
                                    active && "bg-accent font-medium",
                                )}
                            >
                                <span>{item.name}</span>
                                <span className="text-xs opacity-60">{item.shortName}</span>
                            </button>
                        );
                    })}
                </div>
            ) : null}
        </div>
    );
}
