import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings, uploadDiscoverHtml } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    discover_connected_page_url: "",
    discover_prompt_image_url: "",
    discover_prompt_badge_text: "",
    discover_prompt_body: "",
    discover_prompt_button_text: "",
    discover_prompt_cancel_text: "",
    discover_nav_page_url: "",
};

export function DiscoverSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [loaded, setLoaded] = useState(false);
    const [error, setError] = useState("");
    const [uploading, setUploading] = useState(false);
    const lastSavedRef = useRef(null);
    const fileInputRef = useRef(null);

    useEffect(() => {
        getSettings("discover").then((res) => {
            const cfg = res?.data?.discover || res?.data || {};
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
                setError(e?.message || t("discover.save_error"));
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

    const handleFileSelected = async (e) => {
        const file = e.target.files?.[0];
        e.target.value = "";
        if (!file) return;
        setUploading(true);
        setError("");
        try {
            const res = await uploadDiscoverHtml(file);
            const url = res?.data?.url;
            if (url) {
                // 上传接口后端已经直接写库了，这里只是同步本地状态/已保存
                // 基线，避免随后触发的自动保存把同一个值又提交一次。
                setValues((prev) => {
                    const next = { ...prev, discover_connected_page_url: url };
                    lastSavedRef.current = { ...next };
                    return next;
                });
            }
        } catch (err) {
            setError(err?.message || t("discover.upload_error"));
        } finally {
            setUploading(false);
        }
    };

    if (!ready) return null;

    return (
        <div>
            <h3 className="text-lg font-medium">{t("discover.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("discover.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("discover.discover_connected_page_url.title")}</div>
                    <div className="flex gap-2">
                        <Input
                            value={values.discover_connected_page_url ?? ""}
                            placeholder={t("discover.discover_connected_page_url.placeholder")}
                            onChange={(e) => setField("discover_connected_page_url", e.target.value)}
                        />
                        <button
                            type="button"
                            className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md border border-input px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
                            disabled={uploading}
                            onClick={() => fileInputRef.current?.click()}
                        >
                            {uploading ? t("discover.uploading") : t("discover.upload_html")}
                        </button>
                        <input
                            ref={fileInputRef}
                            type="file"
                            accept=".html,.htm"
                            className="hidden"
                            onChange={handleFileSelected}
                        />
                    </div>
                    <p className="text-sm text-muted-foreground">{t("discover.discover_connected_page_url.description")}</p>
                </div>
                <FormField label={t("discover.discover_prompt_image_url.title")} placeholder={t("discover.discover_prompt_image_url.placeholder")} description={t("discover.discover_prompt_image_url.description")} value={values.discover_prompt_image_url ?? ""} onChange={(v) => setField("discover_prompt_image_url", v)} />
                <FormField label={t("discover.discover_prompt_badge_text.title")} placeholder={t("discover.discover_prompt_badge_text.placeholder")} value={values.discover_prompt_badge_text ?? ""} onChange={(v) => setField("discover_prompt_badge_text", v)} />
                <FormField as={Textarea} label={t("discover.discover_prompt_body.title")} placeholder={t("discover.discover_prompt_body.placeholder")} value={values.discover_prompt_body ?? ""} onChange={(v) => setField("discover_prompt_body", v)} />
                <FormField label={t("discover.discover_prompt_button_text.title")} placeholder={t("discover.discover_prompt_button_text.placeholder")} value={values.discover_prompt_button_text ?? ""} onChange={(v) => setField("discover_prompt_button_text", v)} />
                <FormField label={t("discover.discover_prompt_cancel_text.title")} placeholder={t("discover.discover_prompt_cancel_text.placeholder")} value={values.discover_prompt_cancel_text ?? ""} onChange={(v) => setField("discover_prompt_cancel_text", v)} />
                <FormField label={t("discover.discover_nav_page_url.title")} placeholder={t("discover.discover_nav_page_url.placeholder")} description={t("discover.discover_nav_page_url.description")} value={values.discover_nav_page_url ?? ""} onChange={(v) => setField("discover_nav_page_url", v)} />
                {saving && <div className="text-sm text-muted-foreground">{t("discover.saving")}</div>}
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
