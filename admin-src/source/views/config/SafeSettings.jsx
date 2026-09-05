import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    email_verify: false, safe_mode_enable: false, secure_path: "",
    email_whitelist_enable: false, email_whitelist_suffix: [],
    email_gmail_limit_enable: false, captcha_enable: false, captcha_type: "recaptcha",
    recaptcha_key: "", recaptcha_site_key: "",
    recaptcha_v3_secret_key: "", recaptcha_v3_site_key: "", recaptcha_v3_score_threshold: "0.5",
    turnstile_secret_key: "", turnstile_site_key: "",
    register_limit_by_ip_enable: false, register_limit_count: "", register_limit_expire: "",
    password_limit_enable: false, password_limit_count: "", password_limit_expire: "",
};

export function SafeSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [loaded, setLoaded] = useState(false);
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("safe").then((res) => {
            const safe = res?.data?.safe || res?.data || {};
            const next = {};
            for (const [k, v] of Object.entries(DEFAULTS)) {
                if (k in safe) {
                    next[k] = typeof v === "number" ? String(safe[k]) : safe[k];
                } else {
                    next[k] = v;
                }
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
                const payload = { ...data, email_whitelist_suffix: (data.email_whitelist_suffix || []).filter(Boolean) };
                await saveSettings(payload);
                lastSavedRef.current = { ...data };
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

    const captchaType = values.captcha_type;

    return (
        <div>
            <h3 className="text-lg font-medium">{t("safe.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("safe.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <ToggleField label={t("safe.form.emailVerify.label")} description={t("safe.form.emailVerify.description")} checked={values.email_verify} onChange={(v) => setField("email_verify", v)} />
                <ToggleField label={t("safe.form.gmailLimit.label")} description={t("safe.form.gmailLimit.description")} checked={values.email_gmail_limit_enable} onChange={(v) => setField("email_gmail_limit_enable", v)} />
                <ToggleField label={t("safe.form.safeMode.label")} description={t("safe.form.safeMode.description")} checked={values.safe_mode_enable} onChange={(v) => setField("safe_mode_enable", v)} />
                <FormField label={t("safe.form.securePath.label")} placeholder={t("safe.form.securePath.placeholder")} description={t("safe.form.securePath.description")} value={values.secure_path ?? ""} onChange={(v) => setField("secure_path", v)} />
                <ToggleField label={t("safe.form.emailWhitelist.label")} description={t("safe.form.emailWhitelist.description")} checked={values.email_whitelist_enable} onChange={(v) => setField("email_whitelist_enable", v)} />
                {values.email_whitelist_enable && (
                    <FormField as={Textarea} label={t("safe.form.emailWhitelist.suffixes.label")} placeholder={t("safe.form.emailWhitelist.suffixes.placeholder")} description={t("safe.form.emailWhitelist.suffixes.description")} value={(values.email_whitelist_suffix || []).join("\n")} onChange={(v) => setField("email_whitelist_suffix", v.split("\n").filter(Boolean))} />
                )}
                <ToggleField label={t("safe.form.captcha.enable.label")} description={t("safe.form.captcha.enable.description")} checked={values.captcha_enable} onChange={(v) => setField("captcha_enable", v)} />
                {values.captcha_enable && (
                    <>
                        <div className="space-y-1.5">
                            <div className="text-base font-medium">{t("safe.form.captcha.type.label")}</div>
                            <Select value={captchaType || "recaptcha"} onValueChange={(v) => setField("captcha_type", v)}>
                                <SelectTrigger><SelectValue placeholder={t("safe.form.captcha.type.description")} /></SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="recaptcha">{t("safe.form.captcha.type.options.recaptcha")}</SelectItem>
                                    <SelectItem value="recaptcha-v3">{t("safe.form.captcha.type.options.recaptcha-v3")}</SelectItem>
                                    <SelectItem value="turnstile">{t("safe.form.captcha.type.options.turnstile")}</SelectItem>
                                </SelectContent>
                            </Select>
                            <p className="text-sm text-muted-foreground">{t("safe.form.captcha.type.description")}</p>
                        </div>
                        {captchaType === "recaptcha" && (
                            <>
                                <FormField label={t("safe.form.captcha.recaptcha.siteKey.label")} placeholder={t("safe.form.captcha.recaptcha.siteKey.placeholder")} description={t("safe.form.captcha.recaptcha.siteKey.description")} value={values.recaptcha_site_key ?? ""} onChange={(v) => setField("recaptcha_site_key", v)} />
                                <FormField label={t("safe.form.captcha.recaptcha.key.label")} placeholder={t("safe.form.captcha.recaptcha.key.placeholder")} description={t("safe.form.captcha.recaptcha.key.description")} value={values.recaptcha_key ?? ""} onChange={(v) => setField("recaptcha_key", v)} />
                            </>
                        )}
                        {captchaType === "recaptcha-v3" && (
                            <>
                                <FormField label={t("safe.form.captcha.recaptcha_v3.siteKey.label")} placeholder={t("safe.form.captcha.recaptcha_v3.siteKey.placeholder")} description={t("safe.form.captcha.recaptcha_v3.siteKey.description")} value={values.recaptcha_v3_site_key ?? ""} onChange={(v) => setField("recaptcha_v3_site_key", v)} />
                                <FormField label={t("safe.form.captcha.recaptcha_v3.secretKey.label")} placeholder={t("safe.form.captcha.recaptcha_v3.secretKey.placeholder")} description={t("safe.form.captcha.recaptcha_v3.secretKey.description")} value={values.recaptcha_v3_secret_key ?? ""} onChange={(v) => setField("recaptcha_v3_secret_key", v)} />
                                <FormField type="number" step="0.1" min="0" max="1" label={t("safe.form.captcha.recaptcha_v3.scoreThreshold.label")} placeholder={t("safe.form.captcha.recaptcha_v3.scoreThreshold.placeholder")} description={t("safe.form.captcha.recaptcha_v3.scoreThreshold.description")} value={values.recaptcha_v3_score_threshold ?? ""} onChange={(v) => setField("recaptcha_v3_score_threshold", v)} />
                            </>
                        )}
                        {captchaType === "turnstile" && (
                            <>
                                <FormField label={t("safe.form.captcha.turnstile.siteKey.label")} placeholder={t("safe.form.captcha.turnstile.siteKey.placeholder")} description={t("safe.form.captcha.turnstile.siteKey.description")} value={values.turnstile_site_key ?? ""} onChange={(v) => setField("turnstile_site_key", v)} />
                                <FormField label={t("safe.form.captcha.turnstile.secretKey.label")} placeholder={t("safe.form.captcha.turnstile.secretKey.placeholder")} description={t("safe.form.captcha.turnstile.secretKey.description")} value={values.turnstile_secret_key ?? ""} onChange={(v) => setField("turnstile_secret_key", v)} />
                            </>
                        )}
                    </>
                )}
                <ToggleField label={t("safe.form.registerLimit.enable.label")} description={t("safe.form.registerLimit.enable.description")} checked={values.register_limit_by_ip_enable} onChange={(v) => setField("register_limit_by_ip_enable", v)} />
                {values.register_limit_by_ip_enable && (
                    <>
                        <FormField label={t("safe.form.registerLimit.count.label")} placeholder={t("safe.form.registerLimit.count.placeholder")} description={t("safe.form.registerLimit.count.description")} value={values.register_limit_count ?? ""} onChange={(v) => setField("register_limit_count", v)} />
                        <FormField label={t("safe.form.registerLimit.expire.label")} placeholder={t("safe.form.registerLimit.expire.placeholder")} description={t("safe.form.registerLimit.expire.description")} value={values.register_limit_expire ?? ""} onChange={(v) => setField("register_limit_expire", v)} />
                    </>
                )}
                <ToggleField label={t("safe.form.passwordLimit.enable.label")} description={t("safe.form.passwordLimit.enable.description")} checked={values.password_limit_enable} onChange={(v) => setField("password_limit_enable", v)} />
                {values.password_limit_enable && (
                    <>
                        <FormField label={t("safe.form.passwordLimit.count.label")} placeholder={t("safe.form.passwordLimit.count.placeholder")} description={t("safe.form.passwordLimit.count.description")} value={values.password_limit_count ?? ""} onChange={(v) => setField("password_limit_count", v)} />
                        <FormField label={t("safe.form.passwordLimit.expire.label")} placeholder={t("safe.form.passwordLimit.expire.placeholder")} description={t("safe.form.passwordLimit.expire.description")} value={values.password_limit_expire ?? ""} onChange={(v) => setField("password_limit_expire", v)} />
                    </>
                )}
                {saving && <div className="text-sm text-muted-foreground">{t("safe.form.saving")}</div>}
            </div>
        </div>
    );
}

function FormField({ label, placeholder, description, value, onChange, as: As = Input, type = "text", ...rest }) {
    return (
        <div className="space-y-1.5">
            {label && <div className="text-base font-medium">{label}</div>}
            <As type={type} placeholder={placeholder} value={value} onChange={(e) => onChange(e.target.value)} {...rest} />
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
