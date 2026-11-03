import React, { useEffect, useMemo, useRef, useState } from "react";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "../../components/Select.jsx";
import { classNames } from "../../utils/classNames.js";
import {
    FIELDS, PRESETS, OP_KIND, fieldDef, operatorsFor, defaultValue, newCondition, isComplete, describeCondition,
    instantiate, presetActive, togglePreset, readViews, writeViews, addView, removeView,
} from "./userFilter.js";

function safeStorage() {
    try {
        return window.localStorage;
    } catch {
        return null;
    }
}

// 一个值的输入框：按字段类型选择文本 / 数字 / 日期 / 下拉 / 开关。
function ValueInput({ def, value, onChange, planOptions, currency, placeholder, t }) {
    if (def.type === "date") {
        return <Input type="date" value={value ?? ""} onChange={(e) => onChange(e.target.value)} />;
    }
    if (def.type === "number") {
        const unit = def.unit === "currency" ? currency : def.unit;
        return (
            <div className="flex items-center gap-2">
                <Input type="number" value={value ?? ""} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} />
                {unit ? <span className="shrink-0 text-sm text-muted-foreground">{unit}</span> : null}
            </div>
        );
    }
    if (def.type === "plan" || def.type === "enum") {
        const options = def.type === "plan" ? planOptions : def.options.map((o) => ({ value: o.value, label: t(`filter.${o.labelKey}`) }));
        return (
            <Select value={String(value ?? "")} onValueChange={onChange}>
                <SelectTrigger><SelectValue placeholder={t("filter.sheet.value")} /></SelectTrigger>
                <SelectContent>
                    {options.map((o) => <SelectItem key={String(o.value)} value={String(o.value)}>{o.label}</SelectItem>)}
                </SelectContent>
            </Select>
        );
    }
    if (def.type === "bool") {
        return (
            <div className="flex items-center gap-2">
                <Switch checked={Boolean(value)} onCheckedChange={onChange} />
                <span className="text-sm">{value ? t("filter.boolean.true") : t("filter.boolean.false")}</span>
            </div>
        );
    }
    return <Input value={value ?? ""} onChange={(e) => onChange(e.target.value)} placeholder={placeholder || t("filter.builder.text_placeholder")} />;
}

