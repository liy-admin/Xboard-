import React, { useState } from "react";
import { Dialog } from "./Dialog.jsx";
import { Button } from "./Button.jsx";
import { classNames } from "../utils/classNames.js";

export function FormDialog({
    open,
    onOpenChange,
    title,
    description,
    cancelLabel = "取消",
    submitLabel = "提交",
    onSubmit,
    className,
    children,
}) {
    const [submitting, setSubmitting] = useState(false);

    const handleSubmit = async (event) => {
        event.preventDefault();
        setSubmitting(true);
        try {
            await onSubmit?.();
            onOpenChange?.(false);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className={classNames("sm:max-w-lg", className)}>
            <form onSubmit={handleSubmit} className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{title}</h3>
                {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
                <div className="mt-4 space-y-4">{children}</div>
                <div className="mt-6 flex justify-end gap-2">
                    <Button type="button" variant="outline" onClick={() => onOpenChange?.(false)} disabled={submitting}>
                        {cancelLabel}
                    </Button>
                    <Button type="submit" loading={submitting}>
                        {submitLabel}
                    </Button>
                </div>
            </form>
        </Dialog>
    );
}
