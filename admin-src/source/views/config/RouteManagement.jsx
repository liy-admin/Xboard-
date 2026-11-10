import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { fetchRoutes, saveRoute, dropRoute } from "../../api/server.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useClientSort } from "../../components/SortableHead.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { ClientRoutingCard } from "../../components/ClientRoutingCard.jsx";

const ACTIONS = ["block", "dns", "direct", "proxy"];
const EMPTY_FORM = { id: null, remarks: "", match: "", action: "block", action_value: "" };

export function RouteManagement() {
    const { t } = useTranslation("route");
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
        return fetchRoutes()
            .then((res) => setItems(res?.data || []))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
    }, [auth.token]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        if (!keyword) return items;
        return items.filter((r) => (r.remarks || "").toLowerCase().includes(keyword));
    }, [items, search]);

    const { sorted, sort, toggle: toggleSort } = useClientSort(filtered, {
        id: (i) => Number(i.id),
        remarks: (i) => i.remarks || "",
        action_value: (i) => `${i.action || ""} ${i.action_value || ""}`,
        action: (i) => i.action || "",
    });

    const openAdd = () => {
        setForm(EMPTY_FORM);
        setFormError("");
        setFormOpen(true);
    };

    const openEdit = (item) => {
        setForm({
            id: item.id,
            remarks: item.remarks || "",
            match: Array.isArray(item.match) ? item.match.join("\n") : "",
            action: item.action || "block",
            action_value: item.action_value || "",
        });
        setFormError("");
        setFormOpen(true);
    };

    const handleSubmit = async () => {
        setFormError("");
        if (!form.remarks.trim()) {
            const error = t("form.validation.remarks");
            setFormError(error);
            throw new Error(error);
        }
        const match = form.match.split("\n").map((line) => line.trim()).filter(Boolean);
        const isValueAction = form.action === "dns" || form.action === "proxy";
        await saveRoute({
            id: form.id ?? undefined,
            remarks: form.remarks.trim(),
            match,
            action: form.action,
            action_value: isValueAction ? form.action_value : "",
        });
        await load();
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await dropRoute({ id: deleteTarget.id });
        await load();
    };

    const actionValueLabel = (item) => {
        if ((item.action === "dns" || item.action === "proxy") && item.action_value) {
            return t(`columns.action_value.${item.action}`, { value: item.action_value });
        }
        return t(`columns.action_value.${item.action === "block" ? "block" : "direct"}`);
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
                            <SortableHead sortKey="remarks" sort={sort} onSort={toggleSort}>{t("columns.remarks")}</SortableHead>
                            <SortableHead sortKey="action_value" sort={sort} onSort={toggleSort}>{t("columns.action_value.title")}</SortableHead>
                            <SortableHead sortKey="action" sort={sort} onSort={toggleSort}>{t("columns.action")}</SortableHead>
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
                                    <TableCell className="font-medium">{item.remarks}</TableCell>
                                    <TableCell>
                                        <div className="text-sm font-medium">{actionValueLabel(item)}</div>
                                        <div className="text-xs text-muted-foreground">{t("columns.matchRules", { count: Array.isArray(item.match) ? item.match.length : 0 })}</div>
                                    </TableCell>
                                    <TableCell><Badge variant="secondary">{t(`actions.${item.action}`)}</Badge></TableCell>
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

            <hr className="my-6" />
            <ClientRoutingCard />

            <FormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                title={form.id ? t("form.edit") : t("form.create")}
                cancelLabel={t("form.cancel")}
                submitLabel={t("form.submit")}
                onSubmit={handleSubmit}
                className="sm:max-w-lg"
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.remarks")}</div>
                    <Input value={form.remarks} placeholder={t("form.remarksPlaceholder")} onChange={(e) => setForm((p) => ({ ...p, remarks: e.target.value }))} required />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.match")}</div>
                    <Textarea className="min-h-[120px] font-mono text-xs" placeholder={t("form.matchPlaceholder")} value={form.match} onChange={(e) => setForm((p) => ({ ...p, match: e.target.value }))} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.action")}</div>
                    <Select value={form.action} onValueChange={(v) => setForm((p) => ({ ...p, action: v }))}>
                        <SelectTrigger><SelectValue placeholder={t("form.actionPlaceholder")} /></SelectTrigger>
                        <SelectContent>
                            {ACTIONS.map((action) => <SelectItem key={action} value={action}>{t(`actions.${action}`)}</SelectItem>)}
                        </SelectContent>
                    </Select>
                </div>
                {form.action === "dns" && (
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.dns")}</div>
                        <Input value={form.action_value} placeholder={t("form.dnsPlaceholder")} onChange={(e) => setForm((p) => ({ ...p, action_value: e.target.value }))} required />
                    </div>
                )}
                {form.action === "proxy" && (
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.proxy")}</div>
                        <Input value={form.action_value} placeholder={t("form.proxyPlaceholder")} onChange={(e) => setForm((p) => ({ ...p, action_value: e.target.value }))} required />
                    </div>
                )}
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
