import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { getCurrencySymbol, useCurrencySymbol } from "../../store/config.js";
import { fetchOrders, getOrderDetail, payOrder, cancelOrder, updateOrder, assignOrder } from "../../api/order.js";
import { fetchPlans } from "../../api/plan.js";
import { useQueryParam } from "../../composables/useQueryParam.js";
import { navigate } from "../../router/index.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useServerSort } from "../../components/SortableHead.jsx";
import { SearchFilterToolbar } from "../../components/SearchFilterToolbar.jsx";
import { OrderCleanupCard } from "./OrderCleanupCard.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";

const STATUS = { PENDING: 0, PROCESSING: 1, CANCELLED: 2, COMPLETED: 3, DISCOUNTED: 4 };
const STATUS_LABEL = { 0: "PENDING", 1: "PROCESSING", 2: "CANCELLED", 3: "COMPLETED", 4: "DISCOUNTED" };
const STATUS_CLASS = {
    0: "bg-yellow-100 text-yellow-800",
    1: "bg-blue-100 text-blue-800",
    2: "bg-red-100 text-red-800",
    3: "bg-green-100 text-green-800",
    4: "bg-gray-100 text-gray-800",
};

const COMMISSION = { PENDING: 0, PROCESSING: 1, VALID: 2, INVALID: 3 };
const COMMISSION_LABEL = { 0: "PENDING", 1: "PROCESSING", 2: "VALID", 3: "INVALID" };
const COMMISSION_CLASS = {
    0: "bg-yellow-100 text-yellow-800",
    1: "bg-blue-100 text-blue-800",
    2: "bg-green-100 text-green-800",
    3: "bg-red-100 text-red-800",
};

const TYPE = { NEW: 1, RENEWAL: 2, UPGRADE: 3, RESET_FLOW: 4 };
const TYPE_LABEL = { 1: "NEW", 2: "RENEWAL", 3: "UPGRADE", 4: "RESET_FLOW" };

const PERIODS = ["month_price", "quarter_price", "half_year_price", "year_price", "two_year_price", "three_year_price", "onetime_price", "reset_price"];

const PAGE_SIZE = 20;

function formatMoney(cents) {
    const value = typeof cents === "number" ? cents : 0;
    return `${(value / 100).toFixed(0).replace(/\B(?=(\d{3})+(?!\d))/g, ",")}${getCurrencySymbol()}`;
}

