import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { getCurrencySymbol, useCurrencySymbol } from "../../store/config.js";
import { navigate } from "../../router/index.js";
import {
    fetchUsers,
    updateUser,
    resetUserSecret,
    generateUsers,
    destroyUser,
    sendUserMail,
    dumpUserCSV,
    banUser,
    resetUserTraffic,
    fetchUserTrafficHistory,
    getOnlineUsers,
} from "../../api/user.js";
import { fetchPlans } from "../../api/plan.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { UserFilterBar } from "./UserFilterBar.jsx";
import { buildFilter as buildUserFilter } from "./userFilter.js";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/Tabs.jsx";
import { Tooltip, TooltipTrigger, TooltipContent } from "../../components/Tooltip.jsx";
import { Card, CardHeader, CardTitle, CardContent } from "../../components/Card.jsx";

const PAGE_SIZE = 20;
const GB = 1024 * 1024 * 1024;

function formatBytes(value) {
    if (value == null) return "-";
    if (value < 1024) return `${value.toFixed(0)} B`;
    if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
    if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
    if (value < GB * 1024) return `${(value / GB).toFixed(1)} GB`;
    return `${(value / (GB * 1024)).toFixed(2)} TB`;
}

function formatMoney(yuan) {
    // 接口返回的是「元」（后端已除以 100），保存时才乘回分。
    const value = typeof yuan === "number" ? yuan : 0;
    return `${value.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ",")}${getCurrencySymbol()}`;
}

