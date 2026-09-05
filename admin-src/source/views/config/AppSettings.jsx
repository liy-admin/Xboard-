import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Switch } from "../../components/Switch.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    windows_version: "", windows_download_url: "", windows_force_update: false,
    macos_version: "", macos_download_url: "", macos_force_update: false,
    android_version: "", android_download_url: "", android_force_update: false,
};

export function AppSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("app").then((res) => {
            const app = res?.data?.app || res?.data || {};
            const next = { ...DEFAULTS, ...app };
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

    if (!ready) return null;

    return (
        <div>
            <h3 className="text-lg font-medium">{t("app.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("app.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <FormField label={t("app.windows.version.title")} placeholder={t("common.placeholder")} description={t("app.windows.version.description")} value={values.windows_version ?? ""} onChange={(v) => setField("windows_version", v)} />
                <FormField label={t("app.windows.download.title")} placeholder={t("common.placeholder")} description={t("app.windows.download.description")} value={values.windows_download_url ?? ""} onChange={(v) => setField("windows_download_url", v)} />
                <SwitchField label={t("app.windows.force_update.title")} description={t("app.windows.force_update.description")} checked={!!values.windows_force_update} onChange={(v) => setField("windows_force_update", v)} />
                <FormField label={t("app.macos.version.title")} placeholder={t("common.placeholder")} description={t("app.macos.version.description")} value={values.macos_version ?? ""} onChange={(v) => setField("macos_version", v)} />
                <FormField label={t("app.macos.download.title")} placeholder={t("common.placeholder")} description={t("app.macos.download.description")} value={values.macos_download_url ?? ""} onChange={(v) => setField("macos_download_url", v)} />
                <SwitchField label={t("app.macos.force_update.title")} description={t("app.macos.force_update.description")} checked={!!values.macos_force_update} onChange={(v) => setField("macos_force_update", v)} />
                <FormField label={t("app.android.version.title")} placeholder={t("common.placeholder")} description={t("app.android.version.description")} value={values.android_version ?? ""} onChange={(v) => setField("android_version", v)} />
                <FormField label={t("app.android.download.title")} placeholder={t("common.placeholder")} description={t("app.android.download.description")} value={values.android_download_url ?? ""} onChange={(v) => setField("android_download_url", v)} />
                <SwitchField label={t("app.android.force_update.title")} description={t("app.android.force_update.description")} checked={!!values.android_force_update} onChange={(v) => setField("android_force_update", v)} />
                {saving && <div className="text-sm text-muted-foreground">Đang lưu...</div>}
            </div>
        </div>
    );
}

function FormField({ label, placeholder, description, value, onChange, as: As = Input }) {
    return (
        <div className="space-y-1.5">
            {label && <div className="text-base font-medium">{label}</div>}
            <As placeholder={placeholder} value={value} onChange={(e) => onChange(e.target.value)} />
            {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
    );
}

function SwitchField({ label, description, checked, onChange }) {
    return (
        <div className="flex items-center justify-between gap-4 py-1">
            <div className="space-y-0.5">
                {label && <div className="text-base font-medium">{label}</div>}
                {description && <p className="text-sm text-muted-foreground">{description}</p>}
            </div>
            <Switch checked={checked} onCheckedChange={onChange} />
        </div>
    );
}
