import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import {
    fetchGiftCardTemplates,
    createGiftCardTemplate,
    updateGiftCardTemplate,
    deleteGiftCardTemplate,
    fetchGiftCardCodes,
    generateGiftCardCodes,
    toggleGiftCardCode,
    fetchGiftCardUsages,
    fetchGiftCardStatistics,
} from "../../api/giftCard.js";
import { fetchPlans } from "../../api/plan.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Card } from "../../components/Card.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useClientSort, useServerSort } from "../../components/SortableHead.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/Tabs.jsx";
import { classNames } from "../../utils/classNames.js";

const TEMPLATE_TYPES = [1, 2, 3];
const CODE_PAGE_SIZE = 20;

function formatBytes(bytes) {
    if (!bytes) return "0 B";
    const units = ["B", "KB", "MB", "GB", "TB"];
    let value = bytes;
    let i = 0;
    while (value >= 1024 && i < units.length - 1) {
        value /= 1024;
        i += 1;
    }
    return `${value.toFixed(2)} ${units[i]}`;
}

function formatDateTime(seconds) {
    if (!seconds) return "-";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function toDatetimeLocal(seconds) {
    if (!seconds) return "";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function fromDatetimeLocal(value) {
    if (!value) return undefined;
    return Math.floor(new Date(value).getTime() / 1000);
}

function MultiSelectField({ options, value, onChange, placeholder }) {
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
                    {options.map((o) => {
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
                    })}
                </div>
            )}
        </div>
    );
}

function Field({ label, children, description }) {
    return (
        <div className="space-y-1.5">
            <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{label}</div>
            {children}
            {description && <p className="text-xs text-muted-foreground">{description}</p>}
        </div>
    );
}

const EMPTY_TEMPLATE = {
    id: null,
    name: "",
    description: "",
    type: 1,
    status: true,
    sort: 0,
    icon: "",
    background_image: "",
    conditions: { new_user_only: false, new_user_max_days: "", paid_user_only: false, require_invite: false, allowed_plans: [], disallowed_plans: [] },
    limits: { max_use_per_user: "", cooldown_hours: "", invite_reward_rate: "" },
    rewards: { balance: "", transfer_enable: "", expire_days: "", device_limit: "", reset_package: false, plan_id: "", plan_validity_days: "", random_rewards: [] },
    special_config: { start_time: "", end_time: "", festival_bonus: "" },
};

function rewardsSummary(rewards, type, t) {
    const parts = [];
    if (rewards?.balance) parts.push(`${t("template.form.rewards.balance.short_label")}: ${rewards.balance / 100}`);
    if (rewards?.transfer_enable) parts.push(`${t("template.form.rewards.transfer_enable.short_label")}: ${formatBytes(rewards.transfer_enable)}`);
    if (rewards?.expire_days) parts.push(`${t("template.form.rewards.expire_days.short_label")}: ${rewards.expire_days}`);
    if (rewards?.device_limit) parts.push(`${t("template.form.rewards.device_limit.short_label")}: ${rewards.device_limit}`);
    if (type === 2 && rewards?.plan_id) parts.push(`${t("template.form.rewards.plan_id.short_label")}: ${rewards.plan_id}`);
    if (type === 2 && rewards?.plan_validity_days) parts.push(`${t("template.form.rewards.plan_validity_days.short_label")}: ${rewards.plan_validity_days}`);
    if (type === 3 && rewards?.random_rewards?.length) parts.push(t("types.3"));
    return parts;
}