function formatDateTime(seconds) {
    if (!seconds) return "-";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function toDatetimeLocal(seconds) {
    if (!seconds) return "";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function fromDatetimeLocal(value) {
    if (!value) return null;
    return Math.floor(new Date(value).getTime() / 1000);
}

function copyText(text) {
    if (!text) return Promise.resolve();
    return navigator.clipboard?.writeText(String(text)).catch(() => {});
}

function SendMailDialog({ open, onOpenChange, hasSelection, selectedCount, hasFilter, onSend, t }) {
    const [scope, setScope] = useState("all");
    const [subject, setSubject] = useState("");
    const [content, setContent] = useState("");
    const [sending, setSending] = useState(false);

    useEffect(() => {
        if (open) setScope(hasSelection ? "selected" : hasFilter ? "filtered" : "all");
    }, [open, hasSelection, hasFilter]);

    const handleSend = async () => {
        if (!subject.trim() || !content.trim()) return;
        setSending(true);
        try {
            await onSend({ scope, subject, content });
            setSubject("");
            setContent("");
            onOpenChange(false);
        } finally {
            setSending(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-xl">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t("send_mail.title")}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{t("send_mail.description")}</p>
                <div className="mt-4 space-y-3">
                    <Select value={scope} onValueChange={setScope}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="selected" disabled={!hasSelection}>{`${t("filter.selected", { count: selectedCount })}`}</SelectItem>
                            <SelectItem value="filtered" disabled={!hasFilter}>filtered</SelectItem>
                            <SelectItem value="all">all</SelectItem>
                        </SelectContent>
                    </Select>
                    <div>
                        <label className="text-sm font-medium">{t("send_mail.subject")}</label>
                        <Input className="mt-1" value={subject} onChange={(e) => setSubject(e.target.value)} />
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("send_mail.content")}</label>
                        <Textarea className="mt-1 min-h-[200px]" value={content} onChange={(e) => setContent(e.target.value)} />
                    </div>
                </div>
                <div className="mt-6 flex justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={sending}>{t("common:cancel", "取消")}</Button>
                    <Button onClick={handleSend} loading={sending}>{t(sending ? "send_mail.sending" : "send_mail.send")}</Button>
                </div>
            </div>
        </Dialog>
    );
}

function DetailRow({ label, value, valueClassName = "" }) {
    return (
        <div className="flex items-center py-1.5">
            <div className="w-32 shrink-0 text-sm text-muted-foreground">{label}</div>
            <div className={`text-sm break-all ${valueClassName}`}>{value ?? "-"}</div>
        </div>
    );
}

function UserDetailDialog({ open, onOpenChange, user, t }) {
    if (!user) return null;
    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-xl">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t("dialog.title")}</h3>
                <div className="mt-4 space-y-4">
                    <div className="rounded-lg border p-4">
                        <div className="mb-2 text-sm font-medium">{t("dialog.basicInfo")}</div>
                        <DetailRow label={t("dialog.fields.userId")} value={user.id} />
                        <DetailRow label={t("dialog.fields.email")} value={user.email} valueClassName="font-medium" />
                        <DetailRow label={t("dialog.fields.uuid")} value={user.uuid} valueClassName="font-mono text-xs" />
                        <DetailRow label={t("dialog.fields.token")} value={user.token} valueClassName="font-mono text-xs" />
                        <DetailRow label={t("dialog.fields.remarks")} value={user.remarks} />
                    </div>
                    <div className="rounded-lg border p-4">
                        <div className="mb-2 text-sm font-medium">{t("dialog.subscriptionInfo")}</div>
                        <DetailRow label={t("dialog.fields.subscriptionPlan")} value={user.plan?.name} />
                        <DetailRow label={t("dialog.fields.permissionGroup")} value={user.group?.name} />
                        <DetailRow label={t("dialog.fields.expiredAt")} value={user.expired_at ? formatDateTime(user.expired_at) : t("columns.expire_status.permanent")} />
                        <DetailRow label={t("dialog.fields.deviceLimit")} value={user.device_limit || "-"} />
                        <DetailRow label={t("dialog.fields.speedLimit")} value={user.speed_limit ? `${user.speed_limit} Mbps` : "-"} />
                    </div>
                    <div className="rounded-lg border p-4">
                        <div className="mb-2 text-sm font-medium">{t("dialog.trafficInfo")}</div>
                        <DetailRow label={t("dialog.fields.transferEnable")} value={formatBytes(user.transfer_enable)} />
                        <DetailRow label={t("dialog.fields.uploadUsed")} value={formatBytes(user.u)} />
                        <DetailRow label={t("dialog.fields.downloadUsed")} value={formatBytes(user.d)} />
                        <DetailRow label={t("dialog.fields.totalUsed")} value={formatBytes(user.total_used)} />
                        <DetailRow label={t("dialog.fields.lastResetAt")} value={formatDateTime(user.last_reset_at)} />
                        <DetailRow label={t("dialog.fields.nextResetAt")} value={formatDateTime(user.next_reset_at)} />
                        <DetailRow label={t("dialog.fields.resetCount")} value={user.reset_count || 0} />
                    </div>
                    <div className="rounded-lg border p-4">
                        <div className="mb-2 text-sm font-medium">{t("dialog.financialInfo")}</div>
                        <DetailRow label={t("dialog.fields.balance")} value={formatMoney(user.balance)} />
                        <DetailRow label={t("dialog.fields.commissionBalance")} value={formatMoney(user.commission_balance)} />
                        <DetailRow label={t("dialog.fields.commissionRate")} value={user.commission_rate != null ? `${user.commission_rate}%` : "-"} />
                    </div>
                    <div className="rounded-lg border p-4">
                        <div className="mb-2 text-sm font-medium">{t("dialog.activityInfo")}</div>
                        <DetailRow label={t("dialog.fields.lastLoginAt")} value={formatDateTime(user.last_login_at)} />
                        <DetailRow label={t("dialog.fields.lastLoginIp")} value={user.last_login_ip} />
                        <DetailRow label={t("dialog.fields.lastOnlineAt")} value={formatDateTime(user.last_online_at)} />
                        <DetailRow label={t("dialog.fields.onlineCount")} value={user.online_count || 0} />
                    </div>
                    {user.invite_user && (
                        <div className="rounded-lg border p-4">
                            <div className="mb-2 text-sm font-medium">{t("dialog.inviteInfo")}</div>
                            <DetailRow label={t("dialog.fields.inviteUser")} value={user.invite_user.email} />
                            <DetailRow label={t("dialog.fields.inviteUserId")} value={user.invite_user.id} />
                        </div>
                    )}
                    <div className="rounded-lg border p-4">
                        <div className="mb-2 text-sm font-medium">{t("dialog.timeInfo")}</div>
                        <DetailRow label={t("dialog.fields.createdAt")} value={formatDateTime(user.created_at)} />
                        <DetailRow label={t("dialog.fields.updatedAt")} value={formatDateTime(user.updated_at)} />
                    </div>
                    {user.subscribe_url && (
                        <div className="rounded-lg border p-4">
                            <div className="mb-2 text-sm font-medium">{t("dialog.subscriptionUrl")}</div>
                            <p className="break-all text-sm text-primary">{user.subscribe_url}</p>
                        </div>
                    )}
                </div>
                <div className="mt-6 flex justify-end">
                    <Button variant="outline" onClick={() => onOpenChange(false)}>{t("common:close", "关闭")}</Button>
                </div>
            </div>
        </Dialog>
    );
}

function TrafficResetDialog({ open, onOpenChange, user, onSuccess, t }) {
    const [tab, setTab] = useState("reset");
    const [reason, setReason] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [history, setHistory] = useState(null);
    const [loadingHistory, setLoadingHistory] = useState(false);

    useEffect(() => {
        if (!open) {
            setTab("reset");
            setReason("");
            setHistory(null);
        }
    }, [open]);

    useEffect(() => {
        if (open && tab === "history" && user) {
            setLoadingHistory(true);
            fetchUserTrafficHistory(user.id, { limit: 10 })
                .then((res) => setHistory(res?.data || null))
                .finally(() => setLoadingHistory(false));
        }
    }, [open, tab, user]);

    if (!user) return null;

    const handleReset = async () => {
        setSubmitting(true);
        try {
            await resetUserTraffic({ user_id: user.id, reason: reason.trim() || undefined });
            onOpenChange(false);
            onSuccess?.();
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-3xl">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t("traffic_reset.title")}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{t("traffic_reset.description", { email: user.email })}</p>

                <Tabs value={tab} onValueChange={setTab} className="mt-4">
                    <TabsList className="grid w-full grid-cols-2">
                        <TabsTrigger value="reset">{t("traffic_reset.tabs.reset")}</TabsTrigger>
                        <TabsTrigger value="history">{t("traffic_reset.tabs.history")}</TabsTrigger>
                    </TabsList>
                    <TabsContent value="reset" className="mt-4 space-y-4">
                        <div className="grid gap-4 md:grid-cols-2">
                            <Card>
                                <CardHeader><CardTitle className="text-sm">{t("traffic_reset.user_info")}</CardTitle></CardHeader>
                                <CardContent className="space-y-2 text-sm">
                                    <div className="flex justify-between border-b pb-1"><span className="text-muted-foreground">{t("columns.email")}</span><span>{user.email}</span></div>
                                    <div className="flex justify-between border-b pb-1"><span className="text-muted-foreground">{t("columns.used_traffic")}</span><span className="text-destructive">{formatBytes(user.total_used)}</span></div>
                                    <div className="flex justify-between border-b pb-1"><span className="text-muted-foreground">{t("columns.total_traffic")}</span><span>{formatBytes(user.transfer_enable)}</span></div>
                                    <div className="flex justify-between"><span className="text-muted-foreground">{t("columns.expire_time")}</span><span>{user.expired_at ? formatDateTime(user.expired_at) : t("columns.expire_status.permanent")}</span></div>
                                </CardContent>
                            </Card>
                            <Card className="border-amber-200 bg-amber-50">
                                <CardHeader><CardTitle className="text-sm text-amber-800">{t("traffic_reset.warning.title")}</CardTitle></CardHeader>
                                <CardContent>
                                    <ul className="space-y-1 text-xs text-amber-700">
                                        <li>• {t("traffic_reset.warning.irreversible")}</li>
                                        <li>• {t("traffic_reset.warning.reset_to_zero")}</li>
                                        <li>• {t("traffic_reset.warning.logged")}</li>
                                    </ul>
                                </CardContent>
                            </Card>
                        </div>
                        <div>
                            <label className="text-sm font-medium">{t("traffic_reset.reason.label")}</label>
                            <Textarea className="mt-1" value={reason} onChange={(e) => setReason(e.target.value)} placeholder={t("traffic_reset.reason.placeholder")} />
                            <p className="mt-1 text-xs text-muted-foreground">{t("traffic_reset.reason.optional")}</p>
                        </div>
                    </TabsContent>
                    <TabsContent value="history" className="mt-4">
                        {loadingHistory ? (
                            <div className="py-8 text-center text-sm text-muted-foreground">...</div>
                        ) : (
                            <div className="grid gap-4 md:grid-cols-3">
                                {history?.user && (
                                    <Card className="md:col-span-1">
                                        <CardHeader><CardTitle className="text-sm">{t("traffic_reset.history.summary")}</CardTitle></CardHeader>
                                        <CardContent className="space-y-3 text-xs">
                                            <div><div className="text-muted-foreground">{t("traffic_reset.history.reset_count")}</div><p className="text-base font-medium">{history.user.reset_count}</p></div>
                                            <div><div className="text-muted-foreground">{t("traffic_reset.history.last_reset")}</div><p>{history.user.last_reset_at ? formatDateTime(history.user.last_reset_at) : t("traffic_reset.history.never")}</p></div>
                                            <div><div className="text-muted-foreground">{t("traffic_reset.history.next_reset")}</div><p>{history.user.next_reset_at ? formatDateTime(history.user.next_reset_at) : t("traffic_reset.history.no_schedule")}</p></div>
                                        </CardContent>
                                    </Card>
                                )}
                                <Card className="md:col-span-2">
                                    <CardHeader><CardTitle className="text-sm">{t("traffic_reset.history.records")}</CardTitle></CardHeader>
                                    <CardContent className="max-h-[300px] overflow-y-auto">
                                        {history?.history?.length ? (
                                            <div className="space-y-3">
                                                {history.history.map((rec) => (
                                                    <div key={rec.id} className="rounded border bg-muted/30 p-3">
                                                        <div className="flex items-center gap-2">
                                                            <Badge variant="outline" className="text-[10px]">{rec.reset_type_name}</Badge>
                                                            <Badge variant="secondary" className="text-[10px]">{rec.trigger_source_name}</Badge>
                                                        </div>
                                                        <div className="mt-2 grid grid-cols-2 gap-4 text-[11px]">
                                                            <div><div className="text-muted-foreground">{t("traffic_reset.history.reset_time")}</div><p>{formatDateTime(rec.reset_time)}</p></div>
                                                            <div><div className="text-muted-foreground">{t("traffic_reset.history.traffic_cleared")}</div><p className="text-destructive">{rec.old_traffic?.formatted}</p></div>
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        ) : (
                                            <div className="py-8 text-center text-xs text-muted-foreground">{t("traffic_reset.history.no_records")}</div>
                                        )}
                                    </CardContent>
                                </Card>
                            </div>
                        )}
                    </TabsContent>
                </Tabs>

                <div className="mt-6 flex justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange(false)}>{t("common:cancel", "取消")}</Button>
                    {tab === "reset" && <Button variant="destructive" onClick={handleReset} loading={submitting}>{submitting ? t("traffic_reset.resetting") : t("traffic_reset.confirm_reset")}</Button>}
                </div>
            </div>
        </Dialog>
    );
}

const EMPTY_EDIT = {
    email: "", invite_user_email: "", password: "",
    balance: "0", commission_balance: "0", u: "", d: "", transfer_enable: "0",
    expired_at: null, plan_id: "", banned: false,
    commission_type: 0, commission_rate: "", discount: "", speed_limit: "", device_limit: "",
    is_admin: false, is_staff: false, remarks: "",
};

function userToEditForm(user) {
    return {
        id: user.id,
        email: user.email || "",
        invite_user_email: user.invite_user?.email || "",
        password: "",
        balance: String(user.balance || 0),
        commission_balance: String(user.commission_balance || 0),
        u: user.u ? (user.u / GB).toFixed(3) : "",
        d: user.d ? (user.d / GB).toFixed(3) : "",
        transfer_enable: String((user.transfer_enable || 0) / GB),
        expired_at: user.expired_at || null,
        plan_id: user.plan_id != null ? String(user.plan_id) : "",
        banned: Boolean(user.banned),
        commission_type: user.commission_type || 0,
        commission_rate: user.commission_rate != null ? String(user.commission_rate) : "",
        discount: user.discount != null ? String(user.discount) : "",
        speed_limit: user.speed_limit != null ? String(user.speed_limit) : "",
        device_limit: user.device_limit != null ? String(user.device_limit) : "",
        is_admin: Boolean(user.is_admin),
        is_staff: Boolean(user.is_staff),
        remarks: user.remarks || "",
    };
}

function UserEditDialog({ open, onOpenChange, user, plans, onSaved, t }) {
    const [form, setForm] = useState(EMPTY_EDIT);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (open && user) setForm(userToEditForm(user));
        if (open) setError("");
    }, [open, user]);

    const set = (patch) => setForm((prev) => ({ ...prev, ...patch }));

    const applyExpirePreset = (preset) => {
        if (preset === "permanent") return set({ expired_at: null });
        const d = new Date();
        if (preset === "today") d.setHours(23, 59, 59, 999);
        if (preset === "1month") { d.setMonth(d.getMonth() + 1); d.setHours(23, 59, 59, 999); }
        if (preset === "3months") { d.setMonth(d.getMonth() + 3); d.setHours(23, 59, 59, 999); }
        set({ expired_at: Math.floor(d.getTime() / 1000) });
    };

    const handleSubmit = async () => {
        setError("");
        if (!form.email.trim()) { setError(t("edit.form.email_placeholder")); return; }
        setSubmitting(true);
        try {
            const payload = {
                id: form.id,
                email: form.email.trim(),
                invite_user_email: form.invite_user_email.trim() || null,
                balance: Math.round(parseFloat(form.balance || 0) * 100),
                commission_balance: Math.round(parseFloat(form.commission_balance || 0) * 100),
                transfer_enable: Math.round(parseFloat(form.transfer_enable || 0) * GB),
                expired_at: form.expired_at || null,
                plan_id: form.plan_id !== "" ? parseInt(form.plan_id, 10) : null,
                banned: form.banned,
                commission_type: parseInt(form.commission_type, 10) || 0,
                commission_rate: form.commission_rate !== "" ? parseInt(form.commission_rate, 10) : null,
                discount: form.discount !== "" ? parseInt(form.discount, 10) : null,
                speed_limit: form.speed_limit !== "" ? parseInt(form.speed_limit, 10) : null,
                device_limit: form.device_limit !== "" ? parseInt(form.device_limit, 10) : null,
                is_admin: form.is_admin,
                is_staff: form.is_staff,
                remarks: form.remarks || null,
            };
            if (form.password.trim()) payload.password = form.password.trim();
            if (form.u !== "") payload.u = Math.round(parseFloat(form.u) * GB);
            if (form.d !== "") payload.d = Math.round(parseFloat(form.d) * GB);
            await updateUser(payload);
            onOpenChange(false);
            await onSaved();
        } catch (err) {
            setError(err?.message || t("edit.form.email_placeholder"));
        } finally {
            setSubmitting(false);
        }
    };

    if (!user) return null;

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-2xl">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t("edit.title")}</h3>
                {error && <p className="mt-2 text-sm text-destructive">{error}</p>}
                <div className="mt-4 space-y-4">
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.email")}</label>
                        <Input className="mt-1" value={form.email} onChange={(e) => set({ email: e.target.value })} placeholder={t("edit.form.email_placeholder")} />
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.inviter_email")}</label>
                        <Input className="mt-1" value={form.invite_user_email} onChange={(e) => set({ invite_user_email: e.target.value })} placeholder={t("edit.form.inviter_email_placeholder")} />
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.password")}</label>
                        <Input className="mt-1" type="password" value={form.password} onChange={(e) => set({ password: e.target.value })} placeholder={t("edit.form.password_placeholder")} />
                    </div>
                    <div className="grid gap-3 md:grid-cols-2">
                        <div>
                            <label className="text-sm font-medium">{t("edit.form.balance")}</label>
                            <Input className="mt-1" type="number" value={form.balance} onChange={(e) => set({ balance: e.target.value })} />
                        </div>
                        <div>
                            <label className="text-sm font-medium">{t("edit.form.commission_balance")}</label>
                            <Input className="mt-1" type="number" value={form.commission_balance} onChange={(e) => set({ commission_balance: e.target.value })} />
                        </div>
                        <div>
                            <label className="text-sm font-medium">{t("edit.form.upload")} (GB)</label>
                            <Input className="mt-1" type="number" step="any" value={form.u} onChange={(e) => set({ u: e.target.value })} placeholder={t("edit.form.upload_placeholder")} />
                        </div>
                        <div>
                            <label className="text-sm font-medium">{t("edit.form.download")} (GB)</label>
                            <Input className="mt-1" type="number" step="any" value={form.d} onChange={(e) => set({ d: e.target.value })} placeholder={t("edit.form.download_placeholder")} />
                        </div>
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.total_traffic")} (GB)</label>
                        <Input className="mt-1" type="number" value={form.transfer_enable} onChange={(e) => set({ transfer_enable: e.target.value })} />
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.expire_time")}</label>
                        <div className="mt-1 flex flex-wrap gap-2">
                            <Button type="button" variant="outline" size="sm" onClick={() => applyExpirePreset("permanent")}>{t("edit.form.expire_time_permanent")}</Button>
                            <Button type="button" variant="outline" size="sm" onClick={() => applyExpirePreset("today")}>{t("edit.form.expire_time_today")}</Button>
                            <Button type="button" variant="outline" size="sm" onClick={() => applyExpirePreset("1month")}>{t("edit.form.expire_time_1month")}</Button>
                            <Button type="button" variant="outline" size="sm" onClick={() => applyExpirePreset("3months")}>{t("edit.form.expire_time_3months")}</Button>
                        </div>
                        <Input className="mt-2" type="datetime-local" step="1" value={toDatetimeLocal(form.expired_at)} onChange={(e) => set({ expired_at: fromDatetimeLocal(e.target.value) })} />
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.subscription")}</label>
                        <Select value={form.plan_id} onValueChange={(v) => set({ plan_id: v })} className="mt-1">
                            <SelectTrigger><SelectValue placeholder={t("edit.form.subscription_none")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="">{t("edit.form.subscription_none")}</SelectItem>
                                {plans.map((p) => <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>)}
                            </SelectContent>
                        </Select>
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.account_status")}</label>
                        <Select value={form.banned ? "true" : "false"} onValueChange={(v) => set({ banned: v === "true" })} className="mt-1">
                            <SelectTrigger><SelectValue /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="false">{t("columns.status_text.normal")}</SelectItem>
                                <SelectItem value="true">{t("columns.status_text.banned")}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.commission_type")}</label>
                        <Select value={String(form.commission_type)} onValueChange={(v) => set({ commission_type: parseInt(v, 10) })} className="mt-1">
                            <SelectTrigger><SelectValue /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="0">{t("edit.form.commission_type_system")}</SelectItem>
                                <SelectItem value="1">{t("edit.form.commission_type_cycle")}</SelectItem>
                                <SelectItem value="2">{t("edit.form.commission_type_onetime")}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <div className="grid gap-3 md:grid-cols-2">
                        <div>
                            <label className="text-sm font-medium">{t("edit.form.commission_rate")}</label>
                            <Input className="mt-1" type="number" value={form.commission_rate} onChange={(e) => set({ commission_rate: e.target.value })} placeholder={t("edit.form.commission_rate_placeholder")} />
                        </div>
                        <div>
                            <label className="text-sm font-medium">{t("edit.form.discount")}</label>
                            <Input className="mt-1" type="number" value={form.discount} onChange={(e) => set({ discount: e.target.value })} placeholder={t("edit.form.discount_placeholder")} />
                        </div>
                        <div>
                            <label className="text-sm font-medium">{t("edit.form.speed_limit")}</label>
                            <Input className="mt-1" type="number" value={form.speed_limit} onChange={(e) => set({ speed_limit: e.target.value })} placeholder={t("edit.form.speed_limit_placeholder")} />
                        </div>
                        <div>
                            <label className="text-sm font-medium">{t("edit.form.device_limit")}</label>
                            <Input className="mt-1" type="number" value={form.device_limit} onChange={(e) => set({ device_limit: e.target.value })} placeholder={t("edit.form.device_limit_placeholder")} />
                        </div>
                    </div>
                    <div className="flex items-center justify-between rounded-lg border p-3">
                        <span className="text-sm font-medium">{t("edit.form.is_admin")}</span>
                        <Switch checked={form.is_admin} onCheckedChange={(v) => set({ is_admin: v })} />
                    </div>
                    <div className="flex items-center justify-between rounded-lg border p-3">
                        <span className="text-sm font-medium">{t("edit.form.is_staff")}</span>
                        <Switch checked={form.is_staff} onCheckedChange={(v) => set({ is_staff: v })} />
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("edit.form.remarks")}</label>
                        <Textarea className="mt-1 h-24" value={form.remarks} onChange={(e) => set({ remarks: e.target.value })} placeholder={t("edit.form.remarks_placeholder")} />
                    </div>
                </div>
                <div className="mt-6 flex justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>{t("edit.form.cancel")}</Button>
                    <Button onClick={handleSubmit} loading={submitting}>{t("edit.form.submit")}</Button>
                </div>
            </div>
        </Dialog>
    );
}

const EMPTY_GENERATE = { email_prefix: "", email_suffix: "", password: "", expired_at: null, plan_id: "", generate_count: "", download_csv: false };

function UserGenerateDialog({ open, onOpenChange, plans, onSaved, t }) {
    const [form, setForm] = useState(EMPTY_GENERATE);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (open) { setForm(EMPTY_GENERATE); setError(""); }
    }, [open]);

    const set = (patch) => setForm((prev) => ({ ...prev, ...patch }));

    const applyExpirePreset = (preset) => {
        if (preset === "permanent") return set({ expired_at: null });
        const d = new Date();
        if (preset === "1month") d.setMonth(d.getMonth() + 1);
        if (preset === "3months") d.setMonth(d.getMonth() + 3);
        d.setHours(23, 59, 59, 999);
        set({ expired_at: Math.floor(d.getTime() / 1000) });
    };

    const handleSubmit = async () => {
        setError("");
        if (!form.email_suffix.trim()) { setError(t("generate.form.email")); return; }
        if (!form.generate_count && !form.email_prefix.trim()) { setError(t("generate.form.email_prefix")); return; }
        setSubmitting(true);
        try {
            const payload = {
                email_suffix: form.email_suffix.trim(),
                password: form.password.trim() || undefined,
                expired_at: form.expired_at || null,
                plan_id: form.plan_id !== "" ? parseInt(form.plan_id, 10) : null,
                generate_count: form.generate_count ? parseInt(form.generate_count, 10) : null,
                download_csv: form.download_csv,
            };
            if (!form.generate_count) payload.email_prefix = form.email_prefix.trim();
            const res = await generateUsers(payload);
            if (payload.download_csv && res instanceof Blob) {
                const url = window.URL.createObjectURL(res);
                const link = document.createElement("a");
                link.href = url;
                link.download = `users_${Date.now()}.csv`;
                document.body.appendChild(link);
                link.click();
                link.remove();
                window.URL.revokeObjectURL(url);
            }
            onOpenChange(false);
            await onSaved();
        } catch (err) {
            setError(err?.message || t("generate.form.email"));
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-lg">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t("generate.title")}</h3>
                {error && <p className="mt-2 text-sm text-destructive">{error}</p>}
                <div className="mt-4 space-y-4">
                    <div>
                        <label className="text-sm font-medium">{t("generate.form.email")}</label>
                        <div className="mt-1 flex items-center gap-1">
                            {!form.generate_count && (
                                <Input value={form.email_prefix} onChange={(e) => set({ email_prefix: e.target.value })} placeholder={t("generate.form.email_prefix")} className="flex-1" />
                            )}
                            <span className="text-sm text-muted-foreground">@</span>
                            <Input value={form.email_suffix} onChange={(e) => set({ email_suffix: e.target.value })} placeholder={t("generate.form.email_domain")} className="flex-1" />
                        </div>
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("generate.form.password")}</label>
                        <Input className="mt-1" value={form.password} onChange={(e) => set({ password: e.target.value })} placeholder={t("generate.form.password_placeholder")} />
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("generate.form.expire_time")}</label>
                        <div className="mt-1 flex flex-wrap gap-2">
                            <Button type="button" variant="outline" size="sm" onClick={() => applyExpirePreset("permanent")}>{t("generate.form.permanent")}</Button>
                            <Button type="button" variant="outline" size="sm" onClick={() => applyExpirePreset("1month")}>1M</Button>
                            <Button type="button" variant="outline" size="sm" onClick={() => applyExpirePreset("3months")}>3M</Button>
                        </div>
                        <Input className="mt-2" type="datetime-local" step="1" value={toDatetimeLocal(form.expired_at)} onChange={(e) => set({ expired_at: fromDatetimeLocal(e.target.value) })} placeholder={t("generate.form.expire_time_placeholder")} />
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("generate.form.subscription")}</label>
                        <Select value={form.plan_id} onValueChange={(v) => set({ plan_id: v })} className="mt-1">
                            <SelectTrigger><SelectValue placeholder={t("generate.form.subscription_none")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="">{t("generate.form.subscription_none")}</SelectItem>
                                {plans.map((p) => <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>)}
                            </SelectContent>
                        </Select>
                    </div>
                    <div>
                        <label className="text-sm font-medium">{t("generate.form.generate_count")}</label>
                        <Input className="mt-1" type="number" value={form.generate_count} onChange={(e) => set({ generate_count: e.target.value })} placeholder={t("generate.form.generate_count_placeholder")} />
                    </div>
                    <div className="flex items-center justify-between rounded-lg border p-3">
                        <span className="text-sm font-medium">{t("generate.form.download_csv")}</span>
                        <Switch checked={form.download_csv} onCheckedChange={(v) => set({ download_csv: v })} />
                    </div>
                </div>
                <div className="mt-6 flex justify-end gap-2">
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>{t("generate.form.cancel")}</Button>
                    <Button onClick={handleSubmit} loading={submitting}>{t("generate.form.submit")}</Button>
                </div>
            </div>
        </Dialog>
    );
}