function ConditionDialog({ open, onOpenChange, initial, isNew, planOptions, currency, onSave, ctx, t }) {
    const [draft, setDraft] = useState(initial);
    useEffect(() => {
        if (open) setDraft(initial);
    }, [open, initial]);

    const def = fieldDef(draft.field);
    const ops = operatorsFor(def);
    const kind = OP_KIND[draft.op];
    const set = (patch) => setDraft((d) => ({ ...d, ...patch }));

    const setField = (field) => {
        const next = fieldDef(field);
        set({ field, op: operatorsFor(next)[0], value: defaultValue(next), value2: "" });
    };
    const setOp = (op) => {
        // 取值个数相同就保留已填的值，否则清空，避免带着不适用的值
        const same = OP_KIND[op] === kind;
        set({ op, value: same ? draft.value : OP_KIND[op] === "multi" ? [] : defaultValue(def), value2: same ? draft.value2 : "" });
    };
    const togglePlan = (id) => {
        const cur = Array.isArray(draft.value) ? draft.value : [];
        set({ value: cur.includes(String(id)) ? cur.filter((x) => x !== String(id)) : [...cur, String(id)] });
    };

    const complete = isComplete(draft);

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-lg">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t(isNew ? "filter.builder.title_add" : "filter.builder.title_edit")}</h3>

                <div className="mt-4 space-y-4">
                    <div className="space-y-1.5">
                        <label className="text-sm font-medium">{t("filter.builder.field")}</label>
                        <Select value={draft.field} onValueChange={setField}>
                            <SelectTrigger><SelectValue placeholder={t("filter.sheet.field")} /></SelectTrigger>
                            <SelectContent>
                                {FIELDS.map((f) => <SelectItem key={f.field} value={f.field}>{t(`filter.fields.${f.labelKey}`)}</SelectItem>)}
                            </SelectContent>
                        </Select>
                    </div>

                    <div className="space-y-1.5">
                        <label className="text-sm font-medium">{t("filter.builder.operator")}</label>
                        <Select value={draft.op} onValueChange={setOp}>
                            <SelectTrigger><SelectValue placeholder={t("filter.sheet.operator")} /></SelectTrigger>
                            <SelectContent>
                                {ops.map((op) => <SelectItem key={op} value={op}>{t(`filter.builder.ops.${op}`, { n: "N" })}</SelectItem>)}
                            </SelectContent>
                        </Select>
                    </div>

                    {kind === "one" && (
                        <div className="space-y-1.5">
                            <label className="text-sm font-medium">{t("filter.builder.value")}</label>
                            <ValueInput def={def} value={draft.value} onChange={(v) => set({ value: v })} planOptions={planOptions} currency={currency} t={t} />
                        </div>
                    )}
                    {kind === "two" && (
                        <div className="grid grid-cols-2 gap-3">
                            <div className="space-y-1.5">
                                <label className="text-sm font-medium">{t("filter.builder.from")}</label>
                                <ValueInput def={def} value={draft.value} onChange={(v) => set({ value: v })} planOptions={planOptions} currency={currency} t={t} />
                            </div>
                            <div className="space-y-1.5">
                                <label className="text-sm font-medium">{t("filter.builder.to")}</label>
                                <ValueInput def={def} value={draft.value2} onChange={(v) => set({ value2: v })} planOptions={planOptions} currency={currency} t={t} />
                            </div>
                        </div>
                    )}
                    {kind === "days" && (
                        <div className="space-y-1.5">
                            <label className="text-sm font-medium">{t("filter.builder.days_label")}</label>
                            <div className="flex items-center gap-2">
                                <Input type="number" min="1" step="1" value={draft.value ?? ""} onChange={(e) => set({ value: e.target.value })} placeholder={t("filter.builder.days_placeholder")} />
                                <span className="shrink-0 text-sm text-muted-foreground">{t("filter.builder.days_unit")}</span>
                            </div>
                        </div>
                    )}
                    {kind === "multi" && (
                        <div className="space-y-1.5">
                            <label className="text-sm font-medium">{t("filter.builder.pick_plans")}</label>
                            <div className="flex max-h-48 flex-wrap gap-2 overflow-y-auto rounded-md border p-2">
                                {planOptions.length === 0 && <span className="text-sm text-muted-foreground">-</span>}
                                {planOptions.map((o) => {
                                    const on = Array.isArray(draft.value) && draft.value.includes(String(o.value));
                                    return (
                                        <button
                                            key={String(o.value)}
                                            type="button"
                                            onClick={() => togglePlan(o.value)}
                                            className={classNames(
                                                "rounded-full border px-3 py-1 text-xs",
                                                on ? "border-primary bg-primary text-primary-foreground" : "bg-background hover:bg-accent",
                                            )}
                                        >
                                            {on ? "✓ " : ""}{o.label}
                                        </button>
                                    );
                                })}
                            </div>
                        </div>
                    )}
                    {kind === "none" && <p className="text-sm text-muted-foreground">{t("filter.builder.no_value_needed")}</p>}

                    {complete && (
                        <p className="rounded-md bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                            {t("filter.builder.preview", { text: describeCondition(draft, ctx) })}
                        </p>
                    )}
                </div>

                <div className="mt-6 flex justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange(false)}>{t("filter.builder.cancel")}</Button>
                    <Button disabled={!complete} onClick={() => { onSave(draft); onOpenChange(false); }}>{t("filter.builder.confirm")}</Button>
                </div>
            </div>
        </Dialog>
    );
}

