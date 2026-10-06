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
    ios_version: "", ios_download_url: "", ios_force_update: false,
    // 更新文案（通用 + 各平台覆盖）：留空用 App 内置文案；平台专属优先于通用。
    update_title: "", update_message: "", update_button: "", update_changelog: "",
    windows_update_title: "", windows_update_message: "", windows_update_button: "", windows_update_changelog: "",
    macos_update_title: "", macos_update_message: "", macos_update_button: "", macos_update_changelog: "",
    android_update_title: "", android_update_message: "", android_update_button: "", android_update_changelog: "",
    ios_update_title: "", ios_update_message: "", ios_update_button: "", ios_update_changelog: "",
};

// 更新文案分组：通用先，其后各平台覆盖。key 前缀拼上 _title/_message/_button/_changelog
// 即为后端 admin_setting 的键名（与客户端 getVersion 读取的键一一对应）。
const UPDATE_COPY_GROUPS = [
    { prefix: "update", label: "通用（所有平台）" },
    { prefix: "windows_update", label: "Windows 覆盖" },
    { prefix: "macos_update", label: "macOS 覆盖" },
    { prefix: "android_update", label: "Android 覆盖" },
    { prefix: "ios_update", label: "iOS 覆盖" },
];

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
                <FormField label={t("app.ios.version.title")} placeholder={t("common.placeholder")} description={t("app.ios.version.description")} value={values.ios_version ?? ""} onChange={(v) => setField("ios_version", v)} />
                <FormField label={t("app.ios.download.title")} placeholder={t("common.placeholder")} description={t("app.ios.download.description")} value={values.ios_download_url ?? ""} onChange={(v) => setField("ios_download_url", v)} />
                <SwitchField label={t("app.ios.force_update.title")} description={t("app.ios.force_update.description")} checked={!!values.ios_force_update} onChange={(v) => setField("ios_force_update", v)} />

                <hr className="my-6" />
                <h3 className="text-lg font-medium">更新文案</h3>
                <p className="text-sm text-muted-foreground">
                    自定义（强制）更新界面的标题、说明、按钮文字与更新日志。各项留空时使用 App 内置文案；
                    填写了某平台的「覆盖」会优先于「通用」。
                </p>
                {UPDATE_COPY_GROUPS.map(({ prefix, label }) => (
                    <div key={prefix} className="space-y-4 rounded-md border p-4">
                        <div className="text-base font-semibold">{label}</div>
                        <FormField label="标题" placeholder="例如：发现重要更新" value={values[`${prefix}_title`] ?? ""} onChange={(v) => setField(`${prefix}_title`, v)} />
                        <FormField label="说明文案" placeholder="例如：本次为安全性修复，请立即更新后继续使用。" value={values[`${prefix}_message`] ?? ""} onChange={(v) => setField(`${prefix}_message`, v)} />
                        <FormField label="按钮文字" placeholder="例如：立即升级" value={values[`${prefix}_button`] ?? ""} onChange={(v) => setField(`${prefix}_button`, v)} />
                        <FormField label="更新日志" placeholder="本次更新内容（可多行）" value={values[`${prefix}_changelog`] ?? ""} onChange={(v) => setField(`${prefix}_changelog`, v)} />
                    </div>
                ))}
                {saving &&<div className="text-sm text-muted-foreground">Đang lưu...</div>}
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
