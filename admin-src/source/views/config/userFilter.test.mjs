// 运行：node --test views/config/userFilter.test.mjs（建议同时用 TZ=Asia/Shanghai 和 TZ=UTC 各跑一次）
import test from "node:test";
import assert from "node:assert/strict";
import {
    DAY, FIELDS, OP_KIND, PRESETS, MAX_VIEWS, VIEWS_KEY, fieldDef, operatorsFor, isComplete, conditionToExpr, buildFilter,
    localDayStart, presetActive, togglePreset, instantiate, addView, removeView, readViews, writeViews, describeCondition, newCondition,
} from "./userFilter.js";

const NOW = 1_800_000_000;
const c = (field, op, value = "", value2 = "") => ({ id: "x", field, op, value, value2 });
const expr = (field, op, value, value2) => conditionToExpr(c(field, op, value, value2), NOW);
const L = (id, value) => ({ id, value });

test("每个字段都有运算符，每个运算符都声明了取值个数", () => {
    for (const f of FIELDS) {
        const ops = operatorsFor(f);
        assert.ok(ops.length > 0, `${f.field} 没有运算符`);
        for (const op of ops) assert.ok(OP_KIND[op], `${f.field}.${op} 缺少 OP_KIND`);
    }
    // 可为空的列才提供「为空 / 不为空」；不可为空的列不提供（它们永远不为空，选了没有意义）
    assert.ok(operatorsFor(fieldDef("remarks")).includes("empty"));
    assert.ok(operatorsFor(fieldDef("plan_id")).includes("empty"));
    assert.ok(!operatorsFor(fieldDef("email")).includes("empty"));
    assert.ok(!operatorsFor(fieldDef("created_at")).includes("empty"));
    // 「已过期 / 未到期」只给到期时间
    assert.ok(operatorsFor(fieldDef("expired_at")).includes("past"));
    assert.ok(!operatorsFor(fieldDef("created_at")).includes("past"));
});

test("文本运算符", () => {
    assert.deepEqual(expr("email", "contains", " a:b "), L("email", "like:a:b")); // 去首尾空格，内容里的冒号保留
    assert.deepEqual(expr("email", "not_contains", "x"), L("email", "notlike:x"));
    assert.deepEqual(expr("email", "eq", "a@b.c"), L("email", "eq:a@b.c"));
    assert.deepEqual(expr("email", "neq", "a@b.c"), L("email", "neq:a@b.c"));
    assert.deepEqual(expr("email", "starts", "ab"), L("email", "prefix:ab"));
    assert.deepEqual(expr("email", "ends", "163.com"), L("email", "suffix:163.com"));
    assert.deepEqual(expr("invite_user.email", "contains", "x"), L("invite_user.email", "like:x"));
    // 文本列「为空」= NULL 或空串；「不为空」= 非 NULL 且非空串
    assert.deepEqual(expr("remarks", "empty"), { any: [L("remarks", "null:1"), L("remarks", "eq:")] });
    assert.deepEqual(expr("remarks", "not_empty"), { all: [L("remarks", "notnull:1"), L("remarks", "neq:")] });
});

test("数字运算符与单位换算", () => {
    assert.deepEqual(expr("id", "gt", "5"), L("id", "gt:5"));
    assert.deepEqual(expr("online_count", "gte", "2"), L("online_count", "gte:2"));
    assert.deepEqual(expr("online_count", "neq", "0"), L("online_count", "neq:0"));
    assert.deepEqual(expr("transfer_enable", "gt", "1.5"), L("transfer_enable", `gt:${1.5 * 1024 ** 3}`)); // GB → 字节
    assert.deepEqual(expr("balance", "gte", "12.34"), L("balance", "gte:1234")); // 元 → 分，四舍五入
    assert.deepEqual(expr("balance", "eq", "0.1"), L("balance", "eq:10"));
    // 范围：写反了也按从小到大处理
    assert.deepEqual(expr("online_count", "between", "1", "3"), { all: [L("online_count", "gte:1"), L("online_count", "lte:3")] });
    assert.deepEqual(expr("online_count", "between", "9", "2"), { all: [L("online_count", "gte:2"), L("online_count", "lte:9")] });
    assert.deepEqual(expr("device_limit", "empty"), L("device_limit", "null:1")); // 设备数上限为空 = 不限制
    assert.deepEqual(expr("invite_user_id", "not_empty"), L("invite_user_id", "notnull:1"));
});

