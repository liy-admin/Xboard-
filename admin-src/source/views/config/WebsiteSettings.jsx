import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Switch } from "../../components/Switch.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    website_brand_name: "",
    website_brand_tagline: "",
    website_brand_description: "",
    website_announcement_enabled: false,
    website_announcement_text: "",
    website_announcement_link_text: "",
    website_announcement_link_href: "",
    website_hero_title: "",
    website_hero_subtitle: "",
    website_hero_stats: [],
    website_cta_title: "",
    website_cta_subtitle: "",
    website_footer_contact: "",
};

export function WebsiteSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [loaded, setLoaded] = useState(false);
    const [error, setError] = useState("");
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("website").then((res) => {
            const cfg = res?.data?.website || res?.data || {};
            const next = { ...DEFAULTS, ...cfg };
            // 后端存的是 JSON，历史脏数据或空值都可能不是数组，这里兜一层。
            if (!Array.isArray(next.website_hero_stats)) next.website_hero_stats = [];
            setValues(next);
            lastSavedRef.current = { ...next, website_hero_stats: [...next.website_hero_stats] };
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
                lastSavedRef.current = { ...data, website_hero_stats: [...(data.website_hero_stats || [])] };
                setError("");
            } catch (e) {
                setError(e?.message || t("website.save_error"));
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

    const setStats = useCallback((stats) => setField("website_hero_stats", stats), [setField]);

    const updateStat = (index, key, value) => {
        setStats(values.website_hero_stats.map((s, i) => (i === index ? { ...s, [key]: value } : s)));
    };
    const addStat = () => setStats([...values.website_hero_stats, { value: "", label: "" }]);
    const removeStat = (index) => setStats(values.website_hero_stats.filter((_, i) => i !== index));

    useEffect(() => () => doSave.cancel(), [doSave]);

    if (!ready) return null;

    return (
        <div>
            <h3 className="text-lg font-medium">{t("website.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("website.description")}</p>
            <hr className="my-6" />
            <div className="space-y-6">
                <Section title={t("website.section_brand")}>
                    <FormField label={t("website.website_brand_name.title")} placeholder={t("website.website_brand_name.placeholder")} description={t("website.website_brand_name.description")} value={values.website_brand_name ?? ""} onChange={(v) => setField("website_brand_name", v)} />
                    <FormField label={t("website.website_brand_tagline.title")} placeholder={t("website.website_brand_tagline.placeholder")} value={values.website_brand_tagline ?? ""} onChange={(v) => setField("website_brand_tagline", v)} />
                    <FormField as={Textarea} label={t("website.website_brand_description.title")} placeholder={t("website.website_brand_description.placeholder")} description={t("website.website_brand_description.description")} value={values.website_brand_description ?? ""} onChange={(v) => setField("website_brand_description", v)} />
                </Section>

                <Section title={t("website.section_announcement")}>
                    <div className="flex items-center justify-between rounded-lg border p-3">
                        <div className="space-y-0.5 pr-4">
                            <div className="text-base font-medium">{t("website.website_announcement_enabled.title")}</div>
                            <p className="text-sm text-muted-foreground">{t("website.website_announcement_enabled.description")}</p>
                        </div>
                        <Switch checked={!!values.website_announcement_enabled} onCheckedChange={(v) => setField("website_announcement_enabled", v)} />
                    </div>
                    <FormField label={t("website.website_announcement_text.title")} placeholder={t("website.website_announcement_text.placeholder")} value={values.website_announcement_text ?? ""} onChange={(v) => setField("website_announcement_text", v)} />
                    <FormField label={t("website.website_announcement_link_text.title")} placeholder={t("website.website_announcement_link_text.placeholder")} value={values.website_announcement_link_text ?? ""} onChange={(v) => setField("website_announcement_link_text", v)} />
                    <FormField label={t("website.website_announcement_link_href.title")} placeholder={t("website.website_announcement_link_href.placeholder")} value={values.website_announcement_link_href ?? ""} onChange={(v) => setField("website_announcement_link_href", v)} />
                </Section>

                <Section title={t("website.section_hero")}>
                    <FormField label={t("website.website_hero_title.title")} placeholder={t("website.website_hero_title.placeholder")} value={values.website_hero_title ?? ""} onChange={(v) => setField("website_hero_title", v)} />
                    <FormField label={t("website.website_hero_subtitle.title")} placeholder={t("website.website_hero_subtitle.placeholder")} value={values.website_hero_subtitle ?? ""} onChange={(v) => setField("website_hero_subtitle", v)} />
                    <div className="space-y-2">
                        <div className="text-base font-medium">{t("website.stats.title")}</div>
                        <p className="text-sm text-muted-foreground">{t("website.stats.description")}</p>
                        <div className="space-y-2">
                            {values.website_hero_stats.map((stat, index) => (
                                <div key={index} className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center">
                                    <Input className="sm:w-40" placeholder={t("website.stats.value_placeholder")} value={stat.value ?? ""} onChange={(e) => updateStat(index, "value", e.target.value)} />
                                    <Input className="flex-1" placeholder={t("website.stats.label_placeholder")} value={stat.label ?? ""} onChange={(e) => updateStat(index, "label", e.target.value)} />
                                    <button
                                        type="button"
                                        className="inline-flex items-center justify-center whitespace-nowrap rounded-md border border-input px-3 py-2 text-sm font-medium text-destructive hover:bg-accent"
                                        onClick={() => removeStat(index)}
                                    >
                                        {t("website.stats.remove")}
                                    </button>
                                </div>
                            ))}
                        </div>
                        <button
                            type="button"
                            className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md border border-input px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground"
                            onClick={addStat}
                        >
                            {t("website.stats.add")}
                        </button>
                    </div>
                </Section>

                <Section title={t("website.section_cta")}>
                    <FormField label={t("website.website_cta_title.title")} placeholder={t("website.website_cta_title.placeholder")} value={values.website_cta_title ?? ""} onChange={(v) => setField("website_cta_title", v)} />
                    <FormField label={t("website.website_cta_subtitle.title")} placeholder={t("website.website_cta_subtitle.placeholder")} value={values.website_cta_subtitle ?? ""} onChange={(v) => setField("website_cta_subtitle", v)} />
                </Section>

                <Section title={t("website.section_footer")}>
                    <FormField label={t("website.website_footer_contact.title")} placeholder={t("website.website_footer_contact.placeholder")} description={t("website.website_footer_contact.description")} value={values.website_footer_contact ?? ""} onChange={(v) => setField("website_footer_contact", v)} />
                </Section>

                {saving && <div className="text-sm text-muted-foreground">{t("website.saving")}</div>}
                {error && <div className="text-sm text-destructive">{error}</div>}
            </div>
        </div>
    );
}

function Section({ title, children }) {
    return (
        <div className="space-y-4">
            <div className="text-base font-semibold">{title}</div>
            {children}
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
