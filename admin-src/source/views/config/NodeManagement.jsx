import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import {
    getNodes,
    getNodeSyncStats,
    probeNodeLatency,
    getNodeProbeStatus,
    stopNodeProbe,
    dropNode,
    batchDeleteNodes,
    batchUpdateNodes,
    copyNode,
    updateNode,
    sortNodes,
    resetNodeTraffic,
    batchResetNodeTraffic,
    fetchGroups,
    fetchMachines,
    fetchRoutes,
} from "../../api/server.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { Tooltip, TooltipTrigger, TooltipContent } from "../../components/Tooltip.jsx";
import { NodeFormDialog } from "./NodeFormDialog.jsx";
import { NodeImportDialog } from "./NodeImportDialog.jsx";

const PROTOCOL_TYPES = ["shadowsocks", "vmess", "trojan", "hysteria", "vless", "tuic", "socks", "naive", "http", "mieru", "anytls"];

const PROTOCOL_COLOR = {
    shadowsocks: "#489851",
    vmess: "#CB3180",
    trojan: "#EBB749",
    hysteria: "#5684e6",
    vless: "#94a3b8", // 原为近黑色，在深色主题下与背景融为一体
    tuic: "#00C853",
    socks: "#2196F3",
    naive: "#9C27B0",
    http: "#FF5722",
    mieru: "#4CAF50",
    anytls: "#7E57C2",
};

// 3 = 没有 agent 心跳，但面板实测连通（导入的第三方节点）
const STATUS_TONE = { 0: "bg-red-500", 1: "bg-amber-500", 2: "bg-emerald-500", 3: "bg-sky-500" };

function machineStatus(machine) {
    if (!machine) return "inactive";
    if (!machine.is_active) return "inactive";
    if (!machine.last_seen_at) return "offline";
    return Date.now() / 1000 - machine.last_seen_at < 300 ? "online" : "offline";
}

function formatBytes(value) {
    if (value == null) return "-";
    if (value < 1024) return `${value.toFixed(0)} B`;
    if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
    if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
    if (value < 1024 * 1024 * 1024 * 1024) return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    return `${(value / (1024 * 1024 * 1024 * 1024)).toFixed(2)} TB`;
}

function copyText(text) {
    if (!text) return;
    navigator.clipboard?.writeText(String(text)).catch(() => {});
}

function latencyTone(ms) {
    if (ms == null) return "text-muted-foreground";
    if (ms < 150) return "text-emerald-600 dark:text-emerald-400";
    if (ms < 400) return "text-amber-600 dark:text-amber-400";
    return "text-rose-600 dark:text-rose-400";
}

// 剩余秒数 -> 倒计时文案（<1h 显示 m:ss，否则 Hh Mm）
function formatCountdown(sec) {
    if (sec == null || sec < 0) sec = 0;
    if (sec >= 3600) {
        const h = Math.floor(sec / 3600);
        const m = Math.floor((sec % 3600) / 60);
        return `${h}h${m}m`;
    }
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return `${m}:${String(s).padStart(2, "0")}`;
}

function timeAgo(ts) {
    if (!ts) return "";
    const s = Math.max(0, Math.floor(Date.now() / 1000 - ts));
    if (s < 60) return `${s}s`;
    if (s < 3600) return `${Math.floor(s / 60)}m`;
    if (s < 86400) return `${Math.floor(s / 3600)}h`;
    return `${Math.floor(s / 86400)}d`;
}

// 节点管理里的「测速」列：延迟 + 国旗/国家码 + 真实出口IP + 测速时间
function renderLatency(node, t) {
    if (!node.last_latency_at) {
        return <span className="text-xs text-muted-foreground">{t("columns.latency.untested")}</span>;
    }
    const cc = node.country_code ? String(node.country_code).toUpperCase() : null;
    return (
        <div className="flex flex-col items-start gap-0.5 text-xs leading-tight">
            <div className="flex items-center gap-1.5">
                {cc && (
                    <Badge variant="secondary" className="px-1.5 py-0 text-[10px] font-medium tabular-nums">
                        {cc}
                    </Badge>
                )}
                {node.latency_ms == null ? (
                    <span className="font-medium text-rose-600 dark:text-rose-400">{t("columns.latency.unreachable")}</span>
                ) : (
                    <span className={`font-mono font-medium ${latencyTone(node.latency_ms)}`}>{node.latency_ms}ms</span>
                )}
            </div>
            {node.exit_ip && (
                <div className="font-mono text-[10px] text-muted-foreground">{t("columns.latency.exit")} {node.exit_ip}</div>
            )}
            <div className="text-[10px] text-muted-foreground">{t("columns.latency.testedAt")} {timeAgo(node.last_latency_at)}</div>
        </div>
    );
}

