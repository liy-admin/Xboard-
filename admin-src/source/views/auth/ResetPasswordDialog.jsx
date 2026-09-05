import React, { useEffect, useState } from "react";
import { Button } from "../../components/Button.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { copyText } from "../../utils/clipboard.js";
import { useTranslation } from "../../i18n/index.js";

export function ResetPasswordDialog({ open, onOpenChange, command }) {
    const { t } = useTranslation("auth");
    const { t: tCommon } = useTranslation("common");
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        if (!open) {
            setCopied(false);
        }
    }, [open]);

    async function handleCopy() {
        const success = await copyText(command);
        setCopied(success);

        if (success) {
            window.setTimeout(() => setCopied(false), 1500);
        }
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="max-w-[90vw] sm:max-w-lg">
            <div className="p-6">
                <div className="space-y-1.5">
                    <h3 className="text-lg font-semibold leading-none tracking-tight">{t("signIn.resetPassword.title")}</h3>
                    <p className="text-sm text-muted-foreground">{t("signIn.resetPassword.description")}</p>
                </div>

                <div className="mt-4 rounded-md bg-secondary p-4">
                    <pre className="max-w-full overflow-x-auto text-sm">{command}</pre>
                </div>

                <div className="mt-4 flex items-center justify-between gap-3">
                    <span className="text-xs text-muted-foreground">
                        {copied ? tCommon("copy.success") : "\u00A0"}
                    </span>

                    <Button type="button" variant="outline" size="sm" onClick={handleCopy}>
                        {copied ? "Copied" : "Copy command"}
                    </Button>
                </div>
            </div>
        </Dialog>
    );
}
