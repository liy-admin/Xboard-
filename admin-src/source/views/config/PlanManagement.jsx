import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { fetchPlans, savePlan, updatePlan, dropPlan, sortPlans } from "../../api/plan.js";
import { fetchGroups, saveGroup } from "../../api/server.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useClientSort } from "../../components/SortableHead.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { classNames } from "../../utils/classNames.js";

const PERIOD_META = {
    weekly: { months: 0.25, discount: 1.2, unit: "week" },
    monthly: { months: 1, discount: 1, unit: "month" },
    quarterly: { months: 3, discount: 0.95, unit: "quarter" },
    half_yearly: { months: 6, discount: 0.9, unit: "half_year" },
    yearly: { months: 12, discount: 0.85, unit: "year" },
    two_yearly: { months: 24, discount: 0.8, unit: "two_year" },
    three_yearly: { months: 36, discount: 0.75, unit: "three_year" },
    onetime: { months: 1, discount: 1, unit: null },
    reset_traffic: { months: 1, discount: 1, unit: "times" },
};
const PERIOD_KEYS = Object.keys(PERIOD_META);
const RECURRING_PERIODS = PERIOD_KEYS.filter((key) => key !== "onetime" && key !== "reset_traffic");

const RESET_METHOD_OPTIONS = [
    { value: "", label: "follow_system" },
    { value: "0", label: "monthly_first" },
    { value: "1", label: "monthly_reset" },
    { value: "2", label: "no_reset" },
    { value: "3", label: "yearly_first" },
    { value: "4", label: "yearly_reset" },
];

const EMPTY_PRICES = PERIOD_KEYS.reduce((acc, key) => ({ ...acc, [key]: "" }), {});

const EMPTY_FORM = {
    id: null,
    group_id: null,
    name: "",
    tags: [],
    content: "",
    transfer_enable: "",
    prices: { ...EMPTY_PRICES },
    speed_limit: "",
    capacity_limit: "",
    device_limit: "",
    force_update: false,
    reset_traffic_method: null,
};

const formatNumber = (n) => Number(n).toFixed(0).replace(/\B(?=(\d{3})+(?!\d))/g, ",");

function TagInput({ value, onChange, placeholder }) {
    const [draft, setDraft] = useState("");

    const commit = () => {
        const tag = draft.trim();
        if (tag && !value.includes(tag)) {
            onChange([...value, tag]);
        }
        setDraft("");
    };

    return (
        <div className="flex min-h-9 w-full flex-wrap items-center gap-1 rounded-md border border-input bg-background px-2 py-1.5 text-sm shadow-sm">
            {value.map((tag) => (
                <Badge key={tag} variant="secondary" className="gap-1">
                    {tag}
                    <span role="button" tabIndex={-1} onClick={() => onChange(value.filter((t) => t !== tag))}>
                        ×
                    </span>
                </Badge>
            ))}
            <input
                className="min-w-[8rem] flex-1 border-0 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
                value={draft}
                placeholder={value.length === 0 ? placeholder : ""}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => {
                    if (e.key === "Enter") {
                        e.preventDefault();
                        commit();
                    } else if (e.key === "Backspace" && !draft && value.length > 0) {
                        onChange(value.slice(0, -1));
                    }
                }}
                onBlur={commit}
            />
        </div>
    );
}

