import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Switch } from "../../components/Switch.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    ad_banner_enabled: false,
    ad_banner_image_url: "",
    ad_banner_click_url: "",
    ad_banner_countdown_seconds: "5",
};

export function AdBannerSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [loaded, setLoaded] = useState(false);
    const [error, setError] = useState("");
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("ad_banner").then((res) => {
            const cfg = res?.data?.ad_banner || res?.data || {};
            const next = { ...DEFAULTS, ...cfg };
            if (next.ad_banner_countdown_seconds !== undefined && next.ad_banner_countdown_seconds !== null) {
                next.ad_banner_countdown_seconds = String(next.ad_banner_countdown_seconds);
            }
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
                setError(e?.message || t("ad_banner.save_error"));
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
            <h3 className="text-lg font-medium">{t("ad_banner.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("ad_banner.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <ToggleField label={t("ad_banner.ad_banner_enabled.title")} description={t("ad_banner.ad_banner_enabled.description")} checked={values.ad_banner_enabled} onChange={(v) => setField("ad_banner_enabled", v)} />
                <FormField label={t("ad_banner.ad_banner_image_url.title")} placeholder={t("ad_banner.ad_banner_image_url.placeholder")} description={t("ad_banner.ad_banner_image_url.description")} value={values.ad_banner_image_url ?? ""} onChange={(v) => setField("ad_banner_image_url", v)} />
                <FormField label={t("ad_banner.ad_banner_click_url.title")} placeholder={t("ad_banner.ad_banner_click_url.placeholder")} description={t("ad_banner.ad_banner_click_url.description")} value={values.ad_banner_click_url ?? ""} onChange={(v) => setField("ad_banner_click_url", v)} />
                <FormField type="number" label={t("ad_banner.ad_banner_countdown_seconds.title")} placeholder={t("ad_banner.ad_banner_countdown_seconds.placeholder")} description={t("ad_banner.ad_banner_countdown_seconds.description")} value={values.ad_banner_countdown_seconds ?? ""} onChange={(v) => setField("ad_banner_countdown_seconds", v)} />
                {saving && <div className="text-sm text-muted-foreground">{t("ad_banner.saving")}</div>}
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
