import React, { useEffect, useState } from "react";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/Tabs.jsx";
import { saveNode, generateEchKey } from "../../api/server.js";

export const PROTOCOL_TYPES = ["shadowsocks", "vmess", "trojan", "hysteria", "vless", "tuic", "socks", "naive", "http", "mieru", "anytls"];

const CIPHER_PRESETS = [
    "aes-128-gcm", "aes-192-gcm", "aes-256-gcm", "chacha20-ietf-poly1305",
    "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305",
];
const PLUGIN_OPTIONS = ["none", "obfs", "v2ray-plugin", "gost-plugin", "shadow-tls", "restls", "kcptun"];
const NETWORK_OPTIONS = ["tcp", "ws", "grpc", "h2", "httpupgrade", "xhttp"];
const FINGERPRINT_OPTIONS = ["chrome", "firefox", "safari", "ios"];
const MULTIPLEX_PROTOCOLS = ["smux", "yamux", "h2mux"];
const TUIC_CONGESTION = ["bbr", "cubic", "new_reno"];
const TUIC_RELAY_MODE = ["native", "quic"];
const SOCKS_VERSIONS = ["4", "4a", "5"];
const MULTIPLEX_TYPES = new Set(["vmess", "vless", "trojan", "tuic", "anytls"]);
const ECH_TYPES = new Set(["vmess", "vless", "hysteria", "anytls"]);

function defaultMultiplex() {
    return { enabled: false, protocol: "smux", max_connections: 4, min_streams: 4, padding: false, brutal: { enabled: false, up_mbps: 0, down_mbps: 0 } };
}
function defaultEch() {
    return { enabled: false, config: "", key: "", query_server_name: "" };
}
function defaultCertConfig() {
    return { cert_mode: "none", domain: "", email: "", http_port: 80, dns_provider: "", dns_env: "", cert_content: "", key_content: "" };
}

function protocolDefaults(type) {
    switch (type) {
        case "shadowsocks":
            return { cipher: "aes-128-gcm", plugin: "", plugin_opts: "", client_fingerprint: "chrome" };
        case "vmess":
            return { tls: 0, tls_settings: { server_name: "", allow_insecure: false, ech: defaultEch() }, network: "tcp", network_settings: "{}", multiplex: defaultMultiplex(), cert_config: defaultCertConfig() };
        case "vless":
            return {
                tls: 0,
                tls_settings: { server_name: "", allow_insecure: false, ech: defaultEch() },
                reality_settings: { server_name: "", server_port: 443, allow_insecure: false, private_key: "", public_key: "", short_id: "" },
                network: "tcp", network_settings: "{}", flow: "", encryption: { server: "", client: "" }, multiplex: defaultMultiplex(),
            };
        case "trojan":
            return { server_name: "", allow_insecure: false, use_reality: false, reality_settings: { server_name: "", server_port: 443, allow_insecure: false, private_key: "", public_key: "", short_id: "" }, network: "tcp", network_settings: "{}", multiplex: defaultMultiplex() };
        case "hysteria":
            return { version: "2", alpn: "", obfs: { type: "", password: "" }, tls: { server_name: "", allow_insecure: false, ech: defaultEch() }, bandwidth: { up: "", down: "" } };
        case "tuic":
            return { version: "5", password: "", congestion_control: "bbr", udp_relay_mode: "native", tls: { server_name: "", allow_insecure: false, alpn: "" }, multiplex: defaultMultiplex() };
        case "socks":
            return { version: "5", tls: 0, tls_settings: { server_name: "", allow_insecure: false }, network: "tcp" };
        case "naive":
            return { tls: 0, tls_settings: { server_name: "", allow_insecure: false } };
        case "http":
            return { tls: 0, tls_settings: { server_name: "", allow_insecure: false } };
        case "mieru":
            return { transport: "TCP", traffic_pattern: "" };
        case "anytls":
            return { tls: { server_name: "", allow_insecure: false, ech: defaultEch() }, padding_scheme: "", multiplex: defaultMultiplex() };
        default:
            return {};
    }
}

function emptyForm() {
    return {
        id: null, type: "", code: "", name: "", host: "", port: "", server_port: "",
        listen_address: "", listenAddressVisible: false,
        rate: "1", rate_time_enable: false, rate_time_ranges: [],
        transfer_enable_gb: "", group_ids: [], route_ids: [], parent_id: "0",
        machine_id: "", enabled: true, tags: [], tagInput: "", show: true, banned: false,
        custom_outbounds: "[]", custom_routes: "[]",
        protocol_settings: {},
    };
}

function nodeToForm(node) {
    const base = emptyForm();
    if (!node) return base;
    const settings = node.protocol_settings || {};
    const normalized = { ...protocolDefaults(node.type), ...settings };
    if (normalized.network_settings && typeof normalized.network_settings !== "string") {
        normalized.network_settings = JSON.stringify(normalized.network_settings, null, 2);
    }
    return {
        ...base,
        id: node.id ?? null,
        type: node.type || "",
        code: node.code || "",
        name: node.name || "",
        host: node.host || "",
        port: node.port ?? "",
        server_port: node.server_port ?? "",
        listen_address: node.listen_address || "",
        listenAddressVisible: Boolean(node.listen_address),
        rate: node.rate != null ? String(node.rate) : "1",
        rate_time_enable: Boolean(node.rate_time_enable),
        rate_time_ranges: node.rate_time_ranges || [],
        transfer_enable_gb: node.transfer_enable ? String(node.transfer_enable / (1024 * 1024 * 1024)) : "",
        group_ids: (node.group_ids || []).map(String),
        route_ids: (node.route_ids || []).map(String),
        parent_id: node.parent_id != null ? String(node.parent_id) : "0",
        machine_id: node.machine_id != null ? String(node.machine_id) : "",
        enabled: node.enabled == null ? true : Boolean(node.enabled),
        tags: node.tags || [],
        tagInput: "",
        show: Boolean(node.show),
        banned: Boolean(node.banned),
        custom_outbounds: JSON.stringify(node.custom_outbounds || [], null, 2),
        custom_routes: JSON.stringify(node.custom_routes || [], null, 2),
        protocol_settings: normalized,
    };
}

