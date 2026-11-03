// 用户管理的筛选：字段与运算符定义、条件 → 后端筛选表达式的转换、快捷预设、常用筛选的存取。
// 纯函数，不依赖 React，可以直接用 node 测试（见 userFilter.test.mjs）。
//
// 一个「条件」是 { id, field, op, value, value2 }：
//   field  字段名（对应后端筛选白名单里的列，或 invite_user.email 这样的关联字段）
//   op     运算符（见 operatorsFor）
//   value  / value2  用户填写的值（保持用户输入的单位，例如流量是 GB，换算在转换时做）
//
// 后端筛选表达式：
//   叶子     { id, value: "运算符:值" }（value 为数组时表示 IN）
//   条件组   { any: [...] } 表示 OR，{ all: [...] } 表示 AND，可以嵌套

export const DAY = 86400;
const GB = 1024 * 1024 * 1024;

// type：text 文本 / number 数字 / date 日期 / plan 订阅（选项来自套餐列表）/ enum 固定选项 / bool 开关
// nullable：该列可以为空，提供「为空 / 不为空」
// expiry：到期时间，额外提供「已过期 / 未到期」
export const FIELDS = [
    { field: "email", labelKey: "email", type: "text" },
    { field: "id", labelKey: "id", type: "number" },
    { field: "plan_id", labelKey: "plan_id", type: "plan" },
    { field: "transfer_enable", labelKey: "transfer_enable", type: "number", unit: "GB", scale: GB },
    { field: "total_used", labelKey: "total_used", type: "number", unit: "GB", scale: GB },
    { field: "online_count", labelKey: "online_count", type: "number" },
    { field: "device_limit", labelKey: "device_limit", type: "number", nullable: true },
    { field: "expired_at", labelKey: "expired_at", type: "date", nullable: true, expiry: true },
    { field: "created_at", labelKey: "created_at", type: "date" },
    { field: "last_login_at", labelKey: "last_login_at", type: "date", nullable: true },
    { field: "balance", labelKey: "balance", type: "number", unit: "currency", scale: 100 },
    { field: "commission_balance", labelKey: "commission_balance", type: "number", unit: "currency", scale: 100 },
    { field: "uuid", labelKey: "uuid", type: "text" },
    { field: "token", labelKey: "token", type: "text" },
    { field: "remarks", labelKey: "remark", type: "text", nullable: true },
    { field: "invite_user.email", labelKey: "inviter_email", type: "text" },
    { field: "invite_user_id", labelKey: "invite_user_id", type: "number", nullable: true },
    { field: "banned", labelKey: "banned", type: "enum", options: [{ value: "0", labelKey: "status.normal" }, { value: "1", labelKey: "status.banned" }] },
    { field: "is_admin", labelKey: "is_admin", type: "bool" },
    { field: "is_staff", labelKey: "is_staff", type: "bool" },
];

export function fieldDef(field) {
    return FIELDS.find((f) => f.field === field) || null;
}

// 每个运算符需要填几个值：none 不需要 / one 一个 / two 两个（范围）/ days 天数 / multi 多选
export const OP_KIND = {
    contains: "one", not_contains: "one", eq: "one", neq: "one", starts: "one", ends: "one",
    gt: "one", gte: "one", lt: "one", lte: "one", between: "two",
    on: "one", before: "one", after: "one",
    within_next: "days", within_last: "days", older_than: "days",
    past: "none", future: "none",
    is: "one", is_not: "one", in: "multi",
    empty: "none", not_empty: "none",
};

export function operatorsFor(def) {
    if (!def) return [];
    const nullable = def.nullable ? ["empty", "not_empty"] : [];
    switch (def.type) {
        case "text": return ["contains", "not_contains", "eq", "neq", "starts", "ends", ...nullable];
        case "number": return ["eq", "neq", "gt", "gte", "lt", "lte", "between", ...nullable];
        case "date": return ["on", "before", "after", "between", "within_next", "within_last", "older_than", ...(def.expiry ? ["past", "future"] : []), ...nullable];
        case "plan": return ["is", "is_not", "in", "empty", "not_empty"];
        case "enum": return ["is", "is_not"];
        case "bool": return ["is"];
        default: return [];
    }
}