export function NodeManagement() {
    const { t } = useTranslation("server");
    const auth = useAuthState();
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [groups, setGroups] = useState([]);
    const [machines, setMachines] = useState([]);
    const [routes, setRoutes] = useState([]);

    const [search, setSearch] = useState("");
    const [typeFilter, setTypeFilter] = useState("all");
    const [machineFilter, setMachineFilter] = useState("all");
    const [groupFilter, setGroupFilter] = useState("all");

    const [selectedIds, setSelectedIds] = useState(() => new Set());

    const [sortMode, setSortMode] = useState(false);
    const [orderedItems, setOrderedItems] = useState([]);
    const [sort, setSort] = useState({ key: null, dir: "asc" });
    const [sorting, setSorting] = useState(false);
    const dragIndexRef = useRef(null);

    const [singleDeleteTarget, setSingleDeleteTarget] = useState(null);
    const [batchDeleteOpen, setBatchDeleteOpen] = useState(false);
    const [singleResetTarget, setSingleResetTarget] = useState(null);
    const [batchResetOpen, setBatchResetOpen] = useState(false);
    const [formOpen, setFormOpen] = useState(false);
    const [editingNode, setEditingNode] = useState(null);
    const [importOpen, setImportOpen] = useState(false);

    // 节点变更后有多少客户端已经同步过去了。改完节点想知道"生效了没有"时看它。
    const [syncStats, setSyncStats] = useState(null);
    const [probeStatus, setProbeStatus] = useState(null);
    const probePollRef = useRef(null);
    const probing = !!(probeStatus && probeStatus.running);
    const [stoppingProbe, setStoppingProbe] = useState(false);

    // 停止测速：当前正在测的节点会测完，之后不再继续（进度轮询照常，停下后自动收起进度条）
    const stopProbe = async () => {
        setStoppingProbe(true);
        try {
            await stopNodeProbe();
            const res = await getNodeProbeStatus();
            setProbeStatus(res?.data || null);
        } catch {
            /* 下一次轮询会刷新状态 */
        } finally {
            setStoppingProbe(false);
        }
    };

    // 秒级心跳，仅用于空闲时「下次测速」倒计时的实时刷新
    const [nowTs, setNowTs] = useState(() => Math.floor(Date.now() / 1000));
    useEffect(() => {
        const id = setInterval(() => setNowTs(Math.floor(Date.now() / 1000)), 1000);
        return () => clearInterval(id);
    }, []);

    const loadSyncStats = () =>
        getNodeSyncStats()
            .then((res) => setSyncStats(res?.data || null))
            // 统计只是辅助信息，接口不可用（后端还没升级/迁移没跑）时静默隐藏，
            // 不要影响节点管理本身。
            .catch(() => setSyncStats(null));

    const load = () => {
        setLoading(true);
        // 顺带刷新覆盖率统计：所有增删改节点的入口（含表单/导入弹窗保存后）
        // 最终都会走到 load()，放在这里就不用逐个调用点去补，也不会漏。
        // 不 await，避免统计接口拖慢节点列表的刷新。
        loadSyncStats();
        return getNodes()
            .then((res) => setItems(res?.data || []))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
        fetchGroups().then((res) => setGroups(res?.data || []));
        fetchMachines().then((res) => setMachines(res?.data || []));
        fetchRoutes().then((res) => setRoutes(res?.data || []));
        // 刷新后若后台仍在测速，继续显示进度
        getNodeProbeStatus()
            .then((res) => {
                const s = res?.data || null;
                setProbeStatus(s);
                if (s && s.running) startProbePolling();
            })
            .catch(() => {});
    }, [auth.token]);

    const machineById = useMemo(() => new Map(machines.map((m) => [m.id, m])), [machines]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        return items.filter((node) => {
            if (keyword && !`${node.name} ${node.code || ""} ${node.id}`.toLowerCase().includes(keyword)) return false;
            if (typeFilter !== "all" && node.type !== typeFilter) return false;
            if (machineFilter === "__standalone__" && node.machine_id) return false;
            if (machineFilter !== "all" && machineFilter !== "__standalone__" && String(node.machine_id) !== machineFilter) return false;
            if (groupFilter !== "all" && !(node.group_ids || []).map(String).includes(groupFilter)) return false;
            return true;
        });
    }, [items, search, typeFilter, machineFilter, groupFilter]);

    // 点击表头排序
    const toggleSort = (key) =>
        setSort((p) => (p.key === key ? { key, dir: p.dir === "asc" ? "desc" : "asc" } : { key, dir: "asc" }));

    const sortValue = (node, key) => {
        switch (key) {
            case "id": return Number(node.id) || 0;
            case "show": return node.show ? 1 : 0;
            case "name": return (node.name || "").toLowerCase();
            case "machine": return (machineById.get(node.machine_id)?.name || "").toLowerCase();
            case "host": return (node.host || "").toLowerCase();
            case "online": return Number(node.online || 0);
            case "latency": return node.latency_ms == null ? null : Number(node.latency_ms);
            case "delivered": return node.is_delivered ? 1 : 0;
            case "rate": return Number(node.rate || 0);
            case "groups": return (node.groups?.length ?? node.group_ids?.length ?? 0);
            case "type": return (node.type || "").toLowerCase();
            case "traffic": return Number(node.u || 0) + Number(node.d || 0);
            default: return null;
        }
    };

    const sortedItems = useMemo(() => {
        if (!sort.key) return filtered;
        const isNullish = (v) => v === null || v === undefined || v === "" || (typeof v === "number" && !Number.isFinite(v));
        const arr = [...filtered];
        arr.sort((a, b) => {
            const va = sortValue(a, sort.key);
            const vb = sortValue(b, sort.key);
            const an = isNullish(va), bn = isNullish(vb);
            if (an && bn) return 0;
            if (an) return 1;   // 空值永远排最后
            if (bn) return -1;
            const cmp = va < vb ? -1 : va > vb ? 1 : 0;
            return sort.dir === "asc" ? cmp : -cmp;
        });
        return arr;
    }, [filtered, sort, machineById]);

    const displayItems = sortMode ? orderedItems : sortedItems;

    const SortHead = ({ k, children, className }) => (
        <TableHead className={className}>
            <button type="button" onClick={() => toggleSort(k)} className="inline-flex items-center gap-1 select-none hover:text-foreground">
                {children}
                <span className="text-[10px] text-muted-foreground">{sort.key === k ? (sort.dir === "asc" ? "▲" : "▼") : "↕"}</span>
            </button>
        </TableHead>
    );

    const selectedNodes = filtered.filter((n) => selectedIds.has(n.id));
    const hasSelection = selectedNodes.length > 0;

    const toggleRow = (id, checked) => {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (checked) next.add(id);
            else next.delete(id);
            return next;
        });
    };

    const toggleAll = (checked) => {
        setSelectedIds(checked ? new Set(filtered.map((n) => n.id)) : new Set());
    };

    const enterSortMode = () => {
        setOrderedItems(items);
        setSortMode(true);
    };

    const saveOrder = async () => {
        setSorting(true);
        try {
            await sortNodes(orderedItems.map((n, index) => ({ id: n.id, order: index + 1 })));
            setSortMode(false);
            await load();
        } finally {
            setSorting(false);
        }
    };

    const handleShowToggle = async (node, checked) => {
        setItems((prev) => prev.map((n) => (n.id === node.id ? { ...n, show: checked } : n)));
        try {
            await updateNode({ id: node.id, type: node.type, show: checked ? 1 : 0 });
            // 「是否下发」是后端按 show + 可达 + 未超流量 + 按国去重实时算出的，
            // 改完 show 后重新拉取，让下发状态如实刷新（可达的会变「下发中」，不可达的仍是「不下发」）。
            await load();
        } catch {
            setItems((prev) => prev.map((n) => (n.id === node.id ? { ...n, show: !checked } : n)));
        }
    };

    const runBatch = async (action) => {
        const ids = selectedNodes.map((n) => n.id);
        if (!ids.length) return;
        await action(ids);
        setSelectedIds(new Set());
        await load();
    };

    // 测速进行中静默刷新节点列表（不显示加载态），测完一个就能立刻看到它的延迟、出口 IP 和是否下发。
    const probeTickRef = useRef(0);
    const refreshNodesSilently = () =>
        getNodes()
            .then((res) => setItems(res?.data || []))
            .catch(() => {});

    // 轮询测速进度；进行中每两次轮询（约 4 秒）刷新一次列表，结束后停止并完整刷新（含覆盖率统计）
    const pollProbe = () =>
        getNodeProbeStatus()
            .then((res) => {
                const s = res?.data || null;
                setProbeStatus(s);
                if (!s || !s.running) {
                    stopProbePolling();
                    load();
                    return;
                }
                probeTickRef.current += 1;
                if (probeTickRef.current % 2 === 0) refreshNodesSilently();
            })
            .catch(() => {});

    const startProbePolling = () => {
        if (probePollRef.current) return;
        probeTickRef.current = 0;
        pollProbe();
        probePollRef.current = setInterval(pollProbe, 2000);
    };

    const stopProbePolling = () => {
        if (probePollRef.current) {
            clearInterval(probePollRef.current);
            probePollRef.current = null;
        }
    };

    useEffect(() => () => stopProbePolling(), []);

    // 空闲时每 30s 轻量刷新一次测速方案 / 下次计划时间，并捕获后台计划任务已开始测速的情况
    useEffect(() => {
        if (probing) return;
        const id = setInterval(() => {
            getNodeProbeStatus()
                .then((res) => {
                    const s = res?.data || null;
                    setProbeStatus(s);
                    if (s && s.running) startProbePolling();
                })
                .catch(() => {});
        }, 30000);
        return () => clearInterval(id);
    }, [probing]);

    // 手动测速：测选中
    const probeSelected = async () => {
        const ids = selectedNodes.map((n) => n.id);
        if (!ids.length || probing) return;
        await probeNodeLatency(ids);
        setSelectedIds(new Set());
        startProbePolling();
    };

    // 手动测速：测全部
    const probeAll = async () => {
        if (probing) return;
        await probeNodeLatency();
        startProbePolling();
    };

    const handleCopy = async (node) => {
        try {
            await copyNode({ id: node.id });
            await load();
        } catch {
            // Ignore failures; list stays unchanged.
        }
    };

    const handleSingleDelete = async () => {
        if (!singleDeleteTarget) return;
        await dropNode({ id: singleDeleteTarget.id });
        await load();
    };

    const handleSingleReset = async () => {
        if (!singleResetTarget) return;
        await resetNodeTraffic({ id: singleResetTarget.id });
        await load();
    };

    return (
        <div className="p-4 md:p-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="flex flex-col gap-1">
                    <h3 className="text-lg font-medium">{t("manage.title")}</h3>
                    <p className="text-sm text-muted-foreground">{t("manage.description")}</p>
                </div>
                {syncStats && (
                    <div
                        className="rounded-md border px-3 py-2 text-right"
                        title={t("manage.sync.hint", { total: syncStats.total ?? 0 })}
                    >
                        <div className="text-xs text-muted-foreground">{t("manage.sync.label")}</div>
                        <div className="text-sm font-medium">
                            {t("manage.sync.value", {
                                synced: syncStats.synced ?? 0,
                                reported: syncStats.reported ?? 0,
                            })}
                        </div>
                        {syncStats.last_node_change_at ? (
                            <div className="text-xs text-muted-foreground">
                                {t("manage.sync.last_change", {
                                    time: new Date(syncStats.last_node_change_at * 1000).toLocaleString(),
                                })}
                            </div>
                        ) : null}
                    </div>
                )}
            </div>
            <NodeConfigWarnings stats={syncStats} t={t} />
            <hr className="my-6" />

            <div className="flex flex-col gap-3">
                <div className="flex flex-wrap items-center gap-2">
                    <Input
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        placeholder={t("toolbar.search")}
                        className="h-9 w-[200px]"
                        disabled={sortMode}
                    />
                    <Select value={typeFilter} onValueChange={setTypeFilter} className="w-[150px]">
                        <SelectTrigger><SelectValue placeholder={t("toolbar.type")} /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">{t("toolbar.type")}</SelectItem>
                            {PROTOCOL_TYPES.map((type) => <SelectItem key={type} value={type}>{type}</SelectItem>)}
                        </SelectContent>
                    </Select>
                    <Select value={machineFilter} onValueChange={setMachineFilter} className="w-[200px]">
                        <SelectTrigger><SelectValue placeholder={t("columns.deployment.title")} /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">{t("columns.deployment.title")}</SelectItem>
                            <SelectItem value="__standalone__">{t("columns.deployment.standalone")}</SelectItem>
                            {machines.map((m) => <SelectItem key={m.id} value={String(m.id)}>{`${m.name} · SID:${m.id}`}</SelectItem>)}
                        </SelectContent>
                    </Select>
                    <Select value={groupFilter} onValueChange={setGroupFilter} className="w-[150px]">
                        <SelectTrigger><SelectValue placeholder={t("columns.groups.title")} /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">{t("columns.groups.title")}</SelectItem>
                            {groups.map((g) => <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>)}
                        </SelectContent>
                    </Select>
                    {(search || typeFilter !== "all" || machineFilter !== "all" || groupFilter !== "all") && (
                        <Button variant="ghost" size="sm" onClick={() => { setSearch(""); setTypeFilter("all"); setMachineFilter("all"); setGroupFilter("all"); }}>
                            {t("toolbar.reset")}
                        </Button>
                    )}
                    <div className="ml-auto flex items-center gap-2">
                        <Button variant="outline" size="sm" onClick={probeAll} loading={probing} disabled={probing}>{t("toolbar.probe.all")}</Button>
                        <Button variant="outline" size="sm" onClick={() => { setEditingNode(null); setFormOpen(true); }}>{t("form.add_node")}</Button>
                        <Button variant="outline" size="sm" onClick={() => setImportOpen(true)}>{t("import.button")}</Button>
                        <Button variant={sortMode ? "default" : "outline"} size="sm" onClick={() => (sortMode ? saveOrder() : enterSortMode())} loading={sorting}>
                            {sortMode ? t("toolbar.sort.save") : t("toolbar.sort.edit")}
                        </Button>
                    </div>
                </div>

                {probeStatus && probeStatus.running ? (
                    <div className="rounded-md border bg-muted/20 px-3 py-2">
                        <div className="mb-1 flex items-center justify-between text-xs text-muted-foreground">
                            <span className="truncate">
                                {probeStatus.queued && !probeStatus.done
                                    ? t("toolbar.probe.queued")
                                    : t("toolbar.probe.progress", { done: probeStatus.done || 0, total: probeStatus.total || 0, ok: probeStatus.ok || 0 })}
                                {probeStatus.current ? ` · ${probeStatus.current}` : ""}
                            </span>
                            <span className="ml-2 flex shrink-0 items-center gap-2">
                                <span className="font-medium">
                                    {probeStatus.total ? Math.round(((probeStatus.done || 0) / probeStatus.total) * 100) : 0}%
                                </span>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    className="h-6 px-2 text-xs"
                                    onClick={stopProbe}
                                    loading={stoppingProbe}
                                    disabled={stoppingProbe || probeStatus.stopping}
                                >
                                    {probeStatus.stopping ? t("toolbar.probe.stopping") : t("toolbar.probe.stop")}
                                </Button>
                            </span>
                        </div>
                        <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                            <div
                                className="h-full rounded-full bg-emerald-500 transition-all duration-500"
                                style={{ width: `${probeStatus.total ? Math.min(100, ((probeStatus.done || 0) / probeStatus.total) * 100) : (probeStatus.queued ? 8 : 0)}%` }}
                            />
                        </div>
                    </div>
                ) : probeStatus?.scheme ? (
                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md border bg-muted/20 px-3 py-2 text-xs">
                        {probeStatus.stopped && (
                            <span className="text-amber-600">
                                {t("toolbar.probe.stopped", { done: probeStatus.done || 0, total: probeStatus.total || 0 })}
                            </span>
                        )}
                        {probeStatus.scheme.enabled ? (
                            <span className="flex items-center gap-1.5">
                                <span className="text-muted-foreground">{t("toolbar.probe.nextIn")}</span>
                                <span className="font-mono font-medium tabular-nums text-foreground">
                                    {probeStatus.scheme.next_run_at
                                        ? formatCountdown(probeStatus.scheme.next_run_at - nowTs)
                                        : "--"}
                                </span>
                            </span>
                        ) : (
                            <span className="font-medium text-amber-600 dark:text-amber-400">{t("toolbar.probe.disabled")}</span>
                        )}
                        <span className="flex items-center gap-1.5 text-muted-foreground">
                            <span>{t("toolbar.probe.scheme")}</span>
                            <Badge variant="secondary" className="px-1.5 py-0 text-[10px]">
                                {t(`toolbar.probe.mode.${probeStatus.scheme.mode === "tcp" ? "tcp" : "singbox"}`)}
                            </Badge>
                            <span>·</span>
                            <span>{t("toolbar.probe.interval", { min: probeStatus.scheme.interval_minutes })}</span>
                            <span>·</span>
                            <span>
                                {t("toolbar.probe.autoVisibility")}
                                {probeStatus.scheme.auto_visibility ? t("toolbar.probe.on") : t("toolbar.probe.off")}
                            </span>
                        </span>
                    </div>
                ) : null}

                {sortMode ? (
                    <p className="text-xs text-muted-foreground">{t("toolbar.sort.tip")}</p>
                ) : (
                    <div className="flex flex-wrap items-center gap-2 rounded-md border bg-muted/20 px-3 py-2">
                        <span className="text-xs font-medium text-muted-foreground">{t("toolbar.actions")}{hasSelection ? ` (${selectedNodes.length})` : ""}</span>
                        <Button variant="outline" size="sm" disabled={!hasSelection || probing} loading={probing} onClick={probeSelected}>{t("toolbar.probe.selected")}</Button>
                        <Button variant="outline" size="sm" disabled={!hasSelection} onClick={() => runBatch((ids) => batchUpdateNodes({ ids, show: 1 }))}>{t("toolbar.batch_show.menu")}</Button>
                        <Button variant="outline" size="sm" disabled={!hasSelection} onClick={() => runBatch((ids) => batchUpdateNodes({ ids, show: 0 }))}>{t("toolbar.batch_hide.menu")}</Button>
                        <Button variant="outline" size="sm" disabled={!hasSelection} onClick={() => runBatch((ids) => batchUpdateNodes({ ids, enabled: true }))}>{t("toolbar.batch_enable.menu")}</Button>
                        <Button variant="outline" size="sm" disabled={!hasSelection} onClick={() => runBatch((ids) => batchUpdateNodes({ ids, enabled: false }))}>{t("toolbar.batch_disable.menu")}</Button>
                        <Button variant="outline" size="sm" disabled={!hasSelection} onClick={() => setBatchResetOpen(true)}>{t("toolbar.batch_reset_traffic.menu")}</Button>
                        <Button variant="destructive" size="sm" disabled={!hasSelection} onClick={() => setBatchDeleteOpen(true)}>{t("toolbar.batch_delete.menu")}</Button>
                    </div>
                )}
            </div>

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            {sortMode && <TableHead className="w-8"></TableHead>}
                            {!sortMode && (
                                <TableHead className="w-8">
                                    <input
                                        type="checkbox"
                                        checked={filtered.length > 0 && filtered.every((n) => selectedIds.has(n.id))}
                                        onChange={(e) => toggleAll(e.target.checked)}
                                    />
                                </TableHead>
                            )}
                            <SortHead k="id">{t("columns.nodeId")}</SortHead>
                            <SortHead k="show">{t("columns.show")}</SortHead>
                            <SortHead k="name">{t("columns.node")}</SortHead>
                            <SortHead k="machine">{t("columns.deployment.title")}</SortHead>
                            <SortHead k="host">{t("columns.address")}</SortHead>
                            <SortHead k="online">{t("columns.onlineUsers.title")}</SortHead>
                            <SortHead k="latency">{t("columns.latency.title")}</SortHead>
                            <SortHead k="delivered">{t("columns.delivered.title")}</SortHead>
                            <SortHead k="rate">{t("columns.rate.title")}</SortHead>
                            <SortHead k="groups">{t("columns.groups.title")}</SortHead>
                            <SortHead k="type">{t("columns.type")}</SortHead>
                            <SortHead k="traffic">{t("columns.traffic.title")}</SortHead>
                            <TableHead className="w-44 text-right">{t("columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={14}>...</TableEmpty>
                        ) : displayItems.length === 0 ? (
                            <TableEmpty colSpan={14}>-</TableEmpty>
                        ) : (
                            displayItems.map((node, index) => {
                                const machine = node.machine_id ? machineById.get(node.machine_id) : null;
                                const depStatus = machineStatus(machine);
                                const used = (node.u || 0) + (node.d || 0);
                                const limit = node.transfer_enable ?? 0;
                                const pct = limit > 0 ? Math.min((used / limit) * 100, 100) : 0;
                                return (
                                    <TableRow
                                        key={node.id}
                                        draggable={sortMode}
                                        onDragStart={() => { dragIndexRef.current = index; }}
                                        onDragOver={(e) => sortMode && e.preventDefault()}
                                        onDrop={() => {
                                            if (!sortMode) return;
                                            const from = dragIndexRef.current;
                                            if (from === null || from === index) return;
                                            setOrderedItems((prev) => {
                                                const next = [...prev];
                                                const [moved] = next.splice(from, 1);
                                                next.splice(index, 0, moved);
                                                return next;
                                            });
                                            dragIndexRef.current = null;
                                        }}
                                    >
                                        {sortMode && <TableCell className="cursor-move text-muted-foreground">⠿</TableCell>}
                                        {!sortMode && (
                                            <TableCell>
                                                <input type="checkbox" checked={selectedIds.has(node.id)} onChange={(e) => toggleRow(node.id, e.target.checked)} />
                                            </TableCell>
                                        )}
                                        <TableCell>
                                            <div className="flex items-center gap-1.5">
                                                <Badge variant="outline" className="border-2 font-mono text-[10px]" style={{ borderColor: PROTOCOL_COLOR[node.type] }}>
                                                    {node.code ?? node.id}
                                                    {node.parent ? <span className="text-muted-foreground"> → {node.parent.code || node.parent.id}</span> : null}
                                                </Badge>
                                                <button type="button" className="text-[10px] text-muted-foreground hover:text-foreground" onClick={() => copyText(node.code ?? node.id)}>
                                                    复制ID
                                                </button>
                                            </div>
                                        </TableCell>
                                        <TableCell>
                                            <Switch checked={Boolean(node.show)} onCheckedChange={(v) => handleShowToggle(node, v)} />
                                        </TableCell>
                                        <TableCell>
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <div className="flex items-center gap-2">
                                                        <span className={`size-2.5 shrink-0 rounded-full ${STATUS_TONE[node.available_status] || "bg-slate-400"}`} />
                                                        <span className="font-medium">{node.name}</span>
                                                    </div>
                                                </TooltipTrigger>
                                                <TooltipContent>
                                                    <div className="space-y-1 text-xs">
                                                        <p className="font-medium">{t(`columns.status.${node.available_status ?? 1}`)}</p>
                                                        {node.load_status ? (
                                                            <>
                                                                <p>CPU {Math.round(node.load_status.cpu)}%</p>
                                                                <p>{t("columns.loadStatus.memory")}: {formatBytes(node.load_status.mem?.used)} / {formatBytes(node.load_status.mem?.total)}</p>
                                                                <p>{t("columns.loadStatus.disk")}: {formatBytes(node.load_status.disk?.used)} / {formatBytes(node.load_status.disk?.total)}</p>
                                                            </>
                                                        ) : (
                                                            <p>{t("columns.loadStatus.noData")}</p>
                                                        )}
                                                    </div>
                                                </TooltipContent>
                                            </Tooltip>
                                        </TableCell>
                                        <TableCell>
                                            {machine ? (
                                                <div className="flex items-center gap-1.5 text-xs">
                                                    <span className={`size-2 shrink-0 rounded-full ${depStatus === "online" ? "bg-emerald-500" : depStatus === "inactive" ? "bg-slate-400" : "bg-rose-500"}`} />
                                                    <span className="font-medium">{machine.name}</span>
                                                    {!node.enabled && <Badge variant="secondary" className="px-1.5 py-0 text-[10px]">{t("columns.deployment.disabled")}</Badge>}
                                                </div>
                                            ) : (
                                                <span className="text-xs text-muted-foreground">{t("columns.deployment.standalone")}</span>
                                            )}
                                        </TableCell>
                                        <TableCell className="font-mono text-xs">
                                            {node.host}:{node.port}
                                            {node.port !== node.server_port && (
                                                <div className="text-[10px] text-muted-foreground">({t("columns.internalPort")} {node.server_port})</div>
                                            )}
                                        </TableCell>
                                        <TableCell>
                                            <span className={`font-mono tabular-nums ${Number(node.online) > 0 ? "font-medium text-foreground" : "text-muted-foreground"}`}>
                                                {Number(node.online || 0)}
                                            </span>
                                        </TableCell>
                                        <TableCell>{renderLatency(node, t)}</TableCell>
                                        <TableCell>
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <button
                                                        type="button"
                                                        onClick={() => handleShowToggle(node, !node.show)}
                                                        className="cursor-pointer"
                                                    >
                                                        {node.is_delivered ? (
                                                            <Badge className="bg-emerald-500 text-white hover:bg-emerald-600">{t("columns.delivered.yes")}</Badge>
                                                        ) : (
                                                            <Badge variant="secondary" className="text-muted-foreground hover:bg-secondary/80">{t("columns.delivered.no")}</Badge>
                                                        )}
                                                    </button>
                                                </TooltipTrigger>
                                                <TooltipContent>
                                                    <p className="text-xs">{node.show ? t("columns.delivered.toggleOff") : t("columns.delivered.toggleOn")}</p>
                                                </TooltipContent>
                                            </Tooltip>
                                        </TableCell>
                                        <TableCell><Badge variant="secondary">{node.rate}x</Badge></TableCell>
                                        <TableCell>
                                            <div className="flex flex-wrap gap-1">
                                                {(node.groups || []).length === 0 ? (
                                                    <span className="text-xs text-muted-foreground">{t("columns.groups.empty")}</span>
                                                ) : (
                                                    node.groups.map((g) => <Badge key={g.id} variant="secondary" className="text-[10px]">{g.name}</Badge>)
                                                )}
                                            </div>
                                        </TableCell>
                                        <TableCell>
                                            <Badge variant="outline" className="border-2 font-medium" style={{ borderColor: PROTOCOL_COLOR[node.type] }}>{node.type}</Badge>
                                        </TableCell>
                                        <TableCell>
                                            {limit > 0 ? (
                                                <Tooltip>
                                                    <TooltipTrigger asChild>
                                                        <div className="flex items-center gap-2">
                                                            <div className="h-1.5 w-12 rounded-full bg-secondary">
                                                                <div className={`h-full rounded-full ${node.banned || pct > 90 ? "bg-destructive" : "bg-primary"}`} style={{ width: `${pct}%` }} />
                                                            </div>
                                                            <span className="text-xs text-muted-foreground">{formatBytes(used)}</span>
                                                        </div>
                                                    </TooltipTrigger>
                                                    <TooltipContent>
                                                        <div className="space-y-1 text-xs">
                                                            <p>{t("columns.traffic.used")}: {formatBytes(used)}</p>
                                                            <p>{t("columns.traffic.total")}: {formatBytes(limit)}</p>
                                                            <p>{t("columns.traffic.percentage")}: {pct.toFixed(1)}%</p>
                                                        </div>
                                                    </TooltipContent>
                                                </Tooltip>
                                            ) : (
                                                <span className="text-xs text-muted-foreground">{formatBytes(used)}</span>
                                            )}
                                        </TableCell>
                                        <TableCell className="text-right">
                                            <div className="flex justify-end gap-1.5">
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => { setEditingNode(node); setFormOpen(true); }}>{t("columns.actions_dropdown.edit")}</Button>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => handleCopy(node)}>{t("columns.actions_dropdown.copy")}</Button>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => setSingleResetTarget(node)}>{t("columns.actions_dropdown.reset_traffic.confirm")}</Button>
                                                <Button variant="destructive" size="sm" className="h-7 px-2 text-xs" onClick={() => setSingleDeleteTarget(node)}>{t("columns.actions_dropdown.delete.confirm")}</Button>
                                            </div>
                                        </TableCell>
                                    </TableRow>
                                );
                            })
                        )}
                    </TableBody>
                </Table>
            </div>

            <ConfirmDialog
                open={Boolean(singleDeleteTarget)}
                onOpenChange={(open) => !open && setSingleDeleteTarget(null)}
                title={t("columns.actions_dropdown.delete.title")}
                description={t("columns.actions_dropdown.delete.description")}
                confirmLabel={t("columns.actions_dropdown.delete.confirm")}
                onConfirm={handleSingleDelete}
            />
            <ConfirmDialog
                open={Boolean(singleResetTarget)}
                onOpenChange={(open) => !open && setSingleResetTarget(null)}
                title={t("columns.actions_dropdown.reset_traffic.title")}
                description={t("columns.actions_dropdown.reset_traffic.description")}
                confirmLabel={t("columns.actions_dropdown.reset_traffic.confirm")}
                variant="default"
                onConfirm={handleSingleReset}
            />
            <ConfirmDialog
                open={batchDeleteOpen}
                onOpenChange={setBatchDeleteOpen}
                title={t("toolbar.batch_delete.title")}
                description={t("toolbar.batch_delete.description", { count: selectedNodes.length })}
                confirmLabel={t("toolbar.batch_delete.confirm")}
                onConfirm={() => runBatch((ids) => batchDeleteNodes({ ids }))}
            />
            <ConfirmDialog
                open={batchResetOpen}
                onOpenChange={setBatchResetOpen}
                title={t("toolbar.batch_reset_traffic.title")}
                description={t("toolbar.batch_reset_traffic.description", { count: selectedNodes.length })}
                confirmLabel={t("toolbar.batch_reset_traffic.confirm")}
                variant="default"
                onConfirm={() => runBatch((ids) => batchResetNodeTraffic({ ids }))}
            />

            <NodeFormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                node={editingNode}
                groups={groups}
                routes={routes}
                machines={machines}
                nodes={items}
                onSaved={load}
                t={t}
            />
            <NodeImportDialog
                open={importOpen}
                onOpenChange={setImportOpen}
                groups={groups}
                onImported={load}
                t={t}
            />
        </div>
    );
}

