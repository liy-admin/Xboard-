import React, { useState } from "react";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/Tabs.jsx";
import { parseNodes, batchImportNodes } from "../../api/server.js";

const PROTOCOL_LABEL = {
    vmess: "VMess",
    vless: "VLESS",
    trojan: "Trojan",
    shadowsocks: "SS",
    hysteria: "HY2",
    tuic: "TUIC",
};

function emptyState() {
    return {
        tab: "links",
        content: "",
        url: "",
        groupIds: [],
        rate: "1",
        show: true,
        previewing: false,
        preview: null,
        previewError: "",
        importing: false,
        result: null,
    };
}

function errorText(err) {
    return typeof err === "string" ? err : err?.message || String(err);
}

export function NodeImportDialog({ open, onOpenChange, groups, onImported, t }) {
    const [state, setState] = useState(emptyState);
    const set = (patch) => setState((prev) => ({ ...prev, ...patch }));

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
            const res = await batchImportNodes({
                ...buildPayload(),
                defaults: {
                    group_ids: state.groupIds,
                    rate: state.rate !== "" ? Number(state.rate) : undefined,
                    show: state.show ? 1 : 0,
                },
            });
            const data = res?.data || { created: 0, failed: 0, parse_errors: [], results: [] };
            set({ result: data });
            if (data.created > 0) onImported?.();
        } catch (err) {
            set({ previewError: errorText(err) });
        } finally {
            set({ importing: false });
        }
    };

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
                    <TabsContent value="subscription" className="mt-4">
                        <label className="text-sm font-medium">{t("import.subscription.label")}</label>
                        <Input
                            className="mt-1"
                            value={state.url}
                            onChange={(e) => set({ url: e.target.value })}
                            placeholder={t("import.subscription.placeholder")}
                        />
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