function formatDateTime(seconds) {
    if (!seconds) return "-";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

// 支付方式：优先显示支付通道名；没有通道时按余额抵扣 / 后台手动确认区分，其余为 "-"。
function paymentLabel(order, t) {
    if (order.payment?.name) return order.payment.name;
    if (order.balance_amount > 0) return t("table.columns.paymentBalance", "余额");
    if (order.callback_no === "manual_operation") return t("table.columns.paymentManual", "手动确认");
    return "-";
}

function DetailRow({ label, value, valueClassName = "" }) {
    return (
        <div className="flex items-center py-1.5">
            <div className="w-28 shrink-0 text-sm text-muted-foreground">{label}</div>
            <div className={`text-sm ${valueClassName}`}>{value || "-"}</div>
        </div>
    );
}

const EMPTY_ASSIGN = { email: "", plan_id: "", period: "", total_amount: "" };

export function OrderManagement() {
    const { t } = useTranslation("order");
    const auth = useAuthState();
    useCurrencySymbol();
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [page, setPage] = useState(1);
    const { sort, toggle: toggleSort, sortParam } = useServerSort(() => setPage(1));

    // 从用户列表点「订单」进入时地址里带 ?user_id=eq:<ID>：只显示该用户的订单。
    const userIdParam = useQueryParam("user_id");
    const userIdFilter = useMemo(() => {
        const raw = String(userIdParam || "").replace(/^eq:/i, "").trim();
        return /^\d+$/.test(raw) ? raw : "";
    }, [userIdParam]);

    const [search, setSearch] = useState("");
    const [typeFilter, setTypeFilter] = useState("all");
    const [periodFilter, setPeriodFilter] = useState("all");
    const [statusFilter, setStatusFilter] = useState("all");
    const [commissionFilter, setCommissionFilter] = useState("all");

    const [plans, setPlans] = useState([]);
    const [assignOpen, setAssignOpen] = useState(false);
    const [assignForm, setAssignForm] = useState(EMPTY_ASSIGN);
    const [assignError, setAssignError] = useState("");

    const [detailId, setDetailId] = useState(null);
    const [detail, setDetail] = useState(null);

    const buildFilter = () => {
        const filter = [];
        if (userIdFilter) filter.push({ id: "user_id", value: `eq:${userIdFilter}` });
        if (search.trim()) filter.push({ id: "trade_no", value: search.trim() });
        if (typeFilter !== "all") filter.push({ id: "type", value: parseInt(typeFilter, 10) });
        if (periodFilter !== "all") filter.push({ id: "period", value: periodFilter });
        if (statusFilter !== "all") filter.push({ id: "status", value: parseInt(statusFilter, 10) });
        if (commissionFilter !== "all") filter.push({ id: "commission_status", value: parseInt(commissionFilter, 10) });
        return filter;
    };

    const [filtersVersion, setFiltersVersion] = useState(0);

    const load = () => {
        setLoading(true);
        return fetchOrders({ pageSize: PAGE_SIZE, current: page, filter: buildFilter(), sort: sortParam })
            .then((res) => {
                setItems(res?.data || []);
                setTotal(res?.total || 0);
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [auth.token, page, filtersVersion, userIdFilter, sort.key, sort.dir]);

    // 切换用户筛选时回到第一页
    useEffect(() => {
        setPage(1);
    }, [userIdFilter]);

    useEffect(() => {
        fetchPlans().then((res) => setPlans(res?.data || [])).catch(() => setPlans([]));
    }, [auth.token]);

    const refetch = () => {
        if (page === 1) setFiltersVersion((v) => v + 1);
        else setPage(1);
    };

    const applyFilters = () => refetch();

    const resetFilters = () => {
        setSearch("");
        setTypeFilter("all");
        setPeriodFilter("all");
        setStatusFilter("all");
        setCommissionFilter("all");
        refetch();
    };

    useEffect(() => {
        refetch();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [typeFilter, periodFilter, statusFilter, commissionFilter]);

    useEffect(() => {
        if (!detailId) {
            setDetail(null);
            return;
        }
        getOrderDetail({ id: detailId }).then((res) => setDetail(res?.data || null));
    }, [detailId]);

    const planOptions = useMemo(() => plans.map((p) => ({ value: p.id, label: p.name })), [plans]);

    const handleMarkPaid = async (order) => {
        await payOrder({ trade_no: order.trade_no });
        await load();
    };

    const handleCancel = async (order) => {
        await cancelOrder({ trade_no: order.trade_no });
        await load();
    };

    const handleCommissionUpdate = async (order, commission_status) => {
        await updateOrder({ trade_no: order.trade_no, commission_status });
        await load();
        if (detailId) {
            const res = await getOrderDetail({ id: detailId });
            setDetail(res?.data || null);
        }
    };

    const openAssign = () => {
        setAssignForm(EMPTY_ASSIGN);
        setAssignError("");
        setAssignOpen(true);
    };

    const handleAssignSubmit = async () => {
        setAssignError("");
        if (!assignForm.email.trim()) {
            const error = t("dialog.placeholders.email");
            setAssignError(error);
            throw new Error(error);
        }
        if (!assignForm.plan_id) {
            const error = t("dialog.placeholders.plan");
            setAssignError(error);
            throw new Error(error);
        }
        if (!assignForm.period) {
            const error = t("dialog.placeholders.period");
            setAssignError(error);
            throw new Error(error);
        }
        const payload = {
            email: assignForm.email.trim(),
            plan_id: parseInt(assignForm.plan_id, 10),
            period: assignForm.period,
            total_amount: Math.round((parseFloat(assignForm.total_amount) || 0) * 100),
        };
        await assignOrder(payload);
        await load();
    };

    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

    return (
        <div className="p-4 md:p-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-lg font-medium">{t("title")}</h3>
                <p className="text-sm text-muted-foreground">{t("description")}</p>
            </div>
            <hr className="my-6" />

            <OrderCleanupCard onCleaned={load} />

            {userIdFilter ? (
                <div className="mb-4 flex flex-wrap items-center gap-3 rounded-lg border border-primary/30 bg-accent px-4 py-2.5 text-sm">
                    <span className="font-medium text-accent-foreground">
                        {t("userFilter.active", { id: userIdFilter }, "正在查看用户 #{{id}} 的订单")}
                    </span>
                    <span className="text-muted-foreground">{t("userFilter.total", { count: total }, "共 {{count}} 笔")}</span>
                    <Button variant="outline" size="sm" className="ml-auto h-7 px-2 text-xs" type="button"
                        onClick={() => navigate("/finance/order")}>
                        {t("userFilter.clear", "显示全部订单")}
                    </Button>
                </div>
            ) : null}

            <SearchFilterToolbar
                search={search}
                onSearchChange={setSearch}
                searchPlaceholder={t("search.placeholder")}
                resetLabel={t("actions.reset")}
                onReset={resetFilters}
                filters={
                    <>
                        <Select value={typeFilter} onValueChange={setTypeFilter} className="sm:w-36">
                            <SelectTrigger><SelectValue placeholder={t("table.columns.type")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("table.columns.type")}</SelectItem>
                                {Object.entries(TYPE).map(([key, value]) => (
                                    <SelectItem key={key} value={String(value)}>{t(`type.${key}`)}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Select value={periodFilter} onValueChange={setPeriodFilter} className="sm:w-36">
                            <SelectTrigger><SelectValue placeholder={t("table.columns.period")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("table.columns.period")}</SelectItem>
                                {PERIODS.map((p) => (
                                    <SelectItem key={p} value={p}>{t(`period.${p}`)}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Select value={statusFilter} onValueChange={setStatusFilter} className="sm:w-36">
                            <SelectTrigger><SelectValue placeholder={t("table.columns.status")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("table.columns.status")}</SelectItem>
                                {Object.entries(STATUS).map(([key, value]) => (
                                    <SelectItem key={key} value={String(value)}>{t(`status.${key}`)}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Select value={commissionFilter} onValueChange={setCommissionFilter} className="sm:w-36">
                            <SelectTrigger><SelectValue placeholder={t("table.columns.commissionStatus")} /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t("table.columns.commissionStatus")}</SelectItem>
                                {Object.entries(COMMISSION).map(([key, value]) => (
                                    <SelectItem key={key} value={String(value)}>{t(`commission.${key}`)}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Button variant="secondary" size="sm" onClick={applyFilters} type="button">
                            {t("common:search")}
                        </Button>
                    </>
                }
                actions={<Button onClick={openAssign}>{t("dialog.addOrder")}</Button>}
            />

            <div className="mt-4">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <SortableHead sortKey="trade_no" sort={sort} onSort={toggleSort}>{t("table.columns.tradeNo")}</SortableHead>
                            <SortableHead sortKey="type" sort={sort} onSort={toggleSort}>{t("table.columns.type")}</SortableHead>
                            <SortableHead sortKey="plan" sort={sort} onSort={toggleSort}>{t("table.columns.plan")}</SortableHead>
                            <SortableHead sortKey="period" sort={sort} onSort={toggleSort}>{t("table.columns.period")}</SortableHead>
                            <SortableHead sortKey="total_amount" sort={sort} onSort={toggleSort}>{t("table.columns.amount")}</SortableHead>
                            <SortableHead sortKey="status" sort={sort} onSort={toggleSort}>{t("table.columns.status")}</SortableHead>
                            <SortableHead sortKey="payment" sort={sort} onSort={toggleSort}>{t("table.columns.payment", "支付方式")}</SortableHead>
                            <SortableHead sortKey="payer" sort={sort} onSort={toggleSort}>{t("table.columns.payer", "支付人")}</SortableHead>
                            <SortableHead sortKey="created_at" sort={sort} onSort={toggleSort}>{t("table.columns.createdAt")}</SortableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableEmpty colSpan={9}>...</TableEmpty>
                        ) : items.length === 0 ? (
                            <TableEmpty colSpan={9}>-</TableEmpty>
                        ) : (
                            items.map((order) => {
                                const tradeNo = order.trade_no || "";
                                const shortTrade = tradeNo.length > 6 ? `${tradeNo.slice(0, 3)}...${tradeNo.slice(-3)}` : tradeNo;
                                const statusKey = STATUS_LABEL[order.status];
                                const typeKey = TYPE_LABEL[order.type];
                                return (
                                    <TableRow key={order.id}>
                                        <TableCell>
                                            <Button variant="ghost" size="sm" className="h-8 px-2 font-mono text-primary" onClick={() => setDetailId(order.id)}>
                                                {shortTrade}
                                            </Button>
                                        </TableCell>
                                        <TableCell>
                                            <Badge variant="outline">{typeKey ? t(`type.${typeKey}`) : String(order.type ?? "-")}</Badge>
                                        </TableCell>
                                        <TableCell className="max-w-[200px] truncate font-medium">{order.plan?.name || "-"}</TableCell>
                                        <TableCell>
                                            <Badge variant="outline">{order.period ? t(`period.${order.period}`, { defaultValue: order.period }) : "-"}</Badge>
                                        </TableCell>
                                        <TableCell className="font-mono">{formatMoney(order.total_amount)}</TableCell>
                                        <TableCell>
                                            <div className="flex items-center gap-2">
                                                <span className={`w-fit rounded-md px-2 py-0.5 text-xs font-medium ${STATUS_CLASS[order.status] || ""}`}>
                                                    {statusKey ? t(`status.${statusKey}`) : String(order.status ?? "-")}
                                                </span>
                                                {order.status === STATUS.PENDING && (
                                                    <div className="flex gap-1">
                                                        <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => handleMarkPaid(order)}>
                                                            {t("actions.markAsPaid")}
                                                        </Button>
                                                        <Button variant="destructive" size="sm" className="h-7 px-2 text-xs" onClick={() => handleCancel(order)}>
                                                            {t("actions.cancel")}
                                                        </Button>
                                                    </div>
                                                )}
                                            </div>
                                        </TableCell>
                                        <TableCell className="whitespace-nowrap">{paymentLabel(order, t)}</TableCell>
                                        <TableCell className="max-w-[220px]">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                className="h-8 max-w-full justify-start px-2 text-primary"
                                                title={t("table.columns.payerHint", "点击查看该用户的全部订单")}
                                                onClick={() => navigate(`/finance/order?user_id=eq:${order.user_id}`)}
                                            >
                                                <span className="truncate">{order.user?.email || `#${order.user_id}`}</span>
                                            </Button>
                                        </TableCell>
                                        <TableCell className="whitespace-nowrap font-mono text-xs text-muted-foreground">{formatDateTime(order.created_at)}</TableCell>
                                    </TableRow>
                                );
                            })
                        )}
                    </TableBody>
                </Table>
            </div>

            {totalPages > 1 && (
                <div className="mt-4 flex items-center justify-end gap-2">
                    <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
                        ‹
                    </Button>
                    <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
                    <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>
                        ›
                    </Button>
                </div>
            )}

            <FormDialog
                open={assignOpen}
                onOpenChange={setAssignOpen}
                title={t("dialog.assignOrder")}
                cancelLabel={t("dialog.actions.cancel")}
                submitLabel={t("dialog.actions.confirm")}
                onSubmit={handleAssignSubmit}
            >
                {assignError && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{assignError}</div>}
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("dialog.fields.userEmail")}</div>
                    <Input value={assignForm.email} placeholder={t("dialog.placeholders.email")} onChange={(e) => setAssignForm((p) => ({ ...p, email: e.target.value }))} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("dialog.fields.subscriptionPlan")}</div>
                    <Select value={assignForm.plan_id ? String(assignForm.plan_id) : ""} onValueChange={(value) => setAssignForm((p) => ({ ...p, plan_id: value }))}>
                        <SelectTrigger><SelectValue placeholder={t("dialog.placeholders.plan")} /></SelectTrigger>
                        <SelectContent>
                            {planOptions.map((o) => (
                                <SelectItem key={o.value} value={String(o.value)}>{o.label}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("dialog.fields.orderPeriod")}</div>
                    <Select value={assignForm.period} onValueChange={(value) => setAssignForm((p) => ({ ...p, period: value }))}>
                        <SelectTrigger><SelectValue placeholder={t("dialog.placeholders.period")} /></SelectTrigger>
                        <SelectContent>
                            {PERIODS.map((p) => (
                                <SelectItem key={p} value={p}>{t(`period.${p}`)}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("dialog.fields.paymentAmount")}</div>
                    <Input type="number" step="any" min={0} value={assignForm.total_amount} placeholder={t("dialog.placeholders.amount")} onChange={(e) => setAssignForm((p) => ({ ...p, total_amount: e.target.value }))} />
                </div>
            </FormDialog>

            <Dialog open={Boolean(detailId)} onOpenChange={(open) => !open && setDetailId(null)} className="sm:max-w-xl">
                <div className="max-h-[90vh] overflow-y-auto p-6">
                    <h3 className="text-lg font-medium">{t("dialog.title")}</h3>
                    <div className="mt-2 flex items-center justify-between text-sm">
                        <div className="text-muted-foreground">{t("table.columns.tradeNo")}：{detail?.trade_no}</div>
                        {detail?.status !== undefined && STATUS_LABEL[detail.status] && (
                            <span className={`w-fit rounded-md px-2 py-0.5 text-xs font-medium ${STATUS_CLASS[detail.status] || ""}`}>
                                {t(`status.${STATUS_LABEL[detail.status]}`)}
                            </span>
                        )}
                    </div>

                    {detail ? (
                        <div className="mt-4 space-y-4">
                            <div className="rounded-lg border p-4">
                                <div className="mb-2 text-sm font-medium">{t("dialog.basicInfo")}</div>
                                <DetailRow label={t("dialog.fields.userEmail")} value={detail.user?.email} />
                                <DetailRow label={t("dialog.fields.orderPeriod")} value={detail.period ? t(`period.${detail.period}`, { defaultValue: detail.period }) : "-"} />
                                <DetailRow label={t("dialog.fields.subscriptionPlan")} value={detail.plan?.name} valueClassName="font-medium" />
                                <DetailRow label={t("dialog.fields.callbackNo")} value={detail.callback_no} valueClassName="font-mono text-xs" />
                            </div>
                            <div className="rounded-lg border p-4">
                                <div className="mb-2 text-sm font-medium">{t("dialog.amountInfo")}</div>
                                <DetailRow label={t("dialog.fields.paymentAmount")} value={formatMoney(detail.total_amount)} valueClassName="font-medium text-primary" />
                                <hr className="my-2" />
                                <DetailRow label={t("dialog.fields.balancePayment")} value={formatMoney(detail.balance_amount)} />
                                <DetailRow label={t("dialog.fields.discountAmount")} value={formatMoney(detail.discount_amount)} valueClassName="text-green-600" />
                                <DetailRow label={t("dialog.fields.refundAmount")} value={formatMoney(detail.refund_amount)} valueClassName="text-red-600" />
                                <DetailRow label={t("dialog.fields.deductionAmount")} value={formatMoney(detail.surplus_amount)} />
                            </div>
                            <div className="rounded-lg border p-4">
                                <div className="mb-2 text-sm font-medium">{t("dialog.timeInfo")}</div>
                                <DetailRow label={t("dialog.fields.createdAt")} value={formatDateTime(detail.created_at)} valueClassName="font-mono text-xs" />
                                <DetailRow label={t("dialog.fields.updatedAt")} value={formatDateTime(detail.updated_at)} valueClassName="font-mono text-xs" />
                            </div>
                            {detail.commission_status !== undefined && Boolean(detail.commission_balance) && (
                                <div className="rounded-lg border p-4">
                                    <div className="mb-2 text-sm font-medium">{t("dialog.commissionInfo")}</div>
                                    <DetailRow
                                        label={t("dialog.fields.commissionStatus")}
                                        value={
                                            <span className={`w-fit rounded-md px-2 py-0.5 text-xs font-medium ${COMMISSION_CLASS[detail.commission_status] || ""}`}>
                                                {COMMISSION_LABEL[detail.commission_status] ? t(`commission.${COMMISSION_LABEL[detail.commission_status]}`) : t("dialog.commissionStatusActive")}
                                            </span>
                                        }
                                    />
                                    {detail.commission_status === COMMISSION.PENDING && detail.status === STATUS.COMPLETED && (
                                        <div className="flex gap-2 py-1.5">
                                            <Button variant="outline" size="sm" className="h-7 px-2 text-xs" onClick={() => handleCommissionUpdate(detail, COMMISSION.PROCESSING)}>
                                                {t("commission.PROCESSING")}
                                            </Button>
                                            <Button variant="destructive" size="sm" className="h-7 px-2 text-xs" onClick={() => handleCommissionUpdate(detail, COMMISSION.INVALID)}>
                                                {t("commission.INVALID")}
                                            </Button>
                                        </div>
                                    )}
                                    <DetailRow label={t("dialog.fields.commissionAmount")} value={formatMoney(detail.commission_balance)} valueClassName="font-medium text-orange-600" />
                                    {Boolean(detail.actual_commission_balance) && (
                                        <DetailRow label={t("dialog.fields.actualCommissionAmount")} value={formatMoney(detail.actual_commission_balance)} valueClassName="font-medium text-orange-700" />
                                    )}
                                    {detail.invite_user && (
                                        <>
                                            <hr className="my-2" />
                                            <DetailRow label={t("dialog.fields.inviteUser")} value={detail.invite_user.email} />
                                            <DetailRow label={t("dialog.fields.inviteUserId")} value={detail.invite_user.id} valueClassName="font-mono text-xs" />
                                        </>
                                    )}
                                </div>
                            )}
                        </div>
                    ) : (
                        <div className="mt-4 text-sm text-muted-foreground">...</div>
                    )}

                    <div className="mt-6 flex justify-end">
                        <Button variant="outline" onClick={() => setDetailId(null)}>{t("dialog.actions.cancel")}</Button>
                    </div>
                </div>
            </Dialog>
        </div>
    );
}