// 配置问题提示：套餐没设权限组（买了也分不到节点）、有套餐的用户没有权限组、客户端推送未开启。
function NodeConfigWarnings({ stats, t }) {
    if (!stats) return null;
    const plans = stats.plans_without_group || [];
    const usersNoGroup = stats.users_without_group || 0;
    const showPlans = plans.length > 0;
    const showUsers = usersNoGroup > 0;
    const showPush = stats.client_push_enabled === false;
    if (!showPlans && !showUsers && !showPush) return null;
    return (
        <div className="mt-4 space-y-2">
            {(showPlans || showUsers) && (
                <div className="rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-400">
                    <div className="font-medium">{t("manage.warnings.title")}</div>
                    {showPlans && (
                        <div className="mt-1">
                            {t("manage.warnings.plans", { names: plans.map((p) => `${p.name}（${p.users}）`).join("、") })}
                        </div>
                    )}
                    {showUsers && <div className="mt-1">{t("manage.warnings.users", { count: usersNoGroup })}</div>}
                    <div className="mt-1 text-red-600/90 dark:text-red-400/90">{t("manage.warnings.fix")}</div>
                </div>
            )}
            {showPush && (
                <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-400">
                    {t("manage.warnings.push_off")}
                </div>
            )}
        </div>
    );
}
