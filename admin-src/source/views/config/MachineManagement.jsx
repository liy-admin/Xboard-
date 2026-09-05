import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { fetchMachines, saveMachine, dropMachine, getMachineNodes } from "../../api/server.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";

const EMPTY_FORM = { id: null, name: "", notes: "", is_active: true };

const STATUS_BADGE = {
    online: "success",
    offline: "destructive",
    inactive: "secondary",
    never: "outline",
};

const LOAD_THRESHOLDS = {
    cpu: { warn: 70, danger: 85 },
    mem: { warn: 75, danger: 90 },
    disk: { warn: 80, danger: 90 },
};

function machineStatus(machine) {
    if (!machine.is_active) return "inactive";
    if (!machine.last_seen_at) return "never";
    return Date.now() / 1000 - machine.last_seen_at < 300 ? "online" : "offline";
}

function loadPercents(machine) {
    const status = machine.load_status;
    if (!status) return { cpu: 0, mem: 0, disk: 0 };
    return {
        cpu: status.cpu ?? 0,
        mem: status.mem?.total > 0 ? (status.mem.used / status.mem.total) * 100 : 0,
        disk: status.disk?.total > 0 ? (status.disk.used / status.disk.total) * 100 : 0,
    };
}

function isHighLoad(machine) {
    if (!machine.load_status) return false;
    const { cpu, mem, disk } = loadPercents(machine);
    return cpu >= LOAD_THRESHOLDS.cpu.danger || mem >= LOAD_THRESHOLDS.mem.danger || disk >= LOAD_THRESHOLDS.disk.danger;
}

function loadTone(value, key) {
    const t = LOAD_THRESHOLDS[key];
    if (value >= t.danger) return "bg-red-500";
    if (value >= t.warn) return "bg-amber-500";
    return "bg-primary";
}

function relativeTime(seconds) {
    if (!seconds) return "-";
    const diff = Math.floor(Date.now() / 1000 - seconds);
    if (diff < 60) return `${diff}s`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
    return `${Math.floor(diff / 86400)}d`;
}

function formatBytes(value) {
    if (value == null) return "-";
    if (value < 1024) return `${value.toFixed(0)} B`;
    if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
    if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
    if (value < 1024 * 1024 * 1024 * 1024) return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    return `${(value / (1024 * 1024 * 1024 * 1024)).toFixed(2)} TB`;
}

function LoadBar({ label, value, tone }) {
    return (
        <div className="space-y-0.5">
            <div className="flex items-center justify-between gap-2 text-[10px] uppercase tracking-wide text-muted-foreground">
                <span>{label}</span>
                <span className="font-mono">{value.toFixed(0)}%</span>
            </div>
            <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                <div className={`h-full rounded-full ${tone}`} style={{ width: `${Math.min(100, value)}%` }} />
            </div>
        </div>
    );
}