function DeviceCountBar({ count, limit }) {
    const infinite = limit == null || limit === 0;
    const pct = infinite ? 0 : Math.min(((count || 0) / limit) * 100, 100);
    const full = !infinite && (count || 0) >= limit;
    return (
        <div className="w-28 space-y-1">
            <div className="flex justify-between text-xs">
                <span>{count || 0} / {infinite ? "∞" : limit}</span>
            </div>
            {!infinite && (
                <div className="h-1.5 w-full rounded-full bg-secondary">
                    <div className={`h-full rounded-full ${full ? "bg-destructive" : pct > 80 ? "bg-amber-500" : "bg-primary"}`} style={{ width: `${pct}%` }} />
                </div>
            )}
        </div>
    );
}

function OnlineUsersDialog({ open, onOpenChange, t }) {
    const [data, setData] = useState({ total: 0, users: [] });
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (!open) return;
        let cancelled = false;
        const load = () => {
            setLoading(true);
            getOnlineUsers()
                .then((res) => { if (!cancelled) setData(res?.data || { total: 0, users: [] }); })
                .catch(() => {})
                .finally(() => { if (!cancelled) setLoading(false); });
        };
        load();
        const timer = setInterval(load, 30000);
        return () => { cancelled = true; clearInterval(timer); };
    }, [open]);

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="sm:max-w-2xl">
            <div className="max-h-[90vh] overflow-y-auto p-6">
                <h3 className="text-lg font-semibold">{t("online.title")}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{t("online.description", { count: data.total || 0 })}</p>
                <div className="mt-4">
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>{t("online.columns.user")}</TableHead>
                                <TableHead>{t("online.columns.devices")}</TableHead>
                                <TableHead>{t("online.columns.expire")}</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {loading && data.users.length === 0 ? (
                                <TableEmpty colSpan={3}>...</TableEmpty>
                            ) : data.users.length === 0 ? (
                                <TableEmpty colSpan={3}>{t("online.empty")}</TableEmpty>
                            ) : (
                                data.users.map((u) => {
                                    const expired = u.expired_at != null && u.expired_at < Date.now() / 1000;
                                    return (
                                        <TableRow key={u.user_id}>
                                            <TableCell>
                                                <div className="font-medium">{u.email}</div>
                                                <div className="text-xs text-muted-foreground">UID {u.user_id}</div>
                                            </TableCell>
                                            <TableCell><DeviceCountBar count={u.device_count} limit={u.device_limit} /></TableCell>
                                            <TableCell>
                                                <span className={expired ? "text-destructive" : ""}>
                                                    {u.expired_at ? formatDateTime(u.expired_at) : t("columns.expire_status.permanent")}
                                                </span>
                                            </TableCell>
                                        </TableRow>
                                    );
                                })
                            )}
                        </TableBody>
                    </Table>
                </div>
                <div className="mt-6 flex justify-end">
                    <Button variant="outline" onClick={() => onOpenChange(false)}>{t("common:close", "关闭")}</Button>
                </div>
            </div>
        </Dialog>
    );
}