function setDeep(obj, path, value) {
    const keys = path.split(".");
    const clone = structuredClone(obj);
    let cur = clone;
    for (let i = 0; i < keys.length - 1; i++) {
        if (cur[keys[i]] == null || typeof cur[keys[i]] !== "object") cur[keys[i]] = {};
        cur = cur[keys[i]];
    }
    cur[keys[keys.length - 1]] = value;
    return clone;
}

function Field({ label, description, children }) {
    return (
        <div className="space-y-1.5">
            {label && <label className="text-sm font-medium">{label}</label>}
            {children}
            {description && <p className="text-xs text-muted-foreground">{description}</p>}
        </div>
    );
}

function SwitchField({ label, description, checked, onChange }) {
    return (
        <div className="flex items-center justify-between rounded-md border px-3 py-2">
            <div>
                <div className="text-sm font-medium">{label}</div>
                {description && <p className="text-xs text-muted-foreground">{description}</p>}
            </div>
            <Switch checked={checked} onCheckedChange={onChange} />
        </div>
    );
}

function SimpleSelect({ value, onChange, options, placeholder, className }) {
    return (
        <Select value={String(value ?? "")} onValueChange={onChange} className={className}>
            <SelectTrigger><SelectValue placeholder={placeholder} /></SelectTrigger>
            <SelectContent>
                {options.map((opt) => (
                    <SelectItem key={opt.value} value={String(opt.value)}>{opt.label}</SelectItem>
                ))}
            </SelectContent>
        </Select>
    );
}