let seq = 0;
export function uid() {
    seq += 1;
    return `c${Date.now().toString(36)}${seq}`;
}

export function defaultValue(def) {
    return def?.type === "bool" ? false : "";
}

export function newCondition(field = "email") {
    const def = fieldDef(field);
    return { id: uid(), field, op: operatorsFor(def)[0], value: defaultValue(def), value2: "" };
}

// ---------- 校验 ----------

function num(v) {
    if (v === "" || v === null || v === undefined) return NaN;
    const n = Number(v);
    return Number.isFinite(n) ? n : NaN;
}

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

// "YYYY-MM-DD" → 本地时区当天零点的秒级时间戳。直接 new Date("YYYY-MM-DD") 会按 UTC 零点解析，
// 在东八区会差 8 小时，「早于 / 晚于某天」的边界就不对了。
export function localDayStart(value) {
    if (!DATE_RE.test(String(value))) return NaN;
    const [y, m, d] = String(value).split("-").map(Number);
    return Math.floor(new Date(y, m - 1, d).getTime() / 1000);
}

function valueOk(def, v) {
    switch (def.type) {
        case "text": return String(v ?? "").trim() !== "";
        case "number": return !Number.isNaN(num(v));
        case "date": return !Number.isNaN(localDayStart(v));
        case "plan": return v !== "" && v !== null && v !== undefined && !Number.isNaN(num(v));
        case "enum": return def.options.some((o) => o.value === String(v));
        case "bool": return typeof v === "boolean";
        default: return false;
    }
}

export function isComplete(c) {
    const def = fieldDef(c?.field);
    if (!def || !c.op || !operatorsFor(def).includes(c.op)) return false;
    switch (OP_KIND[c.op]) {
        case "none": return true;
        case "one": return valueOk(def, c.value);
        case "two": return valueOk(def, c.value) && valueOk(def, c.value2);
        case "days": return Number.isInteger(num(c.value)) && num(c.value) >= 1;
        case "multi": return Array.isArray(c.value) && c.value.length > 0 && c.value.every((x) => !Number.isNaN(num(x)));
        default: return false;
    }
}

// ---------- 转换为后端筛选表达式 ----------

const leaf = (id, op, v) => ({ id, value: `${op}:${v}` });

function scaled(def, v) {
    const n = num(v);
    return def.scale && def.scale !== 1 ? Math.round(n * def.scale) : n;
}

function textExpr(id, op, raw) {
    const v = String(raw ?? "").trim();
    switch (op) {
        case "contains": return leaf(id, "like", v);
        case "not_contains": return leaf(id, "notlike", v);
        case "eq": return leaf(id, "eq", v);
        case "neq": return leaf(id, "neq", v);
        case "starts": return leaf(id, "prefix", v);
        case "ends": return leaf(id, "suffix", v);
        // 文本列「为空」既可能是 NULL 也可能是空串
        case "empty": return { any: [leaf(id, "null", 1), leaf(id, "eq", "")] };
        case "not_empty": return { all: [leaf(id, "notnull", 1), leaf(id, "neq", "")] };
        default: return null;
    }
}

function numberExpr(def, c) {
    const id = def.field;
    switch (c.op) {
        case "eq": case "neq": case "gt": case "gte": case "lt": case "lte":
            return leaf(id, c.op, scaled(def, c.value));
        case "between": {
            const a = scaled(def, c.value);
            const b = scaled(def, c.value2);
            return { all: [leaf(id, "gte", Math.min(a, b)), leaf(id, "lte", Math.max(a, b))] };
        }
        case "empty": return leaf(id, "null", 1);
        case "not_empty": return leaf(id, "notnull", 1);
        default: return null;
    }
}

