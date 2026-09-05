import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Switch } from "../../components/Switch.jsx";
import { ReferralRecordsPanel } from "./ReferralRecordsPanel.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    invite_force: false, invite_commission: "0", invite_gen_limit: "0",
    invite_never_expire: false, commission_first_time_enable: false,
    commission_auto_check_enable: false, commission_withdraw_limit: "0",
    commission_withdraw_method: ["Alipay", "USDT", "Paypal"],
    withdraw_close_enable: false, commission_distribution_enable: false,
    commission_distribution_l1: 0, commission_distribution_l2: 0, commission_distribution_l3: 0,
    referral_headline: "", referral_subtext: "", referral_duration_percent: "0",
};

export function InviteSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [loaded, setLoaded] = useState(false);
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("invite").then((res) => {
            const inv = res?.data?.invite || res?.data || {};
            const next = {};
            for (const [k, v] of Object.entries(DEFAULTS)) {
                if (k in inv) {
                    next[k] = typeof v === "number" ? String(inv[k]) : inv[k];
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
                await saveSettings(data);
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

    return (
        <div>
            <h3 className="text-lg font-medium">{t("invite.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("invite.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <ToggleField label={t("invite.invite_force.title")} description={t("invite.invite_force.description")} checked={values.invite_force} onChange={(v) => setField("invite_force", v)} />
                <FormField label={t("invite.invite_commission.title")} placeholder={t("invite.invite_commission.placeholder")} description={t("invite.invite_commission.description")} value={values.invite_commission ?? ""} onChange={(v) => setField("invite_commission", v)} />
                <FormField label={t("invite.invite_gen_limit.title")} placeholder={t("invite.invite_gen_limit.placeholder")} description={t("invite.invite_gen_limit.description")} value={values.invite_gen_limit ?? ""} onChange={(v) => setField("invite_gen_limit", v)} />
                <ToggleField label={t("invite.invite_never_expire.title")} description={t("invite.invite_never_expire.description")} checked={values.invite_never_expire} onChange={(v) => setField("invite_never_expire", v)} />
                <ToggleField label={t("invite.commission_first_time.title")} description={t("invite.commission_first_time.description")} checked={values.commission_first_time_enable} onChange={(v) => setField("commission_first_time_enable", v)} />
                <ToggleField label={t("invite.commission_auto_check.title")} description={t("invite.commission_auto_check.description")} checked={values.commission_auto_check_enable} onChange={(v) => setField("commission_auto_check_enable", v)} />
                <FormField label={t("invite.commission_withdraw_limit.title")} placeholder={t("invite.commission_withdraw_limit.placeholder")} description={t("invite.commission_withdraw_limit.description")} value={values.commission_withdraw_limit ?? ""} onChange={(v) => setField("commission_withdraw_limit", v)} />
                <FormField as={Textarea} label={t("invite.commission_withdraw_method.title")} placeholder={t("invite.commission_withdraw_method.placeholder")} description={t("invite.commission_withdraw_method.description")} value={(values.commission_withdraw_method || []).join(",")} onChange={(v) => setField("commission_withdraw_method", v.split(",").map((s) => s.trim()).filter(Boolean))} />
                <ToggleField label={t("invite.withdraw_close.title")} description={t("invite.withdraw_close.description")} checked={values.withdraw_close_enable} onChange={(v) => setField("withdraw_close_enable", v)} />
                <ToggleField label={t("invite.commission_distribution.title")} description={t("invite.commission_distribution.description")} checked={values.commission_distribution_enable} onChange={(v) => setField("commission_distribution_enable", v)} />
                {values.commission_distribution_enable && (
                    <>
                        <FormField label={t("invite.commission_distribution.l1")} placeholder={t("invite.commission_distribution.placeholder")} value={String(values.commission_distribution_l1 ?? 0)} onChange={(v) => setField("commission_distribution_l1", Number(v) || 0)} />
                        <FormField label={t("invite.commission_distribution.l2")} placeholder={t("invite.commission_distribution.placeholder")} value={String(values.commission_distribution_l2 ?? 0)} onChange={(v) => setField("commission_distribution_l2", Number(v) || 0)} />
                        <FormField label={t("invite.commission_distribution.l3")} placeholder={t("invite.commission_distribution.placeholder")} value={String(values.commission_distribution_l3 ?? 0)} onChange={(v) => setField("commission_distribution_l3", Number(v) || 0)} />
                    </>
                )}
                {saving && <div className="text-sm text-muted-foreground">{t("invite.saving")}</div>}
            </div>
            <hr className="my-6" />
            <h3 className="text-lg font-medium">{t("invite.referral_copy.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("invite.referral_copy.description")}</p>
            <div className="space-y-4 mt-4">
                <FormField label={t("invite.referral_headline.title")} placeholder={t("invite.referral_headline.placeholder")} description={t("invite.referral_headline.description")} value={values.referral_headline ?? ""} onChange={(v) => setField("referral_headline", v)} />
                <FormField as={Textarea} label={t("invite.referral_subtext.title")} placeholder={t("invite.referral_subtext.placeholder")} description={t("invite.referral_subtext.description")} value={values.referral_subtext ?? ""} onChange={(v) => setField("referral_subtext", v)} />
                <FormField label={t("invite.referral_duration_percent.title")} placeholder={t("invite.referral_duration_percent.placeholder")} description={t("invite.referral_duration_percent.description")} value={values.referral_duration_percent ?? ""} onChange={(v) => setField("referral_duration_percent", v)} />
            </div>
            <hr className="my-6" />
            <ReferralRecordsPanel />
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
