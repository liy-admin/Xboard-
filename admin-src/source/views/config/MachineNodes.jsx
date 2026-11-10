import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getMachineNodes, getNodes, fetchGroups, fetchRoutes, updateNode, dropNode, testMachineNode, getMachineNodeLinks } from "../../api/server.js";
import { getErrorMessage } from "../../utils/errors.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Switch } from "../../components/Switch.jsx";
import { SortLabel, useClientSort } from "../../components/SortableHead.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { NodeFormDialog } from "./NodeFormDialog.jsx";
import { MachineQuickNode } from "./MachineQuickNode.jsx";

// 节点的传输层安全，供列表一眼看出 Reality 目标站点 / TLS 域名。
function securityLabel(node) {
    const ps = node.protocol_settings || {};
    const tls = typeof ps.tls === "object" ? 1 : Number(ps.tls || 0);
    if (tls === 2) return { text: "Reality", sub: ps.reality_settings?.server_name || "" };
    if (tls === 1) return { text: "TLS", sub: ps.tls_settings?.server_name || ps.tls?.server_name || "" };
    return null;
}

// 测试结果的一行摘要
function TestResult({ state, reality, t }) {
    if (!state) return null;
    if (state.loading) return <div className="mt-1 text-xs text-muted-foreground">{t("detail.testing")}</div>;
    if (state.error) return <div className="mt-1 text-xs text-destructive">{t("detail.testFail")}：{state.error}</div>;
    const r = state.result || {};
    if (r.ok) {
        const parts = [t("detail.testOk")];
        const ms = r.median_latency ?? r.latency_ms;
        if (ms != null) parts.push(t("detail.testLatency", { ms }));
        if (r.tests) parts.push(`${r.success_tests}/${r.tests}`);
        if (r.exit_ip) parts.push(`${t("detail.testExit")} ${r.exit_ip}${r.country_code ? ` ${r.country_code}` : ""}`);
        if (state.mode === "tcp") parts.push(t("detail.testTcpOnly"));
        return <div className="mt-1 text-xs text-emerald-600 dark:text-emerald-400">{parts.join(" · ")}</div>;
    }
    const label = r.untestable ? t("detail.testUntestable") : t("detail.testFail");
    return (
        <div className="mt-1 text-xs text-destructive">
            {label}：{r.error || "-"}
            {!r.untestable && reality && <div className="text-muted-foreground">{t("detail.testRealityHint")}</div>}
        </div>
    );
}