function dateExpr(def, c, nowSec) {
    const id = def.field;
    const days = num(c.value);
    // 0 不是合法的时间戳：为空 / 不为空要把历史数据里的 0 也算进去
    const positive = leaf(id, "gt", 0);
    switch (c.op) {
        case "on": {
            const s = localDayStart(c.value);
            return { all: [leaf(id, "gte", s), leaf(id, "lt", s + DAY)] };
        }
        case "before": return leaf(id, "lt", localDayStart(c.value));
        case "after": return leaf(id, "gte", localDayStart(c.value) + DAY);
        case "between": {
            const a = localDayStart(c.value);
            const b = localDayStart(c.value2);
            return { all: [leaf(id, "gte", Math.min(a, b)), leaf(id, "lt", Math.max(a, b) + DAY)] };
        }
        case "within_next": return { all: [leaf(id, "gt", nowSec), leaf(id, "lte", nowSec + days * DAY)] };
        case "within_last": return { all: [leaf(id, "gte", nowSec - days * DAY), leaf(id, "lte", nowSec)] };
        case "older_than": return def.nullable ? { all: [leaf(id, "lt", nowSec - days * DAY), positive] } : leaf(id, "lt", nowSec - days * DAY);
        case "past": return { all: [leaf(id, "lt", nowSec), positive] };
        case "future": return leaf(id, "gt", nowSec);
        case "empty": return { any: [leaf(id, "null", 1), leaf(id, "eq", 0)] };
        case "not_empty": return { all: [leaf(id, "notnull", 1), positive] };
        default: return null;
    }
}

// 单个条件 → 后端筛选表达式；条件不完整返回 null。nowSec 注入便于测试。
export function conditionToExpr(c, nowSec = Math.floor(Date.now() / 1000)) {
    if (!isComplete(c)) return null;
    const def = fieldDef(c.field);
    const id = def.field;
    switch (def.type) {
        case "text": return textExpr(id, c.op, c.value);
        case "number": return numberExpr(def, c);
        case "date": return dateExpr(def, c, nowSec);
        case "plan":
            switch (c.op) {
                case "is": return leaf(id, "eq", num(c.value));
                case "is_not": return leaf(id, "neq", num(c.value));
                case "in": return { id, value: c.value.map(Number) };
                case "empty": return leaf(id, "null", 1);
                case "not_empty": return leaf(id, "notnull", 1);
                default: return null;
            }
        case "enum": return leaf(id, c.op === "is" ? "eq" : "neq", String(c.value));
        case "bool": return leaf(id, "eq", c.value ? 1 : 0);
        default: return null;
    }
}

// 页面状态 → 发给后端的 filter 数组。mode 为 all 时各条件 AND，any 时整体作为一个 OR 组；邮箱搜索始终是额外的 AND 条件。
export function buildFilter({ conditions = [], mode = "all", search = "", nowSec } = {}) {
    const exprs = conditions.map((c) => conditionToExpr(c, nowSec)).filter(Boolean);
    const out = [];
    if (mode === "any" && exprs.length > 1) out.push({ any: exprs });
    else out.push(...exprs);
    const s = String(search || "").trim();
    if (s) out.push({ id: "email", value: `like:${s}` });
    return out;
}

// ---------- 快捷预设 ----------

export const PRESETS = [
    { key: "online", conditions: [{ field: "online_count", op: "gt", value: "0" }] },
    { key: "no_plan", conditions: [{ field: "plan_id", op: "empty" }] },
    { key: "expired", conditions: [{ field: "expired_at", op: "past" }] },
    { key: "expiring", conditions: [{ field: "expired_at", op: "within_next", value: "7" }] },
    { key: "banned", conditions: [{ field: "banned", op: "is", value: "1" }] },
];

// 条件的内容签名（忽略 id），用来判断两组条件是否相同。
export function signature(conditions) {
    return JSON.stringify(
        (conditions || []).map((c) => [c.field, c.op, Array.isArray(c.value) ? [...c.value].map(String).sort() : String(c.value ?? ""), String(c.value2 ?? "")]),
    );
}

export function instantiate(items) {
    return (items || []).map((c) => ({ id: uid(), field: c.field, op: c.op, value: c.value ?? defaultValue(fieldDef(c.field)), value2: c.value2 ?? "" }));
}

// 预设是可叠加的开关：它的每个条件都已在当前条件里，就算「已选中」。
export function presetActive(conditions, preset) {
    const have = new Set((conditions || []).map((c) => signature([c])));
    return instantiate(preset.conditions).every((c) => have.has(signature([c])));
}

