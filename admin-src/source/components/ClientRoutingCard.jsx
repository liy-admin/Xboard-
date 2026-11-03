import React, { useEffect, useState } from "react";
import { useTranslation } from "../i18n/index.js";
import { useAuthState } from "../store/auth.js";
import { getSettings, saveSettings } from "../api/config.js";
import { Button } from "./Button.jsx";
import { Input } from "./Input.jsx";
import { Switch } from "./Switch.jsx";
import { Badge } from "./Badge.jsx";
import { Textarea } from "./Textarea.jsx";
import { FormDialog } from "./FormDialog.jsx";
import { ConfirmDialog } from "./ConfirmDialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./Select.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "./Table.jsx";

// 客户端分流（下发给 App，hiddify-core custom-rules / rule-set-url-template）。放在「路由管理」页，
// 与给节点用的服务端路由并列；设置存在独立的 client_routing 配置分组里。
//
// 存储格式仍是 sing-box 风格的规则 JSON 数组；界面上每条规则编辑为「一行一个匹配项」，
// 写法与 v2rayN 路由规则相同（domain: / full: / keyword: / regexp: / geosite: / geoip: / IP 段）。

// Loyalsoldier/v2ray-rules-dat 的 sing-box 移植（2dust/sing-box-rules），与 v2rayN 相同。
const LOYALSOLDIER_TEMPLATE = "https://raw.githubusercontent.com/2dust/sing-box-rules/rule-set-{type}/{name}.srs";
const HIDDIFY_GEO_EXAMPLE = "https://raw.githubusercontent.com/hiddify/hiddify-geo/rule-set/country/geosite-cn.srs";

// v2rayN「V3-绕过大陆(Whitelist)」中可通用的部分。其余（geoip-cn / geosite-cn 直连、最终走代理）App 已内置。
const V2RAYN_PRESET = [
    {
        name: "Google 中国域名 / AI 服务走代理",
        outbound: "proxy",
        domain_suffix: ["googleapis.cn", "gstatic.com", "dl.google.com", "openai.com", "chatgpt.com", "cursor.sh", "cursorapi.com", "githubcopilot.com"],
    },
    { name: "局域网域名直连", outbound: "direct", rule_set: ["geosite-private"] },
    {
        name: "国内 DNS 直连",
        outbound: "direct",
        ip_cidr: ["223.5.5.5/32", "223.6.6.6/32", "2400:3200::1/128", "2400:3200:baba::1/128", "119.29.29.29/32", "1.12.12.12/32", "120.53.53.53/32", "2402:4e00::/128", "2402:4e00:1::/128", "180.76.76.76/32", "2400:da00::6666/128", "114.114.114.114/32", "114.114.115.115/32", "180.184.1.1/32", "180.184.2.2/32"],
        domain_suffix: ["alidns.com", "doh.pub", "dot.pub", "360.cn", "onedns.net"],
    },
];

const OUTBOUNDS = ["proxy", "direct", "block"];
const OUTBOUND_BADGE = { proxy: "default", direct: "secondary", block: "destructive" };
// 一行一个匹配项能表达的字段；其余字段（port、process_name…）在编辑时原样保留。
const LINE_FIELDS = ["domain_suffix", "domain", "domain_keyword", "domain_regex", "ip_cidr", "rule_set"];

const IPV4 = /^\d{1,3}(\.\d{1,3}){3}(\/\d{1,2})?$/;
const IPV6 = /^[0-9a-f]*:[0-9a-f:]*(\/\d{1,3})?$/i;