// 服务器详情里的节点管理：查看、一键 / 完整创建、关联已有、编辑、显隐、启停、取消关联、删除、实测、导出链接。
// 改动由后端即时推送给宿主机（sync.nodes / sync.config）。
export function MachineNodes({ machine, machines, onChanged }) {
    const { t } = useTranslation("machine");
    const { t: ts } = useTranslation("server");
    const [nodes, setNodes] = useState([]);
    const nodeSort = useClientSort(nodes, {
        id: (n) => Number(n.id),
        name: (n) => n.name || "",
        type: (n) => n.type || "",
        host: (n) => `${n.host || ""}:${n.port || ""}`,
        show: (n) => (n.show ? 1 : 0),
        enabled: (n) => (n.enabled == null || n.enabled ? 1 : 0),
    });
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [busyId, setBusyId] = useState(null);

    const [quickOpen, setQuickOpen] = useState(false);
    const [formOpen, setFormOpen] = useState(false);
    const [formNode, setFormNode] = useState(null);
    const [refs, setRefs] = useState({ all: [], groups: [], routes: [] });
    const [deleteTarget, setDeleteTarget] = useState(null);
    const [unbindTarget, setUnbindTarget] = useState(null);
    const [bindOpen, setBindOpen] = useState(false);
    const [exportOpen, setExportOpen] = useState(false);
    const [tests, setTests] = useState({});
    const [testingAll, setTestingAll] = useState(false);

    const load = () => {
        setLoading(true);
        return getMachineNodes({ machine_id: machine.id })
            .then((res) => setNodes(res?.data || []))
            .catch((err) => setError(getErrorMessage(err)))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        setQuickOpen(false);
        setError("");
        setTests({});
        load();
    }, [machine.id]);

    // 编辑表单与「关联已有」需要完整节点数据、权限组和路由
    const loadRefs = async () => {
        const [all, groups, routes] = await Promise.all([getNodes(), fetchGroups(), fetchRoutes()]);
        const next = { all: all?.data || [], groups: groups?.data || [], routes: routes?.data || [] };
        setRefs(next);
        return next;
    };

    const changed = async () => {
        await load();
        onChanged?.();
    };

    const run = async (id, action, failKey) => {
        setError("");
        setBusyId(id);
        try {
            await action();
            await changed();
        } catch (err) {
            setError(`${t(failKey)}: ${getErrorMessage(err)}`);
        } finally {
            setBusyId(null);
        }
    };

    const openEdit = async (node) => {
        setError("");
        setBusyId(node.id);
        try {
            const { all } = await loadRefs();
            const full = all.find((n) => n.id === node.id);
            if (!full) throw new Error(t("detail.nodeMissing"));
            setFormNode(full);
            setFormOpen(true);
        } catch (err) {
            setError(getErrorMessage(err));
        } finally {
            setBusyId(null);
        }
    };

    const openCreate = async () => {
        setError("");
        try {
            await loadRefs();
            setFormNode({ machine_id: machine.id, enabled: true, show: true, host: machine.last_ip || "" });
            setFormOpen(true);
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    const openBind = async () => {
        setError("");
        try {
            await loadRefs();
            setBindOpen(true);
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    // 实测一个节点（面板服务器经节点访问外网）
    const testNode = async (node) => {
        setTests((prev) => ({ ...prev, [node.id]: { loading: true } }));
        try {
            const res = await testMachineNode(node.id);
            setTests((prev) => ({ ...prev, [node.id]: { result: res?.data?.result, mode: res?.data?.mode } }));
        } catch (err) {
            setTests((prev) => ({ ...prev, [node.id]: { error: getErrorMessage(err) } }));
        }
    };

    const testAll = async () => {
        setTestingAll(true);
        try {
            for (const node of nodes) {
                await testNode(node);
            }
        } finally {
            setTestingAll(false);
        }
    };

    const toggleShow = (node, checked) =>
        run(node.id, () => updateNode({ id: node.id, type: node.type, show: checked ? 1 : 0 }), "detail.toggleShowError");
    const toggleEnabled = (node, checked) =>
        run(node.id, () => updateNode({ id: node.id, type: node.type, enabled: checked }), "detail.toggleEnabledError");

    return (
        <div>
            <div className="flex flex-wrap items-center justify-between gap-2">
                <h4 className="text-sm font-medium">{t("detail.associatedNodes")}</h4>
                <div className="flex flex-wrap gap-2">
                    {!quickOpen && <Button size="sm" onClick={() => setQuickOpen(true)}>{t("quickNode.button")}</Button>}
                    <Button size="sm" variant="outline" onClick={openCreate}>{t("detail.addNodeToServer")}</Button>
                    <Button size="sm" variant="outline" onClick={openBind}>{t("detail.bindExistingButton")}</Button>
                    <Button size="sm" variant="outline" disabled={nodes.length === 0 || testingAll} onClick={testAll}>
                        {testingAll ? t("detail.testing") : t("detail.testAll")}
                    </Button>
                    <Button size="sm" variant="outline" disabled={nodes.length === 0} onClick={() => setExportOpen(true)}>{t("detail.exportLinks")}</Button>
                </div>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">{t("detail.testHint")}</p>

            {quickOpen && (
                <MachineQuickNode
                    key={machine.id}
                    machine={machine}
                    onCreated={changed}
                    onCancel={() => setQuickOpen(false)}
                />
            )}
            {error && <div className="mt-2 rounded-md bg-destructive/10 p-2 text-sm text-destructive">{error}</div>}

            {loading && nodes.length === 0 ? (
                <p className="mt-2 text-sm text-muted-foreground">...</p>
            ) : nodes.length === 0 ? (
                <p className="mt-2 text-sm text-muted-foreground">{t("detail.noNodes")}</p>
            ) : (
                <div className="mt-2 overflow-x-auto rounded-md border">
                    <table className="w-full text-sm">
                        <thead className="bg-muted/50 text-left text-xs text-muted-foreground">
                            <tr>
                                <th className="p-2"><SortLabel sortKey="id" sort={nodeSort.sort} onSort={nodeSort.toggle}>{t("detail.nodeId")}</SortLabel></th>
                                <th className="p-2"><SortLabel sortKey="name" sort={nodeSort.sort} onSort={nodeSort.toggle}>{t("detail.nodeName")}</SortLabel></th>
                                <th className="p-2"><SortLabel sortKey="type" sort={nodeSort.sort} onSort={nodeSort.toggle}>{t("detail.nodeType")}</SortLabel></th>
                                <th className="p-2"><SortLabel sortKey="host" sort={nodeSort.sort} onSort={nodeSort.toggle}>{t("detail.nodeHost")}</SortLabel></th>
                                <th className="p-2 text-center"><SortLabel sortKey="show" sort={nodeSort.sort} onSort={nodeSort.toggle}>{t("detail.nodeShow")}</SortLabel></th>
                                <th className="p-2 text-center"><SortLabel sortKey="enabled" sort={nodeSort.sort} onSort={nodeSort.toggle}>{t("detail.nodeEnabled")}</SortLabel></th>
                                <th className="p-2 text-right">{t("detail.nodeActions")}</th>
                            </tr>
                        </thead>
                        <tbody>
                            {nodeSort.sorted.map((node) => {
                                const sec = securityLabel(node);
                                const busy = busyId === node.id;
                                const enabled = node.enabled == null ? true : Boolean(node.enabled);
                                const listenPort = node.server_port && String(node.server_port) !== String(node.port) ? node.server_port : null;
                                return (
                                    <tr key={node.id} className={`border-t align-top ${enabled ? "" : "opacity-60"}`}>
                                        <td className="p-2 font-mono">{node.id}</td>
                                        <td className="min-w-[160px] max-w-[300px] break-words p-2">
                                            <div>{node.name}</div>
                                            <TestResult state={tests[node.id]} reality={sec?.text === "Reality"} t={t} />
                                        </td>
                                        <td className="p-2">
                                            <div>{node.type}</div>
                                            {sec && (
                                                <div className="text-xs text-muted-foreground">
                                                    {sec.text}{sec.sub ? ` · ${sec.sub}` : ""}
                                                </div>
                                            )}
                                        </td>
                                        <td className="p-2 font-mono text-xs">
                                            <div className="whitespace-nowrap">{node.host}:{node.port}</div>
                                            {listenPort && <div className="text-muted-foreground">{t("detail.listenPort")}: {listenPort}</div>}
                                        </td>
                                        <td className="p-2 text-center">
                                            <Switch checked={Boolean(node.show)} disabled={busy} onCheckedChange={(v) => toggleShow(node, v)} />
                                        </td>
                                        <td className="p-2 text-center">
                                            <Switch checked={enabled} disabled={busy} onCheckedChange={(v) => toggleEnabled(node, v)} />
                                        </td>
                                        <td className="p-2">
                                            <div className="flex justify-end gap-1">
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" disabled={busy || tests[node.id]?.loading} onClick={() => testNode(node)}>
                                                    {t("detail.testNode")}
                                                </Button>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" disabled={busy} onClick={() => openEdit(node)}>
                                                    {t("detail.editNode")}
                                                </Button>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" disabled={busy} onClick={() => setUnbindTarget(node)}>
                                                    {t("detail.unbindNode")}
                                                </Button>
                                                <Button variant="destructive" size="sm" className="h-7 px-2 text-xs" disabled={busy} onClick={() => setDeleteTarget(node)}>
                                                    {t("messages.deleteButton")}
                                                </Button>
                                            </div>
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                </div>
            )}

            <NodeFormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                node={formNode}
                groups={refs.groups}
                routes={refs.routes}
                machines={machines}
                nodes={refs.all}
                onSaved={changed}
                t={ts}
            />

            <ExportLinksDialog open={exportOpen} onOpenChange={setExportOpen} machine={machine} />

            <BindExistingDialog
                open={bindOpen}
                onOpenChange={setBindOpen}
                machine={machine}
                candidates={refs.all.filter((n) => !n.machine_id)}
                onBound={changed}
            />

            <ConfirmDialog
                open={Boolean(unbindTarget)}
                onOpenChange={(open) => !open && setUnbindTarget(null)}
                title={t("detail.unbindNode")}
                description={t("detail.unbindDescription", { name: unbindTarget?.name || "" })}
                confirmLabel={t("detail.unbindNode")}
                onConfirm={() => {
                    const node = unbindTarget;
                    return run(node.id, () => updateNode({ id: node.id, type: node.type, machine_id: 0 }), "detail.unbindFailed");
                }}
            />

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("detail.deleteNodeConfirm", { name: deleteTarget?.name || "" })}
                description={t("detail.deleteNodeDescription")}
                confirmLabel={t("messages.deleteButton")}
                onConfirm={() => {
                    const node = deleteTarget;
                    return run(node.id, () => dropNode({ id: node.id }), "detail.deleteNodeFailed");
                }}
            />
        </div>
    );
}

// 从未绑定任何服务器的节点里挑选，关联到当前服务器。
function BindExistingDialog({ open, onOpenChange, machine, candidates, onBound }) {
    const { t } = useTranslation("machine");
    const [search, setSearch] = useState("");
    const [selected, setSelected] = useState([]);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (open) {
            setSearch("");
            setSelected([]);
            setError("");
        }
    }, [open]);

    const list = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        if (!keyword) return candidates;
        return candidates.filter((n) => `${n.name} ${n.host} ${n.type}`.toLowerCase().includes(keyword));
    }, [candidates, search]);

    const toggle = (id, checked) => setSelected((prev) => (checked ? [...prev, id] : prev.filter((x) => x !== id)));

    const submit = async () => {
        setSubmitting(true);
        setError("");
        try {
            for (const id of selected) {
                const node = candidates.find((n) => n.id === id);
                await updateNode({ id, type: node?.type, machine_id: machine.id, enabled: true });
            }
            onOpenChange(false);
            await onBound?.();
        } catch (err) {
            setError(`${t("detail.bindFailed")}: ${getErrorMessage(err)}`);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-lg">
            <div className="p-6">
                <h3 className="text-lg font-semibold">{t("detail.bindExistingTitle")}</h3>
                <p className="mt-1 text-xs text-muted-foreground">{t("detail.bindExistingDescription", { name: machine.name })}</p>
                <Input className="mt-4" value={search} placeholder={t("detail.bindSearchPlaceholder")} onChange={(e) => setSearch(e.target.value)} />
                <div className="mt-3 max-h-72 overflow-y-auto rounded-md border">
                    {candidates.length === 0 ? (
                        <p className="p-3 text-sm text-muted-foreground">{t("detail.noUnboundNodes")}</p>
                    ) : list.length === 0 ? (
                        <p className="p-3 text-sm text-muted-foreground">{t("detail.noSearchResults")}</p>
                    ) : (
                        list.map((n) => (
                            <label key={n.id} className="flex cursor-pointer items-center gap-3 border-b p-2 text-sm last:border-b-0 hover:bg-muted/40">
                                <input type="checkbox" checked={selected.includes(n.id)} onChange={(e) => toggle(n.id, e.target.checked)} />
                                <span className="font-mono text-xs text-muted-foreground">#{n.id}</span>
                                <span className="flex-1 truncate">{n.name}</span>
                                <Badge variant="outline" className="text-[10px]">{n.type}</Badge>
                                <span className="font-mono text-xs text-muted-foreground">{n.host}:{n.port}</span>
                            </label>
                        ))
                    )}
                </div>
                {selected.length > 0 && <p className="mt-2 text-xs text-muted-foreground">{t("detail.selectedCount", { count: selected.length })}</p>}
                {error && <div className="mt-2 rounded-md bg-destructive/10 p-2 text-sm text-destructive">{error}</div>}
                <div className="mt-4 flex justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange(false)}>{t("detail.cancel")}</Button>
                    <Button disabled={selected.length === 0 || submitting} onClick={submit}>
                        {submitting ? t("detail.binding") : t("detail.bindConfirm", { count: selected.length })}
                    </Button>
                </div>
            </div>
        </Dialog>
    );
}

// 导出服务器上全部节点的分享链接：缺省每个节点用其权限组里的第一个可用用户，也可指定用户邮箱。
function ExportLinksDialog({ open, onOpenChange, machine }) {
    const { t } = useTranslation("machine");
    const [email, setEmail] = useState("");
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const [copied, setCopied] = useState("");

    const generate = async (forEmail) => {
        setLoading(true);
        setError("");
        try {
            const res = await getMachineNodeLinks(machine.id, forEmail);
            setItems(res?.data || []);
        } catch (err) {
            setItems([]);
            setError(getErrorMessage(err));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (open) {
            setEmail("");
            setCopied("");
            generate("");
        }
    }, [open, machine.id]);

    const links = items.filter((i) => i.link).map((i) => i.link);

    const copy = async (text, key) => {
        try {
            await navigator.clipboard.writeText(text);
            setCopied(key);
            setTimeout(() => setCopied((c) => (c === key ? "" : c)), 1500);
        } catch {
            // 剪贴板不可用时链接仍可手动选中复制
        }
    };

    const download = () => {
        const blob = new Blob([links.join("\n") + "\n"], { type: "text/plain;charset=utf-8" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${machine.name || "server"}-${machine.id}-links.txt`;
        a.click();
        URL.revokeObjectURL(url);
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-3xl">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t("detail.exportTitle")}</h3>
                <p className="mt-1 text-xs text-muted-foreground">{t("detail.exportDescription")}</p>
                <form
                    className="mt-4 flex gap-2"
                    onSubmit={(e) => {
                        e.preventDefault();
                        generate(email.trim());
                    }}
                >
                    <Input value={email} placeholder={t("detail.exportEmailPlaceholder")} onChange={(e) => setEmail(e.target.value)} />
                    <Button type="submit" variant="outline" disabled={loading}>{t("detail.exportGenerate")}</Button>
                </form>
                {error && <div className="mt-3 rounded-md bg-destructive/10 p-2 text-sm text-destructive">{error}</div>}
                <div className="mt-3 space-y-2">
                    {loading ? (
                        <p className="text-sm text-muted-foreground">...</p>
                    ) : items.length === 0 ? (
                        !error && <p className="text-sm text-muted-foreground">{t("detail.noNodes")}</p>
                    ) : (
                        items.map((item) => (
                            <div key={item.id} className="rounded-md border p-2">
                                <div className="flex flex-wrap items-center gap-2 text-sm">
                                    <span className="font-mono text-xs text-muted-foreground">#{item.id}</span>
                                    <span className="font-medium">{item.name}</span>
                                    <Badge variant="outline" className="text-[10px]">{item.type}</Badge>
                                    {!item.enabled && <Badge variant="secondary" className="text-[10px]">{t("detail.exportDisabled")}</Badge>}
                                    {item.user_email && <span className="text-xs text-muted-foreground">{t("detail.exportUser")}: {item.user_email}</span>}
                                    {item.link && (
                                        <Button variant="ghost" size="sm" className="ml-auto h-7 px-2 text-xs" onClick={() => copy(item.link, String(item.id))}>
                                            {copied === String(item.id) ? t("detail.copied") : t("detail.copy")}
                                        </Button>
                                    )}
                                </div>
                                {item.link && <code className="mt-1 block break-all rounded bg-muted/50 p-2 font-mono text-[11px]">{item.link}</code>}
                                {item.warning && <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">{item.warning}</p>}
                                {item.error && <p className="mt-1 text-xs text-destructive">{item.error}</p>}
                            </div>
                        ))
                    )}
                </div>
                <div className="mt-4 flex flex-wrap justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange(false)}>{t("detail.cancel")}</Button>
                    <Button variant="outline" disabled={links.length === 0} onClick={download}>{t("detail.download")}</Button>
                    <Button disabled={links.length === 0} onClick={() => copy(links.join("\n"), "all")}>
                        {copied === "all" ? t("detail.copied") : t("detail.copyAll", { count: links.length })}
                    </Button>
                </div>
            </div>
        </Dialog>
    );
}
