import React, { useEffect, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { fetchDevices, fetchDeviceDetail, setDeviceTrialClaimed } from "../../api/device.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useClientSort, useServerSort } from "../../components/SortableHead.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { getErrorMessage } from "../../utils/errors.js";

const PAGE_SIZE = 20;

function formatDateTime(seconds) {
    if (!seconds) return "-";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function shortDeviceId(id) {
    if (!id) return "-";
    return id.length > 20 ? `${id.slice(0, 8)}…${id.slice(-8)}` : id;
}

function TrialBadge({ claimed, t }) {
    return claimed
        ? <Badge variant="secondary">{t("status.claimed")}</Badge>
        : <Badge variant="success">{t("status.unclaimed")}</Badge>;
}

function sourceLabel(source, t) {
    return t(`source.${source}`, source || "-");
}

function DeviceDetailDialog({ deviceId, open, reloadKey, onOpenChange, onToggleTrial, t }) {
    const [detail, setDetail] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    useEffect(() => {
        if (!open || !deviceId) {
            setDetail(null);
            setError("");
            return;
        }
        setLoading(true);
        fetchDeviceDetail(deviceId)
            .then((res) => setDetail(res?.data || null))
            .catch((err) => setError(getErrorMessage(err, t("messages.failed"))))
            .finally(() => setLoading(false));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open, deviceId, reloadKey]);

    const device = detail?.device;
    const accountSort = useClientSort(detail?.accounts || [], {
        id: (u) => Number(u.id),
        email: (u) => u.email || "",
        plan: (u) => u.plan_name || null,
        expired_at: (u) => Number(u.expired_at) || null,
        created_at: (u) => Number(u.created_at) || null,
    });
    const logSort = useClientSort(detail?.trial_logs || [], {
        granted_at: (l) => Number(l.created_at) || null,
        source: (l) => sourceLabel(l.source, t),
        email: (l) => l.email || null,
        expired_at: (l) => Number(l.expired_at) || null,
        ip: (l) => l.ip || null,
    });
    const accounts = accountSort.sorted;
    const logs = logSort.sorted;

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="flex max-h-[85vh] max-w-4xl flex-col overflow-hidden p-0">
            <div className="flex items-start justify-between gap-3 border-b px-5 py-4">
                <div className="min-w-0 space-y-1">
                    <h2 className="text-lg font-semibold leading-tight">{t("detail.title")}</h2>
                    {device ? (
                        <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                            <span className="break-all font-mono">{device.device_id}</span>
                            <span>·</span>
                            <span>{t("columns.display_code")} {device.display_code}</span>
                            <TrialBadge claimed={device.claimed_trial} t={t} />
                        </div>
                    ) : null}
                </div>
                {device ? (
                    <Button variant="outline" size="sm" onClick={() => onToggleTrial(device)}>
                        {t(device.claimed_trial ? "actions.reset_trial" : "actions.mark_claimed")}
                    </Button>
                ) : null}
            </div>

            <div className="flex-1 space-y-6 overflow-y-auto p-5">
                {loading ? (
                    <div className="text-sm text-muted-foreground">...</div>
                ) : error ? (
                    <div className="text-sm text-destructive">{error}</div>
                ) : (
                    <>
                        <section className="space-y-2">
                            <h3 className="text-sm font-semibold">{t("detail.accounts")}</h3>
                            <Table>
                                <TableHeader>
                                    <TableRow>
                                        <SortableHead sortKey="id" sort={accountSort.sort} onSort={accountSort.toggle}>{t("detail.user_id")}</SortableHead>
                                        <SortableHead sortKey="email" sort={accountSort.sort} onSort={accountSort.toggle}>{t("detail.email")}</SortableHead>
                                        <SortableHead sortKey="plan" sort={accountSort.sort} onSort={accountSort.toggle}>{t("detail.plan")}</SortableHead>
                                        <SortableHead sortKey="expired_at" sort={accountSort.sort} onSort={accountSort.toggle}>{t("detail.expired_at")}</SortableHead>
                                        <SortableHead sortKey="created_at" sort={accountSort.sort} onSort={accountSort.toggle}>{t("detail.created_at")}</SortableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {accounts.length === 0 ? (
                                        <TableEmpty colSpan={5}>{t("detail.no_accounts")}</TableEmpty>
                                    ) : accounts.map((u) => (
                                        <TableRow key={u.id}>
                                            <TableCell><Badge variant="outline">{u.id}</Badge></TableCell>
                                            <TableCell>
                                                <div className="flex flex-wrap items-center gap-1.5">
                                                    <span className="break-all">{u.email}</span>
                                                    {u.is_current ? <Badge variant="default">{t("detail.current")}</Badge> : null}
                                                    {u.is_device_trial ? <Badge variant="outline">{t("detail.device_trial_account")}</Badge> : null}
                                                    {u.banned ? <Badge variant="destructive">{t("user_banned")}</Badge> : null}
                                                </div>
                                            </TableCell>
                                            <TableCell className="text-sm">{u.plan_name || "-"}</TableCell>
                                            <TableCell className="text-sm text-muted-foreground">{formatDateTime(u.expired_at)}</TableCell>
                                            <TableCell className="text-sm text-muted-foreground">{formatDateTime(u.created_at)}</TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </section>

                        <section className="space-y-2">
                            <h3 className="text-sm font-semibold">{t("detail.trial_logs")}</h3>
                            <Table>
                                <TableHeader>
                                    <TableRow>
                                        <SortableHead sortKey="granted_at" sort={logSort.sort} onSort={logSort.toggle}>{t("detail.granted_at")}</SortableHead>
                                        <SortableHead sortKey="source" sort={logSort.sort} onSort={logSort.toggle}>{t("detail.source")}</SortableHead>
                                        <SortableHead sortKey="email" sort={logSort.sort} onSort={logSort.toggle}>{t("detail.email")}</SortableHead>
                                        <SortableHead sortKey="expired_at" sort={logSort.sort} onSort={logSort.toggle}>{t("detail.trial_expired_at")}</SortableHead>
                                        <SortableHead sortKey="ip" sort={logSort.sort} onSort={logSort.toggle}>{t("detail.ip")}</SortableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {logs.length === 0 ? (
                                        <TableEmpty colSpan={5}>{t("detail.no_logs")}</TableEmpty>
                                    ) : logs.map((log) => (
                                        <TableRow key={log.id}>
                                            <TableCell className="text-sm text-muted-foreground">{formatDateTime(log.created_at)}</TableCell>
                                            <TableCell><Badge variant="outline">{sourceLabel(log.source, t)}</Badge></TableCell>
                                            <TableCell className="break-all text-sm">{log.email || "-"} <span className="text-muted-foreground">#{log.user_id}</span></TableCell>
                                            <TableCell className="text-sm text-muted-foreground">{formatDateTime(log.expired_at)}</TableCell>
                                            <TableCell className="font-mono text-xs">{log.ip || "-"}</TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </section>
                    </>
                )}
            </div>
        </Dialog>
    );
}

export function DeviceManagement() {
    const { t } = useTranslation("device");
    const { t: tc } = useTranslation("common");

    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const [page, setPage] = useState(1);
    const [claimed, setClaimed] = useState("all");
    const [keyword, setKeyword] = useState("");
    const [appliedKeyword, setAppliedKeyword] = useState("");
    const [version, setVersion] = useState(0);
    const { sort, toggle: toggleSort, sortParam } = useServerSort(() => setPage(1));

    const [detailId, setDetailId] = useState(null);
    const [detailOpen, setDetailOpen] = useState(false);
    const [toggleTarget, setToggleTarget] = useState(null);
    const [detailVersion, setDetailVersion] = useState(0);

    const load = () => {
        setLoading(true);
        setError("");
        const params = { pageSize: PAGE_SIZE, current: page, keyword: appliedKeyword, sort: sortParam };
        if (claimed !== "all") params.claimed_trial = claimed;
        return fetchDevices(params)
            .then((res) => {
                setItems(res?.data || []);
                setTotal(res?.total ?? 0);
            })
            .catch((err) => setError(getErrorMessage(err, t("messages.failed"))))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [page, version, sort.key, sort.dir]);

    const refetch = () => {
        if (page !== 1) setPage(1);
        else setVersion((v) => v + 1);
    };

    const search = () => {
        setAppliedKeyword(keyword.trim());
        refetch();
    };

    const handleToggle = async () => {
        if (!toggleTarget) return;
        try {
            await setDeviceTrialClaimed(toggleTarget.device_id, !toggleTarget.claimed_trial);
        } catch (err) {
            setError(getErrorMessage(err, t("messages.failed")));
        }
        setToggleTarget(null);
        setDetailVersion((v) => v + 1);
        await load();
    };

    const resetting = Boolean(toggleTarget?.claimed_trial);

    return (
        <div className="p-4 md:p-6 space-y-4">
            <div>
                <h2 className="text-2xl font-bold tracking-tight">{t("title")}</h2>
                <p className="mt-2 text-muted-foreground">{t("description")}</p>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex flex-wrap items-center gap-2">
                    <Select value={claimed} onValueChange={(v) => { setClaimed(v); refetch(); }}>
                        <SelectTrigger className="h-9 w-[150px]">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">{t("filter.all")}</SelectItem>
                            <SelectItem value="1">{t("filter.claimed")}</SelectItem>
                            <SelectItem value="0">{t("filter.unclaimed")}</SelectItem>
                        </SelectContent>
                    </Select>
                    <Input
                        value={keyword}
                        onChange={(e) => setKeyword(e.target.value)}
                        onKeyDown={(e) => { if (e.key === "Enter") search(); }}
                        placeholder={t("search_placeholder")}
                        className="h-9 w-[260px]"
                    />
                    <Button variant="outline" size="sm" onClick={search}>{t("actions.search")}</Button>
                </div>
                <Button variant="outline" size="sm" onClick={load}>↻</Button>
            </div>

            {error ? <div className="text-sm text-destructive">{error}</div> : null}

            <Table>
                <TableHeader>
                    <TableRow>
                        <SortableHead sortKey="device_id" sort={sort} onSort={toggleSort}>{t("columns.device_id")}</SortableHead>
                        <SortableHead sortKey="display_code" sort={sort} onSort={toggleSort}>{t("columns.display_code")}</SortableHead>
                        <SortableHead sortKey="current_user" sort={sort} onSort={toggleSort}>{t("columns.current_user")}</SortableHead>
                        <SortableHead sortKey="claimed_trial" sort={sort} onSort={toggleSort}>{t("columns.claimed_trial")}</SortableHead>
                        <SortableHead sortKey="accounts" sort={sort} onSort={toggleSort}>{t("columns.accounts")}</SortableHead>
                        <SortableHead sortKey="trials" sort={sort} onSort={toggleSort}>{t("columns.trials")}</SortableHead>
                        <SortableHead sortKey="updated_at" sort={sort} onSort={toggleSort}>{t("columns.updated_at")}</SortableHead>
                        <SortableHead sortKey="created_at" sort={sort} onSort={toggleSort}>{t("columns.created_at")}</SortableHead>
                        <TableHead className="text-right">{t("columns.actions")}</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {loading ? (
                        <TableEmpty colSpan={9}>...</TableEmpty>
                    ) : items.length === 0 ? (
                        <TableEmpty colSpan={9}>{t("list.empty")}</TableEmpty>
                    ) : (
                        items.map((d) => (
                            <TableRow key={d.id}>
                                <TableCell className="font-mono text-xs" title={d.device_id}>{shortDeviceId(d.device_id)}</TableCell>
                                <TableCell className="font-mono text-xs">{d.display_code}</TableCell>
                                <TableCell className="max-w-[240px] text-sm">
                                    {d.user_email ? (
                                        <div className="flex flex-wrap items-center gap-1.5">
                                            <span className="truncate">{d.user_email}</span>
                                            {d.user_banned ? <Badge variant="destructive">{t("user_banned")}</Badge> : null}
                                        </div>
                                    ) : <span className="text-muted-foreground">{t("no_user")}</span>}
                                </TableCell>
                                <TableCell><TrialBadge claimed={d.claimed_trial} t={t} /></TableCell>
                                <TableCell className="text-sm">{d.account_count}</TableCell>
                                <TableCell className="text-sm">{d.trial_count}</TableCell>
                                <TableCell className="text-sm text-muted-foreground">{formatDateTime(d.updated_at)}</TableCell>
                                <TableCell className="text-sm text-muted-foreground">{formatDateTime(d.created_at)}</TableCell>
                                <TableCell>
                                    <div className="flex items-center justify-end gap-1">
                                        <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => { setDetailId(d.device_id); setDetailOpen(true); }}>
                                            {t("actions.view")}
                                        </Button>
                                        <Button variant="ghost" size="sm" className="h-7 px-2 text-xs text-muted-foreground" onClick={() => setToggleTarget(d)}>
                                            {t(d.claimed_trial ? "actions.reset_trial" : "actions.mark_claimed")}
                                        </Button>
                                    </div>
                                </TableCell>
                            </TableRow>
                        ))
                    )}
                </TableBody>
            </Table>

            <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">{total}</span>
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>‹</Button>
                    <span className="text-sm">{page}</span>
                    <Button variant="outline" size="sm" disabled={page * PAGE_SIZE >= total} onClick={() => setPage((p) => p + 1)}>›</Button>
                </div>
            </div>

            <DeviceDetailDialog
                deviceId={detailId}
                open={detailOpen}
                reloadKey={detailVersion}
                onOpenChange={setDetailOpen}
                onToggleTrial={setToggleTarget}
                t={t}
            />

            <ConfirmDialog
                open={Boolean(toggleTarget)}
                onOpenChange={(open) => !open && setToggleTarget(null)}
                title={t(resetting ? "actions.reset_confirm_title" : "actions.mark_confirm_title")}
                description={t(resetting ? "actions.reset_confirm_description" : "actions.mark_confirm_description")}
                confirmLabel={t("actions.confirm")}
                cancelLabel={tc("cancel")}
                variant={resetting ? "destructive" : "default"}
                onConfirm={handleToggle}
            />
        </div>
    );
}
