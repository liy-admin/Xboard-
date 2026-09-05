import React, { useCallback, useEffect, useRef, useState } from "react";
import Editor from "@monaco-editor/react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/Tabs.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const TEMPLATE_TABS = [
    { key: "singbox", label: "Sing-box", language: "json" },
    { key: "clash", label: "Clash", language: "yaml" },
    { key: "clashmeta", label: "Clash Meta", language: "yaml" },
    { key: "stash", label: "Stash", language: "yaml" },
    { key: "surge", label: "Surge", language: "ini" },
    { key: "surfboard", label: "Surfboard", language: "ini" },
];

const TEMPLATE_KEYS = TEMPLATE_TABS.map(({ key }) => `subscribe_template_${key}`);

const DEFAULTS = Object.fromEntries(TEMPLATE_KEYS.map((k) => [k, ""]));

export function SubscribeTemplateSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const [activeTab, setActiveTab] = useState("singbox");
    const lastSavedRef = useRef(null);
    const valuesRef = useRef(values);

    useEffect(() => {
        getSettings("subscribe_template").then((res) => {
            const templates = res?.data?.subscribe_template || res?.data || {};
            const next = { ...DEFAULTS };
            for (const key of TEMPLATE_KEYS) {
                if (typeof templates[key] === "string") {
                    next[key] = templates[key];
                }
            }
            setValues(next);
            valuesRef.current = next;
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
            } finally {
                setSaving(false);
            }
        }, 1500),
        [auth.token],
    );

    useEffect(() => () => doSave.cancel(), [doSave]);

    const setField = useCallback((name, value) => {
        setValues((prev) => {
            const next = { ...prev, [name]: value || "" };
            valuesRef.current = next;
            doSave(next);
            return next;
        });
    }, [doSave]);

    const currentTab = TEMPLATE_TABS.find((t) => t.key === activeTab) || TEMPLATE_TABS[0];

    if (!ready) {
        return (
            <div className="flex items-center justify-center py-16">
                <div className="text-sm text-muted-foreground">{t("common.loading")}</div>
            </div>
        );
    }

    return (
        <div>
            <h3 className="text-lg font-medium">{t("subscribe_template.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("subscribe_template.description")}</p>
            <hr className="my-6" />
            <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
                <TabsList className="w-full">
                    {TEMPLATE_TABS.map(({ key, label }) => (
                        <TabsTrigger key={key} value={key} className="text-xs">
                            {label}
                        </TabsTrigger>
                    ))}
                </TabsList>
                {TEMPLATE_TABS.map(({ key, language }) => {
                    const fieldName = `subscribe_template_${key}`;
                    return (
                        <TabsContent key={key} value={key} className="mt-4 space-y-4">
                            <div>
                                <div className="text-base font-medium">{t(`subscribe_template.${key}.title`)}</div>
                                <div className="mt-2">
                                    <Editor
                                        height="500px"
                                        defaultLanguage={language}
                                        value={values[fieldName] || ""}
                                        onChange={(v) => setField(fieldName, v)}
                                        options={{
                                            minimap: { enabled: false },
                                            fontSize: 14,
                                            wordWrap: "on",
                                            scrollBeyondLastLine: false,
                                            automaticLayout: true,
                                        }}
                                    />
                                </div>
                                <p className="mt-1 text-sm text-muted-foreground">{t(`subscribe_template.${key}.description`)}</p>
                            </div>
                        </TabsContent>
                    );
                })}
            </Tabs>
            {saving && (
                <div className="mt-4 flex items-center gap-2 text-sm text-muted-foreground">
                    <div className="h-2 w-2 animate-pulse rounded-full bg-blue-500" />
                    Đang lưu...
                </div>
            )}
        </div>
    );
}