export function PlanManagement() {
    const { t } = useTranslation("subscribe");
    const auth = useAuthState();
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");
    const [groups, setGroups] = useState([]);

    const [sortMode, setSortMode] = useState(false);
    const [orderedItems, setOrderedItems] = useState([]);
    const [sorting, setSorting] = useState(false);
    const dragIndexRef = useRef(null);

    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState(EMPTY_FORM);
    const [formError, setFormError] = useState("");
    const [contentPreview, setContentPreview] = useState(false);

    const [groupDialogOpen, setGroupDialogOpen] = useState(false);
    const [groupDialogName, setGroupDialogName] = useState("");
    const [groupDialogError, setGroupDialogError] = useState("");

    const [deleteTarget, setDeleteTarget] = useState(null);

    const loadGroups = () => fetchGroups().then((res) => setGroups(res?.data || []));

    const load = () => {
        setLoading(true);
        return fetchPlans()
            .then((res) => setItems(res?.data || []))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
        loadGroups();
    }, [auth.token]);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        if (!keyword) return items;
        return items.filter((p) => (p.name || "").toLowerCase().includes(keyword));
    }, [items, search]);

    // 价格列按最低的已定价周期价格排序。
    const minPrice = (p) => {
        const vals = Object.values(p.prices || {}).filter((v) => v !== null && v !== undefined && v !== "").map(Number).filter(Number.isFinite);
        return vals.length ? Math.min(...vals) : null;
    };
    const { sorted, sort, toggle: toggleSort } = useClientSort(filtered, {
        id: (i) => Number(i.id),
        show: (i) => (i.show ? 1 : 0),
        sell: (i) => (i.sell ? 1 : 0),
        renew: (i) => (i.renew ? 1 : 0),
        name: (i) => i.name || "",
        stats: (i) => Number(i.users_count || 0),
        group: (i) => i.group?.name || null,
        price: (i) => minPrice(i),
    });

    const enterSortMode = () => {
        setOrderedItems(items);
        setSortMode(true);
    };

    const saveOrder = async () => {
        setSorting(true);
        try {
            await sortPlans(orderedItems.map((p) => p.id));
            setSortMode(false);
            await load();
        } finally {
            setSorting(false);
        }
    };

    const openAdd = () => {
        setForm(EMPTY_FORM);
        setContentPreview(false);
        setFormError("");
        setFormOpen(true);
    };

    const openEdit = (item) => {
        setForm({
            id: item.id,
            group_id: item.group_id ?? item.group?.id ?? null,
            name: item.name || "",
            tags: Array.isArray(item.tags) ? item.tags : [],
            content: item.content || "",
            transfer_enable: item.transfer_enable ?? "",
            prices: { ...EMPTY_PRICES, ...(item.prices || {}) },
            speed_limit: item.speed_limit ?? "",
            capacity_limit: item.capacity_limit ?? "",
            device_limit: item.device_limit ?? "",
            force_update: false,
            reset_traffic_method: item.reset_traffic_method ?? null,
        });
        setContentPreview(false);
        setFormError("");
        setFormOpen(true);
    };

    const applyBasePrice = (base) => {
        if (isNaN(base)) return;
        setForm((p) => ({
            ...p,
            prices: PERIOD_KEYS.reduce((acc, key) => {
                const meta = PERIOD_META[key];
                if (key === "onetime" || key === "reset_traffic") {
                    return { ...acc, [key]: p.prices[key] };
                }
                return { ...acc, [key]: (base * meta.months * meta.discount).toFixed(2) };
            }, {}),
        }));
    };

    const clearPrices = () => {
        setForm((p) => ({ ...p, prices: { ...EMPTY_PRICES } }));
    };

    const handleSubmit = async () => {
        setFormError("");
        if (!form.name.trim()) {
            const error = `${t("plan.form.name.label")} ${t("plan.form.name.placeholder")}`;
            setFormError(error);
            throw new Error(error);
        }
        if (form.transfer_enable === "" || Number(form.transfer_enable) < 0) {
            const error = `${t("plan.form.transfer.label")} ${t("plan.form.transfer.placeholder")}`;
            setFormError(error);
            throw new Error(error);
        }

        const prices = PERIOD_KEYS.reduce((acc, key) => {
            const raw = form.prices[key];
            acc[key] = raw === "" || raw === null || raw === undefined ? null : Number(raw);
            return acc;
        }, {});

        const payload = {
            id: form.id ?? undefined,
            group_id: form.group_id === null || form.group_id === "" ? null : Number(form.group_id),
            name: form.name.trim(),
            tags: form.tags,
            content: form.content,
            transfer_enable: Number(form.transfer_enable),
            prices,
            speed_limit: form.speed_limit === "" ? null : Number(form.speed_limit),
            capacity_limit: form.capacity_limit === "" ? null : Number(form.capacity_limit),
            device_limit: form.device_limit === "" ? null : Number(form.device_limit),
            reset_traffic_method: form.reset_traffic_method === null || form.reset_traffic_method === "" ? null : Number(form.reset_traffic_method),
            force_update: form.id ? form.force_update : undefined,
        };

        await savePlan(payload);
        await load();
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await dropPlan({ id: deleteTarget.id });
        await load();
    };

    const handleToggle = async (item, field, checked) => {
        setItems((prev) => prev.map((p) => (p.id === item.id ? { ...p, [field]: checked } : p)));
        await updatePlan({ id: item.id, [field]: checked });
    };

    const handleGroupDialogSubmit = async () => {
        setGroupDialogError("");
        const name = groupDialogName.trim();
        if (name.length < 2) {
            const error = t("group:messages.nameValidation.min");
            setGroupDialogError(error);
            throw new Error(error);
        }
        await saveGroup({ name });
        await loadGroups();
        setGroupDialogName("");
    };

    const displayItems = sortMode ? orderedItems : sorted;

    return (
        <div className="p-4 md:p-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-lg font-medium">{t("plan.title")}</h3>
                <p className="text-sm text-muted-foreground">{t("plan.page.description")}</p>
            </div>
            <hr className="my-6" />

            <SearchFilterToolbar
                search={search}
                onSearchChange={setSearch}
                searchPlaceholder={t("plan.search")}
                resetLabel="重置"
                onReset={() => setSearch("")}
                actions={
                    <div className="flex items-center gap-2">
                        <Button onClick={openAdd}>{t("plan.add")}</Button>
                        {items.length > 0 && (
                            <Button variant={sortMode ? "default" : "outline"} onClick={() => (sortMode ? saveOrder() : enterSortMode())} loading={sorting}>
                                {sortMode ? t("plan.sort.save") : t("plan.sort.edit")}
                            </Button>
                        )}
                    </div>
                }
            />

            {items.some((p) => (p.group_id ?? p.group?.id) == null) && (
                <div className="mt-4 rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-400">
                    {t("group:form.missing_banner", {
                        names: items.filter((p) => (p.group_id ?? p.group?.id) == null).map((p) => p.name).join("、"),
                    })}
                </div>
            )}

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            {sortMode && <TableHead className="w-10"></TableHead>}
                            <SortableHead className="w-16" sortKey="id" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("plan.columns.id")}</SortableHead>
                            <SortableHead className="w-16" sortKey="show" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("plan.columns.show")}</SortableHead>
                            <SortableHead className="w-16" sortKey="sell" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("plan.columns.sell")}</SortableHead>
                            <SortableHead className="w-16" title={t("plan.columns.renew_tooltip")} sortKey="renew" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("plan.columns.renew")}</SortableHead>
                            <SortableHead sortKey="name" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("plan.columns.name")}</SortableHead>
                            <SortableHead sortKey="stats" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("plan.columns.stats")}</SortableHead>
                            <SortableHead sortKey="group" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("plan.columns.group")}</SortableHead>
                            <SortableHead sortKey="price" sort={sort} onSort={toggleSort} disabled={sortMode}>{t("plan.columns.price")}</SortableHead>
                            <TableHead className="w-32 text-right">{t("plan.columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={9}>...</TableEmpty>
                        ) : displayItems.length === 0 ? (
                            <TableEmpty colSpan={9}>-</TableEmpty>
                        ) : (
                            displayItems.map((item, index) => (
                                <TableRow
                                    key={item.id}
                                    draggable={sortMode}
                                    onDragStart={(e) => {
                                        dragIndexRef.current = index;
                                        e.dataTransfer.setData("text/plain", String(index));
                                        e.currentTarget.classList.add("opacity-50");
                                    }}
                                    onDragEnd={(e) => e.currentTarget.classList.remove("opacity-50")}
                                    onDragOver={(e) => sortMode && e.preventDefault()}
                                    onDrop={(e) => {
                                        if (!sortMode) return;
                                        e.preventDefault();
                                        const from = dragIndexRef.current;
                                        if (from === null || from === index) return;
                                        setOrderedItems((prev) => {
                                            const next = [...prev];
                                            const [moved] = next.splice(from, 1);
                                            next.splice(index, 0, moved);
                                            return next;
                                        });
                                        dragIndexRef.current = null;
                                    }}
                                >
                                    {sortMode && (
                                        <TableCell className="cursor-move text-muted-foreground">⠿</TableCell>
                                    )}
                                    <TableCell><Badge variant="outline" className="font-mono">{item.id}</Badge></TableCell>
                                    <TableCell><Switch checked={Boolean(item.show)} onCheckedChange={(checked) => handleToggle(item, "show", checked)} /></TableCell>
                                    <TableCell><Switch checked={Boolean(item.sell)} onCheckedChange={(checked) => handleToggle(item, "sell", checked)} /></TableCell>
                                    <TableCell><Switch checked={Boolean(item.renew)} onCheckedChange={(checked) => handleToggle(item, "renew", checked)} /></TableCell>
                                    <TableCell className="font-medium">{item.name}</TableCell>
                                    <TableCell>
                                        <div className="flex items-center gap-2 text-xs">
                                            <span className="rounded-md bg-slate-100 px-2 py-1 font-medium text-slate-700">{item.users_count || 0}</span>
                                            <span className="rounded-md bg-green-50 px-2 py-1 font-medium text-green-700">{item.active_users_count || 0}</span>
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        {item.group?.name ? (
                                            <Badge variant="secondary">{item.group.name}</Badge>
                                        ) : (item.group_id ?? item.group?.id) == null ? (
                                            <Badge variant="outline" className="border-red-300 text-red-600 dark:border-red-800 dark:text-red-400">{t("group:form.missing_badge")}</Badge>
                                        ) : (
                                            <span className="text-xs text-muted-foreground">-</span>
                                        )}
                                    </TableCell>
                                    <TableCell>
                                        {item.prices ? (
                                            <div className="flex flex-wrap items-center gap-1.5">
                                                {PERIOD_KEYS.map((key) => {
                                                    const price = item.prices[key];
                                                    if (price === null || price === undefined) return null;
                                                    const unit = PERIOD_META[key].unit;
                                                    return (
                                                        <Badge key={key} variant="secondary" className="whitespace-nowrap text-xs">
                                                            {t(`plan.columns.price_period.${key}`)} {formatNumber(price)}{unit ? t(`plan.columns.price_period.unit.${unit}`) : "元"}
                                                        </Badge>
                                                    );
                                                })}
                                            </div>
                                        ) : (
                                            <Badge variant="outline" className="border-dashed text-xs text-muted-foreground">{t("plan.columns.price_period.no_price")}</Badge>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button variant="outline" size="sm" onClick={() => openEdit(item)}>
                                                {t("plan.columns.edit")}
                                            </Button>
                                            <Button variant="destructive" size="sm" onClick={() => setDeleteTarget(item)}>
                                                {t("plan.columns.delete")}
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
                title={form.id ? t("plan.form.edit_title") : t("plan.form.add_title")}
                cancelLabel={t("plan.form.submit.cancel")}
                submitLabel={t("plan.form.submit.submit")}
                onSubmit={handleSubmit}
                className="sm:max-w-3xl"
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("plan.form.name.label")}</div>
                        <Input value={form.name} placeholder={t("plan.form.name.placeholder")} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} required />
                    </div>
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("plan.form.tags.label")}</div>
                        <TagInput value={form.tags} onChange={(tags) => setForm((p) => ({ ...p, tags }))} placeholder={t("plan.form.tags.placeholder")} />
                    </div>
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div className="space-y-1.5">
                        <div className="flex items-center justify-between text-sm font-medium">
                            <span>{t("plan.form.group.label")}</span>
                            <button type="button" className="text-xs text-primary underline-offset-2 hover:underline" onClick={() => setGroupDialogOpen(true)}>
                                {t("plan.form.group.add")}
                            </button>
                        </div>
                        <Select value={form.group_id != null ? String(form.group_id) : ""} onValueChange={(v) => setForm((p) => ({ ...p, group_id: v ? Number(v) : null }))}>
                            <SelectTrigger><SelectValue placeholder={t("plan.form.group.placeholder")} /></SelectTrigger>
                            <SelectContent>
                                {groups.map((g) => <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>)}
                            </SelectContent>
                        </Select>
                        {form.group_id == null && (
                            <p className="text-xs text-red-600 dark:text-red-400">{t("group:form.missing_form")}</p>
                        )}
                    </div>
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("plan.form.transfer.label")} ({t("plan.form.transfer.unit")})</div>
                        <Input type="number" min={0} value={form.transfer_enable} placeholder={t("plan.form.transfer.placeholder")} onChange={(e) => setForm((p) => ({ ...p, transfer_enable: e.target.value }))} required />
                    </div>
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("plan.form.speed.label")} ({t("plan.form.speed.unit")})</div>
                        <Input type="number" min={0} value={form.speed_limit} placeholder={t("plan.form.speed.placeholder")} onChange={(e) => setForm((p) => ({ ...p, speed_limit: e.target.value }))} />
                    </div>
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("plan.form.device.label")} ({t("plan.form.device.unit")})</div>
                        <Input type="number" min={0} value={form.device_limit} placeholder={t("plan.form.device.placeholder")} onChange={(e) => setForm((p) => ({ ...p, device_limit: e.target.value }))} />
                    </div>
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("plan.form.capacity.label")} ({t("plan.form.capacity.unit")})</div>
                        <Input type="number" min={0} value={form.capacity_limit} placeholder={t("plan.form.capacity.placeholder")} onChange={(e) => setForm((p) => ({ ...p, capacity_limit: e.target.value }))} />
                    </div>
                    <div className="space-y-1.5">
                        <div className="text-sm font-medium">{t("plan.form.reset_method.label")}</div>
                        <Select value={form.reset_traffic_method != null ? String(form.reset_traffic_method) : ""} onValueChange={(v) => setForm((p) => ({ ...p, reset_traffic_method: v === "" ? null : Number(v) }))}>
                            <SelectTrigger><SelectValue placeholder={t("plan.form.reset_method.placeholder")} /></SelectTrigger>
                            <SelectContent>
                                {RESET_METHOD_OPTIONS.map((opt) => <SelectItem key={opt.label} value={opt.value}>{t(`plan.form.reset_method.options.${opt.label}`)}</SelectItem>)}
                            </SelectContent>
                        </Select>
                        <p className="text-xs text-muted-foreground">{t("plan.form.reset_method.description")}</p>
                    </div>
                </div>

                {form.id && (
                    <div className="flex items-center gap-2">
                        <Switch checked={form.force_update} onCheckedChange={(checked) => setForm((p) => ({ ...p, force_update: checked }))} />
                        <span className="cursor-pointer select-none text-xs" onClick={() => setForm((p) => ({ ...p, force_update: !p.force_update }))}>
                            {t("plan.form.force_update.label")}
                        </span>
                    </div>
                )}

                <div className="space-y-4 rounded-lg border border-dashed p-4">
                    <div className="flex items-center justify-between">
                        <h3 className="text-sm font-medium">{t("plan.form.price.title")}</h3>
                        <div className="flex items-center gap-2">
                            <Input
                                type="number"
                                step="0.01"
                                placeholder={t("plan.form.price.base_price")}
                                className="h-8 w-36 text-xs"
                                onChange={(e) => applyBasePrice(parseFloat(e.target.value))}
                            />
                            <Button variant="outline" size="sm" type="button" title={t("plan.form.price.clear.tooltip")} onClick={clearPrices}>
                                {t("plan.form.price.clear.button")}
                            </Button>
                        </div>
                    </div>
                    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
                        {RECURRING_PERIODS.map((key) => (
                            <div key={key} className="space-y-1">
                                <div className="text-xs font-medium text-muted-foreground">{t(`plan.columns.price_period.${key}`)}</div>
                                <Input type="number" step="0.01" value={form.prices[key]} onChange={(e) => setForm((p) => ({ ...p, prices: { ...p.prices, [key]: e.target.value } }))} className="h-8 text-xs" />
                            </div>
                        ))}
                    </div>
                    <hr />
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        {["onetime", "reset_traffic"].map((key) => (
                            <div key={key} className="space-y-1">
                                <div className="text-xs font-medium text-muted-foreground">{t(`plan.columns.price_period.${key}`)}</div>
                                <Input type="number" step="0.01" value={form.prices[key]} onChange={(e) => setForm((p) => ({ ...p, prices: { ...p.prices, [key]: e.target.value } }))} className="h-8 text-xs" />
                                <p className="text-xs text-muted-foreground">{t(key === "onetime" ? "plan.form.price.onetime_desc" : "plan.form.price.reset_desc")}</p>
                            </div>
                        ))}
                    </div>
                </div>

                <div className="space-y-1.5">
                    <div className="flex items-center justify-between">
                        <div className="text-sm font-medium">{t("plan.form.content.label")}</div>
                        <div className="flex items-center gap-2">
                            <Button variant="outline" size="sm" type="button" onClick={() => setForm((p) => ({ ...p, content: t("plan.form.content.template.content") }))}>
                                {t("plan.form.content.template.button")}
                            </Button>
                            <Button variant="outline" size="sm" type="button" onClick={() => setContentPreview((v) => !v)}>
                                {contentPreview ? t("plan.form.content.preview_button.hide") : t("plan.form.content.preview_button.show")}
                            </Button>
                        </div>
                    </div>
                    <div className={classNames("grid gap-4", contentPreview ? "grid-cols-1 lg:grid-cols-2" : "grid-cols-1")}>
                        <Textarea className="min-h-[200px] font-mono text-xs" value={form.content} onChange={(e) => setForm((p) => ({ ...p, content: e.target.value }))} />
                        {contentPreview && (
                            <div className="min-h-[200px] overflow-y-auto whitespace-pre-wrap rounded-md border bg-muted/20 p-4 text-xs">{form.content}</div>
                        )}
                    </div>
                    <p className="text-xs text-muted-foreground">{t("plan.form.content.description")}</p>
                </div>
            </FormDialog>

            <FormDialog
                open={groupDialogOpen}
                onOpenChange={setGroupDialogOpen}
                title={t("plan.form.group.add")}
                onSubmit={handleGroupDialogSubmit}
                className="sm:max-w-sm"
            >
                {groupDialogError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{groupDialogError}</div>}
                <div className="space-y-1.5">
                    <Input value={groupDialogName} placeholder={t("plan.form.group.placeholder")} onChange={(e) => setGroupDialogName(e.target.value)} required />
                </div>
            </FormDialog>

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("plan.columns.delete_confirm.title")}
                description={t("plan.columns.delete_confirm.description")}
                confirmLabel={t("plan.columns.delete")}
                onConfirm={handleDelete}
            />
        </div>
    );
}
