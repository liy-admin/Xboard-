import React, { useCallback, useEffect, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getCleanupStatus, saveCleanupSettings, runCleanup } from "../../api/order.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Switch } from "../../components/Switch.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";

function formatTime(seconds) {
    if (!seconds) return "-";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// Input / Button 自带 w-full、h-9 等尺寸类且没有"后者覆盖前者"，紧凑尺寸必须用内联样式才可靠。
const COMPACT_BTN = { height: "1.75rem", fontSize: "0.75rem", paddingLeft: "0.75rem", paddingRight: "0.75rem" };
const COMPACT_INPUT = { width: "4.5rem", height: "1.75rem" };

function errorText(e, fallback) {
    return e?.response?.data?.message || e?.message || fallback;
}

// 已取消订单自动清理：紧凑的单行控件条 + 一行状态。
// 规则（后端保证）：只删「已取消」且取消时间超过 N 天的订单，其他状态的订单不受影响。
export function OrderCleanupCard({ onCleaned }) {
    const { t } = useTranslation("order");
    const [status, setStatus] = useState(null);
    const [enable, setEnable] = useState(false);
    const [days, setDays] = useState("30");
    const [saving, setSaving] = useState(false);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [msg, setMsg] = useState(null); // { ok: boolean, text: string }

    const load = useCallback(async () => {
        try {
            const res = await getCleanupStatus();
            const data = res?.data;
            if (data) {
                setStatus(data);
                setEnable(Boolean(data.enable));
                setDays(String(data.days));
            }
        } catch (e) {
            setMsg({ ok: false, text: errorText(e, t("cleanup.loadFailed", "读取清理设置失败")) });
        }
    }, [t]);

    useEffect(() => {
        load();
    }, [load]);

    const max = status?.max ?? 3650;
    const min = status?.min ?? 1;
    const daysNum = Number(days);
    const daysValid = Number.isInteger(daysNum) && daysNum >= min && daysNum <= max;
    const dirty = status ? enable !== Boolean(status.enable) || String(daysNum) !== String(status.days) : false;

    const handleSave = async () => {
        if (!daysValid) {
            setMsg({ ok: false, text: t("cleanup.daysInvalid", { min, max }, "保留天数必须是 {{min}} 到 {{max}} 之间的整数") });
            return;
        }
        setSaving(true);
        setMsg(null);
        try {
            await saveCleanupSettings({ enable, days: daysNum });
            await load();
            setMsg({ ok: true, text: t("cleanup.saved", "已保存") });
        } catch (e) {
            setMsg({ ok: false, text: errorText(e, t("cleanup.saveFailed", "保存失败")) });
        } finally {
            setSaving(false);
        }
    };

    const handleRun = async () => {
        setMsg(null);
        try {
            const res = await runCleanup({ days: status.days });
            const deleted = res?.data?.deleted ?? 0;
            setMsg({ ok: true, text: t("cleanup.runDone", { count: deleted }, "已删除 {{count}} 笔订单") });
            await load();
            onCleaned?.();
        } catch (e) {
            setMsg({ ok: false, text: errorText(e, t("cleanup.runFailed", "清理失败")) });
            throw e;
        }
    };

    const last = status?.last;
    const pending = status?.pending ?? 0;

    return (
        <div className="mb-4 rounded-lg border px-3 py-2.5">
            <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
                <div className="min-w-0 flex-1 basis-64">
                    <div className="flex flex-wrap items-baseline gap-x-2">
                        <span className="text-sm font-medium">{t("cleanup.title", "已取消订单自动清理")}</span>
                        <span className="text-xs text-muted-foreground">
                            {t("cleanup.description", "只删「已取消」且超过设定天数的订单，其他状态不受影响。")}
                        </span>
                    </div>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                    <span className="text-xs text-muted-foreground">{t("cleanup.days", "保留天数")}</span>
                    <Input
                        type="number"
                        min={min}
                        max={max}
                        style={COMPACT_INPUT}
                        className="px-2 py-0 text-sm"
                        value={days}
                        onChange={(e) => setDays(e.target.value)}
                    />
                    <Button size="sm" style={COMPACT_BTN} onClick={handleSave} loading={saving} disabled={saving || !dirty || !daysValid}>
                        {t("cleanup.save", "保存")}
                    </Button>
                    <Button
                        size="sm"
                        variant="outline"
                        style={COMPACT_BTN}
                        onClick={() => setConfirmOpen(true)}
                        disabled={!status || dirty || pending === 0}
                        title={dirty ? t("cleanup.unsavedHint", "请先保存设置") : undefined}
                    >
                        {t("cleanup.run", "立即清理")}
                    </Button>
                    <Switch checked={enable} onCheckedChange={setEnable} aria-label={t("cleanup.enable", "启用自动清理")} />
                </div>
            </div>

            {(status || msg) && (
                <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
                    {status && (
                        <>
                            <span>
                                {enable && !dirty
                                    ? t("cleanup.nextAuto", { time: status.auto_at }, "每天 {{time}} 自动执行")
                                    : t("cleanup.autoOff", "自动清理未开启，只能手动清理")}
                            </span>
                            <span>
                                {t("cleanup.pending", { count: pending, days: status.days }, "当前有 {{count}} 笔已取消超过 {{days}} 天的订单待清理")}
                            </span>
                            <span>
                                {last
                                    ? t(
                                          "cleanup.last",
                                          {
                                              time: formatTime(last.at),
                                              count: last.deleted,
                                              source: last.source === "manual" ? t("cleanup.sourceManual", "手动") : t("cleanup.sourceDaily", "自动"),
                                          },
                                          "上次清理：{{time}}（{{source}}），删除 {{count}} 笔"
                                      )
                                    : t("cleanup.lastNone", "尚未执行过清理")}
                                {last?.truncated ? ` · ${t("cleanup.truncated", "数量较多，剩余部分将在下次继续")}` : ""}
                            </span>
                            {last?.error ? <span className="text-red-600 dark:text-red-400">{last.error}</span> : null}
                        </>
                    )}
                    {msg && (
                        <span className={msg.ok ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"}>
                            {msg.ok ? "✓ " : "✗ "}
                            {msg.text}
                        </span>
                    )}
                </div>
            )}

            <ConfirmDialog
                open={confirmOpen}
                onOpenChange={setConfirmOpen}
                title={t("cleanup.runConfirmTitle", "立即清理已取消订单？")}
                description={t(
                    "cleanup.runConfirmDesc",
                    { count: pending, days: status?.days },
                    "将永久删除 {{count}} 笔已取消超过 {{days}} 天的订单，删除后无法恢复。"
                )}
                confirmLabel={t("cleanup.runConfirmBtn", "确认删除")}
                onConfirm={handleRun}
            />
        </div>
    );
}
