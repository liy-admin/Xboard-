import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    anti_loss_qr_data: "",
    anti_loss_qr_image_url: "",
    anti_loss_qr_logo_url: "",
    anti_loss_title: "",
    anti_loss_body: "",
    anti_loss_save_button_text: "",
    anti_loss_saved_toast: "",
    anti_loss_follow_text: "",
    anti_loss_socials: [],
};

// 客户端能按名字自动配图标的平台，仅作为管理员填写时的提示，不强制。
const KNOWN_SOCIALS = "telegram / facebook / instagram / x / youtube / tiktok / discord / whatsapp / wechat";

export function AntiLossSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [loaded, setLoaded] = useState(false);
    const [error, setError] = useState("");
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("anti_loss").then((res) => {
            const cfg = res?.data?.anti_loss || res?.data || {};
            const next = { ...DEFAULTS, ...cfg };
            // 后端存的是 JSON，历史脏数据或空值都可能不是数组，这里兜一层。
            if (!Array.isArray(next.anti_loss_socials)) next.anti_loss_socials = [];
            setValues(next);
            lastSavedRef.current = { ...next, anti_loss_socials: [...next.anti_loss_socials] };
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
                lastSavedRef.current = { ...data, anti_loss_socials: [...(data.anti_loss_socials || [])] };
                setError("");
            } catch (e) {
                setError(e?.message || t("anti_loss.save_error"));
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

    const setSocials = useCallback((socials) => setField("anti_loss_socials", socials), [setField]);

    const updateSocial = (index, key, value) => {
        setSocials(values.anti_loss_socials.map((s, i) => (i === index ? { ...s, [key]: value } : s)));
    };
    const addSocial = () => setSocials([...values.anti_loss_socials, { name: "", url: "", icon_url: "" }]);
    const removeSocial = (index) => setSocials(values.anti_loss_socials.filter((_, i) => i !== index));

    useEffect(() => () => doSave.cancel(), [doSave]);

    if (!ready) return null;

    return (
        <div>
            <h3 className="text-lg font-medium">{t("anti_loss.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("anti_loss.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <FormField label={t("anti_loss.anti_loss_qr_data.title")} placeholder={t("anti_loss.anti_loss_qr_data.placeholder")} description={t("anti_loss.anti_loss_qr_data.description")} value={values.anti_loss_qr_data ?? ""} onChange={(v) => setField("anti_loss_qr_data", v)} />
                <FormField label={t("anti_loss.anti_loss_qr_image_url.title")} placeholder={t("anti_loss.anti_loss_qr_image_url.placeholder")} description={t("anti_loss.anti_loss_qr_image_url.description")} value={values.anti_loss_qr_image_url ?? ""} onChange={(v) => setField("anti_loss_qr_image_url", v)} />
                <FormField label={t("anti_loss.anti_loss_qr_logo_url.title")} placeholder={t("anti_loss.anti_loss_qr_logo_url.placeholder")} description={t("anti_loss.anti_loss_qr_logo_url.description")} value={values.anti_loss_qr_logo_url ?? ""} onChange={(v) => setField("anti_loss_qr_logo_url", v)} />
                <FormField label={t("anti_loss.anti_loss_title.title")} placeholder={t("anti_loss.anti_loss_title.placeholder")} value={values.anti_loss_title ?? ""} onChange={(v) => setField("anti_loss_title", v)} />
                <FormField as={Textarea} label={t("anti_loss.anti_loss_body.title")} placeholder={t("anti_loss.anti_loss_body.placeholder")} value={values.anti_loss_body ?? ""} onChange={(v) => setField("anti_loss_body", v)} />
                <FormField label={t("anti_loss.anti_loss_save_button_text.title")} placeholder={t("anti_loss.anti_loss_save_button_text.placeholder")} value={values.anti_loss_save_button_text ?? ""} onChange={(v) => setField("anti_loss_save_button_text", v)} />
                <FormField label={t("anti_loss.anti_loss_saved_toast.title")} placeholder={t("anti_loss.anti_loss_saved_toast.placeholder")} value={values.anti_loss_saved_toast ?? ""} onChange={(v) => setField("anti_loss_saved_toast", v)} />
                <FormField label={t("anti_loss.anti_loss_follow_text.title")} placeholder={t("anti_loss.anti_loss_follow_text.placeholder")} value={values.anti_loss_follow_text ?? ""} onChange={(v) => setField("anti_loss_follow_text", v)} />

                <div className="space-y-2">
                    <div className="text-base font-medium">{t("anti_loss.socials.title")}</div>
                    <p className="text-sm text-muted-foreground">{t("anti_loss.socials.description")}{" "}<span className="font-mono text-xs">{KNOWN_SOCIALS}</span></p>
                    <div className="space-y-2">
                        {values.anti_loss_socials.map((social, index) => (
                            <div key={index} className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center">
                                <Input className="sm:w-40" placeholder={t("anti_loss.socials.name_placeholder")} value={social.name ?? ""} onChange={(e) => updateSocial(index, "name", e.target.value)} />
                                <Input className="flex-1" placeholder={t("anti_loss.socials.url_placeholder")} value={social.url ?? ""} onChange={(e) => updateSocial(index, "url", e.target.value)} />
                                <Input className="flex-1" placeholder={t("anti_loss.socials.icon_placeholder")} value={social.icon_url ?? ""} onChange={(e) => updateSocial(index, "icon_url", e.target.value)} />
                                <button
                                    type="button"
                                    className="inline-flex items-center justify-center whitespace-nowrap rounded-md border border-input px-3 py-2 text-sm font-medium text-destructive hover:bg-accent"
                                    onClick={() => removeSocial(index)}
                                >
                                    {t("anti_loss.socials.remove")}
                                </button>
                            </div>
                        ))}
                    </div>
                    <button
                        type="button"
                        className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md border border-input px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground"
                        onClick={addSocial}
                    >
                        {t("anti_loss.socials.add")}
                    </button>
                </div>

                {saving && <div className="text-sm text-muted-foreground">{t("anti_loss.saving")}</div>}
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
