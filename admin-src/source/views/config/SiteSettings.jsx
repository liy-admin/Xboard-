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
import { fetchPlans } from "../../api/plan.js";

const DEFAULTS = {
    app_name: "", app_description: "", app_url: "", force_https: 0,
    logo: "", subscribe_url: "", tos_url: "", stop_register: 0,
    ticket_must_wait_reply: 0, try_out_plan_id: 0, try_out_hour: 0,
    default_device_limit: 4,
    currency: "", currency_symbol: "",
};

export function SiteSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [plans, setPlans] = useState([]);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const lastSavedRef = useRef(null);
    const valuesRef = useRef(values);
    valuesRef.current = values;

    useEffect(() => {
        getSettings("site").then((res) => {
            const site = res?.data?.site || res?.data || {};
            const next = { ...DEFAULTS, ...site };
            setValues(next);
            lastSavedRef.current = { ...next };
            setReady(true);
        }).catch(() => setReady(true));

        fetchPlans().then((res) => setPlans(res?.data || []));
    }, [auth.token]);

    const doSave = useCallback(
        debounce(async (data) => {
            if (isEqual(data, lastSavedRef.current)) return;
            setSaving(true);
            try {
                const payload = {};
                for (const [k, v] of Object.entries(data)) {
                    payload[k] = v === null || v === undefined ? "" : v;
                }
                await saveSettings(payload);
                lastSavedRef.current = { ...data };
            } finally {
                setSaving(false);
            }
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
            <h3 className="text-lg font-medium">{t("site.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("site.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <FormField label={t("site.form.siteName.label")} placeholder={t("site.form.siteName.placeholder")} description={t("site.form.siteName.description")} value={values.app_name ?? ""} onChange={(v) => setField("app_name", v)} />
                <FormField label={t("site.form.siteDescription.label")} placeholder={t("site.form.siteDescription.placeholder")} description={t("site.form.siteDescription.description")} value={values.app_description ?? ""} onChange={(v) => setField("app_description", v)} />
                <FormField label={t("site.form.siteUrl.label")} placeholder={t("site.form.siteUrl.placeholder")} description={t("site.form.siteUrl.description")} value={values.app_url ?? ""} onChange={(v) => setField("app_url", v)} />
                <ToggleField label={t("site.form.forceHttps.label")} description={t("site.form.forceHttps.description")} checked={Boolean(values.force_https)} onChange={(v) => setField("force_https", v ? 1 : 0)} />
                <FormField label={t("site.form.logo.label")} placeholder={t("site.form.logo.placeholder")} description={t("site.form.logo.description")} value={values.logo ?? ""} onChange={(v) => setField("logo", v)} />
                <FormField as={Textarea} label={t("site.form.subscribeUrl.label")} placeholder={t("site.form.subscribeUrl.placeholder")} description={t("site.form.subscribeUrl.description")} value={values.subscribe_url ?? ""} onChange={(v) => setField("subscribe_url", v)} />
                <FormField label={t("site.form.tosUrl.label")} placeholder={t("site.form.tosUrl.placeholder")} description={t("site.form.tosUrl.description")} value={values.tos_url ?? ""} onChange={(v) => setField("tos_url", v)} />
                <ToggleField label={t("site.form.stopRegister.label")} description={t("site.form.stopRegister.description")} checked={Boolean(values.stop_register)} onChange={(v) => setField("stop_register", v ? 1 : 0)} />
                <ToggleField label={t("site.form.ticketMustWaitReply.label")} description={t("site.form.ticketMustWaitReply.description")} checked={Boolean(values.ticket_must_wait_reply)} onChange={(v) => setField("ticket_must_wait_reply", v ? 1 : 0)} />
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("site.form.tryOut.label")}</div>
                    <Select value={String(values.try_out_plan_id ?? "0")} onValueChange={(v) => setField("try_out_plan_id", Number(v))}>
                        <SelectTrigger><SelectValue placeholder={t("site.form.tryOut.placeholder")} /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="0">{t("site.form.tryOut.placeholder")}</SelectItem>
                            {plans.map((p) => <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>)}
                        </SelectContent>
                    </Select>
                    <p className="text-sm text-muted-foreground">{t("site.form.tryOut.description")}</p>
                </div>
                {Number(values.try_out_plan_id) > 0 && (
                    <FormField label={t("site.form.tryOut.duration.label")} placeholder={t("site.form.tryOut.duration.placeholder")} description={t("site.form.tryOut.duration.description")} value={values.try_out_hour ?? ""} onChange={(v) => setField("try_out_hour", v)} />
                )}
                <FormField
                    type="number"
                    label={t("site.form.defaultDeviceLimit.label")}
                    placeholder={t("site.form.defaultDeviceLimit.placeholder")}
                    description={t("site.form.defaultDeviceLimit.description")}
                    value={values.default_device_limit ?? ""}
                    onChange={(v) => setField("default_device_limit", v === "" ? "" : Number(v))}
                />
                <FormField label={t("site.form.currency.label")} placeholder={t("site.form.currency.placeholder")} description={t("site.form.currency.description")} value={values.currency ?? ""} onChange={(v) => setField("currency", v)} />
                <FormField label={t("site.form.currencySymbol.label")} placeholder={t("site.form.currencySymbol.placeholder")} description={t("site.form.currencySymbol.description")} value={values.currency_symbol ?? ""} onChange={(v) => setField("currency_symbol", v)} />
                {saving && <div className="text-sm text-muted-foreground">{t("site.form.saving")}</div>}
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
