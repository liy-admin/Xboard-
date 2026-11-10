import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { fetchNotices, saveNotice, dropNotice, showNotice, sortNotices } from "../../api/notice.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useClientSort } from "../../components/SortableHead.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";

const EMPTY_FORM = { id: null, title: "", content: "", img_url: "", url: "", show: true, tags: [] };

export function NoticeManagement() {
    const { t } = useTranslation("notice");
    const auth = useAuthState();
    const [notices, setNotices] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");
    const [sortMode, setSortMode] = useState(false);
    const [sortedIds, setSortedIds] = useState([]);
    const [sortSaving, setSortSaving] = useState(false);
    const [dragId, setDragId] = useState(null);

    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState(EMPTY_FORM);
    const [formError, setFormError] = useState("");
    const [tagInput, setTagInput] = useState("");

    const [deleteTarget, setDeleteTarget] = useState(null);

    const load = () => {
        setLoading(true);
        return fetchNotices()
            .then((res) => {
                const list = res?.data || [];
                setNotices(list);
                setSortedIds(list.map((n) => n.id));
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
    }, [auth.token]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        if (!keyword) return notices;
        return notices.filter((n) => (n.title || "").toLowerCase().includes(keyword));
    }, [notices, search]);

    const { sorted, sort, toggle: toggleSort } = useClientSort(filtered, {
        id: (i) => Number(i.id),
        show: (i) => (i.show ? 1 : 0),
        title: (i) => i.title || "",
    });

    const orderedForSort = useMemo(() => {
        const byId = new Map(notices.map((n) => [n.id, n]));
        return sortedIds.map((id) => byId.get(id)).filter(Boolean);
    }, [sortedIds, notices]);

    const openAdd = () => {
        setForm(EMPTY_FORM);
        setFormError("");
        setTagInput("");
        setFormOpen(true);
    };

    const openEdit = (notice) => {
        setForm({
            id: notice.id,
            title: notice.title || "",
            content: notice.content || "",
            img_url: notice.img_url || "",
            url: notice.url || "",
            show: Boolean(notice.show),
            tags: Array.isArray(notice.tags) ? notice.tags : [],
        });
        setFormError("");
        setTagInput("");
        setFormOpen(true);
    };

    const addTag = () => {
        const value = tagInput.trim();
        if (!value) return;
        setForm((prev) => (prev.tags.includes(value) ? prev : { ...prev, tags: [...prev.tags, value] }));
        setTagInput("");
    };

    const removeTag = (tag) => {
        setForm((prev) => ({ ...prev, tags: prev.tags.filter((item) => item !== tag) }));
    };

    const handleSubmit = async () => {
        setFormError("");
        try {
            await saveNotice({
                id: form.id ?? undefined,
                title: form.title,
                content: form.content,
                img_url: form.img_url,
                url: form.url,
                show: form.show ? 1 : 0,
                tags: form.tags,
            });
            await load();
        } catch (e) {
            setFormError(e?.message || "提交失败");
            throw e;
        }
    };

    const toggleShow = async (notice) => {
        setNotices((prev) => prev.map((n) => (n.id === notice.id ? { ...n, show: n.show ? 0 : 1 } : n)));
        try {
            await showNotice(notice.id);
        } catch {
            await load();
        }
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await dropNotice(deleteTarget.id);
        await load();
    };

    const startSort = () => {
        setSortedIds(notices.map((n) => n.id));
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
            await sortNotices(sortedIds);
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
                searchPlaceholder={t("table.toolbar.search")}
                resetLabel={t("table.toolbar.reset")}
                onReset={() => setSearch("")}
                actions={
                    sortMode ? (
                        <Button onClick={saveSort} loading={sortSaving}>{t("table.toolbar.sort.save")}</Button>
                    ) : (
                        <>
                            <Button variant="outline" onClick={startSort}>{t("table.toolbar.sort.edit")}</Button>
                            <Button onClick={openAdd}>{t("form.add.button")}</Button>
                        </>
                    )
                }
            />

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <SortableHead className="w-16" sortKey="id" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("table.columns.id")}</SortableHead>
                            <SortableHead className="w-24" sortKey="show" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("table.columns.show")}</SortableHead>
                            <SortableHead sortKey="title" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("table.columns.title")}</SortableHead>
                            <TableHead className="w-32 text-right">{t("table.columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={4}>...</TableEmpty>
                        ) : (sortMode ? orderedForSort : sorted).length === 0 ? (
                            <TableEmpty colSpan={4}>-</TableEmpty>
                        ) : (
                            (sortMode ? orderedForSort : sorted).map((notice) => (
                                <TableRow
                                    key={notice.id}
                                    draggable={sortMode}
                                    onDragStart={() => handleDragStart(notice.id)}
                                    onDragOver={(e) => handleDragOver(e, notice.id)}
                                    className={sortMode ? "cursor-move" : undefined}
                                >
                                    <TableCell>{notice.id}</TableCell>
                                    <TableCell>
                                        <Switch checked={Boolean(notice.show)} onCheckedChange={() => toggleShow(notice)} disabled={sortMode} />
                                    </TableCell>
                                    <TableCell>{notice.title}</TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button variant="outline" size="sm" onClick={() => openEdit(notice)} disabled={sortMode}>
                                                {t("table.actions.edit")}
                                            </Button>
                                            <Button variant="destructive" size="sm" onClick={() => setDeleteTarget(notice)} disabled={sortMode}>
                                                {t("table.actions.delete.title")}
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
                {sortMode && <p className="mt-2 text-sm text-muted-foreground">{t("table.toolbar.sort.hint", null, "拖拽公告进行排序，完成后点击保存")}</p>}
            </div>

            <FormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                title={form.id ? t("form.edit.title") : t("form.add.title")}
                cancelLabel={t("form.buttons.cancel")}
                submitLabel={t("form.buttons.submit")}
                onSubmit={handleSubmit}
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.title.label")}</div>
                    <Input value={form.title} placeholder={t("form.fields.title.placeholder")} onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))} required />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.content.label")}</div>
                    <Textarea rows={6} value={form.content} onChange={(e) => setForm((p) => ({ ...p, content: e.target.value }))} required />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.img_url.label")}</div>
                    <Input value={form.img_url} placeholder={t("form.fields.img_url.placeholder")} onChange={(e) => setForm((p) => ({ ...p, img_url: e.target.value }))} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.url.label")}</div>
                    <Input value={form.url} placeholder={t("form.fields.url.placeholder")} onChange={(e) => setForm((p) => ({ ...p, url: e.target.value }))} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.tags.label")}</div>
                    <div className="flex flex-wrap gap-1.5">
                        {form.tags.map((tag) => (
                            <Badge key={tag} variant="secondary" className="cursor-pointer" onClick={() => removeTag(tag)}>
                                {tag} ×
                            </Badge>
                        ))}
                    </div>
                    <Input
                        value={tagInput}
                        placeholder={t("form.fields.tags.placeholder")}
                        onChange={(e) => setTagInput(e.target.value)}
                        onKeyDown={(e) => {
                            if (e.key === "Enter") {
                                e.preventDefault();
                                addTag();
                            }
                        }}
                    />
                </div>
                <div className="flex items-center justify-between rounded-lg border p-3">
                    <div className="text-sm font-medium">{t("form.fields.show.label")}</div>
                    <Switch checked={form.show} onCheckedChange={(v) => setForm((p) => ({ ...p, show: v }))} />
                </div>
            </FormDialog>

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("table.actions.delete.title")}
                description={t("table.actions.delete.description")}
                confirmLabel={t("table.actions.delete.title")}
                onConfirm={handleDelete}
            />
        </div>
    );
}
