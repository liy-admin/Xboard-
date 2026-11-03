import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings, diagnoseProbe, startSingboxDownload, getSingboxDownloadStatus, getCountriesStatus, refreshCountries } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Button } from "../../components/Button.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../../components/Tooltip.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    server_pull_interval: 0, server_push_interval: 0, server_token: "",
    device_limit_mode: 0, server_ws_enable: false, server_ws_url: "", client_ws_enable: false,
    server_latency_probe_enable: true, server_latency_probe_interval: 3, server_latency_timeout_ms: 2000,
    server_latency_stale_seconds: 600, server_dedup_by_country: false,
    server_country_naming: true, server_auto_manage_visibility: false,
    geoip_method: "api", geoip_maxmind_db_path: "",
    country_refresh_time: "04:00", country_tags: "",
    node_probe_mode: "singbox", node_probe_singbox_path: "",
    node_probe_singbox_auto_download: true, node_probe_singbox_version: "1.14.2",
    node_probe_ip_echo_url: "http://ip-api.com/json/?fields=status,countryCode,query",
    node_probe_latency_url: "https://www.gstatic.com/generate_204",
    node_probe_test_count: 3, node_probe_request_timeout_ms: 10000,
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
    const [diagLoading, setDiagLoading] = useState(false);
    const [diagResult, setDiagResult] = useState(null);

    const [dl, setDl] = useState({ state: "idle" });
    const downloading = dl.state === "downloading";

    const runDownload = async () => {
        try {
            const res = await startSingboxDownload((values.node_probe_singbox_version || "").trim());
            setDl(res?.data ?? { state: "downloading" });
        } catch (e) {
            setDl({ state: "error", message: e?.response?.data?.message || e?.message || "请求失败" });
        }
    };

    // 页面打开时同步一次状态（刷新页面后仍能看到进行中的下载）；下载中每秒轮询，完成后自动重新诊断。
    useEffect(() => {
        getSingboxDownloadStatus().then((res) => res?.data && setDl(res.data)).catch(() => {});
    }, []);
    useEffect(() => {
        if (!downloading) return undefined;
        const timer = setInterval(async () => {
            try {
                const res = await getSingboxDownloadStatus();
                if (res?.data) {
                    setDl(res.data);
                    if (res.data.state === "done") runDiagnose();
                }
            } catch (e) { /* 下一轮再试 */ }
        }, 1000);
        return () => clearInterval(timer);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [downloading]);

    const runDiagnose = async () => {
        setDiagLoading(true);
        setDiagResult(null);
        try {
            const res = await diagnoseProbe();
            setDiagResult(res?.data ?? { error: "no data" });
        } catch (e) {
            setDiagResult({ requestError: e?.response?.data?.message || e?.message || "请求失败" });
        } finally {
            setDiagLoading(false);
        }
    };

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

                <ToggleField label={t("server.client_ws_enable.title")} description={t("server.client_ws_enable.description")} checked={values.client_ws_enable ?? false} onChange={(v) => setField("client_ws_enable", v)} />

                <hr className="my-6" />
                <div className="text-base font-semibold">{t("server.latency_section")}</div>

                <ToggleField label={t("server.server_latency_probe_enable.title")} description={t("server.server_latency_probe_enable.description")} checked={values.server_latency_probe_enable ?? true} onChange={(v) => setField("server_latency_probe_enable", v)} />
                {values.server_latency_probe_enable && (
                    <>
                        <div className="space-y-1.5">
                            <div className="text-base font-medium">{t("server.server_latency_probe_interval.title")}</div>
                            <Input type="number" placeholder="3" value={values.server_latency_probe_interval ?? ""} onChange={(e) => setField("server_latency_probe_interval", e.target.value ? Number(e.target.value) : null)} />
                            <p className="text-sm text-muted-foreground">{t("server.server_latency_probe_interval.description")}</p>
                        </div>
                        <div className="space-y-1.5">
                            <div className="text-base font-medium">{t("server.node_probe_mode.title")}</div>
                            <select
                                className="inline-flex h-9 w-full items-center rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm"
                                value={values.node_probe_mode ?? "tcp"}
                                onChange={(e) => setField("node_probe_mode", e.target.value)}
                            >
                                <option value="tcp">{t("server.node_probe_mode.tcp")}</option>
                                <option value="singbox">{t("server.node_probe_mode.singbox")}</option>
                            </select>
                            <p className="text-sm text-muted-foreground">{t("server.node_probe_mode.description")}</p>
                        </div>
                        {values.node_probe_mode === "singbox" && (
                            <>
                                <div className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-400">
                                    <svg className="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 24 24" width="1.2em" height="1.2em"><path fill="currentColor" d="M12 2a10 10 0 1 0 10 10A10.01 10.01 0 0 0 12 2m0 18a8 8 0 1 1 8-8a8.01 8.01 0 0 1-8 8m0-8.5a1 1 0 0 0-1 1v3a1 1 0 0 0 2 0v-3a1 1 0 0 0-1-1m0-4a1.25 1.25 0 1 0 1.25 1.25A1.25 1.25 0 0 0 12 7.5" /></svg>
                                    <span>{t("server.node_probe_mode.singbox_hint")}</span>
                                </div>

                                <div className="space-y-2 rounded-lg border p-4">
                                    <div className="flex items-center justify-between gap-2">
                                        <div>
                                            <div className="text-base font-medium">{t("server.probe_diagnose.title")}</div>
                                            <p className="text-sm text-muted-foreground">{t("server.probe_diagnose.description")}</p>
                                        </div>
                                        <Button variant="outline" size="sm" onClick={runDiagnose} loading={diagLoading} disabled={diagLoading}>
                                            {t("server.probe_diagnose.button")}
                                        </Button>
                                    </div>
                                    {diagLoading && <p className="text-sm text-muted-foreground">{t("server.probe_diagnose.running")}</p>}
                                    {diagResult && <DiagnoseResult r={diagResult} t={t} />}
                                </div>

                                <div className="space-y-2 rounded-lg border p-4">
                                    <div className="flex items-center justify-between gap-2">
                                        <div>
                                            <div className="text-base font-medium">{t("server.singbox_download.title")}</div>
                                            <p className="text-sm text-muted-foreground">{t("server.singbox_download.description")}</p>
                                        </div>
                                        <Button variant="outline" size="sm" onClick={runDownload} loading={downloading} disabled={downloading}>
                                            {dl.state === "done" || dl.state === "error" ? t("server.singbox_download.button_retry") : t("server.singbox_download.button")}
                                        </Button>
                                    </div>
                                    <DownloadStatus dl={dl} t={t} />
                                </div>

                                <ToggleField label={t("server.node_probe_singbox_auto_download.title")} description={t("server.node_probe_singbox_auto_download.description")} checked={values.node_probe_singbox_auto_download ?? true} onChange={(v) => setField("node_probe_singbox_auto_download", v)} />
                                {values.node_probe_singbox_auto_download && (
                                    <div className="space-y-1.5">
                                        <div className="text-base font-medium">{t("server.node_probe_singbox_version.title")}</div>
                                        <Input placeholder={t("server.node_probe_singbox_version.placeholder")} value={values.node_probe_singbox_version ?? ""} onChange={(e) => setField("node_probe_singbox_version", e.target.value)} />
                                        <p className="text-sm text-muted-foreground">{t("server.node_probe_singbox_version.description")}</p>
                                    </div>
                                )}
                                <div className="space-y-1.5">
                                    <div className="text-base font-medium">{t("server.node_probe_singbox_path.title")}</div>
                                    <Input placeholder={t("server.node_probe_singbox_path.placeholder")} value={values.node_probe_singbox_path ?? ""} onChange={(e) => setField("node_probe_singbox_path", e.target.value)} />
                                    <p className="text-sm text-muted-foreground">{t("server.node_probe_singbox_path.description")}</p>
                                </div>
                                <div className="space-y-1.5">
                                    <div className="text-base font-medium">{t("server.node_probe_latency_url.title")}</div>
                                    <Input placeholder={t("server.node_probe_latency_url.placeholder")} value={values.node_probe_latency_url ?? ""} onChange={(e) => setField("node_probe_latency_url", e.target.value)} />
                                    <p className="text-sm text-muted-foreground">{t("server.node_probe_latency_url.description")}</p>
                                </div>
                                <div className="space-y-1.5">
                                    <div className="text-base font-medium">{t("server.node_probe_test_count.title")}</div>
                                    <Input type="number" placeholder="3" value={values.node_probe_test_count ?? ""} onChange={(e) => setField("node_probe_test_count", e.target.value ? Number(e.target.value) : null)} />
                                    <p className="text-sm text-muted-foreground">{t("server.node_probe_test_count.description")}</p>
                                </div>
                                <div className="space-y-1.5">
                                    <div className="text-base font-medium">{t("server.node_probe_request_timeout_ms.title")}</div>
                                    <Input type="number" placeholder="10000" value={values.node_probe_request_timeout_ms ?? ""} onChange={(e) => setField("node_probe_request_timeout_ms", e.target.value ? Number(e.target.value) : null)} />
                                    <p className="text-sm text-muted-foreground">{t("server.node_probe_request_timeout_ms.description")}</p>
                                </div>
                                <div className="space-y-1.5">
                                    <div className="text-base font-medium">{t("server.node_probe_ip_echo_url.title")}</div>
                                    <Input placeholder={t("server.node_probe_ip_echo_url.placeholder")} value={values.node_probe_ip_echo_url ?? ""} onChange={(e) => setField("node_probe_ip_echo_url", e.target.value)} />
                                    <p className="text-sm text-muted-foreground">{t("server.node_probe_ip_echo_url.description")}</p>
                                </div>
                            </>
                        )}
                        <div className="space-y-1.5">
                            <div className="text-base font-medium">{t("server.server_latency_timeout_ms.title")}</div>
                            <Input type="number" placeholder={t("server.server_latency_timeout_ms.placeholder")} value={values.server_latency_timeout_ms ?? ""} onChange={(e) => setField("server_latency_timeout_ms", e.target.value ? Number(e.target.value) : null)} />
                            <p className="text-sm text-muted-foreground">{t("server.server_latency_timeout_ms.description")}</p>
                        </div>
                    </>
                )}

                <ToggleField label={t("server.server_auto_manage_visibility.title")} description={t("server.server_auto_manage_visibility.description")} checked={values.server_auto_manage_visibility ?? false} onChange={(v) => setField("server_auto_manage_visibility", v)} />

                <ToggleField label={t("server.server_dedup_by_country.title")} description={t("server.server_dedup_by_country.description")} checked={values.server_dedup_by_country ?? false} onChange={(v) => setField("server_dedup_by_country", v)} />
                {values.server_dedup_by_country && (
                    <>
                        <ToggleField label={t("server.server_country_naming.title")} description={t("server.server_country_naming.description")} checked={values.server_country_naming ?? true} onChange={(v) => setField("server_country_naming", v)} />
                        <div className="space-y-1.5">
                            <div className="text-base font-medium">{t("server.server_latency_stale_seconds.title")}</div>
                            <Input type="number" placeholder={t("server.server_latency_stale_seconds.placeholder")} value={values.server_latency_stale_seconds ?? ""} onChange={(e) => setField("server_latency_stale_seconds", e.target.value ? Number(e.target.value) : null)} />
                            <p className="text-sm text-muted-foreground">{t("server.server_latency_stale_seconds.description")}</p>
                        </div>
                        <div className="space-y-1.5">
                            <div className="text-base font-medium">{t("server.geoip_method.title")}</div>
                            <select
                                className="inline-flex h-9 w-full items-center rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm"
                                value={values.geoip_method ?? "api"}
                                onChange={(e) => setField("geoip_method", e.target.value)}
                            >
                                <option value="api">{t("server.geoip_method.api")}</option>
                                <option value="maxmind">{t("server.geoip_method.maxmind")}</option>
                            </select>
                            <p className="text-sm text-muted-foreground">{t("server.geoip_method.description")}</p>
                        </div>
                        {values.geoip_method === "maxmind" && (
                            <div className="space-y-1.5">
                                <div className="text-base font-medium">{t("server.geoip_maxmind_db_path.title")}</div>
                                <Input placeholder={t("server.geoip_maxmind_db_path.placeholder")} value={values.geoip_maxmind_db_path ?? ""} onChange={(e) => setField("geoip_maxmind_db_path", e.target.value)} />
                                <p className="text-sm text-muted-foreground">{t("server.geoip_maxmind_db_path.description")}</p>
                            </div>
                        )}
                    </>
                )}
                <CountriesCard t={t} time={values.country_refresh_time || "04:00"} onTimeChange={(v) => setField("country_refresh_time", v)} tags={values.country_tags ?? ""} onTagsChange={(v) => setField("country_tags", v)} />
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

// sing-box 诊断结果展示
function DiagnoseResult({ r, t }) {
    if (r.requestError) {
        return <div className="rounded-md bg-rose-50 p-3 text-sm text-rose-700 dark:bg-rose-950/40 dark:text-rose-400">{r.requestError}</div>;
    }
    const sb = r.singbox || {};
    const sample = r.sample;
    const Line = ({ ok, children }) => (
        <div className="flex items-start gap-2 text-sm">
            <span className={ok ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}>{ok ? "✓" : "✗"}</span>
            <span className="min-w-0 break-all">{children}</span>
        </div>
    );
    return (
        <div className="space-y-1.5 rounded-md bg-muted/40 p-3">
            <Line ok={sb.ok}>
                {sb.ok
                    ? `sing-box ${t("server.probe_diagnose.ready")}${sb.version ? ` (v${sb.version})` : ""}`
                    : `sing-box ${t("server.probe_diagnose.unavailable")}：${sb.error || ""}`}
            </Line>
            {sb.ok && sb.path && <div className="pl-5 font-mono text-[11px] text-muted-foreground break-all">{sb.path}</div>}
            {sample ? (
                sample.untestable ? (
                    <Line ok={false}>{t("server.probe_diagnose.sample")}「{sample.node_name}」{t("server.probe_diagnose.untestable")}：{sample.error || ""}</Line>
                ) : sample.ok ? (
                    <Line ok>
                        {t("server.probe_diagnose.sample")}「{sample.node_name}」{t("server.probe_diagnose.passed")}：{sample.latency_ms}ms
                        {sample.success_tests != null ? ` (${sample.success_tests}/${sample.tests})` : ""}
                        {sample.country_code ? ` · ${sample.country_code}` : ""}
                        {sample.exit_ip ? ` · ${t("server.probe_diagnose.exit")} ${sample.exit_ip}` : ""}
                    </Line>
                ) : (
                    <Line ok={false}>{t("server.probe_diagnose.sample")}「{sample.node_name}」{t("server.probe_diagnose.failed")}：{sample.error || ""}</Line>
                )
            ) : (
                <div className="pl-5 text-sm text-muted-foreground">{t("server.probe_diagnose.no_node")}</div>
            )}
        </div>
    );
}

function formatBytes(n) {
    if (!n) return "0 B";
    const u = ["B", "KB", "MB", "GB"];
    let i = 0;
    let v = n;
    while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
    return `${v.toFixed(i === 0 ? 0 : 1)} ${u[i]}`;
}

function DownloadStatus({ dl, t }) {
    if (!dl || dl.state === "idle") return null;
    if (dl.state === "downloading") {
        const pct = dl.total > 0 ? Math.min(100, Math.round((dl.downloaded / dl.total) * 100)) : null;
        return (
            <div className="space-y-1.5">
                <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                    <div className={`h-full rounded-full bg-primary transition-all ${pct === null ? "w-1/3 animate-pulse" : ""}`} style={pct === null ? undefined : { width: `${pct}%` }} />
                </div>
                <p className="text-sm text-muted-foreground">
                    {dl.message || t("server.singbox_download.downloading")}
                    {dl.total > 0 ? ` · ${formatBytes(dl.downloaded)} / ${formatBytes(dl.total)} (${pct}%)` : dl.downloaded > 0 ? ` · ${formatBytes(dl.downloaded)}` : ""}
                </p>
            </div>
        );
    }
    if (dl.state === "done") {
        return (
            <div className="space-y-0.5 text-sm text-green-600 dark:text-green-400">
                <div>✓ {t("server.singbox_download.done")}{dl.version ? `（${t("server.singbox_download.version")} ${dl.version}）` : ""}</div>
                {dl.path && <div className="break-all text-muted-foreground">{t("server.singbox_download.path")}：{dl.path}</div>}
            </div>
        );
    }
    return <div className="break-all text-sm text-red-600 dark:text-red-400">✗ {t("server.singbox_download.failed")}：{dl.message}</div>;
}

// 可用国家快照：展示上次更新时间与国家列表，支持手动更新（先测速再生成），更新期间轮询进度。
function CountriesCard({ t, time, onTimeChange, tags, onTagsChange }) {
    const [status, setStatus] = useState(null);
    const [starting, setStarting] = useState(false);
    const [error, setError] = useState("");
    const refreshing = starting || !!status?.refreshing;

    const load = useCallback(async () => {
        try {
            const res = await getCountriesStatus();
            setStatus(res?.data ?? null);
            setError("");
        } catch (e) {
            setError(e?.response?.data?.message || e?.message || "请求失败");
        }
    }, []);

    useEffect(() => { load(); }, [load]);
    useEffect(() => {
        if (!status?.refreshing) return undefined;
        const timer = setInterval(load, 2000);
        return () => clearInterval(timer);
    }, [status?.refreshing, load]);

    const refresh = async () => {
        setStarting(true);
        try {
            await refreshCountries(true);
            await load();
        } catch (e) {
            setError(e?.response?.data?.message || e?.message || "请求失败");
        } finally {
            setStarting(false);
        }
    };

    const snap = status?.snapshot;
    const countries = snap?.countries || [];
    const sourceLabel = snap?.source ? t("server.countries.source_" + snap.source) : "";

    return (
        <div className="space-y-2 rounded-lg border p-4">
            <div className="flex items-center justify-between gap-2">
                <div>
                    <div className="text-base font-medium">{t("server.countries.title")}</div>
                    <p className="text-sm text-muted-foreground">{t("server.countries.description")}</p>
                </div>
                <Button variant="outline" size="sm" onClick={refresh} loading={refreshing} disabled={refreshing}>
                    {t("server.countries.button")}
                </Button>
            </div>
            {refreshing && <p className="text-sm text-muted-foreground">{t("server.countries.refreshing")}</p>}
            {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
            {status && (
                <p className="text-sm text-muted-foreground">
                    {status.exists
                        ? `${t("server.countries.updated")}：${new Date(snap.updated_at * 1000).toLocaleString()}${sourceLabel ? `（${sourceLabel}）` : ""} · ${countries.length} ${t("server.countries.count")}`
                        : t("server.countries.never")}
                    {" · "}{t("server.countries.auto").replace("{time}", time)}
                </p>
            )}
            <div className="flex flex-wrap items-center gap-3 pt-1">
                <div className="text-sm font-medium">{t("server.countries.time_label")}</div>
                <Input type="time" className="w-32" value={time} onChange={(e) => e.target.value && onTimeChange(e.target.value)} />
                <p className="text-sm text-muted-foreground">{t("server.countries.time_desc")}</p>
            </div>
            <div className="flex flex-wrap items-center gap-3 pt-1">
                <div className="text-sm font-medium">{t("server.countries.tags_label")}</div>
                <Input className="w-64" placeholder={t("server.countries.tags_placeholder")} value={tags} onChange={(e) => onTagsChange(e.target.value)} />
                <p className="text-sm text-muted-foreground">{t("server.countries.tags_desc")}</p>
            </div>
            {snap?.kept && <p className="text-sm text-amber-600 dark:text-amber-400">{t("server.countries.kept")}</p>}
            {status?.exists && countries.length === 0 && <p className="text-sm text-muted-foreground">{t("server.countries.empty")}</p>}
            {countries.length > 0 && (
                <div className="flex flex-wrap gap-2 pt-1">
                    {countries.map((c) => (
                        <span key={c.code} className="inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-sm">
                            <span>{c.flag}</span>
                            <span>{c.name}</span>
                            <span className="text-muted-foreground">{c.nodes} {t("server.countries.nodes")}{c.best_latency_ms != null ? ` · ${c.best_latency_ms}ms` : ""}</span>
                        </span>
                    ))}
                </div>
            )}
            {snap?.unknown_country > 0 && <p className="text-sm text-muted-foreground">{snap.unknown_country} {t("server.countries.unknown")}</p>}
        </div>
    );
}