export function UserManagement() {
    const { t } = useTranslation("user");
    const auth = useAuthState();
    useCurrencySymbol();
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [page, setPage] = useState(1);

    const [search, setSearch] = useState("");
    const [conditions, setConditions] = useState([]);
    const [mode, setMode] = useState("all"); // 条件之间：all 全部满足 / any 满足任一
    const [filtersVersion, setFiltersVersion] = useState(0);

    const [plans, setPlans] = useState([]);
    const [selectedIds, setSelectedIds] = useState(() => new Set());

    const [editingUser, setEditingUser] = useState(null);
    const [editOpen, setEditOpen] = useState(false);
    const [generateOpen, setGenerateOpen] = useState(false);
    const [detailUser, setDetailUser] = useState(null);
    const [trafficResetUser, setTrafficResetUser] = useState(null);
    const [sendMailOpen, setSendMailOpen] = useState(false);
    const [batchBanOpen, setBatchBanOpen] = useState(false);
    const [deleteTarget, setDeleteTarget] = useState(null);
    const [onlineOpen, setOnlineOpen] = useState(false);
    const [onlineCount, setOnlineCount] = useState(null);

    // 列表、群发邮件、导出、批量封禁都用这一份条件，保证「所见即所操作」。
    const buildFilter = () => buildUserFilter({ conditions, mode, search });

    const load = () => {
        setLoading(true);
        return fetchUsers({ pageSize: PAGE_SIZE, current: page, filter: buildFilter(), sort: [] })
            .then((res) => {
                setItems(res?.data || []);
                setTotal(res?.total ?? 0);
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [auth.token, page, filtersVersion]);

    useEffect(() => {
        fetchPlans().then((res) => setPlans(res?.data || [])).catch(() => setPlans([]));
    }, [auth.token]);

    useEffect(() => {
        let cancelled = false;
        const poll = () => {
            getOnlineUsers()
                .then((res) => { if (!cancelled) setOnlineCount(res?.data?.total ?? 0); })
                .catch(() => {});
        };
        poll();
        const timer = setInterval(poll, 60000);
        return () => { cancelled = true; clearInterval(timer); };
    }, [auth.token]);

    const refetch = () => {
        if (page === 1) setFiltersVersion((v) => v + 1);
        else setPage(1);
    };

    const planOptions = useMemo(() => plans.map((p) => ({ value: p.id, label: p.name })), [plans]);

    const hasFilter = search.trim() !== "" || conditions.length > 0;
    const hasSelection = selectedIds.size > 0;

    const toggleRow = (id, checked) => {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (checked) next.add(id);
            else next.delete(id);
            return next;
        });
    };

    const toggleAll = (checked) => {
        setSelectedIds(checked ? new Set(items.map((u) => u.id)) : new Set());
    };

    const currentScope = () => (hasSelection ? "selected" : hasFilter ? "filtered" : "all");

    const handleResetSecret = async (user) => {
        await resetUserSecret(user.id);
        await load();
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        await destroyUser(deleteTarget.id);
        await load();
    };

    const handleSendMail = async ({ scope, subject, content }) => {
        if (scope === "selected") {
            for (const id of selectedIds) {
                await sendUserMail({ subject, content, filter: [{ id: "id", value: `eq:${id}` }] });
            }
        } else if (scope === "filtered") {
            await sendUserMail({ subject, content, filter: buildFilter() });
        } else {
            await sendUserMail({ subject, content });
        }
    };

    const handleExportCsv = async () => {
        const scope = currentScope();
        const params = { scope };
        if (scope === "selected") params.user_ids = Array.from(selectedIds);
        if (scope === "filtered") params.filter = buildFilter();
        const res = await dumpUserCSV(params);
        if (res instanceof Blob) {
            const url = window.URL.createObjectURL(res);
            const link = document.createElement("a");
            link.href = url;
            link.download = `users_${Date.now()}.csv`;
            document.body.appendChild(link);
            link.click();
            link.remove();
            window.URL.revokeObjectURL(url);
        }
    };

    const handleBatchBan = async () => {
        const scope = currentScope();
        const params = { scope };
        if (scope === "selected") params.user_ids = Array.from(selectedIds);
        if (scope === "filtered") params.filter = buildFilter();
        await banUser(params);
        setSelectedIds(new Set());
        await load();
    };

    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

    return (
        <div className="p-4 md:p-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-lg font-medium">{t("manage.title")}</h3>
                <p className="text-sm text-muted-foreground">{t("manage.description")}</p>
            </div>
            <hr className="my-6" />

            <div className="flex flex-col gap-3">
                <UserFilterBar
                    t={t}
                    search={search}
                    onSearchChange={setSearch}
                    onSearch={refetch}
                    conditions={conditions}
                    mode={mode}
                    onChange={({ conditions: next, mode: nextMode }) => { setConditions(next); setMode(nextMode); refetch(); }}
                    onReset={() => { setSearch(""); setConditions([]); setMode("all"); refetch(); }}
                    planOptions={planOptions}
                    currency={getCurrencySymbol()}
                    actions={(
                        <>
                            <Button variant="outline" size="sm" onClick={() => setOnlineOpen(true)}>
                                {t("online.button")}{onlineCount != null ? ` (${onlineCount})` : ""}
                            </Button>
                            <Button variant="outline" size="sm" onClick={load}>↻</Button>
                            <Button size="sm" onClick={() => setGenerateOpen(true)}>{t("generate.button")}</Button>
                        </>
                    )}
                />

                <div className="flex flex-wrap items-center gap-2 rounded-md border bg-muted/20 px-3 py-2">
                    <span className="text-xs font-medium text-muted-foreground">{t("actions.title")}{hasSelection ? ` (${selectedIds.size})` : ""}</span>
                    <Button variant="outline" size="sm" onClick={() => setSendMailOpen(true)}>{t("actions.send_email")}</Button>
                    <Button variant="outline" size="sm" onClick={handleExportCsv}>{t("actions.export_csv")}</Button>
                    <Button variant="destructive" size="sm" onClick={() => setBatchBanOpen(true)}>{t("actions.batch_ban")}</Button>
                </div>
            </div>

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-8">
                                <input type="checkbox" checked={items.length > 0 && items.every((u) => selectedIds.has(u.id))} onChange={(e) => toggleAll(e.target.checked)} />
                            </TableHead>
                            <TableHead>{t("columns.id")}</TableHead>
                            <TableHead>{t("columns.email")}</TableHead>
                            <TableHead>{t("columns.online_count")}</TableHead>
                            <TableHead>{t("columns.status")}</TableHead>
                            <TableHead>{t("columns.subscription")}</TableHead>
                            <TableHead>{t("columns.group")}</TableHead>
                            <TableHead>{t("columns.used_traffic")}</TableHead>
                            <TableHead>{t("columns.total_traffic")}</TableHead>
                            <TableHead>{t("columns.expire_time")}</TableHead>
                            <TableHead>{t("columns.balance")}</TableHead>
                            <TableHead>{t("columns.invite_code")}</TableHead>
                            <TableHead>{t("columns.inviter")}</TableHead>
                            <TableHead>{t("columns.register_time")}</TableHead>
                            <TableHead className="w-56 text-right">{t("columns.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={15}>...</TableEmpty>
                        ) : items.length === 0 ? (
                            <TableEmpty colSpan={15}>-</TableEmpty>
                        ) : (
                            items.map((user) => {
                                const usedPct = user.transfer_enable > 0 ? Math.min((user.total_used / user.transfer_enable) * 100, 100) : 0;
                                const isOnline = user.t && Date.now() / 1000 - user.t < 120;
                                const expired = user.expired_at != null && user.expired_at < Date.now() / 1000;
                                return (
                                    <TableRow key={user.id}>
                                        <TableCell><input type="checkbox" checked={selectedIds.has(user.id)} onChange={(e) => toggleRow(user.id, e.target.checked)} /></TableCell>
                                        <TableCell><Badge variant="outline">{user.id}</Badge></TableCell>
                                        <TableCell>
                                            <div className="flex items-center gap-2">
                                                <span className={`size-2.5 shrink-0 rounded-full ${isOnline ? "bg-emerald-500" : "bg-slate-300"}`} />
                                                <button type="button" className="font-medium hover:underline" onClick={() => setDetailUser(user)}>{user.email}</button>
                                                <button type="button" className="text-xs text-muted-foreground hover:text-foreground" onClick={() => copyText(user.email)}>⧉</button>
                                            </div>
                                        </TableCell>
                                        <TableCell className={user.device_limit != null && (user.online_count ?? 0) > user.device_limit ? "font-medium text-destructive" : undefined}>
                                            {user.online_count ?? 0} / {user.device_limit == null ? "∞" : user.device_limit}
                                        </TableCell>
                                        <TableCell>
                                            <Badge className={user.banned ? "bg-destructive/15 text-destructive" : "bg-emerald-500/15 text-emerald-600"}>
                                                {t(user.banned ? "columns.status_text.banned" : "columns.status_text.normal")}
                                            </Badge>
                                        </TableCell>
                                        <TableCell>{user.plan?.name || "-"}</TableCell>
                                        <TableCell>{user.group?.name || "-"}</TableCell>
                                        <TableCell>
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <div className="w-24 space-y-1">
                                                        <div className="flex justify-between text-xs"><span>{formatBytes(user.total_used)}</span><span className="text-muted-foreground">{usedPct.toFixed(0)}%</span></div>
                                                        <div className="h-1.5 w-full rounded-full bg-secondary"><div className={`h-full rounded-full ${usedPct > 90 ? "bg-destructive" : "bg-primary"}`} style={{ width: `${usedPct}%` }} /></div>
                                                    </div>
                                                </TooltipTrigger>
                                                <TooltipContent><p className="text-xs">{t("columns.total_traffic")}: {formatBytes(user.transfer_enable)}</p></TooltipContent>
                                            </Tooltip>
                                        </TableCell>
                                        <TableCell className="text-muted-foreground">{formatBytes(user.transfer_enable)}</TableCell>
                                        <TableCell>
                                            <Badge variant="outline" className={expired ? "border-destructive/50 text-destructive" : ""}>
                                                {user.expired_at == null ? t("columns.expire_status.permanent") : formatDateTime(user.expired_at)}
                                            </Badge>
                                        </TableCell>
                                        <TableCell>{formatMoney(user.balance)}</TableCell>
                                        <TableCell>
                                            {user.codes?.length ? (
                                                <div className="flex flex-wrap gap-1">
                                                    {user.codes.map((c) => (
                                                        <button
                                                            key={c.id}
                                                            type="button"
                                                            title={c.code}
                                                            className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs hover:bg-muted/70"
                                                            onClick={() => copyText(c.code)}
                                                        >
                                                            {c.code}
                                                        </button>
                                                    ))}
                                                </div>
                                            ) : (
                                                <span className="text-muted-foreground">-</span>
                                            )}
                                        </TableCell>
                                        <TableCell>{user.invite_user?.email || "-"}</TableCell>
                                        <TableCell className="whitespace-nowrap text-muted-foreground">{formatDateTime(user.created_at)}</TableCell>
                                        <TableCell className="text-right">
                                            <div className="ml-auto flex w-[300px] flex-wrap justify-end gap-1.5">
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => { setEditingUser(user); setEditOpen(true); }}>{t("columns.actions_menu.edit")}</Button>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => copyText(user.subscribe_url).then(() => {})}>{t("columns.actions_menu.copy_url")}</Button>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => handleResetSecret(user)}>{t("columns.actions_menu.reset_secret")}</Button>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => setTrafficResetUser(user)}>{t("columns.actions_menu.reset_traffic")}</Button>
                                                <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => navigate(`/finance/order?user_id=eq:${user.id}`)}>{t("columns.actions_menu.orders")}</Button>
                                                <Button variant="destructive" size="sm" className="h-7 px-2 text-xs" onClick={() => setDeleteTarget(user)}>{t("columns.actions_menu.delete")}</Button>
                                            </div>
                                        </TableCell>
                                    </TableRow>
                                );
                            })
                        )}
                    </TableBody>
                </Table>
            </div>

            {totalPages > 1 && (
                <div className="mt-4 flex items-center justify-end gap-2">
                    <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>‹</Button>
                    <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
                    <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>›</Button>
                </div>
            )}

            <SendMailDialog
                open={sendMailOpen}
                onOpenChange={setSendMailOpen}
                hasSelection={hasSelection}
                selectedCount={selectedIds.size}
                hasFilter={hasFilter}
                onSend={handleSendMail}
                t={t}
            />
            <UserDetailDialog open={Boolean(detailUser)} onOpenChange={(open) => !open && setDetailUser(null)} user={detailUser} t={t} />
            <TrafficResetDialog
                open={Boolean(trafficResetUser)}
                onOpenChange={(open) => !open && setTrafficResetUser(null)}
                user={trafficResetUser}
                onSuccess={load}
                t={t}
            />
            <UserEditDialog open={editOpen} onOpenChange={setEditOpen} user={editingUser} plans={plans} onSaved={load} t={t} />
            <UserGenerateDialog open={generateOpen} onOpenChange={setGenerateOpen} plans={plans} onSaved={load} t={t} />
            <OnlineUsersDialog open={onlineOpen} onOpenChange={setOnlineOpen} t={t} />

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("columns.actions_menu.delete_confirm_title")}
                description={deleteTarget ? t("columns.actions_menu.delete_confirm_description", { email: deleteTarget.email }) : ""}
                confirmLabel={t("columns.actions_menu.delete")}
                onConfirm={handleDelete}
            />
            <ConfirmDialog
                open={batchBanOpen}
                onOpenChange={setBatchBanOpen}
                title={t("actions.confirm_ban.title")}
                description={t(currentScope() === "filtered" ? "actions.confirm_ban.filtered_description" : "actions.confirm_ban.all_description")}
                confirmLabel={t("actions.confirm_ban.confirm")}
                onConfirm={handleBatchBan}
            />
        </div>
    );
}