// 「添加筛选」的级联菜单：悬停（或点击）展开字段，悬停字段展开条件，悬停条件展开可选值。
// 不需要填值的条件、固定选项的值点一下就直接加上；其余的带着已选的字段和条件打开编辑框。
function AddFilterMenu({ label, planOptions, onPick, onEdit, t }) {
    const [open, setOpen] = useState(false);
    const [fieldKey, setFieldKey] = useState(null);
    const [opKey, setOpKey] = useState(null);
    const timer = useRef(null);

    const show = () => {
        clearTimeout(timer.current);
        setOpen(true);
    };
    const hide = () => {
        clearTimeout(timer.current);
        timer.current = setTimeout(() => {
            setOpen(false);
            setFieldKey(null);
            setOpKey(null);
        }, 180);
    };
    useEffect(() => () => clearTimeout(timer.current), []);

    const def = fieldKey ? fieldDef(fieldKey) : null;
    const ops = def ? operatorsFor(def) : [];
    const kind = opKey ? OP_KIND[opKey] : null;

    // 第三列：这个字段 + 条件下可以直接点选的值；没有就返回 null
    const valueChoices = () => {
        if (!def || !opKey || (opKey !== "is" && opKey !== "is_not")) return null;
        if (def.type === "enum") return def.options.map((o) => ({ value: o.value, label: t(`filter.${o.labelKey}`) }));
        if (def.type === "bool") return [{ value: true, label: t("filter.boolean.true") }, { value: false, label: t("filter.boolean.false") }];
        if (def.type === "plan") return planOptions.map((o) => ({ value: String(o.value), label: o.label }));
        return null;
    };
    const choices = valueChoices();

    const close = () => {
        setOpen(false);
        setFieldKey(null);
        setOpKey(null);
    };
    const pickOp = (op) => {
        if (OP_KIND[op] === "none") {
            onPick({ field: def.field, op, value: defaultValue(def), value2: "" });
            close();
        } else {
            setOpKey(op);
        }
    };
    const itemCls = (on) => classNames("flex w-full items-center justify-between gap-3 rounded px-3 py-1.5 text-left text-sm hover:bg-accent", on && "bg-accent");

    return (
        <div className="relative" onMouseEnter={show} onMouseLeave={hide}>
            <Button variant="outline" size="sm" onClick={() => (open ? close() : show())}>＋ {label}</Button>
            {open && (
                <div className="absolute left-0 top-full z-50 flex pt-1" onMouseEnter={show}>
                    <div className="max-h-80 min-w-[150px] overflow-y-auto rounded-md border bg-background p-1 shadow-lg">
                        {FIELDS.map((f) => (
                            <button key={f.field} type="button" className={itemCls(fieldKey === f.field)} onMouseEnter={() => { setFieldKey(f.field); setOpKey(null); }} onClick={() => { setFieldKey(f.field); setOpKey(null); }}>
                                {t(`filter.fields.${f.labelKey}`)}<span className="text-muted-foreground">›</span>
                            </button>
                        ))}
                    </div>
                    {def && (
                        <div className="ml-1 max-h-80 min-w-[150px] overflow-y-auto rounded-md border bg-background p-1 shadow-lg">
                            {ops.map((op) => {
                                const needsMore = OP_KIND[op] !== "none";
                                const hasValues = (op === "is" || op === "is_not") && ["enum", "bool", "plan"].includes(def.type);
                                return (
                                    <button
                                        key={op}
                                        type="button"
                                        className={itemCls(opKey === op)}
                                        onMouseEnter={() => (needsMore ? setOpKey(op) : setOpKey(null))}
                                        onClick={() => (needsMore && !hasValues ? onEdit({ field: def.field, op }, close) : pickOp(op))}
                                    >
                                        {t(`filter.builder.ops.${op}`, { n: "N" })}
                                        {hasValues ? <span className="text-muted-foreground">›</span> : null}
                                    </button>
                                );
                            })}
                        </div>
                    )}
                    {choices && kind === "one" && (
                        <div className="ml-1 max-h-80 min-w-[130px] overflow-y-auto rounded-md border bg-background p-1 shadow-lg">
                            {choices.length === 0 && <span className="px-3 py-1.5 text-sm text-muted-foreground">-</span>}
                            {choices.map((c) => (
                                <button key={String(c.value)} type="button" className={itemCls(false)} onClick={() => { onPick({ field: def.field, op: opKey, value: c.value, value2: "" }); close(); }}>
                                    {c.label}
                                </button>
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}

// 用户列表的筛选栏：搜索、预设、常用筛选、条件标签、匹配方式。
// 状态由父组件持有；每次变更调用 onChange({ conditions, mode }) 或 onReset()，由父组件刷新列表。
export function UserFilterBar({ t, search, onSearchChange, onSearch, conditions, mode, onChange, onReset, planOptions, currency, actions }) {
    const [editor, setEditor] = useState(null); // { condition, isNew }
    const [saveOpen, setSaveOpen] = useState(false);
    const [saveName, setSaveName] = useState("");
    const [views, setViews] = useState(() => readViews(safeStorage()));

    const hasFilter = search.trim() !== "" || conditions.length > 0;
    const planName = useMemo(() => {
        const m = new Map(planOptions.map((o) => [String(o.value), o.label]));
        return (id) => m.get(String(id)) ?? `#${id}`;
    }, [planOptions]);
    const ctx = useMemo(() => ({ t, planName, currency }), [t, planName, currency]);

    const commit = (nextConditions, nextMode = mode) => onChange({ conditions: nextConditions, mode: nextMode });

    // 值为空 / 不完整的条件一律不生效，也不会加入列表
    const saveCondition = (draft) => {
        if (!isComplete(draft)) return;
        const exists = conditions.some((c) => c.id === draft.id);
        commit(exists ? conditions.map((c) => (c.id === draft.id ? draft : c)) : [...conditions, draft]);
    };

    const persistViews = (next) => {
        setViews(next);
        writeViews(safeStorage(), next);
    };
    const saveView = () => {
        const next = addView(views, saveName, mode, conditions);
        if (next !== views) persistViews(next);
        setSaveOpen(false);
        setSaveName("");
    };

    return (
        <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-2">
                <Input
                    value={search}
                    onChange={(e) => onSearchChange(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && onSearch()}
                    placeholder={t("filter.email_search")}
                    className="h-9 w-[200px]"
                />
                <Button variant="outline" size="sm" onClick={onSearch}>{t("filter.sheet.apply")}</Button>
                <AddFilterMenu
                    label={t("filter.builder.add")}
                    planOptions={planOptions}
                    t={t}
                    onPick={(p) => saveCondition({ ...newCondition(p.field), ...p })}
                    onEdit={(p, close) => {
                        close();
                        setEditor({ condition: { ...newCondition(p.field), op: p.op }, isNew: true });
                    }}
                />
                {hasFilter && <Button variant="ghost" size="sm" onClick={onReset}>{t("filter.reset")}</Button>}
                <div className="ml-auto flex items-center gap-2">{actions}</div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
                <Button variant={!hasFilter ? "default" : "outline"} size="sm" onClick={onReset}>{t("filter.quick.all")}</Button>
                {PRESETS.map((p) => (
                    <Button
                        key={p.key}
                        variant={presetActive(conditions, p) ? "default" : "outline"}
                        size="sm"
                        onClick={() => commit(togglePreset(conditions, p))}
                    >
                        {t(`filter.quick.${p.key}`)}
                    </Button>
                ))}
                {views.length > 0 && <span className="ml-2 text-xs text-muted-foreground">{t("filter.builder.views")}</span>}
                {views.map((v) => (
                    <span key={v.name} className="inline-flex items-center rounded-md border bg-background text-sm">
                        <button type="button" className="px-2.5 py-1 hover:bg-accent" onClick={() => commit(instantiate(v.conditions), v.mode)}>{v.name}</button>
                        <button
                            type="button"
                            title={t("filter.builder.remove_view")}
                            aria-label={t("filter.builder.remove_view")}
                            className="border-l px-1.5 py-1 text-muted-foreground hover:text-destructive"
                            onClick={() => persistViews(removeView(views, v.name))}
                        >✕</button>
                    </span>
                ))}
            </div>

            {conditions.length > 0 && (
                <div className="flex flex-wrap items-center gap-2 rounded-md border bg-muted/20 px-3 py-2">
                    {conditions.map((c) => (
                        <span key={c.id} className="inline-flex items-center rounded-full border bg-background text-xs">
                            <button type="button" className="py-1 pl-3 pr-2 hover:underline" title={t("filter.builder.edit_condition")} onClick={() => setEditor({ condition: c, isNew: false })}>
                                {describeCondition(c, ctx)}
                            </button>
                            <button
                                type="button"
                                title={t("filter.builder.remove_condition")}
                                aria-label={t("filter.builder.remove_condition")}
                                className="rounded-full px-2 py-1 text-muted-foreground hover:text-destructive"
                                onClick={() => commit(conditions.filter((x) => x.id !== c.id))}
                            >✕</button>
                        </span>
                    ))}
                    {conditions.length > 1 && (
                        <span className="ml-1 inline-flex items-center gap-1 text-xs text-muted-foreground">
                            {t("filter.builder.match")}
                            <Button variant={mode === "all" ? "default" : "outline"} size="sm" onClick={() => commit(conditions, "all")}>{t("filter.builder.match_all")}</Button>
                            <Button variant={mode === "any" ? "default" : "outline"} size="sm" onClick={() => commit(conditions, "any")}>{t("filter.builder.match_any")}</Button>
                        </span>
                    )}
                    <Button variant="ghost" size="sm" className="ml-auto" onClick={() => { setSaveName(""); setSaveOpen(true); }}>
                        {t("filter.builder.save_view")}
                    </Button>
                </div>
            )}

            <ConditionDialog
                open={Boolean(editor)}
                onOpenChange={(o) => !o && setEditor(null)}
                initial={editor?.condition ?? newCondition()}
                isNew={editor?.isNew ?? true}
                planOptions={planOptions}
                currency={currency}
                onSave={saveCondition}
                ctx={ctx}
                t={t}
            />

            <Dialog open={saveOpen} onOpenChange={setSaveOpen} className="sm:max-w-md">
                <div className="p-6">
                    <h3 className="text-lg font-semibold">{t("filter.builder.save_view")}</h3>
                    <div className="mt-4 space-y-1.5">
                        <label className="text-sm font-medium">{t("filter.builder.view_name")}</label>
                        <Input
                            value={saveName}
                            maxLength={30}
                            autoFocus
                            onChange={(e) => setSaveName(e.target.value)}
                            onKeyDown={(e) => e.key === "Enter" && saveName.trim() && saveView()}
                            placeholder={t("filter.builder.view_placeholder")}
                        />
                    </div>
                    <div className="mt-6 flex justify-end gap-2">
                        <Button variant="outline" onClick={() => setSaveOpen(false)}>{t("filter.builder.cancel")}</Button>
                        <Button disabled={!saveName.trim()} onClick={saveView}>{t("filter.builder.confirm")}</Button>
                    </div>
                </div>
            </Dialog>
        </div>
    );
}
