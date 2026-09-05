import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { fetchCoupons, generateCoupon, dropCoupon, updateCoupon } from "../../api/coupon.js";
import { fetchPlans } from "../../api/plan.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { Tooltip, TooltipContent, TooltipTrigger } from "../../components/Tooltip.jsx";
import { classNames } from "../../utils/classNames.js";

const TYPE_AMOUNT = 1;
const TYPE_PERCENTAGE = 2;

const PERIODS = ["monthly", "quarterly", "half_yearly", "yearly", "two_yearly", "three_yearly", "onetime", "reset_traffic"];

const PRESETS = [
    { key: "1week", days: 7 },
    { key: "2weeks", days: 14 },
    { key: "1month", days: 30 },
    { key: "3months", days: 90 },
    { key: "6months", days: 180 },
    { key: "1year", days: 365 },
];

const nowSeconds = () => Math.floor(Date.now() / 1000);

const EMPTY_FORM = {
    id: null,
    name: "",
    code: "",
    generate_count: "",
    type: TYPE_AMOUNT,
    value: "",
    started_at: nowSeconds(),
    ended_at: nowSeconds() + 604800,
    limit_use: "",
    limit_use_with_user: "",
    limit_plan_ids: [],
    limit_period: [],
};

function toDatetimeLocal(seconds) {
    if (!seconds) return "";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function fromDatetimeLocal(value) {
    return Math.floor(new Date(value).getTime() / 1000);
}

function formatDateTime(seconds, withYear = false) {
    if (!seconds) return "-";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return withYear
        ? `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
        : `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function MultiSelectField({ options, value, onChange, placeholder, emptyText }) {
    const [open, setOpen] = useState(false);
    const rootRef = useRef(null);

    useEffect(() => {
        if (!open) return;
        const handleDown = (e) => {
            if (!rootRef.current?.contains(e.target)) setOpen(false);
        };
        document.addEventListener("pointerdown", handleDown);
        return () => document.removeEventListener("pointerdown", handleDown);
    }, [open]);

    const selected = options.filter((o) => value.includes(o.value));
    const toggle = (v) => {
        onChange(value.includes(v) ? value.filter((x) => x !== v) : [...value, v]);
    };

    return (
        <div ref={rootRef} className="relative">
            <button
                type="button"
                className="flex min-h-9 w-full flex-wrap items-center gap-1 rounded-md border border-input bg-background px-2 py-1.5 text-left text-sm shadow-sm"
                onClick={() => setOpen((v) => !v)}
            >
                {selected.length === 0 ? (
                    <span className="text-muted-foreground">{placeholder}</span>
                ) : (
                    selected.map((o) => (
                        <Badge key={o.value} variant="secondary" className="gap-1">
                            {o.label}
                            <span role="button" tabIndex={-1} onClick={(e) => { e.stopPropagation(); toggle(o.value); }}>
                                ×
                            </span>
                        </Badge>
                    ))
                )}
            </button>
            {open && (
                <div className="absolute left-0 top-10 z-50 max-h-56 w-full overflow-auto rounded-md border bg-popover p-1 text-popover-foreground shadow-md">
                    {options.length === 0 ? (
                        <div className="px-2 py-1.5 text-sm text-muted-foreground">{emptyText}</div>
                    ) : (
                        options.map((o) => {
                            const isSelected = value.includes(o.value);
                            return (
                                <button
                                    type="button"
                                    key={o.value}
                                    className={classNames(
                                        "flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent",
                                        isSelected && "bg-secondary",
                                    )}
                                    onClick={() => toggle(o.value)}
                                >
                                    {o.label}
                                    {isSelected && " ✓"}
                                </button>
                            );
                        })
                    )}
                </div>
            )}
        </div>
    );
}

function ValidityBadge({ item, t }) {
    const now = Date.now();
    const startMs = item.started_at * 1000;
    const endMs = item.ended_at * 1000;
    const expired = now > endMs;
    const notStarted = now < startMs;
    const daysLeft = Math.ceil((endMs - now) / 86400000);

    let label;
    let color;
    if (expired) {
        label = t("table.validity.expired", { days: Math.abs(daysLeft) });
        color = "bg-red-50 text-red-600";
    } else if (notStarted) {
        label = t("table.validity.notStarted", { days: Math.abs(Math.ceil((startMs - now) / 86400000)) });
        color = "bg-yellow-50 text-yellow-600";
    } else {
        label = t("table.validity.remaining", { days: daysLeft });
        color = "bg-green-50 text-green-600";
    }

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <div className="flex cursor-default flex-col gap-0.5">
                    <span className={classNames("w-fit whitespace-nowrap rounded-md px-1.5 py-0.5 text-xs font-medium", color)}>{label}</span>
                    <span className="hidden text-xs text-muted-foreground md:inline">
                        {formatDateTime(item.started_at)} → {formatDateTime(item.ended_at)}
                    </span>
                </div>
            </TooltipTrigger>
            <TooltipContent>
                <div className="space-y-1 text-xs">
                    <div className="flex items-center justify-between gap-4">
                        <span className="text-muted-foreground">{t("table.validity.startTime")}</span>
                        <span className="font-medium">{formatDateTime(item.started_at, true)}</span>
                    </div>
                    <div className="flex items-center justify-between gap-4">
                        <span className="text-muted-foreground">{t("table.validity.endTime")}</span>
                        <span className="font-medium">{formatDateTime(item.ended_at, true)}</span>
                    </div>
                </div>
            </TooltipContent>
        </Tooltip>
    );
}

export function CouponManagement() {
    const { t } = useTranslation("coupon");
    const auth = useAuthState();
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");
    const [typeFilter, setTypeFilter] = useState("all");
    const [plans, setPlans] = useState([]);

    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState(EMPTY_FORM);
    const [formError, setFormError] = useState("");

    const [deleteTarget, setDeleteTarget] = useState(null);

    const load = () => {
        setLoading(true);
        return fetchCoupons({})
            .then((res) => setItems(res?.data || []))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
    }, [auth.token]);

    useEffect(() => {
        fetchPlans()
            .then((res) => setPlans(res?.data || []))
            .catch(() => setPlans([]));
    }, [auth.token]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        return items.filter((c) => {
            if (keyword && !(c.name || "").toLowerCase().includes(keyword)) return false;
            if (typeFilter !== "all" && String(c.type) !== typeFilter) return false;
            return true;
        });
    }, [items, search, typeFilter]);

    const planOptions = useMemo(() => plans.map((p) => ({ value: p.id, label: p.name })), [plans]);
    const periodOptions = useMemo(() => PERIODS.map((p) => ({ value: p, label: t(`period.${p}`) })), [t]);

    const openAdd = () => {
        setForm({ ...EMPTY_FORM, started_at: nowSeconds(), ended_at: nowSeconds() + 604800 });
        setFormError("");
        setFormOpen(true);
    };

    const openEdit = (item) => {
        setForm({
            id: item.id,
            name: item.name || "",
            code: item.code || "",
            generate_count: "",
            type: item.type,
            value: item.type === TYPE_AMOUNT ? String((item.value || 0) / 100) : String(item.value ?? ""),
            started_at: item.started_at,
            ended_at: item.ended_at,
            limit_use: item.limit_use === null || item.limit_use === undefined ? "" : String(item.limit_use),
            limit_use_with_user: item.limit_use_with_user === null || item.limit_use_with_user === undefined ? "" : String(item.limit_use_with_user),
            limit_plan_ids: item.limit_plan_ids || [],
            limit_period: item.limit_period || [],
        });
        setFormError("");
        setFormOpen(true);
    };

    const applyPreset = (days) => {
        const start = nowSeconds();
        setForm((p) => ({ ...p, started_at: start, ended_at: start + days * 86400 }));
    };

    const handleTypeChange = (value) => {
        const nextType = parseInt(value, 10);
        setForm((p) => {
            const current = parseFloat(p.value);
            if (!p.value || isNaN(current)) return { ...p, type: nextType };
            if (p.type === TYPE_AMOUNT && nextType === TYPE_PERCENTAGE) {
                return { ...p, type: nextType, value: String(current / 100) };
            }
            if (p.type === TYPE_PERCENTAGE && nextType === TYPE_AMOUNT) {
                return { ...p, type: nextType, value: String(current * 100) };
            }
            return { ...p, type: nextType };
        });
    };

    const handleSubmit = async () => {
        setFormError("");
        if (!form.name.trim()) {
            const error = t("form.name.required");
            setFormError(error);
            throw new Error(error);
        }
        if (form.ended_at <= form.started_at) {
            const error = t("form.validity.endTimeError");
            setFormError(error);
            throw new Error(error);
        }

        const rawValue = parseFloat(form.value || 0) || 0;
        const payload = {
            id: form.id ?? undefined,
            name: form.name.trim(),
            code: form.code.trim() || null,
            type: form.type,
            value: form.type === TYPE_AMOUNT ? Math.round(rawValue * 100) : rawValue,
            started_at: form.started_at,
            ended_at: form.ended_at,
            limit_use: form.limit_use === "" ? null : parseInt(form.limit_use, 10),
            limit_use_with_user: form.limit_use_with_user === "" ? null : parseInt(form.limit_use_with_user, 10),
            limit_plan_ids: form.limit_plan_ids,
            limit_period: form.limit_period,
            generate_count: form.id ? null : (form.generate_count === "" ? null : parseInt(form.generate_count, 10)),
        };

        const res = await generateCoupon(payload);
        if (payload.generate_count && typeof res === "string") {
            const blob = new Blob([res], { type: "text/csv;charset=utf-8;" });
            const link = document.createElement("a");
            link.href = window.URL.createObjectURL(blob);
            link.download = `coupons_${Date.now()}.csv`;
            link.click();
            window.URL.revokeObjectURL(link.href);
        }
        await load();
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await dropCoupon({ id: deleteTarget.id });
        await load();
    };

    const handleToggleShow = async (item, checked) => {
        setItems((prev) => prev.map((c) => (c.id === item.id ? { ...c, show: checked } : c)));
        await updateCoupon({ id: item.id, show: checked });
    };

    const showCodeField = !form.generate_count;

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
                onReset={() => { setSearch(""); setTypeFilter("all"); }}
                filters={
                    <Select value={typeFilter} onValueChange={setTypeFilter} className="sm:w-40">
                        <SelectTrigger><SelectValue placeholder={t("table.toolbar.type")} /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">{t("table.toolbar.type")}</SelectItem>
                            <SelectItem value={String(TYPE_AMOUNT)}>{t("table.toolbar.types.1")}</SelectItem>
                            <SelectItem value={String(TYPE_PERCENTAGE)}>{t("table.toolbar.types.2")}</SelectItem>
                        </SelectContent>
                    </Select>
                }
                actions={<Button onClick={openAdd}>{t("form.add")}</Button>}
            />

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-16">{t("table.columns.id")}</TableHead>
                            <TableHead className="w-16">{t("table.columns.show")}</TableHead>
                            <TableHead>{t("table.columns.name")}</TableHead>
                            <TableHead>{t("table.columns.type")}</TableHead>
                            <TableHead>{t("table.columns.code")}</TableHead>
                            <TableHead>{t("table.columns.limitUse")}</TableHead>
                            <TableHead>{t("table.columns.limitUseWithUser")}</TableHead>
                            <TableHead>{t("table.columns.validity")}</TableHead>
                            <TableHead className="w-32 text-right">{t("table.columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={9}>...</TableEmpty>
                        ) : filtered.length === 0 ? (
                            <TableEmpty colSpan={9}>-</TableEmpty>
                        ) : (
                            filtered.map((item) => (
                                <TableRow key={item.id}>
                                    <TableCell><Badge variant="outline" className="font-mono">{item.id}</Badge></TableCell>
                                    <TableCell>
                                        <Switch checked={Boolean(item.show)} onCheckedChange={(checked) => handleToggleShow(item, checked)} />
                                    </TableCell>
                                    <TableCell className="font-medium">{item.name}</TableCell>
                                    <TableCell><Badge variant="outline">{t(`table.toolbar.types.${item.type}`)}</Badge></TableCell>
                                    <TableCell><Badge variant="secondary" className="font-mono">{item.code}</Badge></TableCell>
                                    <TableCell>{item.limit_use === null || item.limit_use === undefined ? t("table.validity.unlimited") : item.limit_use}</TableCell>
                                    <TableCell>{item.limit_use_with_user === null || item.limit_use_with_user === undefined ? t("table.validity.noLimit") : item.limit_use_with_user}</TableCell>
                                    <TableCell><ValidityBadge item={item} t={t} /></TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button variant="outline" size="sm" onClick={() => openEdit(item)}>
                                                {t("table.actions.edit")}
                                            </Button>
                                            <Button variant="destructive" size="sm" onClick={() => setDeleteTarget(item)}>
                                                {t("table.actions.delete")}
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
                submitLabel={t("form.submit.save")}
                onSubmit={handleSubmit}
                className="sm:max-w-2xl"
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}

                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.name.label")}</div>
                    <Input value={form.name} placeholder={t("form.name.placeholder")} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} required />
                </div>

                {!form.id && (
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.generateCount.label")}</div>
                        <Input type="number" min={0} value={form.generate_count} placeholder={t("form.generateCount.placeholder")} onChange={(e) => setForm((p) => ({ ...p, generate_count: e.target.value }))} />
                        <p className="text-xs text-muted-foreground">{t("form.generateCount.description")}</p>
                    </div>
                )}

                {showCodeField && (
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.code.label")}</div>
                        <Input value={form.code} placeholder={t("form.code.placeholder")} onChange={(e) => setForm((p) => ({ ...p, code: e.target.value }))} />
                        <p className="text-xs text-muted-foreground">{t("form.code.description")}</p>
                    </div>
                )}

                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.type.label")}</div>
                    <div className="flex gap-2">
                        <Select value={String(form.type)} onValueChange={handleTypeChange} className="w-40">
                            <SelectTrigger><SelectValue placeholder={t("form.type.placeholder")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value={String(TYPE_AMOUNT)}>{t("table.toolbar.types.1")}</SelectItem>
                                <SelectItem value={String(TYPE_PERCENTAGE)}>{t("table.toolbar.types.2")}</SelectItem>
                            </SelectContent>
                        </Select>
                        <Input type="number" step="any" min={0} value={form.value} placeholder={t("form.value.placeholder")} onChange={(e) => setForm((p) => ({ ...p, value: e.target.value }))} className="flex-1" />
                    </div>
                </div>

                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.validity.label")}</div>
                    <div className="flex flex-wrap gap-2">
                        {PRESETS.map((preset) => (
                            <Button key={preset.key} type="button" variant="secondary" size="sm" onClick={() => applyPreset(preset.days)}>
                                {t(`form.timeRange.presets.${preset.key}`)}
                            </Button>
                        ))}
                    </div>
                    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                        <Input type="datetime-local" step="1" value={toDatetimeLocal(form.started_at)} onChange={(e) => setForm((p) => ({ ...p, started_at: fromDatetimeLocal(e.target.value) }))} className="flex-1" />
                        <span className="text-xs text-muted-foreground">{t("form.validity.to")}</span>
                        <Input type="datetime-local" step="1" value={toDatetimeLocal(form.ended_at)} onChange={(e) => setForm((p) => ({ ...p, ended_at: fromDatetimeLocal(e.target.value) }))} className="flex-1" />
                    </div>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.limitUse.label")}</div>
                        <Input type="number" min={0} value={form.limit_use} placeholder={t("form.limitUse.placeholder")} onChange={(e) => setForm((p) => ({ ...p, limit_use: e.target.value }))} />
                        <p className="text-xs text-muted-foreground">{t("form.limitUse.description")}</p>
                    </div>
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("form.limitUseWithUser.label")}</div>
                        <Input type="number" min={0} value={form.limit_use_with_user} placeholder={t("form.limitUseWithUser.placeholder")} onChange={(e) => setForm((p) => ({ ...p, limit_use_with_user: e.target.value }))} />
                        <p className="text-xs text-muted-foreground">{t("form.limitUseWithUser.description")}</p>
                    </div>
                </div>

                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.limitPeriod.label")}</div>
                    <MultiSelectField
                        options={periodOptions}
                        value={form.limit_period}
                        onChange={(value) => setForm((p) => ({ ...p, limit_period: value }))}
                        placeholder={t("form.limitPeriod.placeholder")}
                        emptyText={t("form.limitPeriod.empty")}
                    />
                    <p className="text-xs text-muted-foreground">{t("form.limitPeriod.description")}</p>
                </div>

                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("form.limitPlan.label")}</div>
                    <MultiSelectField
                        options={planOptions}
                        value={form.limit_plan_ids}
                        onChange={(value) => setForm((p) => ({ ...p, limit_plan_ids: value }))}
                        placeholder={t("form.limitPlan.placeholder")}
                        emptyText={t("form.limitPlan.empty")}
                    />
                    <p className="text-xs text-muted-foreground">{t("form.limitPlan.description")}</p>
                </div>
            </FormDialog>

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("table.actions.deleteConfirm.title")}
                description={t("table.actions.deleteConfirm.description")}
                confirmLabel={t("table.actions.deleteConfirm.confirmText")}
                onConfirm={handleDelete}
            />
        </div>
    );
}