test("日期运算符（按本地时区的整天）", () => {
    const d = localDayStart("2026-10-20");
    assert.equal(d, Math.floor(new Date(2026, 9, 20).getTime() / 1000));
    assert.deepEqual(expr("expired_at", "on", "2026-10-20"), { all: [L("expired_at", `gte:${d}`), L("expired_at", `lt:${d + DAY}`)] });
    assert.deepEqual(expr("expired_at", "before", "2026-10-20"), L("expired_at", `lt:${d}`));
    assert.deepEqual(expr("expired_at", "after", "2026-10-20"), L("expired_at", `gte:${d + DAY}`));
    const e = localDayStart("2026-10-25");
    assert.deepEqual(expr("created_at", "between", "2026-10-20", "2026-10-25"), { all: [L("created_at", `gte:${d}`), L("created_at", `lt:${e + DAY}`)] });
    assert.deepEqual(expr("created_at", "between", "2026-10-25", "2026-10-20"), { all: [L("created_at", `gte:${d}`), L("created_at", `lt:${e + DAY}`)] });
    assert.equal(localDayStart("2026-13-40") > 0, true); // 非法日期会被 Date 滚动到下个月，但格式合法
    assert.ok(Number.isNaN(localDayStart("2026/10/20")));
    assert.ok(Number.isNaN(localDayStart("")));
});

test("相对日期", () => {
    assert.deepEqual(expr("expired_at", "within_next", "7"), { all: [L("expired_at", `gt:${NOW}`), L("expired_at", `lte:${NOW + 7 * DAY}`)] });
    assert.deepEqual(expr("created_at", "within_last", "3"), { all: [L("created_at", `gte:${NOW - 3 * DAY}`), L("created_at", `lte:${NOW}`)] });
    // 可为空的日期「N 天前」要排除 0 / 空值；注册时间不会为空，不需要
    assert.deepEqual(expr("last_login_at", "older_than", "30"), { all: [L("last_login_at", `lt:${NOW - 30 * DAY}`), L("last_login_at", "gt:0")] });
    assert.deepEqual(expr("created_at", "older_than", "30"), L("created_at", `lt:${NOW - 30 * DAY}`));
    assert.deepEqual(expr("expired_at", "past"), { all: [L("expired_at", `lt:${NOW}`), L("expired_at", "gt:0")] }); // 永久有效的不算已过期
    assert.deepEqual(expr("expired_at", "future"), L("expired_at", `gt:${NOW}`));
    // 日期「为空」包括历史数据里的 0
    assert.deepEqual(expr("expired_at", "empty"), { any: [L("expired_at", "null:1"), L("expired_at", "eq:0")] });
    assert.deepEqual(expr("last_login_at", "not_empty"), { all: [L("last_login_at", "notnull:1"), L("last_login_at", "gt:0")] });
});

test("订阅、选项、开关", () => {
    assert.deepEqual(expr("plan_id", "is", "3"), L("plan_id", "eq:3"));
    assert.deepEqual(expr("plan_id", "is_not", "3"), L("plan_id", "neq:3")); // 后端 neq 对空值安全，没有套餐的用户也会被包含
    assert.deepEqual(expr("plan_id", "in", ["1", "2"]), { id: "plan_id", value: [1, 2] });
    assert.deepEqual(expr("plan_id", "empty"), L("plan_id", "null:1"));
    assert.deepEqual(expr("plan_id", "not_empty"), L("plan_id", "notnull:1"));
    assert.deepEqual(expr("banned", "is", "1"), L("banned", "eq:1"));
    assert.deepEqual(expr("banned", "is_not", "1"), L("banned", "neq:1"));
    assert.deepEqual(expr("is_admin", "is", true), L("is_admin", "eq:1"));
    assert.deepEqual(expr("is_staff", "is", false), L("is_staff", "eq:0"));
});

test("不完整的条件不会生成表达式", () => {
    assert.equal(expr("email", "contains", ""), null);
    assert.equal(expr("email", "contains", "   "), null);
    assert.equal(expr("id", "gt", ""), null);
    assert.equal(expr("id", "gt", "abc"), null);
    assert.equal(expr("id", "between", "1", ""), null);
    assert.equal(expr("expired_at", "before", "not-a-date"), null);
    assert.equal(expr("expired_at", "within_next", "0"), null); // 天数至少 1
    assert.equal(expr("expired_at", "within_next", "1.5"), null);
    assert.equal(expr("plan_id", "in", []), null);
    assert.equal(expr("plan_id", "is", ""), null);
    assert.equal(expr("is_admin", "is", ""), null);
    assert.equal(expr("banned", "is", "2"), null);
    assert.equal(expr("email", "empty"), null); // email 不可为空，没有这个运算符
    assert.equal(expr("nope", "eq", "1"), null);
    assert.equal(isComplete(c("expired_at", "past")), true);
});

