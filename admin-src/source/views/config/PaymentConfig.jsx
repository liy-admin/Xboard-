import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import {
    fetchPayments,
    getPaymentMethods,
    getPaymentForm,
    savePayment,
    dropPayment,
    showPayment,
    sortPayments,
    uploadPaymentIcon,
} from "../../api/payment.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Tooltip, TooltipTrigger, TooltipContent } from "../../components/Tooltip.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useClientSort } from "../../components/SortableHead.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { DynamicConfigFields, configSchemaToValues } from "../../components/DynamicConfigFields.jsx";

const EMPTY_FORM = {
    id: null, name: "", icon: "", notify_domain: "",
    handling_fee_percent: "", handling_fee_fixed: "", payment: "",
};

export function PaymentConfig() {
    const { t } = useTranslation("payment");
    const auth = useAuthState();
    const [payments, setPayments] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");
    const [methods, setMethods] = useState([]);

    const [sortMode, setSortMode] = useState(false);
    const [sortedIds, setSortedIds] = useState([]);
    const [sortSaving, setSortSaving] = useState(false);
    const [dragId, setDragId] = useState(null);

    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState(EMPTY_FORM);
    const [formError, setFormError] = useState("");
    const [configSchema, setConfigSchema] = useState({});
    const [configValues, setConfigValues] = useState({});

    const [deleteTarget, setDeleteTarget] = useState(null);

    const [iconUploading, setIconUploading] = useState(false);
    const [iconUploadError, setIconUploadError] = useState("");
    const iconFileRef = useRef(null);

    const load = () => {
        setLoading(true);
        return fetchPayments()
            .then((res) => {
                const list = res?.data || [];
                setPayments(list);
                setSortedIds(list.map((p) => p.id));
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
        getPaymentMethods().then((res) => setMethods(res?.data || []));
    }, [auth.token]);

    useEffect(() => {
        if (!formOpen || !form.payment) return;
        getPaymentForm({ payment: form.payment, ...(form.id ? { id: Number(form.id) } : {}) })
            .then((res) => {
                const schema = res?.data || {};
                setConfigSchema(schema);
                setConfigValues(configSchemaToValues(schema));
            })
            .catch(() => {
                setConfigSchema({});
                setConfigValues({});
            });
    }, [form.payment, form.id, formOpen]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        if (!keyword) return payments;
        return payments.filter((p) => (p.name || "").toLowerCase().includes(keyword));
    }, [payments, search]);

    const { sorted, sort, toggle: toggleSort } = useClientSort(filtered, {
        id: (i) => Number(i.id),
        enable: (i) => (i.enable ? 1 : 0),
        name: (i) => i.name || "",
        payment: (i) => i.payment || "",
        notify_url: (i) => i.notify_url || "",
    });

    const orderedForSort = useMemo(() => {
        const byId = new Map(payments.map((p) => [p.id, p]));
        return sortedIds.map((id) => byId.get(id)).filter(Boolean);
    }, [sortedIds, payments]);

    const openAdd = () => {
        setForm(EMPTY_FORM);
        setFormError("");
        setIconUploadError("");
        setConfigSchema({});
        setConfigValues({});
        setFormOpen(true);
    };

    const openEdit = (payment) => {
        setForm({
            id: payment.id,
            name: payment.name || "",
            icon: payment.icon || "",
            notify_domain: payment.notify_domain || "",
            handling_fee_percent: payment.handling_fee_percent ?? "",
            handling_fee_fixed: payment.handling_fee_fixed != null ? payment.handling_fee_fixed / 100 : "",
            payment: payment.payment || "",
        });
        setFormError("");
        setIconUploadError("");
        setFormOpen(true);
    };

    const handleIconFileSelected = async (e) => {
        const file = e.target.files?.[0];
        e.target.value = "";
        if (!file) return;
        setIconUploading(true);
        setIconUploadError("");
        try {
            const res = await uploadPaymentIcon(file);
            const url = res?.data?.url;
            if (url) {
                setForm((p) => ({ ...p, icon: url }));
            }
        } catch (err) {
            setIconUploadError(err?.message || t("form.fields.icon.upload_error"));
        } finally {
            setIconUploading(false);
        }
    };

    const handleSubmit = async () => {
        setFormError("");
        try {
            await savePayment({
                id: form.id ?? undefined,
                name: form.name,
                icon: form.icon,
                notify_domain: form.notify_domain,
                handling_fee_percent: form.handling_fee_percent === "" ? null : Number(form.handling_fee_percent),
                handling_fee_fixed: form.handling_fee_fixed === "" ? null : Math.round(Number(form.handling_fee_fixed) * 100),
                payment: form.payment,
                config: configValues,
            });
            await load();
        } catch (e) {
            setFormError(e?.message || "提交失败");
            throw e;
        }
    };

    const toggleEnable = async (payment) => {
        setPayments((prev) => prev.map((p) => (p.id === payment.id ? { ...p, enable: !p.enable } : p)));
        try {
            await showPayment({ id: payment.id });
        } catch {
            await load();
        }
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await dropPayment({ id: deleteTarget.id });
        await load();
    };

    const startSort = () => {
        setSortedIds(payments.map((p) => p.id));
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
            await sortPayments({ ids: sortedIds });
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
            {sortMode && <p className="mt-2 text-sm text-muted-foreground">{t("table.toolbar.sort.hint")}</p>}

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <SortableHead className="w-16" sortKey="id" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("table.columns.id")}</SortableHead>
                            <SortableHead className="w-20" sortKey="enable" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("table.columns.enable")}</SortableHead>
                            <SortableHead sortKey="name" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("table.columns.name")}</SortableHead>
                            <SortableHead sortKey="payment" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("table.columns.payment")}</SortableHead>
                            <SortableHead sortKey="notify_url" sort={sort} onSort={toggleSort} disabled={sortMode}>
                                <Tooltip>
                                    <TooltipTrigger>{t("table.columns.notify_url")}</TooltipTrigger>
                                    <TooltipContent>{t("table.columns.notify_url_tooltip")}</TooltipContent>
                                </Tooltip>
                            </SortableHead>
                            <TableHead className="w-40 text-right">{t("table.columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={6}>...</TableEmpty>
                        ) : (sortMode ? orderedForSort : sorted).length === 0 ? (
                            <TableEmpty colSpan={6}>-</TableEmpty>
                        ) : (
                            (sortMode ? orderedForSort : sorted).map((payment) => (
                                <TableRow
                                    key={payment.id}
                                    draggable={sortMode}
                                    onDragStart={() => handleDragStart(payment.id)}
                                    onDragOver={(e) => handleDragOver(e, payment.id)}
                                    className={sortMode ? "cursor-move" : undefined}
                                >
                                    <TableCell><Badge variant="outline" className="font-mono">{payment.id}</Badge></TableCell>
                                    <TableCell>
                                        <Switch checked={Boolean(payment.enable)} onCheckedChange={() => toggleEnable(payment)} disabled={sortMode} />
                                    </TableCell>
                                    <TableCell className="max-w-[200px] truncate font-medium">{payment.name}</TableCell>
                                    <TableCell className="max-w-[200px] truncate">{payment.payment}</TableCell>
                                    <TableCell className="max-w-[260px] truncate">{payment.notify_url}</TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button variant="outline" size="sm" onClick={() => openEdit(payment)} disabled={sortMode}>
                                                {t("table.actions.edit")}
                                            </Button>
                                            <Button variant="destructive" size="sm" onClick={() => setDeleteTarget(payment)} disabled={sortMode}>
                                                {t("table.actions.delete.title")}
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
                title={form.id ? t("form.edit.title") : t("form.add.title")}
                cancelLabel={t("form.buttons.cancel")}
                submitLabel={t("form.buttons.submit")}
                onSubmit={handleSubmit}
                className="sm:max-w-2xl"
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.name.label")}</div>
                    <Input value={form.name} placeholder={t("form.fields.name.placeholder")} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} required />
                    <p className="text-xs text-muted-foreground">{t("form.fields.name.description")}</p>
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.icon.label")}</div>
                    <div className="flex gap-2">
                        <Input value={form.icon} placeholder={t("form.fields.icon.placeholder")} onChange={(e) => setForm((p) => ({ ...p, icon: e.target.value }))} />
                        <button
                            type="button"
                            className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md border border-input px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
                            disabled={iconUploading}
                            onClick={() => iconFileRef.current?.click()}
                        >
                            {iconUploading ? t("form.fields.icon.uploading") : t("form.fields.icon.upload")}
                        </button>
                        <input
                            ref={iconFileRef}
                            type="file"
                            accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml"
                            className="hidden"
                            onChange={handleIconFileSelected}
                        />
                    </div>
                    {form.icon && (
                        <img src={form.icon} alt="" className="h-10 w-10 rounded border object-contain" />
                    )}
                    {iconUploadError && <p className="text-xs text-destructive">{iconUploadError}</p>}
                    <p className="text-xs text-muted-foreground">{t("form.fields.icon.description")}</p>
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.notify_domain.label")}</div>
                    <Input value={form.notify_domain} placeholder={t("form.fields.notify_domain.placeholder")} onChange={(e) => setForm((p) => ({ ...p, notify_domain: e.target.value }))} />
                    <p className="text-xs text-muted-foreground">{t("form.fields.notify_domain.description")}</p>
                </div>
                <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.fields.handling_fee_percent.label")}</div>
                        <Input type="number" value={form.handling_fee_percent} placeholder={t("form.fields.handling_fee_percent.placeholder")} onChange={(e) => setForm((p) => ({ ...p, handling_fee_percent: e.target.value }))} />
                    </div>
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.fields.handling_fee_fixed.label")}</div>
                        <Input type="number" value={form.handling_fee_fixed} placeholder={t("form.fields.handling_fee_fixed.placeholder")} onChange={(e) => setForm((p) => ({ ...p, handling_fee_fixed: e.target.value }))} />
                    </div>
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.fields.payment.label")}</div>
                    <Select value={form.payment} onValueChange={(v) => setForm((p) => ({ ...p, payment: v }))}>
                        <SelectTrigger><SelectValue placeholder={t("form.fields.payment.placeholder")} /></SelectTrigger>
                        <SelectContent>
                            {methods.map((m) => <SelectItem key={m} value={m}>{m}</SelectItem>)}
                        </SelectContent>
                    </Select>
                    <p className="text-xs text-muted-foreground">{t("form.fields.payment.description")}</p>
                </div>
                {Object.keys(configSchema).length > 0 && (
                    <div className="space-y-4 border-t pt-4">
                        <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{t("form.sections.payment_config")}</div>
                        <DynamicConfigFields schema={configSchema} values={configValues} onChange={setConfigValues} />
                    </div>
                )}
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
