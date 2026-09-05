import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Button } from "../../components/Button.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../../components/Tooltip.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    server_pull_interval: 0, server_push_interval: 0, server_token: "",
    device_limit_mode: 0, server_ws_enable: false, server_ws_url: "",
};

function generateToken() {
    const len = Math.floor(Math.random() * 17) + 16;
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    let token = "";
    for (let i = 0; i < len; i++) {
        token += chars.charAt(Math.floor(Math.random() * 62));
    }
    return token;
}

export function ServerSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const lastSavedRef = useRef(null);

    useEffect(() => {
        Promise.all([
            getSettings("server"),
            getSettings("site"),
        ]).then(([serverRes, siteRes]) => {
            const server = serverRes?.data?.server || serverRes?.data || {};
            const siteUrl = siteRes?.data?.site?.app_url || "";
            const next = { ...DEFAULTS, ...server };
            if (!next.server_ws_url && siteUrl) {
                next.server_ws_url = siteUrl;
            }
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
            <h3 className="text-lg font-medium">{t("server.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("server.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("server.server_token.title")}</div>
                    <div className="relative">
                        <Input
                            placeholder={t("server.server_token.placeholder")}
                            value={values.server_token ?? ""}
                            onChange={(e) => setField("server_token", e.target.value)}
                            className="pr-10"
                        />
                        <TooltipProvider>
                            <Tooltip>
                                <TooltipTrigger asChild>
                                    <Button
                                        type="button"
                                        variant="ghost"
                                        size="icon"
                                        className="absolute right-0 top-0 h-full px-3 py-2"
                                        onClick={(e) => {
                                            e.preventDefault();
                                            setField("server_token", generateToken());
                                        }}
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground hover:text-foreground">
                                            <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" /><path d="M21 3v5h-5" />
                                        </svg>
                                    </Button>
                                </TooltipTrigger>
                                <TooltipContent>{t("server.server_token.generate_tooltip")}</TooltipContent>
                            </Tooltip>
                        </TooltipProvider>
                    </div>
                    <p className="text-sm text-muted-foreground">{t("server.server_token.description")}</p>
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("server.server_pull_interval.title")}</div>
                    <Input type="number" placeholder={t("server.server_pull_interval.placeholder")} value={values.server_pull_interval ?? ""} onChange={(e) => setField("server_pull_interval", e.target.value ? Number(e.target.value) : null)} />
                    <p className="text-sm text-muted-foreground">{t("server.server_pull_interval.description")}</p>
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("server.server_push_interval.title")}</div>
                    <Input type="number" placeholder={t("server.server_push_interval.placeholder")} value={values.server_push_interval ?? ""} onChange={(e) => setField("server_push_interval", e.target.value ? Number(e.target.value) : null)} />
                    <p className="text-sm text-muted-foreground">{t("server.server_push_interval.description")}</p>
                </div>
                <ToggleField label={t("server.server_ws_enable.title")} description={t("server.server_ws_enable.description")} checked={values.server_ws_enable ?? false} onChange={(v) => setField("server_ws_enable", v)} />
                {values.server_ws_enable && (
                    <div className="space-y-1.5">
                        <div className="text-base font-medium">{t("server.server_ws_url.title")}</div>
                        <Input placeholder={t("server.server_ws_url.placeholder")} value={values.server_ws_url ?? ""} onChange={(e) => setField("server_ws_url", e.target.value)} />
                        <p className="text-sm text-muted-foreground">{t("server.server_ws_url.description")}</p>
                    </div>
                )}
                <div className="flex items-start gap-2 rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-950/40 dark:text-blue-400">
                    <svg className="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 24 24" width="1.2em" height="1.2em"><path fill="currentColor" d="M12 2a10 10 0 1 0 10 10A10.01 10.01 0 0 0 12 2m0 18a8 8 0 1 1 8-8a8.01 8.01 0 0 1-8 8m0-8.5a1 1 0 0 0-1 1v3a1 1 0 0 0 2 0v-3a1 1 0 0 0-1-1m0-4a1.25 1.25 0 1 0 1.25 1.25A1.25 1.25 0 0 0 12 7.5" /></svg>
                    <span>{t("server.server_ws_enable.supported_clients")}</span>
                </div>
                {saving && <div className="text-sm text-muted-foreground">{t("server.saving")}</div>}
            </div>
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