test("buildFilter：全部 / 任一 / 搜索", () => {
    const a = c("online_count", "gt", "0");
    const b = c("plan_id", "empty");
    const bad = c("email", "contains", "");
    assert.deepEqual(buildFilter({ conditions: [a, b], mode: "all", nowSec: NOW }), [L("online_count", "gt:0"), L("plan_id", "null:1")]);
    assert.deepEqual(buildFilter({ conditions: [a, b], mode: "any", nowSec: NOW }), [{ any: [L("online_count", "gt:0"), L("plan_id", "null:1")] }]);
    // 只有一个条件时「任一」不需要包一层
    assert.deepEqual(buildFilter({ conditions: [a], mode: "any", nowSec: NOW }), [L("online_count", "gt:0")]);
    // 不完整的条件被忽略；搜索始终是额外的 AND，不会被并进 OR 组
    assert.deepEqual(buildFilter({ conditions: [a, bad, b], mode: "any", search: " x@y ", nowSec: NOW }),
        [{ any: [L("online_count", "gt:0"), L("plan_id", "null:1")] }, L("email", "like:x@y")]);
    assert.deepEqual(buildFilter({ search: "a:b" }), [L("email", "like:a:b")]);
    assert.deepEqual(buildFilter({}), []);
    // 「任一」里带范围条件时，范围内部仍是 AND
    const r = c("online_count", "between", "1", "3");
    assert.deepEqual(buildFilter({ conditions: [r, b], mode: "any", nowSec: NOW }),
        [{ any: [{ all: [L("online_count", "gte:1"), L("online_count", "lte:3")] }, L("plan_id", "null:1")] }]);
});

test("预设是可叠加的开关", () => {
    const [online, noPlan, expired] = PRESETS;
    for (const p of PRESETS) {
        const inst = instantiate(p.conditions);
        assert.ok(inst.every(isComplete), `${p.key} 的条件必须完整`);
        assert.equal(presetActive(inst, p), true);
        assert.equal(presetActive([], p), false);
    }
    // 点击：追加；再点击：移除；其他条件保持不变
    const mine = [c("remarks", "contains", "vip")];
    let cur = togglePreset(mine, online);
    assert.equal(cur.length, 2);
    assert.equal(presetActive(cur, online), true);
    cur = togglePreset(cur, noPlan); // 两个预设可以同时选中
    assert.equal(cur.length, 3);
    assert.equal(presetActive(cur, online) && presetActive(cur, noPlan), true);
    assert.equal(presetActive(cur, expired), false);
    cur = togglePreset(cur, online);
    assert.equal(presetActive(cur, online), false);
    assert.equal(presetActive(cur, noPlan), true);
    assert.deepEqual(cur.map((x) => x.field), ["remarks", "plan_id"]);
    // 重复点击不会重复添加
    assert.equal(togglePreset(togglePreset([], online), online).length, 0);
    // 用户改过值就不再算选中
    const edited = instantiate(online.conditions);
    edited[0].value = "1";
    assert.equal(presetActive(edited, online), false);
    // 预设转换后的语义：已过期 = 已过且非永久；7 天内到期 = 未来 7 天
    assert.deepEqual(conditionToExpr(instantiate(PRESETS[2].conditions)[0], NOW), { all: [L("expired_at", `lt:${NOW}`), L("expired_at", "gt:0")] });
});

function fakeStorage(initial) {
    const m = new Map(initial ? [[VIEWS_KEY, initial]] : []);
    return { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => m.set(k, v), dump: () => m.get(VIEWS_KEY) };
}

