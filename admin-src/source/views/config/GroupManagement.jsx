import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { fetchGroups, saveGroup, dropGroup } from "../../api/server.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useClientSort } from "../../components/SortableHead.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";

const NAME_PATTERN = /^[a-zA-Z0-9一-龥_-]+$/;
const EMPTY_FORM = { id: null, name: "" };

export function GroupManagement() {
    const { t } = useTranslation("group");
    const auth = useAuthState();
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");

    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState(EMPTY_FORM);
    const [formError, setFormError] = useState("");

    const [deleteTarget, setDeleteTarget] = useState(null);

    const load = () => {
        setLoading(true);
        return fetchGroups()
            .then((res) => setItems(res?.data || []))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
    }, [auth.token]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        if (!keyword) return items;
        return items.filter((g) => (g.name || "").toLowerCase().includes(keyword));
    }, [items, search]);

    const { sorted, sort, toggle: toggleSort } = useClientSort(filtered, {
        id: (i) => Number(i.id),
        name: (i) => i.name || "",
        users_count: (i) => Number(i.users_count ?? 0),
        server_count: (i) => Number(i.server_count ?? 0),
    });

    const openAdd = () => {
        setForm(EMPTY_FORM);
        setFormError("");
        setFormOpen(true);
    };

    const openEdit = (item) => {
        setForm({ id: item.id, name: item.name || "" });
        setFormError("");
        setFormOpen(true);
    };

    const validateName = (name) => {
        if (name.length < 2) return t("messages.nameValidation.min");
        if (name.length > 50) return t("messages.nameValidation.max");
        if (!NAME_PATTERN.test(name)) return t("messages.nameValidation.pattern");
        return "";
    };

    const handleSubmit = async () => {
        setFormError("");
        const error = validateName(form.name.trim());
        if (error) {
            setFormError(error);
            throw new Error(error);
        }
        await saveGroup({ id: form.id ?? undefined, name: form.name.trim() });
        await load();
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await dropGroup({ id: deleteTarget.id });
        await load();
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
                actions={<Button onClick={openAdd}>{t("form.add")}</Button>}
            />

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <SortableHead className="w-16" sortKey="id" sort={sort} onSort={toggleSort}>{t("columns.id")}</SortableHead>
                            <SortableHead sortKey="name" sort={sort} onSort={toggleSort}>{t("columns.name")}</SortableHead>
                            <SortableHead className="w-32" sortKey="users_count" sort={sort} onSort={toggleSort}>{t("columns.usersCount")}</SortableHead>
                            <SortableHead className="w-32" sortKey="server_count" sort={sort} onSort={toggleSort}>{t("columns.serverCount")}</SortableHead>
                            <TableHead className="w-32 text-right">{t("columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={5}>...</TableEmpty>
                        ) : sorted.length === 0 ? (
                            <TableEmpty colSpan={5}>-</TableEmpty>
                        ) : (
                            sorted.map((item) => (
                                <TableRow key={item.id}>
                                    <TableCell><Badge variant="outline" className="font-mono">{item.id}</Badge></TableCell>
                                    <TableCell className="font-medium">{item.name}</TableCell>
                                    <TableCell>{item.users_count ?? 0}</TableCell>
                                    <TableCell>{item.server_count ?? 0}</TableCell>
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
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            <FormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                title={form.id ? t("form.edit") : t("form.create")}
                description={form.id ? t("form.editDescription") : t("form.createDescription")}
                cancelLabel={t("form.cancel")}
                submitLabel={form.id ? t("form.update") : t("form.create")}
                onSubmit={handleSubmit}
                className="sm:max-w-md"
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.name")}</div>
                    <Input value={form.name} placeholder={t("form.namePlaceholder")} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} required />
                    <p className="text-xs text-muted-foreground">{t("form.nameDescription")}</p>
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
