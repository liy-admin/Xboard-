import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings, setTelegramWebhook } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Button } from "../../components/Button.jsx";
import { Switch } from "../../components/Switch.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    telegram_bot_enable: false, telegram_bot_token: "",
    telegram_webhook_url: "", telegram_discuss_link: "",
};

export function TelegramSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [webhookLoading, setWebhookLoading] = useState(false);
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("telegram").then((res) => {
            const tg = res?.data?.telegram || res?.data || {};
            const next = { ...DEFAULTS, ...tg };
            setValues(next);
            lastSavedRef.current = { ...next };
            setReady(true);
        }).catch(() => setReady(true));
    }, [auth.token]);

    const doSave = useCallback(
        debounce(async (data) => {
            if (isEqual(data, lastSavedRef.current)) return;
            setSaving(true);
            try {
                await saveSettings(data);
                lastSavedRef.current = { ...data };
            } finally { setSaving(false); }
        }, 1000),
        [auth.token],
    );

    const setField = useCallback((name, value) => {
        setValues((prev) => {
            const next = { ...prev, [name]: value };
            doSave(next);
            return next;
        });
    }, [doSave]);

    useEffect(() => () => doSave.cancel(), [doSave]);

    const handleSetWebhook = async () => {
        setWebhookLoading(true);
        try {
            const res = await setTelegramWebhook();
            if (res?.data) {
                // Webhook set successfully
            }
        } finally {
            setWebhookLoading(false);
        }
    };

    if (!ready) return null;

    const hasToken = Boolean(values.telegram_bot_token?.trim());
    const webhookUrl = values.telegram_webhook_url?.trim();

    return (
        <div>
            <h3 className="text-lg font-medium">{t("telegram.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("telegram.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <FormField label={t("telegram.bot_token.title")} placeholder={t("telegram.bot_token.placeholder")} description={t("telegram.bot_token.description")} value={values.telegram_bot_token ?? ""} onChange={(v) => setField("telegram_bot_token", v)} />
                {hasToken && (
                    <div className="space-y-1.5">
                        <div className="text-base font-medium">{t("telegram.webhook_url.title")}</div>
                        <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
                            <Input className="sm:flex-1" placeholder={t("telegram.webhook_url.placeholder")} value={values.telegram_webhook_url ?? ""} onChange={(e) => setField("telegram_webhook_url", e.target.value)} />
                            <Button type="button" className="sm:shrink-0" loading={webhookLoading} disabled={webhookLoading} onClick={handleSetWebhook}>
                                {webhookLoading ? t("telegram.webhook.setting") : t("telegram.webhook.button")}
                            </Button>
                        </div>
                        <div className="space-y-1.5 text-sm text-muted-foreground">
                            <p>{t("telegram.webhook_url.description")}</p>
                            <a href="https://core.telegram.org/bots/webhooks" target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline">
                                {t("telegram.webhook_url.docs")}
                                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M7 7h10v10" /><path d="M7 17 17 7" /></svg>
                            </a>
                        </div>
                    </div>
                )}
                <ToggleField label={t("telegram.bot_enable.title")} description={t("telegram.bot_enable.description")} checked={values.telegram_bot_enable ?? false} onChange={(v) => setField("telegram_bot_enable", v)} />
                <FormField label={t("telegram.discuss_link.title")} placeholder={t("telegram.discuss_link.placeholder")} description={t("telegram.discuss_link.description")} value={values.telegram_discuss_link ?? ""} onChange={(v) => setField("telegram_discuss_link", v)} />
                {saving && <div className="text-sm text-muted-foreground">Đang lưu...</div>}
            </div>
        </div>
    );
}

function FormField({ label, placeholder, description, value, onChange, type = "text", as: As = Input }) {
    return (
        <div className="space-y-1.5">
            {label && <div className="text-base font-medium">{label}</div>}
            <As type={type} placeholder={placeholder} value={value} onChange={(e) => onChange(e.target.value)} />
            {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
    );
}

function ToggleField({ label, description, checked, onChange }) {
    return (
        <div className="flex flex-row items-center justify-between rounded-lg border p-4">
            <div className="space-y-0.5">
                <div className="text-base font-medium">{label}</div>
                {description && <p className="text-sm text-muted-foreground">{description}</p>}
            </div>
            <Switch checked={checked} onCheckedChange={onChange} />
        </div>
    );
}