function lineToMatcher(line) {
    const s = line.trim();
    if (!s || s.startsWith("#")) return null;
    let m;
    if (/^https?:\/\//i.test(s)) return ["rule_set", s];
    if ((m = s.match(/^(geosite|geoip):(.+)$/i))) return ["rule_set", `${m[1].toLowerCase()}-${m[2].trim().toLowerCase()}`];
    if (/^(geosite|geoip)-\S+$/i.test(s)) return ["rule_set", s.toLowerCase()];
    if ((m = s.match(/^full:(.+)$/i))) return ["domain", m[1].trim()];
    if ((m = s.match(/^keyword:(.+)$/i))) return ["domain_keyword", m[1].trim()];
    if ((m = s.match(/^regexp:(.+)$/i))) return ["domain_regex", m[1].trim()];
    if ((m = s.match(/^domain:(.+)$/i))) return ["domain_suffix", m[1].trim()];
    if (IPV4.test(s)) return ["ip_cidr", s.includes("/") ? s : `${s}/32`];
    if (s.includes(":") && IPV6.test(s)) return ["ip_cidr", s.includes("/") ? s : `${s}/128`];
    return ["domain_suffix", s.replace(/^\*?\./, "")];
}

function ruleToLines(rule) {
    const out = [];
    (rule.rule_set || []).forEach((v) => out.push(v.replace(/^(geosite|geoip)-/, "$1:")));
    (rule.domain_suffix || []).forEach((v) => out.push(v));
    (rule.domain || []).forEach((v) => out.push(`full:${v}`));
    (rule.domain_keyword || []).forEach((v) => out.push(`keyword:${v}`));
    (rule.domain_regex || []).forEach((v) => out.push(`regexp:${v}`));
    (rule.ip_cidr || []).forEach((v) => out.push(v.replace(/\/32$/, "").replace(/\/128$/, "")));
    return out;
}

function extraFields(rule) {
    const extra = {};
    Object.keys(rule || {}).forEach((k) => {
        if (k !== "name" && k !== "outbound" && !LINE_FIELDS.includes(k)) extra[k] = rule[k];
    });
    return extra;
}

function buildRule({ name, outbound, lines, extra }) {
    const rule = { ...(extra || {}) };
    if (name.trim()) rule.name = name.trim();
    rule.outbound = outbound;
    lines.split("\n").forEach((line) => {
        const m = lineToMatcher(line);
        if (!m) return;
        const [field, value] = m;
        rule[field] = rule[field] || [];
        if (!rule[field].includes(value)) rule[field].push(value);
    });
    return rule;
}

function parseStoredRules(v) {
    if (Array.isArray(v)) return v;
    if (typeof v === "string" && v.trim()) {
        try {
            const arr = JSON.parse(v);
            return Array.isArray(arr) ? arr : [];
        } catch (e) {
            return [];
        }
    }
    return [];
}

const errMsg = (e) => e?.response?.data?.message || e?.message || "请求失败";

function SectionTitle({ n, title, desc }) {
    return (
        <div className="space-y-0.5">
            <div className="flex items-center gap-2 text-base font-medium">
                <span className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground">{n}</span>
                {title}
            </div>
            {desc && <p className="text-sm text-muted-foreground">{desc}</p>}
        </div>
    );
}

// 匹配顺序说明：让管理员一眼看清自定义规则插在哪里。
function FlowExplainer({ t }) {
    const steps = ["flow_lan", "flow_custom", "flow_cn", "flow_rest"];
    return (
        <ol className="space-y-1 rounded-md bg-muted/40 p-3 text-sm">
            {steps.map((k, i) => (
                <li key={k} className={k === "flow_custom" ? "font-medium text-foreground" : "text-muted-foreground"}>
                    {i + 1}. {t(`server.client_routing.${k}`)}
                </li>
            ))}
            <li className="pt-1 text-xs text-muted-foreground">{t("server.client_routing.flow_safe")}</li>
        </ol>
    );
}

// ① 中国直连数据来源
const modeOf = (tpl) => (tpl === LOYALSOLDIER_TEMPLATE ? "loyal" : tpl === "" ? "builtin" : "custom");

// 读取服务端当前保存的模板（保存后回读确认，界面始终以服务端为准）。
async function fetchTemplate() {
    const res = await getSettings("client_routing");
    const v = res?.data?.client_routing?.client_ruleset_url_template;
    return typeof v === "string" ? v : "";
}

function SourceSection({ t, initial }) {
    // saved = 服务端已保存（App 实际使用）的模板；editing = 正在填写、尚未保存的自定义地址。
    const [saved, setSaved] = useState(initial);
    const [editing, setEditing] = useState(modeOf(initial) === "custom");
    const [custom, setCustom] = useState(modeOf(initial) === "custom" ? initial : "");
    const [error, setError] = useState("");
    const [justSaved, setJustSaved] = useState(false);
    const [busy, setBusy] = useState(false);
    const savedMode = modeOf(saved);

    const save = async (value) => {
        setBusy(true);
        setJustSaved(false);
        try {
            await saveSettings({ client_ruleset_url_template: value });
            const confirmed = await fetchTemplate();
            setSaved(confirmed);
            if (confirmed !== value) {
                setError(t("server.client_routing.source_not_persisted"));
                return;
            }
            setError("");
            setJustSaved(true);
            setEditing(modeOf(confirmed) === "custom");
        } catch (e) {
            setError(errMsg(e));
        } finally {
            setBusy(false);
        }
    };

    const choose = (next) => {
        setError("");
        if (next === "loyal") save(LOYALSOLDIER_TEMPLATE);
        else if (next === "builtin") save("");
        else {
            setEditing(true);
            setJustSaved(false);
        }
    };

    // 选中高亮：正在编辑自定义时高亮「自定义」，否则高亮服务端已保存的那一项。
    const selected = editing ? "custom" : savedMode;
    const preview = saved ? saved.replaceAll("{type}", "geosite").replaceAll("{name}", "geosite-cn") : HIDDIFY_GEO_EXAMPLE;
    const options = [
        { key: "loyal", label: t("server.client_routing.source_loyalsoldier"), desc: t("server.client_routing.source_loyalsoldier_desc") },
        { key: "builtin", label: t("server.client_routing.source_builtin"), desc: t("server.client_routing.source_builtin_desc") },
        { key: "custom", label: t("server.client_routing.source_custom"), desc: t("server.client_routing.source_custom_desc") },
    ];

    return (
        <div className="space-y-3">
            <SectionTitle n={1} title={t("server.client_routing.source_title")} desc={t("server.client_routing.source_help")} />
            <div className="grid gap-2 sm:grid-cols-3">
                {options.map((o) => {
                    const active = savedMode === o.key;
                    const pending = editing && o.key === "custom" && !active;
                    return (
                        <button
                            key={o.key}
                            type="button"
                            disabled={busy}
                            onClick={() => choose(o.key)}
                            className={`rounded-lg border p-3 text-left transition-colors ${selected === o.key ? "border-primary bg-primary/5 ring-1 ring-primary" : "hover:bg-muted/50"} ${pending ? "border-dashed" : ""}`}
                        >
                            <div className="flex items-center justify-between gap-2">
                                <span className="text-sm font-medium">{selected === o.key ? "● " : "○ "}{o.label}</span>
                                {active && <span className="rounded bg-green-600/10 px-1.5 py-0.5 text-[11px] text-green-700 dark:text-green-400">{t("server.client_routing.source_active")}</span>}
                                {pending && <span className="rounded bg-amber-500/10 px-1.5 py-0.5 text-[11px] text-amber-700 dark:text-amber-400">{t("server.client_routing.source_unsaved")}</span>}
                            </div>
                            <div className="mt-1 text-xs text-muted-foreground">{o.desc}</div>
                        </button>
                    );
                })}
            </div>
            {editing && (
                <div className="space-y-1.5">
                    <div className="flex flex-col gap-2 sm:flex-row">
                        <Input value={custom} placeholder="https://example.com/rule-set-{type}/{name}.srs" onChange={(e) => { setCustom(e.target.value); setJustSaved(false); }} className="font-mono text-xs" />
                        <Button size="sm" onClick={() => save(custom.trim())} loading={busy} disabled={busy || !custom.trim()}>{t("server.client_routing.save")}</Button>
                        {savedMode !== "custom" && (
                            <Button size="sm" variant="outline" onClick={() => { setEditing(false); setError(""); }} disabled={busy}>{t("server.client_routing.cancel")}</Button>
                        )}
                    </div>
                    {savedMode !== "custom" && <p className="text-xs text-amber-600 dark:text-amber-400">{t("server.client_routing.source_custom_hint")}</p>}
                </div>
            )}
            <p className="break-all font-mono text-[11px] text-muted-foreground">{t("server.client_routing.source_preview")}：{preview}</p>
            {error && <p className="break-all text-sm text-red-600 dark:text-red-400">{error}</p>}
            {justSaved && <p className="text-sm text-green-600 dark:text-green-400">✓ {t("server.client_routing.saved")}</p>}
        </div>
    );
}

// ② 直连 DNS（极速模式）：中国域名与直连规则命中的域名用它解析。
const DNS_PRESETS = { ali: "223.5.5.5", tencent: "119.29.29.29", app: "" };
const dnsModeOf = (v) => (v === DNS_PRESETS.ali ? "ali" : v === DNS_PRESETS.tencent ? "tencent" : v === "" ? "app" : "custom");

async function fetchClientRoutingKey(key) {
    const res = await getSettings("client_routing");
    const v = res?.data?.client_routing?.[key];
    return typeof v === "string" ? v : "";
}

function DnsSection({ t, initial }) {
    const [saved, setSaved] = useState(initial);
    const [editing, setEditing] = useState(dnsModeOf(initial) === "custom");
    const [custom, setCustom] = useState(dnsModeOf(initial) === "custom" ? initial : "");
    const [error, setError] = useState("");
    const [justSaved, setJustSaved] = useState(false);
    const [busy, setBusy] = useState(false);
    const savedMode = dnsModeOf(saved);

    const save = async (value) => {
        setBusy(true);
        setJustSaved(false);
        try {
            await saveSettings({ client_direct_dns: value });
            const confirmed = await fetchClientRoutingKey("client_direct_dns");
            setSaved(confirmed);
            if (confirmed !== value) {
                setError(t("server.client_routing.source_not_persisted"));
                return;
            }
            setError("");
            setJustSaved(true);
            setEditing(dnsModeOf(confirmed) === "custom");
        } catch (e) {
            setError(errMsg(e));
        } finally {
            setBusy(false);
        }
    };

    const choose = (next) => {
        setError("");
        if (next === "custom") {
            setEditing(true);
            setJustSaved(false);
            return;
        }
        save(DNS_PRESETS[next]);
    };

    const selected = editing ? "custom" : savedMode;
    const options = [
        { key: "ali", label: t("server.client_routing.dns_ali"), desc: "223.5.5.5" },
        { key: "tencent", label: t("server.client_routing.dns_tencent"), desc: "119.29.29.29" },
        { key: "app", label: t("server.client_routing.dns_app"), desc: t("server.client_routing.dns_app_desc") },
        { key: "custom", label: t("server.client_routing.source_custom"), desc: t("server.client_routing.dns_custom_desc") },
    ];

    return (
        <div className="space-y-3">
            <SectionTitle n={2} title={t("server.client_routing.dns_title")} desc={t("server.client_routing.dns_help")} />
            <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
                {options.map((o) => {
                    const active = savedMode === o.key;
                    const pending = editing && o.key === "custom" && !active;
                    return (
                        <button
                            key={o.key}
                            type="button"
                            disabled={busy}
                            onClick={() => choose(o.key)}
                            className={`rounded-lg border p-3 text-left transition-colors ${selected === o.key ? "border-primary bg-primary/5 ring-1 ring-primary" : "hover:bg-muted/50"} ${pending ? "border-dashed" : ""}`}
                        >
                            <div className="flex items-center justify-between gap-2">
                                <span className="text-sm font-medium">{selected === o.key ? "● " : "○ "}{o.label}</span>
                                {active && <span className="rounded bg-green-600/10 px-1.5 py-0.5 text-[11px] text-green-700 dark:text-green-400">{t("server.client_routing.source_active")}</span>}
                                {pending && <span className="rounded bg-amber-500/10 px-1.5 py-0.5 text-[11px] text-amber-700 dark:text-amber-400">{t("server.client_routing.source_unsaved")}</span>}
                            </div>
                            <div className="mt-1 break-all text-xs text-muted-foreground">{o.desc}</div>
                        </button>
                    );
                })}
            </div>
            {editing && (
                <div className="space-y-1.5">
                    <div className="flex flex-col gap-2 sm:flex-row">
                        <Input value={custom} placeholder="223.5.5.5 / https://dns.alidns.com/dns-query" onChange={(e) => { setCustom(e.target.value); setJustSaved(false); }} className="font-mono text-xs" />
                        <Button size="sm" onClick={() => save(custom.trim())} loading={busy} disabled={busy || !custom.trim()}>{t("server.client_routing.save")}</Button>
                        {savedMode !== "custom" && (
                            <Button size="sm" variant="outline" onClick={() => { setEditing(false); setError(""); }} disabled={busy}>{t("server.client_routing.cancel")}</Button>
                        )}
                    </div>
                    {savedMode !== "custom" && <p className="text-xs text-amber-600 dark:text-amber-400">{t("server.client_routing.source_custom_hint")}</p>}
                </div>
            )}
            {error && <p className="break-all text-sm text-red-600 dark:text-red-400">{error}</p>}
            {justSaved && <p className="text-sm text-green-600 dark:text-green-400">✓ {t("server.client_routing.saved")}</p>}
        </div>
    );
}

// ③ 自定义规则：表格 + 弹窗编辑，改动即保存（顺序有意义，可上下移动）。
function RulesSection({ t, initialEnabled, initialRules }) {
    const [enabled, setEnabled] = useState(initialEnabled);
    const [rules, setRules] = useState(initialRules);
    const [error, setError] = useState("");
    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState({ index: -1, name: "", outbound: "proxy", lines: "", extra: {} });
    const [formError, setFormError] = useState("");
    const [deleteIndex, setDeleteIndex] = useState(-1);
    const [jsonOpen, setJsonOpen] = useState(false);
    const [jsonText, setJsonText] = useState("");

    const persist = async (next) => {
        await saveSettings({ client_routing_rules: JSON.stringify(next) });
        setRules(next);
        setError("");
    };
    const persistOrShow = async (next) => {
        try { await persist(next); } catch (e) { setError(errMsg(e)); }
    };

    const onEnabledChange = async (v) => {
        setEnabled(v);
        try { await saveSettings({ client_routing_enable: v }); } catch (e) { setEnabled(!v); setError(errMsg(e)); }
    };

    const openAdd = () => {
        setForm({ index: -1, name: "", outbound: "proxy", lines: "", extra: {} });
        setFormError("");
        setFormOpen(true);
    };
    const openEdit = (i) => {
        const r = rules[i];
        setForm({ index: i, name: r.name || "", outbound: r.outbound || "proxy", lines: ruleToLines(r).join("\n"), extra: extraFields(r) });
        setFormError("");
        setFormOpen(true);
    };

    const submit = async () => {
        const rule = buildRule(form);
        const hasMatcher = Object.keys(rule).some((k) => k !== "name" && k !== "outbound");
        if (!hasMatcher) {
            const msg = t("server.client_routing.need_match");
            setFormError(msg);
            throw new Error(msg);
        }
        const next = [...rules];
        if (form.index >= 0) next[form.index] = rule; else next.push(rule);
        try {
            await persist(next);
        } catch (e) {
            setFormError(errMsg(e));
            throw e;
        }
    };

    const move = (i, d) => {
        const j = i + d;
        if (j < 0 || j >= rules.length) return;
        const next = [...rules];
        [next[i], next[j]] = [next[j], next[i]];
        persistOrShow(next);
    };

    const addPreset = () => {
        const names = new Set(rules.map((r) => r.name));
        persistOrShow([...rules, ...V2RAYN_PRESET.filter((r) => !names.has(r.name))]);
    };

    const saveJson = async () => {
        let arr;
        try {
            arr = JSON.parse(jsonText || "[]");
            if (!Array.isArray(arr)) throw new Error("must be an array");
        } catch (e) {
            setError(t("server.client_routing.invalid_json") + e.message);
            return;
        }
        try { await persist(arr); setJsonOpen(false); } catch (e) { setError(errMsg(e)); }
    };

    const summary = (r) => {
        const lines = ruleToLines(r);
        const extra = Object.keys(extraFields(r));
        const head = lines.slice(0, 3).join("、");
        const more = lines.length > 3 ? ` ${t("server.client_routing.more", { count: lines.length })}` : "";
        return head + more + (extra.length ? ` +${extra.join(",")}` : "");
    };

    return (
        <div className="space-y-3">
            <div className="flex items-start justify-between gap-3">
                <SectionTitle n={3} title={t("server.client_routing.rules_title")} desc={t("server.client_routing.rules_desc")} />
                <Switch checked={enabled} onCheckedChange={onEnabledChange} aria-label={t("server.client_routing.enable")} />
            </div>
            {!enabled ? (
                <p className="text-sm text-muted-foreground">{t("server.client_routing.disabled_hint")}</p>
            ) : (
                <>
                    <div className="flex flex-wrap gap-2">
                        <Button size="sm" onClick={openAdd}>{t("server.client_routing.add")}</Button>
                        <Button size="sm" variant="outline" onClick={addPreset}>{t("server.client_routing.preset")}</Button>
                        <Button size="sm" variant="ghost" onClick={() => { setJsonText(JSON.stringify(rules, null, 2)); setJsonOpen((v) => !v); }}>
                            {t("server.client_routing.json_toggle")}
                        </Button>
                    </div>
                    {error && <p className="break-all text-sm text-red-600 dark:text-red-400">{error}</p>}
                    {jsonOpen ? (
                        <div className="space-y-2">
                            <Textarea className="min-h-[240px] font-mono text-xs" spellCheck={false} value={jsonText} onChange={(e) => setJsonText(e.target.value)} />
                            <div className="flex gap-2">
                                <Button size="sm" onClick={saveJson}>{t("server.client_routing.save")}</Button>
                                <Button size="sm" variant="outline" onClick={() => setJsonOpen(false)}>{t("server.client_routing.cancel")}</Button>
                            </div>
                        </div>
                    ) : (
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead className="w-12">#</TableHead>
                                    <TableHead>{t("server.client_routing.col_name")}</TableHead>
                                    <TableHead className="w-24">{t("server.client_routing.col_action")}</TableHead>
                                    <TableHead>{t("server.client_routing.col_match")}</TableHead>
                                    <TableHead className="w-48 text-right" />
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {rules.length === 0 ? (
                                    <TableEmpty colSpan={5}>{t("server.client_routing.empty")}</TableEmpty>
                                ) : rules.map((r, i) => (
                                    <TableRow key={i}>
                                        <TableCell className="text-muted-foreground">{i + 1}</TableCell>
                                        <TableCell className="font-medium">{r.name || "-"}</TableCell>
                                        <TableCell><Badge variant={OUTBOUND_BADGE[r.outbound] || "outline"}>{t(`server.client_routing.out_${r.outbound || "proxy"}`)}</Badge></TableCell>
                                        <TableCell className="max-w-md truncate text-xs text-muted-foreground" title={ruleToLines(r).join("\n")}>{summary(r)}</TableCell>
                                        <TableCell className="text-right">
                                            <div className="flex justify-end gap-1">
                                                <Button size="sm" variant="ghost" disabled={i === 0} onClick={() => move(i, -1)} aria-label="up">↑</Button>
                                                <Button size="sm" variant="ghost" disabled={i === rules.length - 1} onClick={() => move(i, 1)} aria-label="down">↓</Button>
                                                <Button size="sm" variant="outline" onClick={() => openEdit(i)}>{t("server.client_routing.edit")}</Button>
                                                <Button size="sm" variant="destructive" onClick={() => setDeleteIndex(i)}>{t("server.client_routing.delete")}</Button>
                                            </div>
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    )}
                </>
            )}

            <FormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                title={form.index >= 0 ? t("server.client_routing.edit") : t("server.client_routing.add")}
                cancelLabel={t("server.client_routing.cancel")}
                submitLabel={t("server.client_routing.save")}
                onSubmit={submit}
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("server.client_routing.col_name")}</div>
                    <Input value={form.name} placeholder={t("server.client_routing.name_placeholder")} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("server.client_routing.col_action")}</div>
                    <Select value={form.outbound} onValueChange={(v) => setForm((p) => ({ ...p, outbound: v }))}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                            {OUTBOUNDS.map((o) => <SelectItem key={o} value={o}>{t(`server.client_routing.out_${o}_long`)}</SelectItem>)}
                        </SelectContent>
                    </Select>
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("server.client_routing.col_match")}</div>
                    <Textarea className="min-h-[160px] font-mono text-xs" spellCheck={false} placeholder={"googleapis.cn\nkeyword:openai\ngeosite:netflix\n8.8.8.8"} value={form.lines} onChange={(e) => setForm((p) => ({ ...p, lines: e.target.value }))} />
                    <ul className="space-y-0.5 text-xs text-muted-foreground">
                        {["fmt_domain", "fmt_full", "fmt_keyword", "fmt_geosite", "fmt_ip"].map((k) => <li key={k}>• {t(`server.client_routing.${k}`)}</li>)}
                    </ul>
                    {Object.keys(form.extra).length > 0 && (
                        <p className="text-xs text-amber-600 dark:text-amber-400">{t("server.client_routing.extra_kept")}: {Object.keys(form.extra).join(", ")}</p>
                    )}
                </div>
            </FormDialog>

            <ConfirmDialog
                open={deleteIndex >= 0}
                onOpenChange={(open) => !open && setDeleteIndex(-1)}
                title={t("server.client_routing.delete_confirm")}
                confirmLabel={t("server.client_routing.delete")}
                onConfirm={() => persistOrShow(rules.filter((_, i) => i !== deleteIndex))}
            />
        </div>
    );
}

export function ClientRoutingCard() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [data, setData] = useState(null);

    useEffect(() => {
        getSettings("client_routing").then((res) => {
            const v = res?.data?.client_routing || {};
            setData({
                enabled: !!v.client_routing_enable,
                rules: parseStoredRules(v.client_routing_rules),
                template: typeof v.client_ruleset_url_template === "string" ? v.client_ruleset_url_template : "",
                dns: typeof v.client_direct_dns === "string" ? v.client_direct_dns : "",
            });
        }).catch(() => setData({ enabled: false, rules: [], template: LOYALSOLDIER_TEMPLATE, dns: DNS_PRESETS.ali }));
    }, [auth.token]);

    if (!data) return null;

    return (
        <div className="space-y-5 rounded-lg border p-4">
            <div className="space-y-2">
                <div className="text-lg font-medium">{t("server.client_routing.title")}</div>
                <p className="text-sm text-muted-foreground">{t("server.client_routing.description")}</p>
                <FlowExplainer t={t} />
            </div>
            <SourceSection t={t} initial={data.template} />
            <hr />
            <DnsSection t={t} initial={data.dns} />
            <hr />
            <RulesSection t={t} initialEnabled={data.enabled} initialRules={data.rules} />
        </div>
    );
}
