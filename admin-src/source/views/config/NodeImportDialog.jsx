import React, { useState, useEffect } from "react";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/Tabs.jsx";
import {
    parseNodes,
    batchImportNodes,
    getImportSubscriptions,
    saveImportSubscriptions,
    getPluginSubscription,
    countNodesByTag,
    deleteNodesByTag,
} from "../../api/server.js";

const PROTOCOL_LABEL = {
    vmess: "VMess",
    vless: "VLESS",
    trojan: "Trojan",
    shadowsocks: "SS",
    hysteria: "HY2",
    tuic: "TUIC",
};

// 只保留预设需要持久化的字段（后端返回的列表可能带有额外字段）。
function stripPreset(p) {
    return {
        id: p.id,
        name: p.name,
        url: p.url,
        tag: p.tag,
        group_ids: p.group_ids,
        rate: p.rate,
        show: p.show,
        schedule_enabled: p.schedule_enabled,
        interval_hours: p.interval_hours,
        refetch_balala: p.refetch_balala,
        last_run_at: p.last_run_at,
    };
}

function emptyState() {
    return {
        tab: "links",
        content: "",
        url: "",
        tag: "",
        groupIds: [],
        rate: "1",
        show: true,
        previewing: false,
        preview: null,
        previewError: "",
        importing: false,
        result: null,
        // 订阅预设
        presets: [],
        selectedPresetId: "",
        savingPreset: false,
        presetName: "",
        presetSchedule: false,
        presetInterval: "24",
        presetRefetch: false,
        showSaveForm: false,
        // 编辑已选中的预设
        editName: "",
        nodeCount: null,
        confirmDeleteNodes: false,
        deletingNodes: false,
        presetNote: "",
        pluginLoading: false,
        pluginNote: "",
    };
}

function errorText(err) {
    return typeof err === "string" ? err : err?.message || String(err);
}

