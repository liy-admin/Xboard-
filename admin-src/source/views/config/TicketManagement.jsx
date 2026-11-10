import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { navigate } from "../../router/index.js";
import { fetchTickets, fetchTicketById, replyTicket, closeTicket } from "../../api/ticket.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableEmpty } from "../../components/Table.jsx";
import { SortableHead, useServerSort } from "../../components/SortableHead.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";

const PAGE_SIZE = 20;

const STATUS_OPENING = 0;
const STATUS_CLOSED = 1;
const LEVEL_LOW = 0;
const LEVEL_MIDDLE = 1;
const LEVEL_HIGH = 2;

function formatDateTime(seconds) {
    if (!seconds) return "-";
    const d = new Date(seconds * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function levelLabel(level, t) {
    return t(level === LEVEL_LOW ? "level.low" : level === LEVEL_MIDDLE ? "level.medium" : "level.high");
}

function levelBadgeVariant(level) {
    return level === LEVEL_LOW ? "default" : level === LEVEL_MIDDLE ? "secondary" : "destructive";
}

function statusInfo(ticket, t) {
    if (ticket.status === STATUS_CLOSED) {
        return { label: t("status.closed"), variant: "secondary" };
    }
    if (ticket.reply_status === 1) {
        return { label: t("status.replied"), variant: "secondary" };
    }
    return { label: t("status.pending"), variant: "destructive" };
}

function TicketDetailDialog({ ticketId, statusFilter, open, onOpenChange, onChanged, t, tc }) {
    const [ticket, setTicket] = useState(null);
    const [loading, setLoading] = useState(false);
    const [message, setMessage] = useState("");
    const [sending, setSending] = useState(false);
    const [closeOpen, setCloseOpen] = useState(false);
    const scrollRef = useRef(null);
    const inputRef = useRef(null);

    const load = () => {
        if (!ticketId) return Promise.resolve();
        setLoading(true);
        return fetchTicketById(ticketId)
            .then((res) => setTicket(res?.data || null))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        if (open && ticketId) {
            load();
        } else {
            setMessage("");
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open, ticketId]);

    useEffect(() => {
        if (!open || !scrollRef.current) return;
        const el = scrollRef.current;
        el.scrollTop = el.scrollHeight;
    }, [open, ticket?.messages]);

    const handleSend = async () => {
        const value = message.trim();
        if (!value || sending || !ticketId) return;
        setSending(true);
        try {
            await replyTicket({ id: ticketId, message: value });
            setMessage("");
            await load();
            onChanged?.();
            inputRef.current?.focus();
        } finally {
            setSending(false);
        }
    };

    const handleClose = async () => {
        await closeTicket(ticketId);
        setCloseOpen(false);
        await load();
        onChanged?.();
    };

    const closed = ticket?.status === STATUS_CLOSED;
    const info = ticket ? statusInfo(ticket, t) : null;

    return (
        <Dialog open={open} onOpenChange={onOpenChange} className="flex h-[85vh] max-w-4xl flex-col gap-0 overflow-hidden p-0">
            {loading || !ticket ? (
                <div className="flex h-full items-center justify-center text-sm text-muted-foreground">...</div>
            ) : (
                <div className="flex h-full flex-col">
                    <div className="flex flex-col gap-3 border-b px-5 py-4">
                        <div className="flex flex-wrap items-start justify-between gap-3">
                            <div className="min-w-0 space-y-1.5">
                                <div className="flex flex-wrap items-center gap-2">
                                    <h2 className="truncate text-lg font-semibold leading-tight">{ticket.subject}</h2>
                                    <Badge variant={info.variant}>{info.label}</Badge>
                                </div>
                                <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                                    <span>{ticket.user?.email}</span>
                                    <span>·</span>
                                    <span>{t("detail.created_at")} {formatDateTime(ticket.created_at)}</span>
                                    <span>·</span>
                                    <Badge variant="outline" className="text-xs">{levelLabel(ticket.level, t)}</Badge>
                                </div>
                            </div>
                            <div className="flex items-center gap-1.5">
                                {ticket.user ? (
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => {
                                            onOpenChange(false);
                                            navigate(`/finance/order?user_id=${ticket.user.id}`);
                                        }}
                                    >
                                        {t("detail.order_records")}
                                    </Button>
                                ) : null}
                                {!closed ? (
                                    <Button variant="outline" size="sm" onClick={() => setCloseOpen(true)}>
                                        {t("actions.close_ticket")}
                                    </Button>
                                ) : null}
                            </div>
                        </div>
                    </div>

                    <div ref={scrollRef} className="flex-1 space-y-4 overflow-y-auto p-4 md:p-6">
                        {!ticket.messages || ticket.messages.length === 0 ? (
                            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">{t("detail.no_messages")}</div>
                        ) : (
                            ticket.messages.map((m) => {
                                const fromAdmin = m.is_from_admin ?? !m.is_me;
                                return (
                                    <div key={m.id} className={`flex ${fromAdmin ? "justify-end" : "justify-start"}`}>
                                        <div className={`max-w-[60%] rounded-lg p-3 ${fromAdmin ? "rounded-tr-none bg-primary text-primary-foreground" : "rounded-tl-none bg-secondary text-secondary-foreground"}`}>
                                            <div className="text-[10px] font-medium opacity-70">{t(fromAdmin ? "detail.sender_admin" : "detail.sender_user")}</div>
                                            <div className="whitespace-pre-wrap break-words text-sm">{m.message}</div>
                                            <div className="mt-1 text-right text-[10px] opacity-70">{formatDateTime(m.created_at)}</div>
                                        </div>
                                    </div>
                                );
                            })
                        )}
                    </div>

                    <div className="border-t p-4">
                        {closed ? <p className="mb-2 text-xs text-muted-foreground">{t("detail.input.closed_hint")}</p> : null}
                        <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
                            <Textarea
                                ref={inputRef}
                                disabled={sending}
                                placeholder={t(closed ? "detail.input.closed_reply_placeholder" : "detail.input.reply_placeholder")}
                                className="flex-1 resize-none"
                                rows={2}
                                value={message}
                                onChange={(e) => setMessage(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === "Enter" && !e.shiftKey) {
                                        e.preventDefault();
                                        handleSend();
                                    }
                                }}
                            />
                            <Button className="w-full sm:w-auto" disabled={sending || !message.trim()} onClick={handleSend}>
                                {t(sending ? "detail.input.sending" : "detail.input.send")}
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            <ConfirmDialog
                open={closeOpen}
                onOpenChange={setCloseOpen}
                title={t("actions.close_confirm_title")}
                description={t("actions.close_confirm_description")}
                confirmLabel={t("actions.close_confirm_button")}
                cancelLabel={tc("cancel")}
                variant="destructive"
                onConfirm={handleClose}
            />
        </Dialog>
    );
}

export function TicketManagement() {
    const { t } = useTranslation("ticket");
    const { t: tc } = useTranslation("common");

    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const [page, setPage] = useState(1);
    const { sort, toggle: toggleSort, sortParam } = useServerSort(() => setPage(1));
    const [status, setStatus] = useState(String(STATUS_OPENING));
    const [level, setLevel] = useState("all");
    const [search, setSearch] = useState("");
    const [filtersVersion, setFiltersVersion] = useState(0);

    const [detailId, setDetailId] = useState(null);
    const [detailOpen, setDetailOpen] = useState(false);
    const [closeTarget, setCloseTarget] = useState(null);

    const buildFilter = () => {
        const filter = [{ id: "status", value: status }];
        if (level !== "all") filter.push({ id: "level", value: level });
        return filter;
    };

    const load = () => {
        setLoading(true);
        return fetchTickets({ pageSize: PAGE_SIZE, current: page, filter: buildFilter(), sort: sortParam })
            .then((res) => {
                setItems(res?.data || []);
                setTotal(res?.total ?? 0);
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [page, filtersVersion, sort.key, sort.dir]);

    const refetch = () => {
        if (page !== 1) setPage(1);
        else setFiltersVersion((v) => v + 1);
    };

    const filtered = useMemo(() => {
        const q = search.trim().toLowerCase();
        if (!q) return items;
        return items.filter((it) => (it.subject || "").toLowerCase().includes(q) || (it.user?.email || "").toLowerCase().includes(q));
    }, [items, search]);

    const openDetail = (id) => {
        setDetailId(id);
        setDetailOpen(true);
    };

    const handleClose = async () => {
        if (!closeTarget) return;
        await closeTicket(closeTarget.id);
        setCloseTarget(null);
        await load();
    };

    return (
        <div className="p-4 md:p-6 space-y-4">
            <div>
                <h2 className="text-2xl font-bold tracking-tight">{t("title")}</h2>
                <p className="mt-2 text-muted-foreground">{t("description")}</p>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex flex-wrap items-center gap-2">
                    <Select value={status} onValueChange={(v) => { setStatus(v); refetch(); }}>
                        <SelectTrigger className="h-9 w-[150px]">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value={String(STATUS_OPENING)}>{t("status.processing")}</SelectItem>
                            <SelectItem value={String(STATUS_CLOSED)}>{t("status.closed")}</SelectItem>
                        </SelectContent>
                    </Select>
                    <Select value={level} onValueChange={(v) => { setLevel(v); refetch(); }}>
                        <SelectTrigger className="h-9 w-[140px]">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">{t("columns.level")}</SelectItem>
                            <SelectItem value={String(LEVEL_LOW)}>{t("level.low")}</SelectItem>
                            <SelectItem value={String(LEVEL_MIDDLE)}>{t("level.medium")}</SelectItem>
                            <SelectItem value={String(LEVEL_HIGH)}>{t("level.high")}</SelectItem>
                        </SelectContent>
                    </Select>
                    <Input
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        placeholder={t("list.search_placeholder")}
                        className="h-9 w-[200px]"
                    />
                </div>
                <Button variant="outline" size="sm" onClick={load}>↻</Button>
            </div>

            <Table>
                <TableHeader>
                    <TableRow>
                        <SortableHead sortKey="id" sort={sort} onSort={toggleSort}>{t("columns.id")}</SortableHead>
                        <SortableHead sortKey="subject" sort={sort} onSort={toggleSort}>{t("columns.subject")}</SortableHead>
                        <SortableHead sortKey="level" sort={sort} onSort={toggleSort}>{t("columns.level")}</SortableHead>
                        <SortableHead sortKey="status" sort={sort} onSort={toggleSort}>{t("columns.status")}</SortableHead>
                        <SortableHead sortKey="updated_at" sort={sort} onSort={toggleSort}>{t("columns.updated_at")}</SortableHead>
                        <SortableHead sortKey="created_at" sort={sort} onSort={toggleSort}>{t("columns.created_at")}</SortableHead>
                        <TableHead className="text-right">{t("columns.actions")}</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {loading ? (
                        <TableEmpty colSpan={7}>...</TableEmpty>
                    ) : filtered.length === 0 ? (
                        <TableEmpty colSpan={7}>{search.trim() ? t("list.no_search_results") : t("list.no_tickets")}</TableEmpty>
                    ) : (
                        filtered.map((ticket) => {
                            const info = statusInfo(ticket, t);
                            return (
                                <TableRow key={ticket.id}>
                                    <TableCell><Badge variant="outline">{ticket.id}</Badge></TableCell>
                                    <TableCell className="max-w-[300px] truncate font-medium">{ticket.subject}</TableCell>
                                    <TableCell><Badge variant={levelBadgeVariant(ticket.level)}>{levelLabel(ticket.level, t)}</Badge></TableCell>
                                    <TableCell><Badge variant={info.variant}>{info.label}</Badge></TableCell>
                                    <TableCell className="text-sm text-muted-foreground">{formatDateTime(ticket.updated_at)}</TableCell>
                                    <TableCell className="text-sm text-muted-foreground">{formatDateTime(ticket.created_at)}</TableCell>
                                    <TableCell>
                                        <div className="flex items-center justify-end gap-1">
                                            <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => openDetail(ticket.id)}>
                                                {t("actions.view_details")}
                                            </Button>
                                            {ticket.status !== STATUS_CLOSED ? (
                                                <Button variant="ghost" size="sm" className="h-7 px-2 text-xs text-muted-foreground hover:text-destructive" onClick={() => setCloseTarget(ticket)}>
                                                    {t("actions.close_ticket")}
                                                </Button>
                                            ) : null}
                                        </div>
                                    </TableCell>
                                </TableRow>
                            );
                        })
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

            <TicketDetailDialog
                ticketId={detailId}
                statusFilter={status}
                open={detailOpen}
                onOpenChange={setDetailOpen}
                onChanged={load}
                t={t}
                tc={tc}
            />

            <ConfirmDialog
                open={Boolean(closeTarget)}
                onOpenChange={(open) => !open && setCloseTarget(null)}
                title={t("actions.close_confirm_title")}
                description={t("actions.close_confirm_description")}
                confirmLabel={t("actions.close_confirm_button")}
                cancelLabel={tc("cancel")}
                variant="destructive"
                onConfirm={handleClose}
            />
        </div>
    );
}