export function MachineManagement() {
    const { t } = useTranslation("machine");
    const auth = useAuthState();
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");
    const [statusFilter, setStatusFilter] = useState("all");
    const [nodeFilter, setNodeFilter] = useState("all");

    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState(EMPTY_FORM);
    const [formError, setFormError] = useState("");
    const [formType, setFormType] = useState("add");

    const [tokenInfo, setTokenInfo] = useState(null);
    const [deleteTarget, setDeleteTarget] = useState(null);

    const [detailTarget, setDetailTarget] = useState(null);
    const [detailNodes, setDetailNodes] = useState([]);
    const [detailLoading, setDetailLoading] = useState(false);

    const load = () => {
        setLoading(true);
        return fetchMachines()
            .then((res) => setItems(res?.data || []))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
    }, [auth.token]);

    const overview = useMemo(() => {
        const total = items.length;
        const online = items.filter((m) => machineStatus(m) === "online").length;
        const offline = items.filter((m) => ["offline", "never"].includes(machineStatus(m))).length;
        const highLoad = items.filter(isHighLoad).length;
        const hostedNodes = items.reduce((sum, m) => sum + (m.servers_count ?? 0), 0);
        return { total, online, offline, highLoad, hostedNodes };
    }, [items]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        return items.filter((m) => {
            if (keyword && !`${m.name} ${m.notes || ""} sid:${m.id}`.toLowerCase().includes(keyword)) return false;
            if (statusFilter !== "all" && machineStatus(m) !== statusFilter) return false;
            if (nodeFilter === "with_nodes" && !(m.servers_count > 0)) return false;
            if (nodeFilter === "idle_nodes" && m.servers_count > 0) return false;
            if (nodeFilter === "high_load" && !isHighLoad(m)) return false;
            return true;
        });
    }, [items, search, statusFilter, nodeFilter]);

    const openAdd = () => {
        setForm(EMPTY_FORM);
        setFormType("add");
        setFormError("");
        setFormOpen(true);
    };

    const openEdit = (item) => {
        setForm({ id: item.id, name: item.name || "", notes: item.notes || "", is_active: item.is_active ?? true });
        setFormType("edit");
        setFormError("");
        setFormOpen(true);
    };

    const handleSubmit = async () => {
        setFormError("");
        if (!form.name.trim()) {
            const error = t("form.nameError");
            setFormError(error);
            throw new Error(error);
        }
        const res = await saveMachine({
            id: form.id ?? undefined,
            name: form.name.trim(),
            notes: form.notes,
            is_active: form.is_active,
        });
        if (formType === "add" && res?.data?.token) {
            setTokenInfo({ token: res.data.token, installCommand: res.data.install_command ?? null });
        }
        await load();
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await dropMachine({ id: deleteTarget.id });
        await load();
    };

    const openDetail = (item) => {
        setDetailTarget(item);
        setDetailNodes([]);
        setDetailLoading(true);
        getMachineNodes({ id: item.id })
            .then((res) => setDetailNodes(res?.data || []))
            .finally(() => setDetailLoading(false));
    };

    const copyText = async (text) => {
        try {
            await navigator.clipboard.writeText(text);
        } catch {
            // Ignore clipboard failures; text remains selectable for manual copy.
        }
    };

    return (
        <div className="p-4 md:p-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-lg font-medium">{t("title")}</h3>
                <p className="text-sm text-muted-foreground">{t("description")}</p>
            </div>
            <hr className="my-6" />

            <div className="mb-4 flex flex-wrap items-center gap-2 rounded-xl border bg-card px-3 py-2.5 shadow-sm">
                {[
                    { key: "total", label: t("overview.total"), value: overview.total },
                    { key: "online", label: t("overview.online"), value: overview.online },
                    { key: "offline", label: t("overview.offline"), value: overview.offline },
                    { key: "highLoad", label: t("overview.high_load"), value: overview.highLoad },
                    { key: "hostedNodes", label: t("columns.nodes"), value: overview.hostedNodes },
                ].map((stat) => (
                    <div key={stat.key} className="min-w-[130px] flex-1 rounded-lg border bg-muted/20 px-3 py-2">
                        <p className="text-[11px] uppercase tracking-wide text-muted-foreground">{stat.label}</p>
                        <div className="mt-1 text-xl font-semibold leading-none">{stat.value}</div>
                    </div>
                ))}
            </div>

            <SearchFilterToolbar
                search={search}
                onSearchChange={setSearch}
                searchPlaceholder={t("toolbar.search")}
                resetLabel={t("toolbar.reset")}
                onReset={() => {
                    setSearch("");
                    setStatusFilter("all");
                    setNodeFilter("all");
                }}
                filters={
                    <>
                        <Select value={statusFilter} onValueChange={setStatusFilter} className="w-[140px]">
                            <SelectTrigger><SelectValue placeholder={t("toolbar.status")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("toolbar.status")}</SelectItem>
                                <SelectItem value="online">{t("columns.online")}</SelectItem>
                                <SelectItem value="offline">{t("columns.offline")}</SelectItem>
                                <SelectItem value="inactive">{t("columns.inactive")}</SelectItem>
                                <SelectItem value="never">{t("columns.never")}</SelectItem>
                            </SelectContent>
                        </Select>
                        <Select value={nodeFilter} onValueChange={setNodeFilter} className="w-[150px]">
                            <SelectTrigger><SelectValue placeholder={t("toolbar.nodes")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("toolbar.nodes")}</SelectItem>
                                <SelectItem value="with_nodes">{t("toolbar.with_nodes")}</SelectItem>
                                <SelectItem value="idle_nodes">{t("toolbar.idle_nodes")}</SelectItem>
                                <SelectItem value="high_load">{t("toolbar.high_load")}</SelectItem>
                            </SelectContent>
                        </Select>
                    </>
                }
                actions={<Button onClick={openAdd}>{t("form.add")}</Button>}
            />
            <p className="mt-2 text-xs text-muted-foreground">{t("toolbar.tip")}</p>

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t("columns.name")}</TableHead>
                            <TableHead>{t("columns.status")}</TableHead>
                            <TableHead className="w-[200px]">{t("columns.load")}</TableHead>
                            <TableHead>{t("columns.nodes")}</TableHead>
                            <TableHead>{t("columns.lastSeen")}</TableHead>
                            <TableHead className="w-40 text-right">{t("columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={6}>...</TableEmpty>
                        ) : filtered.length === 0 ? (
                            <TableEmpty colSpan={6}>-</TableEmpty>
                        ) : (
                            filtered.map((item) => {
                                const status = machineStatus(item);
                                const percents = loadPercents(item);
                                return (
                                    <TableRow key={item.id}>
                                        <TableCell className="max-w-[300px]">
                                            <div className="flex flex-wrap items-center gap-2">
                                                <span className="truncate font-medium">{item.name}</span>
                                                <Badge variant="outline" className="font-mono text-[10px]">SID:{item.id}</Badge>
                                                {isHighLoad(item) && (
                                                    <Badge variant="secondary" className="border-0 bg-amber-500/10 text-amber-600 dark:text-amber-400">
                                                        {t("overview.high_load")}
                                                    </Badge>
                                                )}
                                            </div>
                                            {item.notes && <p className="mt-1 max-w-[300px] truncate text-xs text-muted-foreground">{item.notes}</p>}
                                        </TableCell>
                                        <TableCell><Badge variant={STATUS_BADGE[status]}>{t(`columns.${status}`)}</Badge></TableCell>
                                        <TableCell>
                                            {item.load_status ? (
                                                <div className="w-[180px] space-y-1.5">
                                                    <LoadBar label="CPU" value={percents.cpu} tone={loadTone(percents.cpu, "cpu")} />
                                                    <LoadBar label="MEM" value={percents.mem} tone={loadTone(percents.mem, "mem")} />
                                                    <div className="flex items-center justify-between text-[10px] text-muted-foreground">
                                                        <span>DISK</span>
                                                        <span className="font-mono">{formatBytes(item.load_status.disk?.used)} / {formatBytes(item.load_status.disk?.total)}</span>
                                                    </div>
                                                </div>
                                            ) : (
                                                <span className="text-xs text-muted-foreground">{t("columns.noData")}</span>
                                            )}
                                        </TableCell>
                                        <TableCell>
                                            <div className="space-y-1">
                                                <div className="font-mono text-sm font-semibold">{item.servers_count ?? 0}</div>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => openDetail(item)}>
                                                    {t("detail.title")}
                                                </Button>
                                            </div>
                                        </TableCell>
                                        <TableCell className="text-xs">
                                            <div>{relativeTime(item.last_seen_at)}</div>
                                            <div className="text-muted-foreground">
                                                {item.load_status?.updated_at ? `${t("columns.lastReport")}: ${relativeTime(item.load_status.updated_at)}` : t("columns.noData")}
                                            </div>
                                        </TableCell>
                                        <TableCell className="text-right">
                                            <div className="flex justify-end gap-2">
                                                <Button variant="outline" size="sm" onClick={() => openEdit(item)}>
                                                    {t("form.edit")}
                                                </Button>
                                                <Button variant="destructive" size="sm" onClick={() => setDeleteTarget(item)}>
                                                    {t("messages.deleteButton")}
                                                </Button>
                                            </div>
                                        </TableCell>
                                    </TableRow>
                                );
                            })
                        )}
                    </TableBody>
                </Table>
            </div>

            <FormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                title={t(formType === "edit" ? "form.edit" : "form.create")}
                description={t(formType === "edit" ? "form.editDescription" : "form.createDescription")}
                cancelLabel={t("form.cancel")}
                submitLabel={t(formType === "edit" ? "form.update" : "form.submit")}
                onSubmit={handleSubmit}
                className="sm:max-w-md"
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.name")}</div>
                    <Input value={form.name} placeholder={t("form.namePlaceholder")} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} required />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.notes")}</div>
                    <Textarea className="min-h-[80px]" placeholder={t("form.notesPlaceholder")} value={form.notes} onChange={(e) => setForm((p) => ({ ...p, notes: e.target.value }))} />
                </div>
                <div className="flex items-center justify-between rounded-lg border p-3">
                    <div className="space-y-0.5">
                        <div className="text-sm font-medium">{t("form.isActive")}</div>
                        <div className="text-xs text-muted-foreground">{t("form.isActiveDescription")}</div>
                    </div>
                    <Switch checked={form.is_active} onCheckedChange={(v) => setForm((p) => ({ ...p, is_active: v }))} />
                </div>
            </FormDialog>

            <Dialog open={Boolean(tokenInfo)} onOpenChange={(open) => !open && setTokenInfo(null)} className="sm:max-w-lg">
                <div className="p-6">
                    <h3 className="text-lg font-semibold">{t("token.title")}</h3>
                    <p className="mt-1 text-xs text-muted-foreground">{t("token.createdHint")}</p>
                    <div className="mt-4 flex items-center gap-2 rounded-lg border bg-muted/50 p-3">
                        <code className="flex-1 break-all font-mono text-xs">{tokenInfo?.token}</code>
                        <Button variant="ghost" size="sm" onClick={() => copyText(tokenInfo?.token)}>{t("token.copy")}</Button>
                    </div>
                    {tokenInfo?.installCommand && (
                        <div className="mt-4 space-y-2">
                            <span className="text-xs font-medium text-muted-foreground">{t("install.title")}</span>
                            <div className="flex items-start gap-2 rounded-lg border bg-zinc-950 p-3">
                                <code className="flex-1 break-all font-mono text-xs text-green-400">{tokenInfo.installCommand}</code>
                                <Button variant="ghost" size="sm" className="text-zinc-400 hover:text-white" onClick={() => copyText(tokenInfo.installCommand)}>{t("install.copy")}</Button>
                            </div>
                            <p className="text-[10px] text-muted-foreground">{t("install.hint")}</p>
                        </div>
                    )}
                    <div className="mt-6 flex justify-end">
                        <Button onClick={() => setTokenInfo(null)}>{t("form.submit")}</Button>
                    </div>
                </div>
            </Dialog>

            <Dialog open={Boolean(detailTarget)} onOpenChange={(open) => !open && setDetailTarget(null)} className="sm:max-w-2xl">
                {detailTarget && (
                    <div className="max-h-[90vh] overflow-y-auto p-6">
                        <div className="flex items-center gap-2">
                            <h3 className="text-lg font-semibold">{detailTarget.name}</h3>
                            <Badge variant="outline" className="font-mono">SID:{detailTarget.id}</Badge>
                            <Badge variant={STATUS_BADGE[machineStatus(detailTarget)]}>{t(`columns.${machineStatus(detailTarget)}`)}</Badge>
                        </div>
                        <div className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
                            <span>{t("columns.lastSeen")}: {relativeTime(detailTarget.last_seen_at)}</span>
                            <span>•</span>
                            <span>{t("columns.nodes")}: {detailTarget.servers_count ?? 0}</span>
                        </div>
                        {detailTarget.notes && <p className="mt-2 text-sm text-muted-foreground">{detailTarget.notes}</p>}

                        <hr className="my-4" />
                        <h4 className="text-sm font-medium">{t("detail.associatedNodes")}</h4>
                        {detailLoading ? (
                            <p className="mt-2 text-sm text-muted-foreground">...</p>
                        ) : detailNodes.length === 0 ? (
                            <p className="mt-2 text-sm text-muted-foreground">{t("detail.noNodes")}</p>
                        ) : (
                            <div className="mt-2 overflow-auto rounded-md border">
                                <table className="w-full text-sm">
                                    <thead className="bg-muted/50 text-left text-xs text-muted-foreground">
                                        <tr>
                                            <th className="p-2">{t("detail.nodeId")}</th>
                                            <th className="p-2">{t("detail.nodeName")}</th>
                                            <th className="p-2">{t("detail.nodeType")}</th>
                                            <th className="p-2">{t("detail.nodeHost")}</th>
                                            <th className="p-2">{t("detail.nodePort")}</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {detailNodes.map((node) => (
                                            <tr key={node.id} className="border-t">
                                                <td className="p-2 font-mono">{node.id}</td>
                                                <td className="p-2">{node.name}</td>
                                                <td className="p-2">{node.type}</td>
                                                <td className="p-2 font-mono">{node.host}</td>
                                                <td className="p-2 font-mono">{node.port}</td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        )}
                        <div className="mt-6 flex justify-end">
                            <Button variant="outline" onClick={() => setDetailTarget(null)}>{t("detail.cancel")}</Button>
                        </div>
                    </div>
                )}
            </Dialog>

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("messages.deleteConfirm")}
                description={t("messages.deleteDescription")}
                confirmLabel={t("messages.deleteButton")}
                onConfirm={handleDelete}
            />
        </div>
    );
}