export function NodeImportDialog({ open, onOpenChange, groups, onImported, t }) {
    const [state, setState] = useState(emptyState);
    const set = (patch) => setState((prev) => ({ ...prev, ...patch }));

    // 弹窗打开时加载订阅预设
    useEffect(() => {
        if (!open) return;
        let alive = true;
        getImportSubscriptions()
            .then((res) => {
                if (alive) set({ presets: Array.isArray(res?.data) ? res.data : [] });
            })
            .catch(() => {});
        return () => {
            alive = false;
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open]);

    const close = () => {
        onOpenChange(false);
        setState(emptyState());
    };

    const toggleGroup = (id) => {
        setState((prev) => {
            const has = prev.groupIds.includes(id);
            return { ...prev, groupIds: has ? prev.groupIds.filter((g) => g !== id) : [...prev.groupIds, id] };
        });
    };

    const buildPayload = () =>
        state.tab === "links" ? { type: "links", content: state.content } : { type: "subscription", url: state.url };

    const hasInput = state.tab === "links" ? state.content.trim() : state.url.trim();

    // 应用一个预设到当前表单
    const applyPreset = (p) => {
        set({
            tab: "subscription",
            url: p.url || "",
            tag: p.tag || "",
            groupIds: Array.isArray(p.group_ids) ? p.group_ids : [],
            rate: p.rate != null ? String(p.rate) : "1",
            show: p.show === 1 || p.show === true,
            selectedPresetId: p.id || "",
            presetSchedule: !!p.schedule_enabled,
            presetInterval: p.interval_hours != null ? String(p.interval_hours) : "24",
            presetRefetch: !!p.refetch_balala,
            editName: p.name || "",
            nodeCount: null,
            confirmDeleteNodes: false,
            presetNote: "",
            preview: null,
            result: null,
            previewError: "",
            pluginNote: "",
        });
        refreshNodeCount(p.tag);
    };

    const selectedPreset = state.presets.find((p) => p.id === state.selectedPresetId) || null;

    // 统计带该订阅标识的节点数（删除前确认用）
    const refreshNodeCount = (tag) => {
        if (!tag || !String(tag).trim()) {
            set({ nodeCount: null });
            return;
        }
        countNodesByTag(String(tag).trim())
            .then((res) => set({ nodeCount: res?.data?.count ?? 0 }))
            .catch(() => set({ nodeCount: null }));
    };

    // 持久化整个预设列表
    const persistPresets = async (list) => {
        const res = await saveImportSubscriptions(list);
        set({ presets: Array.isArray(res?.data) ? res.data : list });
    };

    const handleSavePreset = async () => {
        const name = state.presetName.trim();
        if (!name) {
            set({ previewError: t("import.subscription.preset_name_empty") });
            return;
        }
        if (!state.url.trim()) {
            set({ previewError: t("import.preview_empty") });
            return;
        }
        set({ savingPreset: true, previewError: "" });
        try {
            const newItem = {
                name,
                url: state.url.trim(),
                tag: state.tag.trim(),
                group_ids: state.groupIds,
                rate: state.rate !== "" ? Number(state.rate) : 1,
                show: state.show ? 1 : 0,
                schedule_enabled: state.presetSchedule,
                interval_hours: state.presetInterval !== "" ? Number(state.presetInterval) : 24,
                refetch_balala: state.presetRefetch,
            };
            await persistPresets([...state.presets.map(stripPreset), newItem]);
            set({ showSaveForm: false, presetName: "" });
        } catch (err) {
            set({ previewError: errorText(err) });
        } finally {
            set({ savingPreset: false });
        }
    };

    // 把当前表单（链接、标识、默认参数、定时设置、名称）保存回选中的预设
    const handleUpdatePreset = async () => {
        if (!selectedPreset) return;
        if (!state.url.trim()) {
            set({ previewError: t("import.preview_empty") });
            return;
        }
        set({ savingPreset: true, previewError: "", presetNote: "" });
        try {
            const updated = {
                ...stripPreset(selectedPreset),
                name: state.editName.trim() || selectedPreset.name,
                url: state.url.trim(),
                tag: state.tag.trim(),
                group_ids: state.groupIds,
                rate: state.rate !== "" ? Number(state.rate) : 1,
                show: state.show ? 1 : 0,
                schedule_enabled: state.presetSchedule,
                interval_hours: state.presetInterval !== "" ? Number(state.presetInterval) : 24,
                refetch_balala: state.presetRefetch,
            };
            await persistPresets(state.presets.map((p) => (p.id === selectedPreset.id ? updated : stripPreset(p))));
            set({ presetNote: t("import.subscription.preset_updated"), confirmDeleteNodes: false });
            refreshNodeCount(updated.tag);
        } catch (err) {
            set({ previewError: errorText(err) });
        } finally {
            set({ savingPreset: false });
        }
    };

    // 删除选中预设已保存的标识对应的全部节点（预设本身保留）
    const handleDeleteNodes = async () => {
        const tag = (selectedPreset?.tag || "").trim();
        if (!tag) return;
        set({ deletingNodes: true, previewError: "", presetNote: "" });
        try {
            const res = await deleteNodesByTag(tag);
            set({
                confirmDeleteNodes: false,
                presetNote: t("import.subscription.deleted_nodes", { count: res?.data?.deleted ?? 0 }),
            });
            refreshNodeCount(tag);
            onImported?.();
        } catch (err) {
            set({ previewError: errorText(err) });
        } finally {
            set({ deletingNodes: false });
        }
    };

    const handleDeletePreset = async (id) => {
        set({ savingPreset: true, previewError: "" });
        try {
            await persistPresets(state.presets.filter((p) => p.id !== id).map(stripPreset));
            if (state.selectedPresetId === id) set({ selectedPresetId: "", nodeCount: null, confirmDeleteNodes: false, presetNote: "" });
        } catch (err) {
            set({ previewError: errorText(err) });
        } finally {
            set({ savingPreset: false });
        }
    };

    const handleFetchFromPlugin = async () => {
        set({ pluginLoading: true, previewError: "", pluginNote: "" });
        try {
            const res = await getPluginSubscription();
            const url = res?.data?.subscribe_url || "";
            if (!url) {
                set({ previewError: t("import.subscription.plugin_empty") });
                return;
            }
            const fetchedAt = res?.data?.fetched_at;
            set({
                url,
                tab: "subscription",
                preview: null,
                result: null,
                pluginNote: fetchedAt ? t("import.subscription.plugin_fetched_at", { time: fetchedAt }) : "",
            });
        } catch (err) {
            set({ previewError: errorText(err) });
        } finally {
            set({ pluginLoading: false });
        }
    };

    const handlePreview = async () => {
        if (!hasInput) {
            set({ previewError: t("import.preview_empty") });
            return;
        }
        set({ previewing: true, previewError: "", result: null });
        try {
            const res = await parseNodes(buildPayload());
            set({ preview: res?.data || { servers: [], parse_errors: [] } });
        } catch (err) {
            set({ previewError: errorText(err) });
        } finally {
            set({ previewing: false });
        }
    };

    const handleImport = async () => {
        if (!hasInput) {
            set({ previewError: t("import.preview_empty") });
            return;
        }
        set({ importing: true, previewError: "" });
        try {
            const defaults = {
                group_ids: state.groupIds,
                rate: state.rate !== "" ? Number(state.rate) : undefined,
                show: state.show ? 1 : 0,
            };
            // 订阅来源标识：仅订阅导入、且填写了标识时附加为节点 tag
            const tag = state.tab === "subscription" ? state.tag.trim() : "";
            if (tag) defaults.tags = [tag];

            const res = await batchImportNodes({ ...buildPayload(), defaults });
            const data = res?.data || { created: 0, failed: 0, parse_errors: [], results: [] };
            set({ result: data });
            if (data.created > 0) onImported?.();
        } catch (err) {
            set({ previewError: errorText(err) });
        } finally {
            set({ importing: false });
        }
    };

    const renderScheduleFields = () => (
        <>
                                    <div className="flex items-center justify-between">
                                        <div className="text-xs text-muted-foreground">
                                            {t("import.subscription.schedule_enable")}
                                            <span className="ml-1 opacity-70">{t("import.subscription.schedule_hint")}</span>
                                        </div>
                                        <Switch
                                            checked={state.presetSchedule}
                                            onCheckedChange={(v) => set({ presetSchedule: v })}
                                        />
                                    </div>

                                    {state.presetSchedule && (
                                        <div className="grid gap-3 sm:grid-cols-2">
                                            <div>
                                                <label className="text-xs text-muted-foreground">
                                                    {t("import.subscription.schedule_interval")}
                                                </label>
                                                <Input
                                                    className="mt-1"
                                                    type="number"
                                                    min="1"
                                                    value={state.presetInterval}
                                                    onChange={(e) => set({ presetInterval: e.target.value })}
                                                />
                                            </div>
                                            <div className="flex items-center justify-between rounded-md border px-3">
                                                <span className="text-xs text-muted-foreground">
                                                    {t("import.subscription.schedule_refetch")}
                                                </span>
                                                <Switch
                                                    checked={state.presetRefetch}
                                                    onCheckedChange={(v) => set({ presetRefetch: v })}
                                                />
                                            </div>
                                        </div>
                                    )}
                                    {state.presetSchedule && !state.tag.trim() && (
                                        <p className="text-xs text-amber-600">{t("import.subscription.schedule_need_tag")}</p>
                                    )}
        </>
    );

    return (
        <Dialog open={open} onOpenChange={(v) => (v ? onOpenChange(v) : close())} className="sm:max-w-2xl">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t("import.title")}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{t("import.description")}</p>

                <Tabs value={state.tab} onValueChange={(v) => set({ tab: v, preview: null, result: null, previewError: "" })} className="mt-4">
                    <TabsList className="grid w-full grid-cols-2">
                        <TabsTrigger value="links">{t("import.tabs.links")}</TabsTrigger>
                        <TabsTrigger value="subscription">{t("import.tabs.subscription")}</TabsTrigger>
                    </TabsList>
                    <TabsContent value="links" className="mt-4">
                        <label className="text-sm font-medium">{t("import.links.label")}</label>
                        <Textarea
                            className="mt-1 min-h-[160px] font-mono text-xs"
                            value={state.content}
                            onChange={(e) => set({ content: e.target.value })}
                            placeholder={t("import.links.placeholder")}
                        />
                    </TabsContent>
                    <TabsContent value="subscription" className="mt-4 space-y-3">
                        {/* 已保存订阅预设 */}
                        <div>
                            <div className="mb-1 flex items-center justify-between">
                                <span className="text-xs text-muted-foreground">{t("import.subscription.saved")}</span>
                                <button
                                    type="button"
                                    className="text-xs text-primary hover:underline disabled:opacity-50"
                                    onClick={handleFetchFromPlugin}
                                    disabled={state.pluginLoading}
                                >
                                    {state.pluginLoading ? t("import.subscription.plugin_loading") : t("import.subscription.from_plugin")}
                                </button>
                            </div>
                            {state.presets.length > 0 ? (
                                <div className="flex flex-wrap gap-2">
                                    {state.presets.map((p) => (
                                        <span
                                            key={p.id}
                                            className={`group inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs ${
                                                state.selectedPresetId === p.id
                                                    ? "border-primary bg-primary/10 text-primary"
                                                    : "text-muted-foreground"
                                            }`}
                                        >
                                            <button type="button" onClick={() => applyPreset(p)} title={p.url}>
                                                {p.schedule_enabled && (
                                                    <span title={t("import.subscription.scheduled_badge", { hours: p.interval_hours || 24 })}>⏱ </span>
                                                )}
                                                {p.name}
                                            </button>
                                            <button
                                                type="button"
                                                className="opacity-50 hover:opacity-100 hover:text-destructive"
                                                onClick={() => handleDeletePreset(p.id)}
                                                disabled={state.savingPreset}
                                                aria-label="delete"
                                            >
                                                ×
                                            </button>
                                        </span>
                                    ))}
                                </div>
                            ) : (
                                <p className="text-xs text-muted-foreground/70">{t("import.subscription.no_saved")}</p>
                            )}
                        </div>

                        <div>
                            <label className="text-sm font-medium">{t("import.subscription.label")}</label>
                            <Input
                                className="mt-1"
                                value={state.url}
                                onChange={(e) => set({ url: e.target.value, presetNote: "" })}
                                placeholder={t("import.subscription.placeholder")}
                            />
                            {state.pluginNote && <p className="mt-1 text-xs text-muted-foreground">{state.pluginNote}</p>}
                        </div>

                        <div>
                            <label className="text-sm font-medium">{t("import.subscription.tag_label")}</label>
                            <Input
                                className="mt-1"
                                value={state.tag}
                                onChange={(e) => set({ tag: e.target.value })}
                                placeholder={t("import.subscription.tag_placeholder")}
                            />
                            <p className="mt-1 text-xs text-muted-foreground">{t("import.subscription.tag_hint")}</p>
                        </div>

                        {/* 编辑选中的预设：更换链接 / 改名 / 改定时，删除它导入的节点 */}
                        {selectedPreset && (
                            <div className="space-y-3 rounded-lg border border-primary/40 bg-primary/5 p-3">
                                <div className="text-xs font-medium text-primary">
                                    {t("import.subscription.editing", { name: selectedPreset.name })}
                                </div>
                                <div className="flex items-center gap-2">
                                    <Input
                                        className="flex-1"
                                        value={state.editName}
                                        onChange={(e) => set({ editName: e.target.value })}
                                        placeholder={t("import.subscription.preset_name_placeholder")}
                                    />
                                    <Button size="sm" onClick={handleUpdatePreset} loading={state.savingPreset}>
                                        {t("import.subscription.update_preset")}
                                    </Button>
                                </div>
                                <p className="text-xs text-muted-foreground">{t("import.subscription.update_preset_hint")}</p>
                                {renderScheduleFields()}
                                <div className="flex items-center justify-between gap-2 border-t pt-3">
                                    <span className="text-xs text-muted-foreground">
                                        {!selectedPreset.tag
                                            ? t("import.subscription.delete_nodes_no_tag")
                                            : state.nodeCount == null
                                              ? t("import.subscription.nodes_counting")
                                              : t("import.subscription.nodes_with_tag", { tag: selectedPreset.tag, count: state.nodeCount })}
                                    </span>
                                    {!state.confirmDeleteNodes && (
                                        <Button
                                            size="sm"
                                            variant="outline"
                                            className="shrink-0 text-destructive"
                                            disabled={!selectedPreset.tag || !state.nodeCount || state.deletingNodes}
                                            onClick={() => set({ confirmDeleteNodes: true, presetNote: "" })}
                                        >
                                            {t("import.subscription.delete_nodes")}
                                        </Button>
                                    )}
                                </div>
                                {state.confirmDeleteNodes && (
                                    <div className="rounded-md border border-destructive/40 bg-destructive/5 p-2">
                                        <p className="text-xs text-destructive">
                                            {t("import.subscription.delete_nodes_confirm", { tag: selectedPreset.tag, count: state.nodeCount })}
                                        </p>
                                        <div className="mt-2 flex justify-end gap-2">
                                            <Button size="sm" variant="outline" onClick={() => set({ confirmDeleteNodes: false })}>
                                                {t("import.subscription.cancel")}
                                            </Button>
                                            <Button size="sm" variant="destructive" loading={state.deletingNodes} onClick={handleDeleteNodes}>
                                                {t("import.subscription.delete_nodes_confirm_btn")}
                                            </Button>
                                        </div>
                                    </div>
                                )}
                                {state.presetNote && <p className="text-xs text-emerald-600">{state.presetNote}</p>}
                            </div>
                        )}

                        {/* 保存为预设（含定时导入设置） */}
                        <div>
                            {state.showSaveForm ? (
                                <div className="space-y-3 rounded-lg border p-3">
                                    <div className="flex items-center gap-2">
                                        <Input
                                            className="flex-1"
                                            value={state.presetName}
                                            onChange={(e) => set({ presetName: e.target.value })}
                                            placeholder={t("import.subscription.preset_name_placeholder")}
                                        />
                                        <Button size="sm" onClick={handleSavePreset} loading={state.savingPreset}>
                                            {t("import.subscription.preset_save")}
                                        </Button>
                                        <Button
                                            size="sm"
                                            variant="outline"
                                            onClick={() => set({ showSaveForm: false, presetName: "" })}
                                        >
                                            {t("import.close")}
                                        </Button>
                                    </div>

                                    {renderScheduleFields()}
                                </div>
                            ) : (
                                <button
                                    type="button"
                                    className="text-xs text-primary hover:underline"
                                    onClick={() => set({ showSaveForm: true, presetName: state.tag || "" })}
                                >
                                    {t(selectedPreset ? "import.subscription.save_as_new_preset" : "import.subscription.save_as_preset")}
                                </button>
                            )}
                        </div>
                    </TabsContent>
                </Tabs>

                <div className="mt-4 space-y-3 rounded-lg border p-3">
                    <div className="text-sm font-medium">{t("import.defaults.title")}</div>
                    <div>
                        <div className="mb-1 text-xs text-muted-foreground">{t("import.defaults.groups")}</div>
                        <div className="flex flex-wrap gap-2">
                            {(groups || []).map((g) => (
                                <button
                                    key={g.id}
                                    type="button"
                                    onClick={() => toggleGroup(g.id)}
                                    className={`rounded-full border px-2.5 py-1 text-xs ${
                                        state.groupIds.includes(g.id) ? "border-primary bg-primary/10 text-primary" : "text-muted-foreground"
                                    }`}
                                >
                                    {g.name}
                                </button>
                            ))}
                        </div>
                    </div>
                    <div className="grid gap-3 sm:grid-cols-2">
                        <div>
                            <label className="text-xs text-muted-foreground">{t("import.defaults.rate")}</label>
                            <Input className="mt-1" type="number" step="any" value={state.rate} onChange={(e) => set({ rate: e.target.value })} />
                        </div>
                        <div className="flex items-center justify-between rounded-md border px-3">
                            <span className="text-xs text-muted-foreground">{t("import.defaults.show")}</span>
                            <div className="flex items-center gap-2">
                                <span className="text-xs">{t(state.show ? "import.defaults.show_true" : "import.defaults.show_false")}</span>
                                <Switch checked={state.show} onCheckedChange={(v) => set({ show: v })} />
                            </div>
                        </div>
                    </div>
                </div>

                {state.previewError && <p className="mt-3 text-sm text-destructive">{state.previewError}</p>}

                {state.preview && (
                    <div className="mt-4 rounded-lg border p-3">
                        <p className="text-sm">
                            {t("import.preview_result", { count: state.preview.servers?.length || 0, errors: state.preview.parse_errors?.length || 0 })}
                        </p>
                        {state.preview.servers?.length > 0 && (
                            <div className="mt-2 flex flex-wrap gap-1.5">
                                {state.preview.servers.map((s) => (
                                    <Badge key={s.index} variant={s.success ? "secondary" : "destructive"} className="text-[10px]">
                                        {PROTOCOL_LABEL[s.type] || s.type || "?"} · {s.name || s.host}
                                        {s.code ? ` · ${s.code}` : ""}
                                    </Badge>
                                ))}
                            </div>
                        )}
                        {state.preview.parse_errors?.length > 0 && (
                            <div className="mt-2">
                                <div className="text-xs font-medium text-destructive">{t("import.errors_title")}</div>
                                <ul className="mt-1 space-y-0.5 text-xs text-destructive">
                                    {state.preview.parse_errors.map((e, i) => (
                                        <li key={i}>{errorText(e)}</li>
                                    ))}
                                </ul>
                            </div>
                        )}
                    </div>
                )}

                {state.result && (
                    <div className="mt-4 rounded-lg border p-3">
                        <p className="text-sm">{t("import.import_result", { created: state.result.created, failed: state.result.failed })}</p>
                        {state.result.parse_errors?.length > 0 && (
                            <ul className="mt-1 space-y-0.5 text-xs text-destructive">
                                {state.result.parse_errors.map((e, i) => (
                                    <li key={i}>{errorText(e)}</li>
                                ))}
                            </ul>
                        )}
                    </div>
                )}

                <div className="mt-6 flex justify-end gap-2">
                    <Button variant="outline" onClick={close}>
                        {t("import.close")}
                    </Button>
                    <Button variant="outline" onClick={handlePreview} loading={state.previewing} disabled={state.importing}>
                        {state.previewing ? t("import.previewing") : t("import.preview")}
                    </Button>
                    <Button onClick={handleImport} loading={state.importing} disabled={state.previewing}>
                        {state.importing ? t("import.importing") : t("import.import")}
                    </Button>
                </div>
            </div>
        </Dialog>
    );
}
