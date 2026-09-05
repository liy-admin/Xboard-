import React, { useState } from "react";
import { Dialog } from "./Dialog.jsx";
import { Button } from "./Button.jsx";

export function ConfirmDialog({
    open,
    onOpenChange,
    title,
    description,
    confirmLabel = "确认",
    cancelLabel = "取消",
    variant = "destructive",
    onConfirm,
}) {
    const [loading, setLoading] = useState(false);

    const handleConfirm = async () => {
        setLoading(true);
        try {
            await onConfirm?.();
            onOpenChange?.(false);
        } finally {
            setLoading(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-md">
            <div className="p-6">
                <h3 className="text-lg font-semibold">{title}</h3>
                {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
                <div className="mt-6 flex justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange?.(false)} disabled={loading}>
                        {cancelLabel}
                    </Button>
                    <Button variant={variant} onClick={handleConfirm} loading={loading}>
                        {confirmLabel}
                    </Button>
                </div>
            </div>
        </Dialog>
    );
}
