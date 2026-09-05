import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import {
    fetchKnowledge,
    fetchKnowledgeById,
    saveKnowledge,
    dropKnowledge,
    showKnowledge,
    sortKnowledge,
} from "../../api/knowledge.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";

const LANGUAGES = ["en-US", "ja-JP", "ko-KR", "vi-VN", "zh-CN", "zh-TW", "ru-RU"];
const EMPTY_FORM = { id: null, title: "", category: "", language: "zh-CN", show: true, body: "" };

export function KnowledgeManagement() {
    const { t } = useTranslation("knowledge");
    const auth = useAuthState();
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");

    const [sortMode, setSortMode] = useState(false);
    const [sortedIds, setSortedIds] = useState([]);
    const [sortSaving, setSortSaving] = useState(false);
    const [dragId, setDragId] = useState(null);

    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState(EMPTY_FORM);
    const [formError, setFormError] = useState("");

    const [deleteTarget, setDeleteTarget] = useState(null);

    const load = () => {
        setLoading(true);
        return fetchKnowledge()
            .then((res) => {
                const list = res?.data || [];
                setItems(list);
                setSortedIds(list.map((k) => k.id));
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
    }, [auth.token]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        if (!keyword) return items;
        return items.filter((k) => (k.title || "").toLowerCase().includes(keyword));
    }, [items, search]);

    const orderedForSort = useMemo(() => {
        const byId = new Map(items.map((k) => [k.id, k]));
        return sortedIds.map((id) => byId.get(id)).filter(Boolean);
    }, [sortedIds, items]);

    const openAdd = () => {
        setForm(EMPTY_FORM);
        setFormError("");
        setFormOpen(true);
    };

    const openEdit = async (item) => {
        setForm({
            id: item.id,
            title: item.title || "",
            category: item.category || "",
            language: item.language || "zh-CN",
            show: Boolean(item.show),
            body: "",
        });
        setFormError("");
        setFormOpen(true);
        const res = await fetchKnowledgeById(item.id);
        const full = res?.data || {};
        setForm((prev) => ({
            ...prev,
            title: full.title ?? prev.title,
            category: full.category ?? prev.category,
            language: full.language ?? prev.language,
            show: full.show != null ? Boolean(full.show) : prev.show,
            body: full.body || "",
        }));
    };

    const handleSubmit = async () => {
        setFormError("");
        try {
            await saveKnowledge({
                id: form.id ?? undefined,
                title: form.title,
                category: form.category,
                language: form.language,
                show: form.show ? 1 : 0,
                body: form.body,
            });
            await load();
        } catch (e) {
            setFormError(e?.message || "提交失败");
            throw e;
        }
    };

    const toggleShow = async (item) => {
        setItems((prev) => prev.map((k) => (k.id === item.id ? { ...k, show: k.show ? 0 : 1 } : k)));
        try {
            await showKnowledge({ id: item.id });
        } catch {
            await load();
        }
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await dropKnowledge({ id: deleteTarget.id });
        await load();
    };

    const startSort = () => {
        setSortedIds(items.map((k) => k.id));
        setSortMode(true);
    };

    const handleDragStart = (id) => setDragId(id);
    const handleDragOver = (event, overId) => {
        event.preventDefault();
        if (dragId === null || dragId === overId) return;
        setSortedIds((prev) => {
            const next = [...prev];
            const fromIndex = next.indexOf(dragId);
            const toIndex = next.indexOf(overId);
            if (fromIndex === -1 || toIndex === -1) return prev;
            next.splice(fromIndex, 1);
            next.splice(toIndex, 0, dragId);
            return next;
        });
    };

    const saveSort = async () => {
        setSortSaving(true);
        try {
            await sortKnowledge({ ids: sortedIds });
            await load();
            setSortMode(false);
        } finally {
            setSortSaving(false);
        }
    };

    return (
        <div className="p-4 md:p-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-lg font-medium">{t("title")}</h3>
                <p className="text-sm text-muted-foreground">{t("description")}</p>
            </div>
            <hr className="my-6" />

            <SearchFilterToolbar
                search={search}
                onSearchChange={setSearch}
                searchPlaceholder={t("toolbar.searchPlaceholder")}
                resetLabel={t("toolbar.reset")}
                onReset={() => setSearch("")}
                actions={
                    sortMode ? (
                        <Button onClick={saveSort} loading={sortSaving}>{t("toolbar.saveSort")}</Button>
                    ) : (
                        <>
                            <Button variant="outline" onClick={startSort}>{t("toolbar.editSort")}</Button>
                            <Button onClick={openAdd}>{t("form.add")}</Button>
                        </>
                    )
                }
            />
            {sortMode && <p className="mt-2 text-sm text-muted-foreground">{t("toolbar.sortModeHint")}</p>}

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-16">{t("columns.id")}</TableHead>
                            <TableHead className="w-20">{t("columns.status")}</TableHead>
                            <TableHead>{t("columns.title")}</TableHead>
                            <TableHead>{t("columns.category")}</TableHead>
                            <TableHead className="w-32 text-right">{t("columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={5}>...</TableEmpty>
                        ) : (sortMode ? orderedForSort : filtered).length === 0 ? (
                            <TableEmpty colSpan={5}>-</TableEmpty>
                        ) : (
                            (sortMode ? orderedForSort : filtered).map((item) => (
                                <TableRow
                                    key={item.id}
                                    draggable={sortMode}
                                    onDragStart={() => handleDragStart(item.id)}
                                    onDragOver={(e) => handleDragOver(e, item.id)}
                                    className={sortMode ? "cursor-move" : undefined}
                                >
                                    <TableCell><Badge variant="outline" className="font-mono">{item.id}</Badge></TableCell>
                                    <TableCell>
                                        <Switch checked={Boolean(item.show)} onCheckedChange={() => toggleShow(item)} disabled={sortMode} />
                                    </TableCell>
                                    <TableCell className="line-clamp-2 font-medium">{item.title}</TableCell>
                                    <TableCell>{item.category && <Badge variant="secondary">{item.category}</Badge>}</TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button variant="outline" size="sm" onClick={() => openEdit(item)} disabled={sortMode}>
                                                {t("form.edit")}
                                            </Button>
                                            <Button variant="destructive" size="sm" onClick={() => setDeleteTarget(item)} disabled={sortMode}>
                                                {t("messages.deleteButton")}
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            <FormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                title={form.id ? t("form.edit") : t("form.add")}
                cancelLabel={t("form.cancel")}
                submitLabel={t("form.submit")}
                onSubmit={handleSubmit}
                className="sm:max-w-2xl"
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}
                <div className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.title")}</div>
                        <Input value={form.title} placeholder={t("form.titlePlaceholder")} onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))} required />
                    </div>
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.category")}</div>
                        <Input value={form.category} placeholder={t("form.categoryPlaceholder")} onChange={(e) => setForm((p) => ({ ...p, category: e.target.value }))} />
                    </div>
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.language")}</div>
                        <Select value={form.language} onValueChange={(v) => setForm((p) => ({ ...p, language: v }))}>
                            <SelectTrigger><SelectValue placeholder={t("form.languagePlaceholder")} /></SelectTrigger>
                            <SelectContent>
                                {LANGUAGES.map((code) => <SelectItem key={code} value={code}>{t(`languages.${code}`)}</SelectItem>)}
                            </SelectContent>
                        </Select>
                    </div>
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.show")}</div>
                        <div className="flex h-9 items-center">
                            <Switch checked={form.show} onCheckedChange={(v) => setForm((p) => ({ ...p, show: v }))} />
                        </div>
                    </div>
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.content")}</div>
                    <Textarea className="min-h-[300px] font-mono text-xs" value={form.body} onChange={(e) => setForm((p) => ({ ...p, body: e.target.value }))} />
                </div>
            </FormDialog>

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
