import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Switch } from "../../components/Switch.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    lucky_draw_enabled: true,
    lucky_draw_cooldown_seconds: "3600",
    lucky_draw_reward_hours: "1",
    lucky_draw_daily_target_winners: "500",
    lucky_draw_min_probability_percent: "1",
    lucky_draw_max_probability_percent: "50",
};

export function LuckyDrawSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [loaded, setLoaded] = useState(false);
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("lucky_draw").then((res) => {
            const cfg = res?.data?.lucky_draw || res?.data || {};
            const next = {};
            for (const [k, v] of Object.entries(DEFAULTS)) {
                if (k in cfg) {
                    next[k] = typeof v === "number" || typeof v === "string" && v !== "" && !isNaN(v)
                        ? String(cfg[k])
                        : cfg[k];
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
            <h3 className="text-lg font-medium">{t("lucky_draw.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("lucky_draw.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <ToggleField label={t("lucky_draw.lucky_draw_enabled.title")} description={t("lucky_draw.lucky_draw_enabled.description")} checked={values.lucky_draw_enabled} onChange={(v) => setField("lucky_draw_enabled", v)} />
                <FormField type="number" label={t("lucky_draw.lucky_draw_cooldown_seconds.title")} placeholder={t("lucky_draw.lucky_draw_cooldown_seconds.placeholder")} description={t("lucky_draw.lucky_draw_cooldown_seconds.description")} value={values.lucky_draw_cooldown_seconds ?? ""} onChange={(v) => setField("lucky_draw_cooldown_seconds", v)} />
                <FormField type="number" label={t("lucky_draw.lucky_draw_reward_hours.title")} placeholder={t("lucky_draw.lucky_draw_reward_hours.placeholder")} description={t("lucky_draw.lucky_draw_reward_hours.description")} value={values.lucky_draw_reward_hours ?? ""} onChange={(v) => setField("lucky_draw_reward_hours", v)} />
                <FormField type="number" label={t("lucky_draw.lucky_draw_daily_target_winners.title")} placeholder={t("lucky_draw.lucky_draw_daily_target_winners.placeholder")} description={t("lucky_draw.lucky_draw_daily_target_winners.description")} value={values.lucky_draw_daily_target_winners ?? ""} onChange={(v) => setField("lucky_draw_daily_target_winners", v)} />
                <FormField type="number" label={t("lucky_draw.lucky_draw_min_probability_percent.title")} placeholder={t("lucky_draw.lucky_draw_min_probability_percent.placeholder")} description={t("lucky_draw.lucky_draw_min_probability_percent.description")} value={values.lucky_draw_min_probability_percent ?? ""} onChange={(v) => setField("lucky_draw_min_probability_percent", v)} />
                <FormField type="number" label={t("lucky_draw.lucky_draw_max_probability_percent.title")} placeholder={t("lucky_draw.lucky_draw_max_probability_percent.placeholder")} description={t("lucky_draw.lucky_draw_max_probability_percent.description")} value={values.lucky_draw_max_probability_percent ?? ""} onChange={(v) => setField("lucky_draw_max_probability_percent", v)} />
                {saving && <div className="text-sm text-muted-foreground">{t("lucky_draw.saving")}</div>}
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