function MultiSelectField({ values, onChange, options, placeholder, emptyLabel }) {
    const [open, setOpen] = useState(false);
    const selected = new Set((values || []).map(String));
    const toggle = (id) => {
        const idStr = String(id);
        const next = selected.has(idStr) ? values.filter((v) => String(v) !== idStr) : [...values, idStr];
        onChange(next);
    };
    return (
        <div className="relative">
            <button type="button" className="flex min-h-9 w-full flex-wrap items-center gap-1 rounded-md border border-input bg-transparent px-3 py-1.5 text-sm shadow-sm" onClick={() => setOpen((v) => !v)}>
                {values?.length ? values.map((id) => {
                    const opt = options.find((o) => String(o.value) === String(id));
                    return <Badge key={id} variant="secondary" className="text-[10px]">{opt?.label || id}</Badge>;
                }) : <span className="text-muted-foreground">{placeholder}</span>}
            </button>
            {open && (
                <div className="absolute left-0 top-10 z-50 max-h-56 w-full overflow-y-auto rounded-md border bg-popover p-1 shadow-md">
                    {options.length === 0 && <div className="px-2 py-1.5 text-sm text-muted-foreground">{emptyLabel}</div>}
                    {options.map((opt) => (
                        <button key={opt.value} type="button" className="flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent" onClick={() => toggle(opt.value)}>
                            {opt.label}{selected.has(String(opt.value)) && " ✓"}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}

function EchFields({ t, value, onChange, nodeName }) {
    const ech = value || defaultEch();
    const [generating, setGenerating] = useState(false);
    const update = (patch) => onChange({ ...ech, ...patch });
    const handleGenerate = async () => {
        setGenerating(true);
        try {
            const res = await generateEchKey(nodeName || "");
            const data = res?.data || res || {};
            update({ key: data.key || "", config: data.config || "" });
        } finally {
            setGenerating(false);
        }
    };
    return (
        <div className="space-y-3 rounded-md border p-3">
            <SwitchField label="ECH" description={t("dynamic_form.ech.description")} checked={ech.enabled} onChange={(v) => update({ enabled: v })} />
            {ech.enabled && (
                <>
                    <div className="flex justify-end">
                        <Button type="button" variant="outline" size="sm" onClick={handleGenerate} loading={generating}>{t("dynamic_form.ech.generate")}</Button>
                    </div>
                    <Field label={t("dynamic_form.ech.config.label")} description={t("dynamic_form.ech.config.description")}>
                        <Textarea rows={3} value={ech.config} placeholder={t("dynamic_form.ech.config.placeholder")} onChange={(e) => update({ config: e.target.value })} />
                    </Field>
                    <Field label={t("dynamic_form.ech.key.label")} description={t("dynamic_form.ech.key.description")}>
                        <Textarea rows={2} value={ech.key} placeholder={t("dynamic_form.ech.key.placeholder")} onChange={(e) => update({ key: e.target.value })} />
                    </Field>
                    <Field label={t("dynamic_form.ech.query_server_name.label")}>
                        <Input value={ech.query_server_name} placeholder={t("dynamic_form.ech.query_server_name.placeholder")} onChange={(e) => update({ query_server_name: e.target.value })} />
                    </Field>
                </>
            )}
        </div>
    );
}

function ProtocolFields({ t, type, settings, update, nodeName }) {
    const s = (key) => key.split(".").reduce((acc, k) => (acc == null ? acc : acc[k]), settings);
    const set = (key, value) => update(`protocol_settings.${key}`, value);

    if (type === "shadowsocks") {
        return (
            <div className="space-y-4">
                <Field label={t("dynamic_form.shadowsocks.cipher.label")} description={t("dynamic_form.shadowsocks.cipher.description")}>
                    <SimpleSelect value={s("cipher")} onChange={(v) => set("cipher", v)} placeholder={t("dynamic_form.shadowsocks.cipher.placeholder")} options={CIPHER_PRESETS.map((c) => ({ value: c, label: c }))} />
                </Field>
                <Field label={t("dynamic_form.shadowsocks.plugin.label")}>
                    <SimpleSelect value={s("plugin") || "none"} onChange={(v) => set("plugin", v === "none" ? "" : v)} placeholder={t("dynamic_form.shadowsocks.plugin.placeholder")} options={PLUGIN_OPTIONS.map((p) => ({ value: p, label: p }))} />
                </Field>
                {s("plugin") && (
                    <Field label={t("dynamic_form.shadowsocks.plugin_opts.label")} description={t("dynamic_form.shadowsocks.plugin_opts.description")}>
                        <Input value={s("plugin_opts")} placeholder={t("dynamic_form.shadowsocks.plugin_opts.placeholder")} onChange={(e) => set("plugin_opts", e.target.value)} />
                    </Field>
                )}
                {(s("plugin") === "shadow-tls" || s("plugin") === "restls") && (
                    <Field label={t("dynamic_form.shadowsocks.client_fingerprint")} description={t("dynamic_form.shadowsocks.client_fingerprint_description")}>
                        <SimpleSelect value={s("client_fingerprint") || "chrome"} onChange={(v) => set("client_fingerprint", v)} placeholder={t("dynamic_form.shadowsocks.client_fingerprint_placeholder")} options={FINGERPRINT_OPTIONS.map((f) => ({ value: f, label: f }))} />
                    </Field>
                )}
            </div>
        );
    }

    if (type === "vmess" || type === "vless") {
        const tlsOptions = type === "vless"
            ? [{ value: "0", label: t("dynamic_form.vless.tls.none") }, { value: "1", label: t("dynamic_form.vless.tls.tls") }, { value: "2", label: t("dynamic_form.vless.tls.reality") }]
            : [{ value: "0", label: t("dynamic_form.vmess.tls.disabled") }, { value: "1", label: t("dynamic_form.vmess.tls.enabled") }];
        const ns = `dynamic_form.${type}`;
        return (
            <div className="space-y-4">
                <Field label={t(`${ns}.tls.label`)}>
                    <SimpleSelect value={String(s("tls") ?? 0)} onChange={(v) => set("tls", parseInt(v, 10))} placeholder={t(`${ns}.tls.placeholder`)} options={tlsOptions} />
                </Field>
                {String(s("tls")) === "1" && (
                    <div className="space-y-3 rounded-md border p-3">
                        <Field label={t(`${ns}.tls_settings.server_name.label`)}>
                            <Input value={s("tls_settings.server_name")} placeholder={t(`${ns}.tls_settings.server_name.placeholder`)} onChange={(e) => set("tls_settings.server_name", e.target.value)} />
                        </Field>
                        <SwitchField label={t(`${ns}.tls_settings.allow_insecure`)} checked={Boolean(s("tls_settings.allow_insecure"))} onChange={(v) => set("tls_settings.allow_insecure", v)} />
                        <EchFields t={t} value={s("tls_settings.ech")} onChange={(v) => set("tls_settings.ech", v)} nodeName={nodeName} />
                    </div>
                )}
                {type === "vless" && String(s("tls")) === "2" && (
                    <div className="space-y-3 rounded-md border p-3">
                        <Field label={t("dynamic_form.vless.reality_settings.server_name.label")}>
                            <Input value={s("reality_settings.server_name")} placeholder={t("dynamic_form.vless.reality_settings.server_name.placeholder")} onChange={(e) => set("reality_settings.server_name", e.target.value)} />
                        </Field>
                        <Field label={t("dynamic_form.vless.reality_settings.server_port.label")}>
                            <Input type="number" value={s("reality_settings.server_port")} placeholder={t("dynamic_form.vless.reality_settings.server_port.placeholder")} onChange={(e) => set("reality_settings.server_port", e.target.value)} />
                        </Field>
                        <SwitchField label={t("dynamic_form.vless.reality_settings.allow_insecure")} checked={Boolean(s("reality_settings.allow_insecure"))} onChange={(v) => set("reality_settings.allow_insecure", v)} />
                        <Field label={t("dynamic_form.vless.reality_settings.private_key.label")}>
                            <Input value={s("reality_settings.private_key")} onChange={(e) => set("reality_settings.private_key", e.target.value)} />
                        </Field>
                        <Field label={t("dynamic_form.vless.reality_settings.public_key.label")}>
                            <Input value={s("reality_settings.public_key")} onChange={(e) => set("reality_settings.public_key", e.target.value)} />
                        </Field>
                        <Field label={t("dynamic_form.vless.reality_settings.short_id.label")} description={t("dynamic_form.vless.reality_settings.short_id.description")}>
                            <Input value={s("reality_settings.short_id")} placeholder={t("dynamic_form.vless.reality_settings.short_id.placeholder")} onChange={(e) => set("reality_settings.short_id", e.target.value)} />
                        </Field>
                    </div>
                )}
                <Field label={t(`${ns}.network.label`)}>
                    <SimpleSelect value={s("network") || "tcp"} onChange={(v) => set("network", v)} placeholder={t(`${ns}.network.placeholder`)} options={NETWORK_OPTIONS.map((n) => ({ value: n, label: n }))} />
                </Field>
                {s("network") && s("network") !== "tcp" && (
                    <Field label={t("network_settings.json_config_placeholder")}>
                        <Textarea rows={4} className="font-mono text-xs" value={s("network_settings") || "{}"} placeholder={t("network_settings.json_config_placeholder")} onChange={(e) => set("network_settings", e.target.value)} />
                    </Field>
                )}
                {type === "vless" && (
                    <>
                        <Field label={t("dynamic_form.vless.flow.label")}>
                            <Input value={s("flow")} placeholder={t("dynamic_form.vless.flow.placeholder")} onChange={(e) => set("flow", e.target.value)} />
                        </Field>
                        <div className="space-y-3 rounded-md border p-3">
                            <p className="text-sm font-medium">{t("dynamic_form.vless.encryption.label")}</p>
                            <p className="text-xs text-muted-foreground">{t("dynamic_form.vless.encryption.description")}</p>
                            <Field label={t("dynamic_form.vless.encryption.server_label")}>
                                <Input value={s("encryption.server")} placeholder={t("dynamic_form.vless.encryption.server_placeholder")} onChange={(e) => set("encryption.server", e.target.value)} />
                            </Field>
                            <Field label={t("dynamic_form.vless.encryption.client_label")}>
                                <Input value={s("encryption.client")} placeholder={t("dynamic_form.vless.encryption.client_placeholder")} onChange={(e) => set("encryption.client", e.target.value)} />
                            </Field>
                            <p className="text-xs text-muted-foreground">{t("dynamic_form.vless.encryption.generate_hint")}</p>
                        </div>
                    </>
                )}
            </div>
        );
    }

    if (type === "trojan") {
        return (
            <div className="space-y-4">
                <Field label={t("dynamic_form.trojan.server_name.label")}>
                    <Input value={s("server_name")} placeholder={t("dynamic_form.trojan.server_name.placeholder")} onChange={(e) => set("server_name", e.target.value)} />
                </Field>
                <SwitchField label={t("dynamic_form.trojan.allow_insecure")} checked={Boolean(s("allow_insecure"))} onChange={(v) => set("allow_insecure", v)} />
                <SwitchField label="Reality" checked={Boolean(s("use_reality"))} onChange={(v) => set("use_reality", v)} />
                {s("use_reality") && (
                    <div className="space-y-3 rounded-md border p-3">
                        <Field label={t("dynamic_form.trojan.reality_settings.server_name.label")}>
                            <Input value={s("reality_settings.server_name")} placeholder={t("dynamic_form.trojan.reality_settings.server_name.placeholder")} onChange={(e) => set("reality_settings.server_name", e.target.value)} />
                        </Field>
                        <Field label={t("dynamic_form.trojan.reality_settings.server_port.label")}>
                            <Input type="number" value={s("reality_settings.server_port")} placeholder={t("dynamic_form.trojan.reality_settings.server_port.placeholder")} onChange={(e) => set("reality_settings.server_port", e.target.value)} />
                        </Field>
                        <Field label={t("dynamic_form.trojan.reality_settings.private_key.label")}>
                            <Input value={s("reality_settings.private_key")} onChange={(e) => set("reality_settings.private_key", e.target.value)} />
                        </Field>
                        <Field label={t("dynamic_form.trojan.reality_settings.public_key.label")}>
                            <Input value={s("reality_settings.public_key")} onChange={(e) => set("reality_settings.public_key", e.target.value)} />
                        </Field>
                        <Field label={t("dynamic_form.trojan.reality_settings.short_id.label")}>
                            <Input value={s("reality_settings.short_id")} placeholder={t("dynamic_form.trojan.reality_settings.short_id.placeholder")} onChange={(e) => set("reality_settings.short_id", e.target.value)} />
                        </Field>
                    </div>
                )}
                <Field label={t("dynamic_form.trojan.network.label")}>
                    <SimpleSelect value={s("network") || "tcp"} onChange={(v) => set("network", v)} placeholder={t("dynamic_form.trojan.network.placeholder")} options={NETWORK_OPTIONS.map((n) => ({ value: n, label: n }))} />
                </Field>
                {s("network") && s("network") !== "tcp" && (
                    <Field label={t("network_settings.json_config_placeholder")}>
                        <Textarea rows={4} className="font-mono text-xs" value={s("network_settings") || "{}"} onChange={(e) => set("network_settings", e.target.value)} />
                    </Field>
                )}
            </div>
        );
    }

    if (type === "hysteria") {
        return (
            <div className="space-y-4">
                <Field label={t("dynamic_form.hysteria.version.label")}>
                    <SimpleSelect value={s("version") || "2"} onChange={(v) => set("version", v)} placeholder={t("dynamic_form.hysteria.version.placeholder")} options={[{ value: "1", label: "Hysteria 1" }, { value: "2", label: "Hysteria 2" }]} />
                </Field>
                <Field label={t("dynamic_form.hysteria.alpn.label")}>
                    <Input value={s("alpn")} placeholder={t("dynamic_form.hysteria.alpn.placeholder")} onChange={(e) => set("alpn", e.target.value)} />
                </Field>
                <Field label={t("dynamic_form.hysteria.obfs.type.label")}>
                    <SimpleSelect value={s("obfs.type") || "none"} onChange={(v) => set("obfs.type", v === "none" ? "" : v)} placeholder={t("dynamic_form.hysteria.obfs.type.placeholder")} options={[{ value: "none", label: "-" }, { value: "salamander", label: t("dynamic_form.hysteria.obfs.type.salamander") }]} />
                </Field>
                {s("obfs.type") && (
                    <Field label={t("dynamic_form.hysteria.obfs.password.label")}>
                        <Input value={s("obfs.password")} placeholder={t("dynamic_form.hysteria.obfs.password.placeholder")} onChange={(e) => set("obfs.password", e.target.value)} />
                    </Field>
                )}
                <Field label={t("dynamic_form.hysteria.tls.server_name.label")}>
                    <Input value={s("tls.server_name")} placeholder={t("dynamic_form.hysteria.tls.server_name.placeholder")} onChange={(e) => set("tls.server_name", e.target.value)} />
                </Field>
                <SwitchField label={t("dynamic_form.hysteria.tls.allow_insecure")} checked={Boolean(s("tls.allow_insecure"))} onChange={(v) => set("tls.allow_insecure", v)} />
                <EchFields t={t} value={s("tls.ech")} onChange={(v) => set("tls.ech", v)} nodeName={nodeName} />
                <div className="grid grid-cols-2 gap-3">
                    <Field label={`${t("dynamic_form.hysteria.bandwidth.up.label")} (${t("dynamic_form.hysteria.bandwidth.up.suffix")})`} description={t("dynamic_form.hysteria.bandwidth.up.bbr_tip")}>
                        <Input type="number" value={s("bandwidth.up")} placeholder={t("dynamic_form.hysteria.bandwidth.up.placeholder")} onChange={(e) => set("bandwidth.up", e.target.value)} />
                    </Field>
                    <Field label={`${t("dynamic_form.hysteria.bandwidth.down.label")} (${t("dynamic_form.hysteria.bandwidth.down.suffix")})`} description={t("dynamic_form.hysteria.bandwidth.down.bbr_tip")}>
                        <Input type="number" value={s("bandwidth.down")} placeholder={t("dynamic_form.hysteria.bandwidth.down.placeholder")} onChange={(e) => set("bandwidth.down", e.target.value)} />
                    </Field>
                </div>
            </div>
        );
    }

    if (type === "tuic") {
        return (
            <div className="space-y-4">
                <Field label={t("dynamic_form.tuic.version.label")}>
                    <SimpleSelect value={s("version") || "5"} onChange={(v) => set("version", v)} placeholder={t("dynamic_form.tuic.version.placeholder")} options={[{ value: "5", label: "5" }]} />
                </Field>
                <Field label={t("dynamic_form.tuic.password.label")}>
                    <Input value={s("password")} placeholder={t("dynamic_form.tuic.password.placeholder")} onChange={(e) => set("password", e.target.value)} />
                </Field>
                <Field label={t("dynamic_form.tuic.congestion_control.label")}>
                    <SimpleSelect value={s("congestion_control") || "bbr"} onChange={(v) => set("congestion_control", v)} placeholder={t("dynamic_form.tuic.congestion_control.placeholder")} options={TUIC_CONGESTION.map((c) => ({ value: c, label: c }))} />
                </Field>
                <Field label={t("dynamic_form.tuic.udp_relay_mode.label")}>
                    <SimpleSelect value={s("udp_relay_mode") || "native"} onChange={(v) => set("udp_relay_mode", v)} placeholder={t("dynamic_form.tuic.udp_relay_mode.placeholder")} options={TUIC_RELAY_MODE.map((c) => ({ value: c, label: c }))} />
                </Field>
                <Field label={t("dynamic_form.tuic.tls.server_name.label")}>
                    <Input value={s("tls.server_name")} placeholder={t("dynamic_form.tuic.tls.server_name.placeholder")} onChange={(e) => set("tls.server_name", e.target.value)} />
                </Field>
                <SwitchField label={t("dynamic_form.tuic.tls.allow_insecure")} checked={Boolean(s("tls.allow_insecure"))} onChange={(v) => set("tls.allow_insecure", v)} />
                <Field label={t("dynamic_form.tuic.tls.alpn.label")} description={t("dynamic_form.tuic.tls.alpn.placeholder")}>
                    <Input value={s("tls.alpn")} onChange={(e) => set("tls.alpn", e.target.value)} />
                </Field>
            </div>
        );
    }

    if (type === "socks") {
        return (
            <div className="space-y-4">
                <Field label={t("dynamic_form.socks.version.label")}>
                    <SimpleSelect value={s("version") || "5"} onChange={(v) => set("version", v)} placeholder={t("dynamic_form.socks.version.placeholder")} options={SOCKS_VERSIONS.map((v) => ({ value: v, label: v }))} />
                </Field>
                <Field label={t("dynamic_form.socks.tls.label")}>
                    <SimpleSelect value={String(s("tls") ?? 0)} onChange={(v) => set("tls", parseInt(v, 10))} placeholder={t("dynamic_form.socks.tls.placeholder")} options={[{ value: "0", label: t("dynamic_form.socks.tls.disabled") }, { value: "1", label: t("dynamic_form.socks.tls.enabled") }]} />
                </Field>
                {String(s("tls")) === "1" && (
                    <div className="space-y-3 rounded-md border p-3">
                        <Field label={t("dynamic_form.socks.tls_settings.server_name.label")}>
                            <Input value={s("tls_settings.server_name")} placeholder={t("dynamic_form.socks.tls_settings.server_name.placeholder")} onChange={(e) => set("tls_settings.server_name", e.target.value)} />
                        </Field>
                        <SwitchField label={t("dynamic_form.socks.tls_settings.allow_insecure")} checked={Boolean(s("tls_settings.allow_insecure"))} onChange={(v) => set("tls_settings.allow_insecure", v)} />
                    </div>
                )}
                <Field label={t("dynamic_form.socks.network.label")}>
                    <SimpleSelect value={s("network") || "tcp"} onChange={(v) => set("network", v)} placeholder={t("dynamic_form.socks.network.placeholder")} options={[{ value: "tcp", label: "tcp" }]} />
                </Field>
            </div>
        );
    }

    if (type === "naive" || type === "http") {
        const ns = `dynamic_form.${type}`;
        return (
            <div className="space-y-4">
                <Field label={t(`${ns}.tls.label`)}>
                    <SimpleSelect value={String(s("tls") ?? 0)} onChange={(v) => set("tls", parseInt(v, 10))} placeholder={t(`${ns}.tls.placeholder`)} options={[{ value: "0", label: t(`${ns}.tls.disabled`) }, { value: "1", label: t(`${ns}.tls.enabled`) }]} />
                </Field>
                {String(s("tls")) === "1" && (
                    <div className="space-y-3 rounded-md border p-3">
                        <Field label={t(`${ns}.tls_settings.server_name.label`)}>
                            <Input value={s("tls_settings.server_name")} placeholder={t(`${ns}.tls_settings.server_name.placeholder`)} onChange={(e) => set("tls_settings.server_name", e.target.value)} />
                        </Field>
                        <SwitchField label={t(`${ns}.tls_settings.allow_insecure`)} checked={Boolean(s("tls_settings.allow_insecure"))} onChange={(v) => set("tls_settings.allow_insecure", v)} />
                    </div>
                )}
            </div>
        );
    }

    if (type === "mieru") {
        return (
            <div className="space-y-4">
                <Field label={t("dynamic_form.mieru.transport.label")}>
                    <SimpleSelect value={s("transport") || "TCP"} onChange={(v) => set("transport", v)} placeholder={t("dynamic_form.mieru.transport.placeholder")} options={[{ value: "TCP", label: "TCP" }]} />
                </Field>
                <Field label={t("dynamic_form.mieru.traffic_pattern.label")}>
                    <Textarea rows={3} value={s("traffic_pattern")} placeholder={t("dynamic_form.mieru.traffic_pattern.placeholder")} onChange={(e) => set("traffic_pattern", e.target.value)} />
                </Field>
            </div>
        );
    }

    if (type === "anytls") {
        return (
            <div className="space-y-4">
                <Field label={t("dynamic_form.anytls.tls.server_name.label")}>
                    <Input value={s("tls.server_name")} placeholder={t("dynamic_form.anytls.tls.server_name.placeholder")} onChange={(e) => set("tls.server_name", e.target.value)} />
                </Field>
                <SwitchField label={t("dynamic_form.anytls.tls.allow_insecure")} checked={Boolean(s("tls.allow_insecure"))} onChange={(v) => set("tls.allow_insecure", v)} />
                <EchFields t={t} value={s("tls.ech")} onChange={(v) => set("tls.ech", v)} nodeName={nodeName} />
                <Field label={t("dynamic_form.anytls.padding_scheme.label")} description={t("dynamic_form.anytls.padding_scheme.description")}>
                    <Textarea rows={4} className="font-mono text-xs" value={s("padding_scheme")} placeholder={t("dynamic_form.anytls.padding_scheme.placeholder")} onChange={(e) => set("padding_scheme", e.target.value)} />
                </Field>
            </div>
        );
    }

    return <p className="text-sm text-muted-foreground">{t("form.type.select_prompt")}</p>;
}

function MultiplexFields({ t, value, update }) {
    const mp = value || defaultMultiplex();
    const set = (patch) => update("protocol_settings.multiplex", { ...mp, ...patch });
    return (
        <div className="space-y-4">
            <SwitchField label={t("dynamic_form.multiplex.enabled.label")} description={t("dynamic_form.multiplex.enabled.description")} checked={Boolean(mp.enabled)} onChange={(v) => set({ enabled: v })} />
            {mp.enabled && (
                <>
                    <Field label={t("dynamic_form.multiplex.protocol.label")}>
                        <SimpleSelect value={mp.protocol || "smux"} onChange={(v) => set({ protocol: v })} options={MULTIPLEX_PROTOCOLS.map((p) => ({ value: p, label: p }))} />
                    </Field>
                    <div className="grid grid-cols-2 gap-3">
                        <Field label={t("dynamic_form.multiplex.max_connections.label")}>
                            <Input type="number" value={mp.max_connections} onChange={(e) => set({ max_connections: e.target.value })} />
                        </Field>
                        <Field label={t("dynamic_form.multiplex.min_streams.label")}>
                            <Input type="number" value={mp.min_streams} onChange={(e) => set({ min_streams: e.target.value })} />
                        </Field>
                    </div>
                    <SwitchField label={t("dynamic_form.multiplex.padding.label")} checked={Boolean(mp.padding)} onChange={(v) => set({ padding: v })} />
                    <SwitchField label={t("dynamic_form.multiplex.brutal.enabled.label")} description={t("dynamic_form.multiplex.brutal.description")} checked={Boolean(mp.brutal?.enabled)} onChange={(v) => set({ brutal: { ...mp.brutal, enabled: v } })} />
                    {mp.brutal?.enabled && (
                        <div className="grid grid-cols-2 gap-3">
                            <Field label={t("dynamic_form.multiplex.brutal.up_mbps.label")}>
                                <Input type="number" value={mp.brutal.up_mbps} onChange={(e) => set({ brutal: { ...mp.brutal, up_mbps: e.target.value } })} />
                            </Field>
                            <Field label={t("dynamic_form.multiplex.brutal.down_mbps.label")}>
                                <Input type="number" value={mp.brutal.down_mbps} onChange={(e) => set({ brutal: { ...mp.brutal, down_mbps: e.target.value } })} />
                            </Field>
                        </div>
                    )}
                </>
            )}
        </div>
    );
}

export function NodeFormDialog({ open, onOpenChange, node, groups, routes, machines, nodes, onSaved, t }) {
    const [form, setForm] = useState(emptyForm);
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [tab, setTab] = useState("basic");

    useEffect(() => {
        if (open) {
            setForm(nodeToForm(node));
            setError("");
            setTab("basic");
        }
    }, [open, node]);

    const update = (path, value) => setForm((prev) => setDeep(prev, path, value));

    const handleTypeChange = (type) => {
        setForm((prev) => ({ ...prev, type, protocol_settings: protocolDefaults(type) }));
    };

    const addTag = () => {
        const value = form.tagInput.trim();
        if (!value) return;
        if (form.tags.includes(value)) { update("tagInput", ""); return; }
        setForm((prev) => ({ ...prev, tags: [...prev.tags, value], tagInput: "" }));
    };
    const removeTag = (tag) => setForm((prev) => ({ ...prev, tags: prev.tags.filter((x) => x !== tag) }));

    const addRateRule = () => {
        setForm((prev) => ({ ...prev, rate_time_ranges: [...prev.rate_time_ranges, { start: "", end: "", rate: "1" }] }));
    };
    const updateRateRule = (index, patch) => {
        setForm((prev) => ({ ...prev, rate_time_ranges: prev.rate_time_ranges.map((r, i) => (i === index ? { ...r, ...patch } : r)) }));
    };
    const removeRateRule = (index) => {
        setForm((prev) => ({ ...prev, rate_time_ranges: prev.rate_time_ranges.filter((_, i) => i !== index) }));
    };

    const handleSubmit = async () => {
        setError("");
        if (!form.type) { setError(t("form.type.select_error")); return; }
        if (!form.name.trim()) { setError(t("form.name.error")); return; }
        if (!form.host.trim()) { setError(t("form.host.error")); return; }
        if (!String(form.port).trim()) { setError(t("form.port.error")); return; }
        if (!String(form.server_port).trim()) { setError(t("form.server_port.error")); return; }
        const rateNum = parseFloat(form.rate);
        if (form.rate === "" || Number.isNaN(rateNum)) { setError(t("form.rate.error_numeric")); return; }
        if (rateNum < 0) { setError(t("form.rate.error_gte_zero")); return; }

        let customOutbounds;
        let customRoutes;
        try {
            customOutbounds = form.custom_outbounds ? JSON.parse(form.custom_outbounds) : [];
            if (!Array.isArray(customOutbounds)) throw new Error();
        } catch {
            setError(t("dynamic_form.routing.error.must_be_array"));
            return;
        }
        try {
            customRoutes = form.custom_routes ? JSON.parse(form.custom_routes) : [];
            if (!Array.isArray(customRoutes)) throw new Error();
        } catch {
            setError(t("dynamic_form.routing.error.must_be_array"));
            return;
        }

        const protocolSettings = structuredClone(form.protocol_settings);
        if (typeof protocolSettings.network_settings === "string") {
            try {
                protocolSettings.network_settings = protocolSettings.network_settings.trim() ? JSON.parse(protocolSettings.network_settings) : {};
            } catch {
                setError(t("network_settings.validation.invalid_json"));
                return;
            }
        }

        const payload = {
            ...(form.id ? { id: form.id } : {}),
            type: form.type,
            code: form.code || undefined,
            name: form.name.trim(),
            host: form.host.trim(),
            port: form.port,
            server_port: form.server_port,
            listen_address: form.listenAddressVisible ? form.listen_address : "",
            rate: form.rate,
            rate_time_enable: form.rate_time_enable,
            rate_time_ranges: form.rate_time_ranges,
            transfer_enable: form.transfer_enable_gb ? Math.round(parseFloat(form.transfer_enable_gb) * 1024 * 1024 * 1024) : 0,
            group_ids: form.group_ids,
            route_ids: form.route_ids,
            parent_id: form.parent_id,
            machine_id: form.machine_id || null,
            enabled: form.machine_id ? form.enabled : null,
            tags: form.tags,
            show: form.show,
            banned: form.banned,
            custom_outbounds: customOutbounds,
            custom_routes: customRoutes,
            protocol_settings: protocolSettings,
        };

        setSubmitting(true);
        try {
            await saveNode(payload);
        } catch (e) {
            setError(e?.message || t("form.type.select_error"));
            setSubmitting(false);
            return;
        }
        setSubmitting(false);
        onOpenChange(false);
        await onSaved?.();
    };

    const groupOptions = (groups || []).map((g) => ({ value: g.id, label: g.name }));
    const routeOptions = (routes || []).map((r) => ({ value: r.id, label: r.remarks || `#${r.id}` }));
    const parentOptions = [{ value: "0", label: t("form.parent.none") }, ...(nodes || []).filter((n) => n.id !== form.id).map((n) => ({ value: n.id, label: n.name }))];
    const machineOptions = [{ value: "", label: t("form.machine.none") }, ...(machines || []).map((m) => ({ value: m.id, label: m.name }))];

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-3xl">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{form.id ? t("form.edit_node") : t("form.add_node")}</h3>

                {error && <div className="mt-3 rounded-md bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}

                <Tabs value={tab} onValueChange={setTab} className="mt-4">
                    <TabsList>
                        <TabsTrigger value="basic">{t("manage.title")}</TabsTrigger>
                        <TabsTrigger value="protocol">{t("columns.type")}</TabsTrigger>
                        <TabsTrigger value="advanced">{t("dynamic_form.advanced.trigger_label")}</TabsTrigger>
                    </TabsList>

                    <TabsContent value="basic">
                        <div className="space-y-4 pt-4">
                            <Field label={t("form.type.placeholder")}>
                                <SimpleSelect value={form.type} onChange={handleTypeChange} placeholder={t("form.type.placeholder")} options={PROTOCOL_TYPES.map((p) => ({ value: p, label: p }))} />
                            </Field>
                            <div className="grid grid-cols-2 gap-3">
                                <Field label={t("form.name.label")}>
                                    <Input value={form.name} placeholder={t("form.name.placeholder")} onChange={(e) => update("name", e.target.value)} />
                                </Field>
                                <Field label={`${t("form.code.label")} ${t("form.code.optional")}`}>
                                    <Input value={form.code} placeholder={t("form.code.placeholder")} onChange={(e) => update("code", e.target.value)} />
                                </Field>
                            </div>
                            <Field label={t("form.host.label")}>
                                <Input value={form.host} placeholder={t("form.host.placeholder")} onChange={(e) => update("host", e.target.value)} />
                            </Field>
                            <div className="grid grid-cols-2 gap-3">
                                <Field label={t("form.port.label")}>
                                    <Input type="number" value={form.port} placeholder={t("form.port.placeholder")} onChange={(e) => update("port", e.target.value)} />
                                </Field>
                                <Field label={t("form.server_port.label")}>
                                    <div className="flex gap-2">
                                        <Input type="number" value={form.server_port} placeholder={t("form.server_port.placeholder")} onChange={(e) => update("server_port", e.target.value)} />
                                        <Button type="button" variant="outline" size="sm" onClick={() => update("port", form.server_port)}>{t("form.port.sync")}</Button>
                                    </div>
                                </Field>
                            </div>

                            {form.listenAddressVisible ? (
                                <Field label={t("form.listen_address.label")} description={t("form.listen_address.description")}>
                                    <div className="flex gap-2">
                                        <Input value={form.listen_address} placeholder={t("form.listen_address.placeholder")} onChange={(e) => update("listen_address", e.target.value)} />
                                        <Button type="button" variant="outline" size="sm" onClick={() => update("listenAddressVisible", false)}>{t("form.listen_address.hide")}</Button>
                                    </div>
                                </Field>
                            ) : (
                                <Button type="button" variant="ghost" size="sm" onClick={() => update("listenAddressVisible", true)}>{t("form.listen_address.show")}</Button>
                            )}

                            <div className="grid grid-cols-2 gap-3">
                                <Field label={t("form.rate.label")}>
                                    <Input type="number" step="0.01" value={form.rate} onChange={(e) => update("rate", e.target.value)} />
                                </Field>
                                <Field label={t("form.traffic_limit.label")} description={t("form.traffic_limit.hint")}>
                                    <Input type="number" value={form.transfer_enable_gb} placeholder={t("form.traffic_limit.placeholder")} onChange={(e) => update("transfer_enable_gb", e.target.value)} />
                                </Field>
                            </div>

                            <SwitchField label={t("form.dynamic_rate.enable_label")} description={t("form.dynamic_rate.enable_description")} checked={form.rate_time_enable} onChange={(v) => update("rate_time_enable", v)} />
                            {form.rate_time_enable && (
                                <div className="space-y-2 rounded-md border p-3">
                                    <div className="flex items-center justify-between">
                                        <p className="text-sm font-medium">{t("form.dynamic_rate.rules_label")}</p>
                                        <Button type="button" variant="outline" size="sm" onClick={addRateRule}>{t("form.dynamic_rate.add_rule")}</Button>
                                    </div>
                                    {form.rate_time_ranges.length === 0 && <p className="text-xs text-muted-foreground">{t("form.dynamic_rate.no_rules")}</p>}
                                    {form.rate_time_ranges.map((rule, index) => (
                                        <div key={index} className="flex items-center gap-2">
                                            <Input placeholder={t("form.dynamic_rate.start_time")} value={rule.start} onChange={(e) => updateRateRule(index, { start: e.target.value })} />
                                            <Input placeholder={t("form.dynamic_rate.end_time")} value={rule.end} onChange={(e) => updateRateRule(index, { end: e.target.value })} />
                                            <Input type="number" step="0.01" placeholder={t("form.dynamic_rate.multiplier")} value={rule.rate} onChange={(e) => updateRateRule(index, { rate: e.target.value })} />
                                            <Button type="button" variant="ghost" size="sm" onClick={() => removeRateRule(index)}>×</Button>
                                        </div>
                                    ))}
                                </div>
                            )}

                            <Field label={t("form.groups.label")}>
                                <MultiSelectField values={form.group_ids} onChange={(v) => update("group_ids", v)} options={groupOptions} placeholder={t("form.groups.placeholder")} emptyLabel={t("form.groups.empty")} />
                            </Field>
                            <Field label={t("form.route.label")}>
                                <MultiSelectField values={form.route_ids} onChange={(v) => update("route_ids", v)} options={routeOptions} placeholder={t("form.route.placeholder")} emptyLabel={t("form.route.empty")} />
                            </Field>
                            <div className="grid grid-cols-2 gap-3">
                                <Field label={t("form.parent.label")}>
                                    <SimpleSelect value={form.parent_id} onChange={(v) => update("parent_id", v)} placeholder={t("form.parent.placeholder")} options={parentOptions} />
                                </Field>
                                <Field label={t("form.machine.label")} description={t("form.machine.enabled_hint")}>
                                    <SimpleSelect value={form.machine_id} onChange={(v) => update("machine_id", v)} placeholder={t("form.machine.placeholder")} options={machineOptions} />
                                </Field>
                            </div>
                            {form.machine_id && <SwitchField label={t("form.machine.enabled_hint")} checked={form.enabled} onChange={(v) => update("enabled", v)} />}

                            <Field label={t("form.tags.label")}>
                                <div className="flex flex-wrap items-center gap-1.5 rounded-md border p-2">
                                    {form.tags.map((tag) => (
                                        <Badge key={tag} variant="secondary" className="gap-1 text-[10px]">
                                            {tag}
                                            <button type="button" onClick={() => removeTag(tag)}>×</button>
                                        </Badge>
                                    ))}
                                    <input
                                        className="min-w-[8rem] flex-1 bg-transparent text-sm outline-none"
                                        value={form.tagInput}
                                        placeholder={t("form.tags.placeholder")}
                                        onChange={(e) => update("tagInput", e.target.value)}
                                        onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addTag(); } }}
                                    />
                                </div>
                            </Field>

                            <div className="grid grid-cols-2 gap-3">
                                <SwitchField label={t("columns.show")} checked={form.show} onChange={(v) => update("show", v)} />
                                <SwitchField label={t("form.banned.label")} description={t("form.banned.description")} checked={form.banned} onChange={(v) => update("banned", v)} />
                            </div>
                        </div>
                    </TabsContent>

                    <TabsContent value="protocol">
                        <div className="pt-4">
                            <ProtocolFields t={t} type={form.type} settings={form.protocol_settings} update={update} nodeName={form.name} />
                        </div>
                    </TabsContent>

                    <TabsContent value="advanced">
                        <div className="space-y-6 pt-4">
                            {form.protocol_settings?.cert_config && (
                                <div className="space-y-3">
                                    <p className="text-sm font-medium">{t("dynamic_form.cert_config.tab")}</p>
                                    <Field label={t("dynamic_form.cert_config.cert_mode.label")} description={t("dynamic_form.cert_config.cert_mode.description")}>
                                        <SimpleSelect
                                            value={form.protocol_settings.cert_config.cert_mode || "none"}
                                            onChange={(v) => update("protocol_settings.cert_config.cert_mode", v)}
                                            options={["none", "self", "http", "dns", "content"].map((m) => ({ value: m, label: m }))}
                                        />
                                    </Field>
                                    {form.protocol_settings.cert_config.cert_mode !== "none" && (
                                        <>
                                            <Field label={t("dynamic_form.cert_config.domain.label")}>
                                                <Input value={form.protocol_settings.cert_config.domain} onChange={(e) => update("protocol_settings.cert_config.domain", e.target.value)} />
                                            </Field>
                                            {(form.protocol_settings.cert_config.cert_mode === "http" || form.protocol_settings.cert_config.cert_mode === "dns") && (
                                                <Field label={t("dynamic_form.cert_config.email.label")}>
                                                    <Input value={form.protocol_settings.cert_config.email} onChange={(e) => update("protocol_settings.cert_config.email", e.target.value)} />
                                                </Field>
                                            )}
                                            {form.protocol_settings.cert_config.cert_mode === "http" && (
                                                <Field label={t("dynamic_form.cert_config.http_port.label")} description={t("dynamic_form.cert_config.http_port.description")}>
                                                    <Input type="number" value={form.protocol_settings.cert_config.http_port} onChange={(e) => update("protocol_settings.cert_config.http_port", e.target.value)} />
                                                </Field>
                                            )}
                                            {form.protocol_settings.cert_config.cert_mode === "dns" && (
                                                <>
                                                    <Field label={t("dynamic_form.cert_config.dns_provider.label")}>
                                                        <Input value={form.protocol_settings.cert_config.dns_provider} onChange={(e) => update("protocol_settings.cert_config.dns_provider", e.target.value)} />
                                                    </Field>
                                                    <Field label={t("dynamic_form.cert_config.dns_env.label")} description={t("dynamic_form.cert_config.dns_env.description_short")}>
                                                        <Textarea rows={3} value={form.protocol_settings.cert_config.dns_env} onChange={(e) => update("protocol_settings.cert_config.dns_env", e.target.value)} />
                                                    </Field>
                                                </>
                                            )}
                                            {form.protocol_settings.cert_config.cert_mode === "content" && (
                                                <>
                                                    <Field label={t("dynamic_form.cert_config.cert_content.label")}>
                                                        <Textarea rows={3} value={form.protocol_settings.cert_config.cert_content} onChange={(e) => update("protocol_settings.cert_config.cert_content", e.target.value)} />
                                                    </Field>
                                                    <Field label={t("dynamic_form.cert_config.key_content.label")}>
                                                        <Textarea rows={3} value={form.protocol_settings.cert_config.key_content} onChange={(e) => update("protocol_settings.cert_config.key_content", e.target.value)} />
                                                    </Field>
                                                </>
                                            )}
                                        </>
                                    )}
                                    <hr className="my-2" />
                                </div>
                            )}

                            {MULTIPLEX_TYPES.has(form.type) && (
                                <div className="space-y-3">
                                    <p className="text-sm font-medium">{t("dynamic_form.advanced.multiplex_tab")}</p>
                                    <MultiplexFields t={t} value={form.protocol_settings.multiplex} update={update} />
                                    <hr className="my-2" />
                                </div>
                            )}

                            <div className="space-y-3">
                                <p className="text-sm font-medium">{t("dynamic_form.advanced.route_tab")}</p>
                                <Field label={t("dynamic_form.routing.outbounds")}>
                                    <Textarea rows={4} className="font-mono text-xs" value={form.custom_outbounds} onChange={(e) => update("custom_outbounds", e.target.value)} />
                                </Field>
                                <Field label={t("dynamic_form.routing.routes")}>
                                    <Textarea rows={4} className="font-mono text-xs" value={form.custom_routes} onChange={(e) => update("custom_routes", e.target.value)} />
                                </Field>
                            </div>
                        </div>
                    </TabsContent>
                </Tabs>

                <div className="mt-6 flex justify-end gap-2">
                    <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>{t("form.cancel")}</Button>
                    <Button type="button" onClick={handleSubmit} loading={submitting}>{t("form.submit")}</Button>
                </div>
            </div>
        </Dialog>
    );
}
