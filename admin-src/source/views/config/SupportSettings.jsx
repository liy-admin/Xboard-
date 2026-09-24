import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Switch } from "../../components/Switch.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    support_chatwoot_enabled: false,
    support_chatwoot_base_url: "",
    support_chatwoot_website_token: "",
    support_chatwoot_locale: "zh_CN",
};

export function SupportSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [loaded, setLoaded] = useState(false);
    const [error, setError] = useState("");
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("support").then((res) => {
            const cfg = res?.data?.support || res?.data || {};
            const next = { ...DEFAULTS, ...cfg };
            setValues(next);
            lastSavedRef.current = { ...next };
            setReady(true);
            setLoaded(true);
        }).catch(() => { setReady(true); setLoaded(true); });
    }, [auth.token]);

    const doSave = useCallback(
        debounce(async (data) => {
            if (!loaded || isEqual(data, lastSavedRef.current)) return;
            setSaving(true);
            try {
                await saveSettings(data);
                lastSavedRef.current = { ...data };
                setError("");
            } catch (e) {
                setError(e?.message || t("support.save_error"));
            } finally { setSaving(false); }
        }, 1000),
        [auth.token, loaded],
    );

    const setField = useCallback((name, value) => {
        setValues((prev) => {
            const next = { ...prev, [name]: value };
            doSave(next);
            return next;
        });
    }, [doSave]);

    useEffect(() => () => doSave.cancel(), [doSave]);

    if (!ready) return null;

    return (
        <div>
            <h3 className="text-lg font-medium">{t("support.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("support.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <ToggleField label={t("support.support_chatwoot_enabled.title")} description={t("support.support_chatwoot_enabled.description")} checked={values.support_chatwoot_enabled} onChange={(v) => setField("support_chatwoot_enabled", v)} />
                <FormField label={t("support.support_chatwoot_base_url.title")} placeholder={t("support.support_chatwoot_base_url.placeholder")} description={t("support.support_chatwoot_base_url.description")} value={values.support_chatwoot_base_url ?? ""} onChange={(v) => setField("support_chatwoot_base_url", v)} />
                <FormField label={t("support.support_chatwoot_website_token.title")} placeholder={t("support.support_chatwoot_website_token.placeholder")} description={t("support.support_chatwoot_website_token.description")} value={values.support_chatwoot_website_token ?? ""} onChange={(v) => setField("support_chatwoot_website_token", v)} />
                <FormField label={t("support.support_chatwoot_locale.title")} placeholder={t("support.support_chatwoot_locale.placeholder")} description={t("support.support_chatwoot_locale.description")} value={values.support_chatwoot_locale ?? ""} onChange={(v) => setField("support_chatwoot_locale", v)} />
                {saving && <div className="text-sm text-muted-foreground">{t("support.saving")}</div>}
                {error && <div className="text-sm text-destructive">{error}</div>}
            </div>
        </div>
    );
}

function FormField({ label, placeholder, description, value, onChange, as: As = Input, type = "text" }) {
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