function TemplatesTab({ t, plans, onChanged }) {
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");
    const [typeFilter, setTypeFilter] = useState("all");

    const [formOpen, setFormOpen] = useState(false);
    const [form, setForm] = useState(EMPTY_TEMPLATE);
    const [formError, setFormError] = useState("");
    const [deleteTarget, setDeleteTarget] = useState(null);

    const load = () => {
        setLoading(true);
        return fetchGiftCardTemplates({ per_page: 1000, page: 1 })
            .then((res) => setItems(res?.data || []))
            .finally(() => setLoading(false));
    };

    useEffect(() => { load(); }, []);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        return items.filter((item) => {
            if (keyword && !(item.name || "").toLowerCase().includes(keyword)) return false;
            if (typeFilter !== "all" && String(item.type) !== typeFilter) return false;
            return true;
        });
    }, [items, search, typeFilter]);

    const { sorted, sort, toggle: toggleSort } = useClientSort(filtered, {
        id: (i) => Number(i.id),
        name: (i) => i.name || "",
        type: (i) => Number(i.type),
        status: (i) => (i.status ? 1 : 0),
        sort: (i) => (i.sort == null ? null : Number(i.sort)),
        rewards: (i) => rewardsSummary(i.rewards, i.type, t).join(" "),
        created_at: (i) => Number(i.created_at) || null,
    });

    const planOptions = useMemo(() => plans.map((p) => ({ value: p.id, label: p.name })), [plans]);

    const openAdd = () => {
        setForm(EMPTY_TEMPLATE);
        setFormError("");
        setFormOpen(true);
    };

    const openEdit = (item) => {
        const rewards = item.rewards || {};
        const conditions = item.conditions && !Array.isArray(item.conditions) ? item.conditions : {};
        const limits = item.limits && !Array.isArray(item.limits) ? item.limits : {};
        const specialConfig = item.special_config && !Array.isArray(item.special_config) ? item.special_config : {};
        setForm({
            id: item.id,
            name: item.name || "",
            description: item.description || "",
            type: item.type,
            status: Boolean(item.status),
            sort: item.sort ?? 0,
            icon: item.icon || "",
            background_image: item.background_image || "",
            conditions: {
                new_user_only: Boolean(conditions.new_user_only),
                new_user_max_days: conditions.new_user_max_days ?? "",
                paid_user_only: Boolean(conditions.paid_user_only),
                require_invite: Boolean(conditions.require_invite),
                allowed_plans: conditions.allowed_plans || [],
                disallowed_plans: conditions.disallowed_plans || [],
            },
            limits: {
                max_use_per_user: limits.max_use_per_user ?? "",
                cooldown_hours: limits.cooldown_hours ?? "",
                invite_reward_rate: limits.invite_reward_rate ?? "",
            },
            rewards: {
                balance: rewards.balance ? rewards.balance / 100 : "",
                transfer_enable: typeof rewards.transfer_enable === "number" ? rewards.transfer_enable / 1024 / 1024 / 1024 : "",
                expire_days: rewards.expire_days ?? "",
                device_limit: rewards.device_limit ?? "",
                reset_package: Boolean(rewards.reset_package),
                plan_id: rewards.plan_id ?? "",
                plan_validity_days: rewards.plan_validity_days ?? "",
                random_rewards: (rewards.random_rewards || []).map((r) => ({
                    weight: r.weight ?? "",
                    balance: r.balance ? r.balance / 100 : "",
                    transfer_enable: typeof r.transfer_enable === "number" ? r.transfer_enable / 1024 / 1024 / 1024 : "",
                    expire_days: r.expire_days ?? "",
                    device_limit: r.device_limit ?? "",
                })),
            },
            special_config: {
                start_time: specialConfig.start_time ?? "",
                end_time: specialConfig.end_time ?? "",
                festival_bonus: specialConfig.festival_bonus ?? "",
            },
        });
        setFormError("");
        setFormOpen(true);
    };

    const num = (v) => (v === "" || v === null || v === undefined ? undefined : parseFloat(v));
    const int = (v) => (v === "" || v === null || v === undefined ? undefined : parseInt(v, 10));

    const handleSubmit = async () => {
        setFormError("");
        if (!form.name.trim()) {
            const error = t("template.form.name.required");
            setFormError(error);
            throw new Error(error);
        }

        const rewards = {
            balance: num(form.rewards.balance) !== undefined ? Math.round(num(form.rewards.balance) * 100) : undefined,
            transfer_enable: num(form.rewards.transfer_enable) !== undefined ? Math.round(num(form.rewards.transfer_enable) * 1024 * 1024 * 1024) : undefined,
            expire_days: int(form.rewards.expire_days),
            device_limit: int(form.rewards.device_limit),
            reset_package: form.rewards.reset_package,
            plan_id: int(form.rewards.plan_id),
            plan_validity_days: int(form.rewards.plan_validity_days),
            random_rewards: form.type === 3
                ? form.rewards.random_rewards.map((r) => ({
                    weight: num(r.weight),
                    balance: num(r.balance) !== undefined ? Math.round(num(r.balance) * 100) : undefined,
                    transfer_enable: num(r.transfer_enable) !== undefined ? Math.round(num(r.transfer_enable) * 1024 * 1024 * 1024) : undefined,
                    expire_days: int(r.expire_days),
                    device_limit: int(r.device_limit),
                }))
                : undefined,
        };

        const payload = {
            name: form.name.trim(),
            description: form.description || "",
            type: form.type,
            status: form.status,
            sort: int(form.sort) ?? 0,
            icon: form.icon || "",
            background_image: form.background_image || "",
            conditions: {
                new_user_only: form.conditions.new_user_only,
                new_user_max_days: int(form.conditions.new_user_max_days),
                paid_user_only: form.conditions.paid_user_only,
                require_invite: form.conditions.require_invite,
                allowed_plans: form.conditions.allowed_plans,
                disallowed_plans: form.conditions.disallowed_plans,
            },
            limits: {
                max_use_per_user: int(form.limits.max_use_per_user),
                cooldown_hours: int(form.limits.cooldown_hours),
                invite_reward_rate: num(form.limits.invite_reward_rate),
            },
            rewards,
            special_config: {
                start_time: fromDatetimeLocal(form.special_config.start_time),
                end_time: fromDatetimeLocal(form.special_config.end_time),
                festival_bonus: num(form.special_config.festival_bonus),
            },
        };

        if (form.id) {
            await updateGiftCardTemplate({ id: form.id, ...payload });
        } else {
            await createGiftCardTemplate(payload);
        }
        await load();
        onChanged?.();
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await deleteGiftCardTemplate({ id: deleteTarget.id });
        await load();
        onChanged?.();
    };

    const handleToggleStatus = async (item, checked) => {
        setItems((prev) => prev.map((i) => (i.id === item.id ? { ...i, status: checked } : i)));
        await updateGiftCardTemplate({ id: item.id, status: checked });
    };

    const addRandomReward = () => {
        setForm((p) => ({ ...p, rewards: { ...p.rewards, random_rewards: [...p.rewards.random_rewards, { weight: 10, balance: "", transfer_enable: "", expire_days: "", device_limit: "" }] } }));
    };
    const removeRandomReward = (index) => {
        setForm((p) => ({ ...p, rewards: { ...p.rewards, random_rewards: p.rewards.random_rewards.filter((_, i) => i !== index) } }));
    };
    const updateRandomReward = (index, key, value) => {
        setForm((p) => ({
            ...p,
            rewards: { ...p.rewards, random_rewards: p.rewards.random_rewards.map((r, i) => (i === index ? { ...r, [key]: value } : r)) },
        }));
    };

    return (
        <div className="space-y-4">
            <SearchFilterToolbar
                search={search}
                onSearchChange={setSearch}
                searchPlaceholder={t("common.search")}
                resetLabel={t("common.reset")}
                onReset={() => { setSearch(""); setTypeFilter("all"); }}
                filters={
                    <Select value={typeFilter} onValueChange={setTypeFilter} className="sm:w-40">
                        <SelectTrigger><SelectValue placeholder={t("template.table.columns.type")} /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">{t("template.table.columns.type")}</SelectItem>
                            {[1, 2, 3, 4].map((v) => (
                                <SelectItem key={v} value={String(v)}>{t(`types.${v}`)}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                }
                actions={<Button onClick={openAdd}>{t("template.form.add")}</Button>}
            />

            <Table>
                <TableHeader>
                    <TableRow>
                        <SortableHead className="w-16" sortKey="id" sort={sort} onSort={toggleSort}>{t("template.table.columns.id")}</SortableHead>
                        <SortableHead sortKey="name" sort={sort} onSort={toggleSort}>{t("template.table.columns.name")}</SortableHead>
                        <SortableHead sortKey="type" sort={sort} onSort={toggleSort}>{t("template.table.columns.type")}</SortableHead>
                        <SortableHead sortKey="status" sort={sort} onSort={toggleSort}>{t("template.table.columns.status")}</SortableHead>
                        <SortableHead className="w-16" sortKey="sort" sort={sort} onSort={toggleSort}>{t("template.table.columns.sort")}</SortableHead>
                        <SortableHead sortKey="rewards" sort={sort} onSort={toggleSort}>{t("template.table.columns.rewards")}</SortableHead>
                        <SortableHead sortKey="created_at" sort={sort} onSort={toggleSort}>{t("template.table.columns.created_at")}</SortableHead>
                        <TableHead className="text-right">{t("template.table.columns.actions")}</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {loading ? (
                        <TableEmpty colSpan={8}>...</TableEmpty>
                    ) : sorted.length === 0 ? (
                        <TableEmpty colSpan={8}>-</TableEmpty>
                    ) : (
                        sorted.map((item) => {
                            const summary = rewardsSummary(item.rewards, item.type, t);
                            return (
                                <TableRow key={item.id}>
                                    <TableCell><Badge variant="outline" className="font-mono">{item.id}</Badge></TableCell>
                                    <TableCell className="font-medium">{item.name}</TableCell>
                                    <TableCell><Badge variant="outline">{t(`types.${item.type}`, { defaultValue: String(item.type) })}</Badge></TableCell>
                                    <TableCell><Switch checked={Boolean(item.status)} onCheckedChange={(checked) => handleToggleStatus(item, checked)} /></TableCell>
                                    <TableCell>{item.sort ?? 0}</TableCell>
                                    <TableCell>
                                        {summary.length === 0 ? (
                                            <Badge variant="secondary">{t("template.table.columns.no_rewards")}</Badge>
                                        ) : (
                                            <div className="flex flex-col gap-1">
                                                {summary.map((s, i) => <Badge key={i} variant="outline" className="w-fit whitespace-nowrap">{s}</Badge>)}
                                            </div>
                                        )}
                                    </TableCell>
                                    <TableCell className="whitespace-nowrap text-xs text-muted-foreground">{formatDateTime(item.created_at)}</TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            <Button variant="outline" size="sm" onClick={() => openEdit(item)}>{t("template.actions.edit")}</Button>
                                            <Button variant="destructive" size="sm" onClick={() => setDeleteTarget(item)}>{t("template.actions.delete")}</Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            );
                        })
                    )}
                </TableBody>
            </Table>

            <FormDialog
                open={formOpen}
                onOpenChange={setFormOpen}
                title={form.id ? t("template.form.edit") : t("template.form.add")}
                submitLabel={t("template.form.submit.save")}
                onSubmit={handleSubmit}
                className="sm:max-w-3xl"
            >
                {formError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{formError}</div>}

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <Field label={t("template.form.name.label")}>
                        <Input value={form.name} placeholder={t("template.form.name.placeholder")} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} />
                    </Field>
                    <Field label={t("template.form.type.label")}>
                        <Select value={String(form.type)} onValueChange={(v) => setForm((p) => ({ ...p, type: parseInt(v, 10) }))}>
                            <SelectTrigger><SelectValue placeholder={t("template.form.type.placeholder")} /></SelectTrigger>
                            <SelectContent>
                                {TEMPLATE_TYPES.map((v) => (
                                    <SelectItem key={v} value={String(v)}>{t(`types.${v}`)}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </Field>
                </div>

                <Field label={t("template.form.description.label")}>
                    <Textarea value={form.description} placeholder={t("template.form.description.placeholder")} onChange={(e) => setForm((p) => ({ ...p, description: e.target.value }))} />
                </Field>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <Field label={t("template.form.sort.label")}>
                        <Input type="number" value={form.sort} placeholder={t("template.form.sort.placeholder")} onChange={(e) => setForm((p) => ({ ...p, sort: e.target.value }))} />
                    </Field>
                    <div className="flex flex-row items-center justify-between rounded-md border border-dashed p-3">
                        <div className="space-y-0.5">
                            <div className="text-xs font-semibold">{t("template.form.status.label")}</div>
                            <p className="text-[10px] text-muted-foreground">{t("template.form.status.description")}</p>
                        </div>
                        <Switch checked={form.status} onCheckedChange={(checked) => setForm((p) => ({ ...p, status: checked }))} />
                    </div>
                </div>

                <div className="space-y-4 rounded-xl border bg-card/50 p-4">
                    <h4 className="text-sm font-semibold">{t("template.form.rewards.title")}</h4>
                    {form.type === 1 && (
                        <>
                            <div className="grid grid-cols-2 gap-4">
                                <Field label={t("template.form.rewards.balance.label")}>
                                    <Input type="number" step="any" value={form.rewards.balance} placeholder={t("template.form.rewards.balance.placeholder")} onChange={(e) => setForm((p) => ({ ...p, rewards: { ...p.rewards, balance: e.target.value } }))} />
                                </Field>
                                <Field label={t("template.form.rewards.transfer_enable.label")}>
                                    <Input type="number" step="any" value={form.rewards.transfer_enable} placeholder={t("template.form.rewards.transfer_enable.placeholder")} onChange={(e) => setForm((p) => ({ ...p, rewards: { ...p.rewards, transfer_enable: e.target.value } }))} />
                                </Field>
                                <Field label={t("template.form.rewards.expire_days.label")}>
                                    <Input type="number" value={form.rewards.expire_days} placeholder={t("template.form.rewards.expire_days.placeholder")} onChange={(e) => setForm((p) => ({ ...p, rewards: { ...p.rewards, expire_days: e.target.value } }))} />
                                </Field>
                                <Field label={t("template.form.rewards.device_limit.label")}>
                                    <Input type="number" value={form.rewards.device_limit} placeholder={t("template.form.rewards.device_limit.placeholder")} onChange={(e) => setForm((p) => ({ ...p, rewards: { ...p.rewards, device_limit: e.target.value } }))} />
                                </Field>
                            </div>
                            <div className="flex flex-row items-center justify-between rounded-md border border-dashed p-3">
                                <div className="space-y-0.5">
                                    <div className="text-xs font-semibold">{t("template.form.rewards.reset_package.label")}</div>
                                    <p className="text-[10px] text-muted-foreground">{t("template.form.rewards.reset_package.description")}</p>
                                </div>
                                <Switch checked={form.rewards.reset_package} onCheckedChange={(checked) => setForm((p) => ({ ...p, rewards: { ...p.rewards, reset_package: checked } }))} />
                            </div>
                        </>
                    )}
                    {form.type === 2 && (
                        <div className="grid grid-cols-2 gap-4">
                            <Field label={t("template.form.rewards.plan_id.label")}>
                                <Select value={form.rewards.plan_id ? String(form.rewards.plan_id) : ""} onValueChange={(v) => setForm((p) => ({ ...p, rewards: { ...p.rewards, plan_id: v } }))}>
                                    <SelectTrigger><SelectValue placeholder={t("template.form.rewards.plan_id.placeholder")} /></SelectTrigger>
                                    <SelectContent>
                                        {planOptions.map((o) => (
                                            <SelectItem key={o.value} value={String(o.value)}>{o.label}</SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </Field>
                            <Field label={t("template.form.rewards.plan_validity_days.label")}>
                                <Input type="number" value={form.rewards.plan_validity_days} placeholder={t("template.form.rewards.plan_validity_days.placeholder")} onChange={(e) => setForm((p) => ({ ...p, rewards: { ...p.rewards, plan_validity_days: e.target.value } }))} />
                            </Field>
                        </div>
                    )}
                    {form.type === 4 && (
                        <p className="rounded-lg bg-muted/50 p-3 text-xs text-muted-foreground">{t("template.form.rewards.task_card.description")}</p>
                    )}
                    {form.type === 3 && (
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{t("template.form.rewards.random_rewards.label")}</div>
                                <Button type="button" variant="outline" size="sm" onClick={addRandomReward}>{t("template.form.rewards.random_rewards.add")}</Button>
                            </div>
                            {form.rewards.random_rewards.map((r, i) => (
                                <div key={i} className="grid grid-cols-5 items-end gap-2 rounded-lg border border-dashed bg-muted/20 p-3">
                                    <Field label={t("template.form.rewards.random_rewards.weight")}>
                                        <Input type="number" value={r.weight} onChange={(e) => updateRandomReward(i, "weight", e.target.value)} />
                                    </Field>
                                    <Field label={t("template.form.rewards.balance.label")}>
                                        <Input type="number" step="any" value={r.balance} onChange={(e) => updateRandomReward(i, "balance", e.target.value)} />
                                    </Field>
                                    <Field label={t("template.form.rewards.transfer_enable.label")}>
                                        <Input type="number" step="any" value={r.transfer_enable} onChange={(e) => updateRandomReward(i, "transfer_enable", e.target.value)} />
                                    </Field>
                                    <Field label={t("template.form.rewards.expire_days.label")}>
                                        <Input type="number" value={r.expire_days} onChange={(e) => updateRandomReward(i, "expire_days", e.target.value)} />
                                    </Field>
                                    <Button type="button" variant="ghost" size="sm" className="text-destructive" onClick={() => removeRandomReward(i)}>×</Button>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                <div className="space-y-4 rounded-xl border bg-card/50 p-4">
                    <h4 className="text-sm font-semibold">{t("template.form.conditions.title")}</h4>
                    <Field label={t("template.form.conditions.new_user_max_days.label")}>
                        <Input type="number" value={form.conditions.new_user_max_days} placeholder={t("template.form.conditions.new_user_max_days.placeholder")} onChange={(e) => setForm((p) => ({ ...p, conditions: { ...p.conditions, new_user_max_days: e.target.value } }))} />
                    </Field>
                    <div className="grid grid-cols-3 gap-4">
                        <div className="flex flex-col items-center justify-between gap-2 rounded-md border border-dashed p-3">
                            <div className="text-center text-[10px] font-semibold uppercase">{t("template.form.conditions.new_user_only.label")}</div>
                            <Switch checked={form.conditions.new_user_only} onCheckedChange={(checked) => setForm((p) => ({ ...p, conditions: { ...p.conditions, new_user_only: checked } }))} />
                        </div>
                        <div className="flex flex-col items-center justify-between gap-2 rounded-md border border-dashed p-3">
                            <div className="text-center text-[10px] font-semibold uppercase">{t("template.form.conditions.paid_user_only.label")}</div>
                            <Switch checked={form.conditions.paid_user_only} onCheckedChange={(checked) => setForm((p) => ({ ...p, conditions: { ...p.conditions, paid_user_only: checked } }))} />
                        </div>
                        <div className="flex flex-col items-center justify-between gap-2 rounded-md border border-dashed p-3">
                            <div className="text-center text-[10px] font-semibold uppercase">{t("template.form.conditions.require_invite.label")}</div>
                            <Switch checked={form.conditions.require_invite} onCheckedChange={(checked) => setForm((p) => ({ ...p, conditions: { ...p.conditions, require_invite: checked } }))} />
                        </div>
                    </div>
                    <Field label={t("template.form.conditions.allowed_plans.label")}>
                        <MultiSelectField options={planOptions} value={form.conditions.allowed_plans} onChange={(value) => setForm((p) => ({ ...p, conditions: { ...p.conditions, allowed_plans: value } }))} placeholder={t("template.form.conditions.allowed_plans.placeholder")} />
                    </Field>
                    <Field label={t("template.form.conditions.disallowed_plans.label")}>
                        <MultiSelectField options={planOptions} value={form.conditions.disallowed_plans} onChange={(value) => setForm((p) => ({ ...p, conditions: { ...p.conditions, disallowed_plans: value } }))} placeholder={t("template.form.conditions.disallowed_plans.placeholder")} />
                    </Field>
                </div>

                <div className="space-y-4 rounded-xl border bg-card/50 p-4">
                    <h4 className="text-sm font-semibold">{t("template.form.limits.title")}</h4>
                    <div className="grid grid-cols-2 gap-4">
                        <Field label={t("template.form.limits.max_use_per_user.label")}>
                            <Input type="number" value={form.limits.max_use_per_user} placeholder={t("template.form.limits.max_use_per_user.placeholder")} onChange={(e) => setForm((p) => ({ ...p, limits: { ...p.limits, max_use_per_user: e.target.value } }))} />
                        </Field>
                        <Field label={t("template.form.limits.cooldown_hours.label")}>
                            <Input type="number" value={form.limits.cooldown_hours} placeholder={t("template.form.limits.cooldown_hours.placeholder")} onChange={(e) => setForm((p) => ({ ...p, limits: { ...p.limits, cooldown_hours: e.target.value } }))} />
                        </Field>
                    </div>
                    <Field label={t("template.form.limits.invite_reward_rate.label")} description={t("template.form.limits.invite_reward_rate.description")}>
                        <Input type="number" step="any" value={form.limits.invite_reward_rate} placeholder={t("template.form.limits.invite_reward_rate.placeholder")} onChange={(e) => setForm((p) => ({ ...p, limits: { ...p.limits, invite_reward_rate: e.target.value } }))} />
                    </Field>
                </div>

                <div className="space-y-4 rounded-xl border bg-card/50 p-4">
                    <h4 className="text-sm font-semibold">{t("template.form.special_config.title")}</h4>
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <Field label={t("template.form.special_config.start_time.label")}>
                            <Input type="datetime-local" value={toDatetimeLocal(form.special_config.start_time)} onChange={(e) => setForm((p) => ({ ...p, special_config: { ...p.special_config, start_time: fromDatetimeLocal(e.target.value) } }))} />
                        </Field>
                        <Field label={t("template.form.special_config.end_time.label")}>
                            <Input type="datetime-local" value={toDatetimeLocal(form.special_config.end_time)} onChange={(e) => setForm((p) => ({ ...p, special_config: { ...p.special_config, end_time: fromDatetimeLocal(e.target.value) } }))} />
                        </Field>
                    </div>
                    <Field label={t("template.form.special_config.festival_bonus.label")}>
                        <Input type="number" step="any" value={form.special_config.festival_bonus} placeholder={t("template.form.special_config.festival_bonus.placeholder")} onChange={(e) => setForm((p) => ({ ...p, special_config: { ...p.special_config, festival_bonus: e.target.value } }))} />
                    </Field>
                </div>

                <div className="space-y-4 rounded-xl border bg-card/50 p-4">
                    <h4 className="text-sm font-semibold">{t("template.form.display.title")}</h4>
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <Field label={t("template.form.icon.label")}>
                            <Input value={form.icon} placeholder={t("template.form.icon.placeholder")} onChange={(e) => setForm((p) => ({ ...p, icon: e.target.value }))} />
                        </Field>
                        <Field label={t("template.form.background_image.label")}>
                            <Input value={form.background_image} placeholder={t("template.form.background_image.placeholder")} onChange={(e) => setForm((p) => ({ ...p, background_image: e.target.value }))} />
                        </Field>
                    </div>
                </div>
            </FormDialog>

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("template.actions.deleteConfirm.title")}
                description={t("template.actions.deleteConfirm.description")}
                confirmLabel={t("template.actions.deleteConfirm.confirmText")}
                onConfirm={handleDelete}
            />
        </div>
    );
}

const EMPTY_GENERATE = { template_id: "", count: 10, prefix: "", expires_hours: 720, max_usage: 1, download_csv: false };

function CodesTab({ t, templates }) {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState("");
    const [statusFilter, setStatusFilter] = useState("all");
    const [templateFilter, setTemplateFilter] = useState("all");
    const codeSort = useServerSort(() => setPage(1));
    const { sort, toggle: toggleSort } = codeSort;

    const [genOpen, setGenOpen] = useState(false);
    const [genForm, setGenForm] = useState(EMPTY_GENERATE);
    const [genError, setGenError] = useState("");

    const buildFilter = () => {
        const filter = [];
        if (search.trim()) filter.push({ id: "code", value: search.trim() });
        if (statusFilter !== "all") filter.push({ id: "status", value: parseInt(statusFilter, 10) });
        if (templateFilter !== "all") filter.push({ id: "template_id", value: parseInt(templateFilter, 10) });
        return filter;
    };

    const load = () => {
        setLoading(true);
        return fetchGiftCardCodes({ per_page: CODE_PAGE_SIZE, page, filter: buildFilter(), sort: codeSort.sortParam })
            .then((res) => {
                setItems(res?.data || []);
                setTotal(res?.total || 0);
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => { load(); }, [page, sort.key, sort.dir]);

    const refetch = () => {
        if (page === 1) load();
        else setPage(1);
    };

    const resetFilters = () => {
        setSearch("");
        setStatusFilter("all");
        setTemplateFilter("all");
        refetch();
    };

    const openGenerate = () => {
        setGenForm(EMPTY_GENERATE);
        setGenError("");
        setGenOpen(true);
    };

    const handleGenerate = async () => {
        setGenError("");
        if (!genForm.template_id) {
            const error = t("code.form.template_id.placeholder");
            setGenError(error);
            throw new Error(error);
        }
        const payload = {
            template_id: parseInt(genForm.template_id, 10),
            count: parseInt(genForm.count, 10) || 0,
            prefix: genForm.prefix || undefined,
            expires_hours: parseInt(genForm.expires_hours, 10) || 0,
            max_usage: parseInt(genForm.max_usage, 10) || 0,
            download_csv: genForm.download_csv,
        };
        const res = await generateGiftCardCodes(payload);
        if (payload.download_csv && res instanceof Blob) {
            const url = window.URL.createObjectURL(res);
            const link = document.createElement("a");
            link.href = url;
            link.download = `gift_codes_${Date.now()}.csv`;
            link.click();
            window.URL.revokeObjectURL(url);
        }
        await load();
    };

    const handleToggle = async (item, enable) => {
        setItems((prev) => prev.map((c) => (c.id === item.id ? { ...c, status: enable ? 0 : 2 } : c)));
        try {
            await toggleGiftCardCode({ id: item.id, action: enable ? "enable" : "disable" });
        } catch {
            await load();
        }
    };

    const totalPages = Math.max(1, Math.ceil(total / CODE_PAGE_SIZE));

    return (
        <div className="space-y-4">
            <SearchFilterToolbar
                search={search}
                onSearchChange={setSearch}
                searchPlaceholder={t("common.search")}
                resetLabel={t("common.reset")}
                onReset={resetFilters}
                filters={
                    <>
                        <Select value={templateFilter} onValueChange={(v) => { setTemplateFilter(v); refetch(); }} className="sm:w-40">
                            <SelectTrigger><SelectValue placeholder={t("code.form.template_id.label")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("code.form.template_id.label")}</SelectItem>
                                {templates.map((tpl) => (
                                    <SelectItem key={tpl.id} value={String(tpl.id)}>{tpl.name}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); refetch(); }} className="sm:w-32">
                            <SelectTrigger><SelectValue placeholder={t("code.table.columns.status")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("code.table.columns.status")}</SelectItem>
                                {["0", "1", "2", "3"].map((v) => (
                                    <SelectItem key={v} value={v}>{t(`code.status.${v}`)}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Button variant="secondary" size="sm" type="button" onClick={refetch}>{t("common.search")}</Button>
                    </>
                }
                actions={<Button onClick={openGenerate}>{t("code.form.generate")}</Button>}
            />

            <Table>
                <TableHeader>
                    <TableRow>
                        <SortableHead className="w-16" sortKey="id" sort={sort} onSort={toggleSort}>{t("code.table.columns.id")}</SortableHead>
                        <SortableHead sortKey="code" sort={sort} onSort={toggleSort}>{t("code.table.columns.code")}</SortableHead>
                        <SortableHead sortKey="template_name" sort={sort} onSort={toggleSort}>{t("code.table.columns.template_name")}</SortableHead>
                        <SortableHead sortKey="status" sort={sort} onSort={toggleSort}>{t("code.table.columns.status")}</SortableHead>
                        <SortableHead sortKey="expires_at" sort={sort} onSort={toggleSort}>{t("code.table.columns.expires_at")}</SortableHead>
                        <SortableHead sortKey="usage_count" sort={sort} onSort={toggleSort}>{t("code.table.columns.usage_count")}</SortableHead>
                        <SortableHead sortKey="max_usage" sort={sort} onSort={toggleSort}>{t("code.table.columns.max_usage")}</SortableHead>
                        <SortableHead sortKey="created_at" sort={sort} onSort={toggleSort}>{t("code.table.columns.created_at")}</SortableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {loading ? (
                        <TableEmpty colSpan={8}>...</TableEmpty>
                    ) : items.length === 0 ? (
                        <TableEmpty colSpan={8}>-</TableEmpty>
                    ) : (
                        items.map((code) => (
                            <TableRow key={code.id}>
                                <TableCell><Badge variant="outline" className="font-mono">{code.id}</Badge></TableCell>
                                <TableCell><Badge variant="secondary" className="font-mono">{code.code}</Badge></TableCell>
                                <TableCell>{code.template_name || "-"}</TableCell>
                                <TableCell>
                                    <div className="flex items-center gap-2">
                                        <Badge variant={code.status === 1 ? "secondary" : code.status === 2 || code.status === 3 ? "destructive" : "default"}>
                                            {t(`code.status.${code.status}`)}
                                        </Badge>
                                        {(code.status === 0 || code.status === 2) && (
                                            <Switch checked={code.status === 0} onCheckedChange={(checked) => handleToggle(code, checked)} />
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell className="text-sm text-muted-foreground">{formatDateTime(code.expires_at)}</TableCell>
                                <TableCell><Badge variant="outline">{code.usage_count ?? 0}</Badge></TableCell>
                                <TableCell><Badge variant="outline">{code.max_usage ?? 0}</Badge></TableCell>
                                <TableCell className="text-sm text-muted-foreground">{formatDateTime(code.created_at)}</TableCell>
                            </TableRow>
                        ))
                    )}
                </TableBody>
            </Table>

            {totalPages > 1 && (
                <div className="flex items-center justify-end gap-2">
                    <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>‹</Button>
                    <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
                    <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>›</Button>
                </div>
            )}

            <FormDialog
                open={genOpen}
                onOpenChange={setGenOpen}
                title={t("code.form.generate")}
                submitLabel={t("code.form.submit.generate")}
                onSubmit={handleGenerate}
            >
                {genError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{genError}</div>}
                <Field label={t("code.form.template_id.label")}>
                    <Select value={genForm.template_id ? String(genForm.template_id) : ""} onValueChange={(v) => setGenForm((p) => ({ ...p, template_id: v }))}>
                        <SelectTrigger><SelectValue placeholder={t("code.form.template_id.placeholder")} /></SelectTrigger>
                        <SelectContent>
                            {templates.map((tpl) => (
                                <SelectItem key={tpl.id} value={String(tpl.id)}>{tpl.name}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </Field>
                <div className="grid grid-cols-2 gap-4">
                    <Field label={t("code.form.count.label")}>
                        <Input type="number" min={1} max={10000} value={genForm.count} onChange={(e) => setGenForm((p) => ({ ...p, count: e.target.value }))} />
                    </Field>
                    <Field label={t("code.form.prefix.label")}>
                        <Input value={genForm.prefix} onChange={(e) => setGenForm((p) => ({ ...p, prefix: e.target.value }))} />
                    </Field>
                </div>
                <div className="grid grid-cols-2 gap-4">
                    <Field label={t("code.form.expires_hours.label")}>
                        <Input type="number" min={1} value={genForm.expires_hours} onChange={(e) => setGenForm((p) => ({ ...p, expires_hours: e.target.value }))} />
                    </Field>
                    <Field label={t("code.form.max_usage.label")}>
                        <Input type="number" min={1} value={genForm.max_usage} onChange={(e) => setGenForm((p) => ({ ...p, max_usage: e.target.value }))} />
                    </Field>
                </div>
                <div className="flex flex-row items-center justify-between rounded-lg border p-4">
                    <div className="text-sm font-medium">{t("code.form.download_csv")}</div>
                    <Switch checked={genForm.download_csv} onCheckedChange={(checked) => setGenForm((p) => ({ ...p, download_csv: checked }))} />
                </div>
            </FormDialog>
        </div>
    );
}

function UsagesTab({ t }) {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState("");
    const usageSort = useServerSort(() => setPage(1));
    const { sort, toggle: toggleSort } = usageSort;

    const load = () => {
        setLoading(true);
        const filter = search.trim() ? [{ id: "user_email", value: search.trim() }] : [];
        return fetchGiftCardUsages({ per_page: CODE_PAGE_SIZE, page, filter, sort: usageSort.sortParam })
            .then((res) => {
                setItems(res?.data || []);
                setTotal(res?.total || 0);
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => { load(); }, [page, sort.key, sort.dir]);

    const refetch = () => {
        if (page === 1) load();
        else setPage(1);
    };

    const totalPages = Math.max(1, Math.ceil(total / CODE_PAGE_SIZE));

    return (
        <div className="space-y-4">
            <SearchFilterToolbar
                search={search}
                onSearchChange={setSearch}
                searchPlaceholder={t("common.search")}
                onReset={() => { setSearch(""); refetch(); }}
                resetLabel={t("common.reset")}
                actions={<Button variant="secondary" size="sm" onClick={refetch}>{t("common.search")}</Button>}
            />
            <Table>
                <TableHeader>
                    <TableRow>
                        <SortableHead className="w-16" sortKey="id" sort={sort} onSort={toggleSort}>{t("usage.table.columns.id")}</SortableHead>
                        <SortableHead sortKey="code" sort={sort} onSort={toggleSort}>{t("usage.table.columns.code")}</SortableHead>
                        <SortableHead sortKey="template_name" sort={sort} onSort={toggleSort}>{t("usage.table.columns.template_name")}</SortableHead>
                        <SortableHead sortKey="user_email" sort={sort} onSort={toggleSort}>{t("usage.table.columns.user_email")}</SortableHead>
                        <SortableHead sortKey="rewards_given" sort={sort} onSort={toggleSort}>{t("usage.table.columns.rewards_given")}</SortableHead>
                        <SortableHead sortKey="invite_rewards" sort={sort} onSort={toggleSort}>{t("usage.table.columns.invite_rewards")}</SortableHead>
                        <SortableHead sortKey="multiplier_applied" sort={sort} onSort={toggleSort}>{t("usage.table.columns.multiplier_applied")}</SortableHead>
                        <SortableHead sortKey="ip_address" sort={sort} onSort={toggleSort}>{t("usage.table.columns.ip_address")}</SortableHead>
                        <SortableHead sortKey="created_at" sort={sort} onSort={toggleSort}>{t("usage.table.columns.created_at")}</SortableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {loading ? (
                        <TableEmpty colSpan={9}>...</TableEmpty>
                    ) : items.length === 0 ? (
                        <TableEmpty colSpan={9}>-</TableEmpty>
                    ) : (
                        items.map((usage) => (
                            <TableRow key={usage.id}>
                                <TableCell><Badge variant="outline" className="font-mono">{usage.id}</Badge></TableCell>
                                <TableCell><Badge variant="secondary" className="font-mono">{usage.code}</Badge></TableCell>
                                <TableCell>{usage.template_name || "-"}</TableCell>
                                <TableCell>{usage.user_email || "-"}</TableCell>
                                <TableCell className="text-xs text-muted-foreground">{usage.rewards_given ? JSON.stringify(usage.rewards_given) : "-"}</TableCell>
                                <TableCell className="text-xs text-muted-foreground">{usage.invite_rewards ? JSON.stringify(usage.invite_rewards) : "-"}</TableCell>
                                <TableCell>{usage.multiplier_applied ?? "-"}</TableCell>
                                <TableCell className="font-mono text-xs">{usage.ip_address || "-"}</TableCell>
                                <TableCell className="text-sm text-muted-foreground">{formatDateTime(usage.created_at)}</TableCell>
                            </TableRow>
                        ))
                    )}
                </TableBody>
            </Table>
            {totalPages > 1 && (
                <div className="flex items-center justify-end gap-2">
                    <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>‹</Button>
                    <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
                    <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>›</Button>
                </div>
            )}
        </div>
    );
}

function StatisticsTab({ t }) {
    const [stats, setStats] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        setLoading(true);
        fetchGiftCardStatistics({})
            .then((res) => setStats(res?.data?.total_stats || null))
            .finally(() => setLoading(false));
    }, []);

    const cards = [
        { title: t("statistics.total.templates_count"), value: stats?.templates_count },
        { title: t("statistics.total.active_templates_count"), value: stats?.active_templates_count },
        { title: t("statistics.total.codes_count"), value: stats?.codes_count },
        { title: t("statistics.total.used_codes_count"), value: stats?.used_codes_count },
    ];

    return (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {cards.map((c, i) => (
                <Card key={i} className="p-4">
                    <div className="text-sm font-medium text-muted-foreground">{c.title}</div>
                    <div className="mt-2 text-2xl font-bold">{loading ? "..." : c.value ?? 0}</div>
                </Card>
            ))}
        </div>
    );
}

export function GiftCardManagement() {
    const { t } = useTranslation("giftCard");
    const auth = useAuthState();
    const [activeTab, setActiveTab] = useState("templates");
    const [templates, setTemplates] = useState([]);
    const [plans, setPlans] = useState([]);

    const loadTemplates = () => {
        fetchGiftCardTemplates({ per_page: 1000, page: 1 }).then((res) => setTemplates(res?.data || []));
    };

    useEffect(() => {
        loadTemplates();
        fetchPlans().then((res) => setPlans(res?.data || [])).catch(() => setPlans([]));
    }, [auth.token]);

    return (
        <div className="p-4 md:p-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-lg font-medium">{t("title")}</h3>
                <p className="text-sm text-muted-foreground">{t("description")}</p>
            </div>
            <hr className="my-6" />

            <Tabs value={activeTab} onValueChange={setActiveTab}>
                <TabsList>
                    <TabsTrigger value="templates">{t("tabs.templates")}</TabsTrigger>
                    <TabsTrigger value="codes">{t("tabs.codes")}</TabsTrigger>
                    <TabsTrigger value="usages">{t("tabs.usages")}</TabsTrigger>
                    <TabsTrigger value="statistics">{t("tabs.statistics")}</TabsTrigger>
                </TabsList>
                <TabsContent value="templates" className="mt-6">
                    <TemplatesTab t={t} plans={plans} onChanged={loadTemplates} />
                </TabsContent>
                <TabsContent value="codes" className="mt-6">
                    <CodesTab t={t} templates={templates} />
                </TabsContent>
                <TabsContent value="usages" className="mt-6">
                    <UsagesTab t={t} />
                </TabsContent>
                <TabsContent value="statistics" className="mt-6">
                    <StatisticsTab t={t} />
                </TabsContent>
            </Tabs>
        </div>
    );
}