// 点击预设：已选中则移除它的条件，否则追加（已有的相同条件不重复添加）。其他条件保持不变，可以同时选多个预设。
export function togglePreset(conditions, preset) {
    const items = instantiate(preset.conditions);
    const sigs = new Set(items.map((c) => signature([c])));
    const current = conditions || [];
    if (presetActive(current, preset)) return current.filter((c) => !sigs.has(signature([c])));
    const have = new Set(current.map((c) => signature([c])));
    return [...current, ...items.filter((c) => !have.has(signature([c])))];
}

// ---------- 常用筛选（存在浏览器本地） ----------

export const VIEWS_KEY = "xboard_user_filter_views_v1";
export const MAX_VIEWS = 12;
const MAX_VIEW_CONDITIONS = 20;

function validView(v) {
    if (!v || typeof v.name !== "string" || !v.name.trim() || v.name.length > 30) return false;
    if (v.mode !== "all" && v.mode !== "any") return false;
    if (!Array.isArray(v.conditions) || v.conditions.length === 0 || v.conditions.length > MAX_VIEW_CONDITIONS) return false;
    return v.conditions.every((c) => c && fieldDef(c.field) && operatorsFor(fieldDef(c.field)).includes(c.op));
}

export function readViews(storage) {
    try {
        const raw = storage?.getItem(VIEWS_KEY);
        if (!raw) return [];
        const arr = JSON.parse(raw);
        return Array.isArray(arr) ? arr.filter(validView).slice(0, MAX_VIEWS) : [];
    } catch {
        return [];
    }
}

export function writeViews(storage, views) {
    try {
        storage.setItem(VIEWS_KEY, JSON.stringify(views));
        return true;
    } catch {
        return false;
    }
}

// 保存一个常用筛选：同名覆盖，超过上限时丢弃最早的。返回新数组；条件不合法或名字为空时返回原数组。
export function addView(views, name, mode, conditions) {
    const trimmed = String(name || "").trim().slice(0, 30);
    const clean = (conditions || []).filter(isComplete).map(({ field, op, value, value2 }) => ({ field, op, value, value2: value2 ?? "" }));
    const view = { name: trimmed, mode: mode === "any" ? "any" : "all", conditions: clean };
    if (!validView(view)) return views;
    return [...views.filter((v) => v.name !== trimmed), view].slice(-MAX_VIEWS);
}

export function removeView(views, name) {
    return views.filter((v) => v.name !== name);
}

// ---------- 文字描述（条件标签） ----------

// ctx：{ t, planName(id), currency }，t 是翻译函数（key, params）。
export function describeCondition(c, ctx) {
    const def = fieldDef(c.field);
    if (!def) return "";
    const { t } = ctx;
    const label = t(`filter.fields.${def.labelKey}`);
    // 只有「N 天」类运算符的文案里有 {{n}}
    const op = OP_KIND[c.op] === "days" ? t(`filter.builder.ops.${c.op}`, { n: c.value }) : t(`filter.builder.ops.${c.op}`);
    const unit = def.unit === "currency" ? ctx.currency || "" : def.unit || "";
    const fmt = (v) => {
        switch (def.type) {
            case "plan": return ctx.planName ? ctx.planName(v) : String(v);
            case "enum": {
                const o = def.options.find((x) => x.value === String(v));
                return o ? t(`filter.${o.labelKey}`) : String(v);
            }
            case "bool": return t(v ? "filter.boolean.true" : "filter.boolean.false");
            default: return unit ? `${v} ${unit}` : String(v);
        }
    };
    // 开关类字段直接写「管理员 是 / 管理员 否」，不需要运算符。
    if (def.type === "bool") return `${label} ${fmt(c.value)}`;
    switch (OP_KIND[c.op]) {
        case "none":
        case "days": return `${label} ${op}`;
        case "two": return `${label} ${op} ${fmt(c.value)} ~ ${fmt(c.value2)}`;
        case "multi": return `${label} ${op} ${c.value.map(fmt).join("、")}`;
        default: return `${label} ${op} ${fmt(c.value)}`;
    }
}