test("常用筛选：保存、覆盖、删除、上限", () => {
    const cond = [c("online_count", "gt", "0")];
    let views = addView([], " 在线用户 ", "all", cond);
    assert.equal(views.length, 1);
    assert.equal(views[0].name, "在线用户");
    assert.ok(!("id" in views[0].conditions[0]), "不保存条件 id");
    // 同名覆盖
    views = addView(views, "在线用户", "any", [c("plan_id", "empty")]);
    assert.equal(views.length, 1);
    assert.equal(views[0].mode, "any");
    // 名字为空 / 没有有效条件：保持原样
    assert.equal(addView(views, "  ", "all", cond), views);
    assert.equal(addView(views, "x", "all", [c("email", "contains", "")]), views);
    assert.equal(addView(views, "x", "all", []), views);
    // 删除
    assert.deepEqual(removeView(views, "在线用户"), []);
    // 超过上限丢弃最早的
    let many = [];
    for (let i = 0; i < MAX_VIEWS + 3; i++) many = addView(many, `v${i}`, "all", cond);
    assert.equal(many.length, MAX_VIEWS);
    assert.equal(many[0].name, "v3");
    assert.equal(many.at(-1).name, `v${MAX_VIEWS + 2}`);
});

test("常用筛选：读写对坏数据和不可用的存储是安全的", () => {
    const st = fakeStorage();
    assert.deepEqual(readViews(st), []);
    const views = addView([], "a", "all", [c("online_count", "gt", "0")]);
    assert.equal(writeViews(st, views), true);
    assert.deepEqual(readViews(st), views);
    // 脏数据：非 JSON、不是数组、字段或运算符不存在、名字超长、模式不对
    assert.deepEqual(readViews(fakeStorage("not json")), []);
    assert.deepEqual(readViews(fakeStorage('{"a":1}')), []);
    const junk = [
        { name: "ok", mode: "all", conditions: [{ field: "online_count", op: "gt", value: "0" }] },
        { name: "坏字段", mode: "all", conditions: [{ field: "password", op: "eq", value: "x" }] },
        { name: "坏运算符", mode: "all", conditions: [{ field: "email", op: "drop_table", value: "x" }] },
        { name: "x".repeat(31), mode: "all", conditions: [{ field: "id", op: "eq", value: "1" }] },
        { name: "坏模式", mode: "xor", conditions: [{ field: "id", op: "eq", value: "1" }] },
        { name: "无条件", mode: "all", conditions: [] },
        null,
    ];
    assert.deepEqual(readViews(fakeStorage(JSON.stringify(junk))).map((v) => v.name), ["ok"]);
    // 存储抛异常（隐私模式 / 配额满）时不崩溃
    const broken = { getItem() { throw new Error("denied"); }, setItem() { throw new Error("full"); } };
    assert.deepEqual(readViews(broken), []);
    assert.equal(writeViews(broken, views), false);
    assert.deepEqual(readViews(undefined), []);
});

test("条件标签文字", () => {
    const t = (key, p) => (p && p.n !== undefined && p.n !== "" ? `${key}(${p.n})` : key);
    const ctx = { t, planName: (id) => `套餐${id}`, currency: "¥" };
    assert.equal(describeCondition(c("online_count", "gt", "0"), ctx), "filter.fields.online_count filter.builder.ops.gt 0");
    assert.equal(describeCondition(c("plan_id", "empty"), ctx), "filter.fields.plan_id filter.builder.ops.empty");
    assert.equal(describeCondition(c("plan_id", "in", ["1", "2"]), ctx), "filter.fields.plan_id filter.builder.ops.in 套餐1、套餐2");
    assert.equal(describeCondition(c("expired_at", "within_next", "7"), ctx), "filter.fields.expired_at filter.builder.ops.within_next(7)");
    assert.equal(describeCondition(c("balance", "gte", "10", ""), ctx), "filter.fields.balance filter.builder.ops.gte 10 ¥");
    assert.equal(describeCondition(c("transfer_enable", "between", "1", "5"), ctx), "filter.fields.transfer_enable filter.builder.ops.between 1 GB ~ 5 GB");
    assert.equal(describeCondition(c("banned", "is", "1"), ctx), "filter.fields.banned filter.builder.ops.is filter.status.banned");
    assert.equal(describeCondition(c("is_admin", "is", true), ctx), "filter.fields.is_admin filter.boolean.true");
    assert.equal(describeCondition(c("nope", "eq", "1"), ctx), "");
});

test("newCondition 使用该字段的第一个运算符和合适的默认值", () => {
    const a = newCondition("email");
    assert.equal(a.op, "contains");
    assert.equal(a.value, "");
    const b = newCondition("is_admin");
    assert.equal(b.value, false);
    assert.equal(newCondition("expired_at").op, "on");
    assert.notEqual(newCondition("id").id, newCondition("id").id);
});
