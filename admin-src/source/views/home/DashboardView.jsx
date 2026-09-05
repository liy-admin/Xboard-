import React, { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { fetchDashboardStats, fetchOrderStats, fetchQueueStats, fetchTrafficRank } from "../../api/dashboard.js";
import { navigate } from "../../router/index.js";
import { clearAuthSession, useAuthState } from "../../store/auth.js";
import { getCurrencySymbol, useCurrencySymbol } from "../../store/config.js";
import { useTranslation } from "../../i18n/index.js";
import { DateRangePicker } from "../../components/DateRangePicker.jsx";
import { SIDEBAR_NAV_GROUPS, SidebarIcon } from "../../components/Sidebar.jsx";
import { classNames } from "../../utils/classNames.js";
import { persistTheme, resolveTheme } from "../../utils/theme.js";

const ICONS = {
    search: <><circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" /></>,
    moon: <path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z" />,
    sun: <><circle cx="12" cy="12" r="4" /><path d="M12 2v2" /><path d="M12 20v2" /><path d="m4.93 4.93 1.41 1.41" /><path d="m17.66 17.66 1.41 1.41" /><path d="M2 12h2" /><path d="M20 12h2" /><path d="m6.34 17.66-1.41 1.41" /><path d="m19.07 4.93-1.41 1.41" /></>,
    logout: <><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><polyline points="16 17 21 12 16 7" /><line x1="21" x2="9" y1="12" y2="12" /></>,
    cog: <><path d="M12 20h9" /><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z" /></>,
    downLine: <><path d="M12 17V3" /><path d="m6 11 6 6 6-6" /><path d="M19 21H5" /></>,
    bar: <><path d="M3 3v18h18" /><path d="M18 17V9" /><path d="M13 17V5" /><path d="M8 17v-3" /></>,
    trend: <><polyline points="22 7 13.5 15.5 8.5 10.5 2 17" /><polyline points="16 7 22 7 22 13" /></>,
    messages: <><path d="M14 9a2 2 0 0 1-2 2H6l-4 4V4c0-1.1.9-2 2-2h8a2 2 0 0 1 2 2z" /><path d="M18 9h2a2 2 0 0 1 2 2v11l-4-4h-6a2 2 0 0 1-2-2v-1" /></>,
    bell: <><path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" /><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" /></>,
    users: <><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></>,
    upload: <><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="17 8 12 3 7 8" /><line x1="12" x2="12" y1="3" y2="15" /></>,
    download: <><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="7 10 12 15 17 10" /><line x1="12" x2="12" y1="15" y2="3" /></>,
    network: <><rect x="16" y="16" width="6" height="6" rx="1" /><rect x="2" y="16" width="6" height="6" rx="1" /><rect x="9" y="2" width="6" height="6" rx="1" /><path d="M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3" /><path d="M12 12V8" /></>,
    activity: <path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2" />,
    timer: <><line x1="10" x2="14" y1="2" y2="2" /><line x1="12" x2="15" y1="14" y2="11" /><circle cx="12" cy="14" r="8" /></>,
    refresh: <><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" /><path d="M21 3v5h-5" /></>,
    alert: <><circle cx="12" cy="12" r="10" /><line x1="12" x2="12" y1="8" y2="12" /><line x1="12" x2="12.01" y1="16" y2="16" /></>,
    cpu: <><rect width="16" height="16" x="4" y="4" rx="2" /><rect width="6" height="6" x="9" y="9" rx="1" /><path d="M15 2v2" /><path d="M15 20v2" /><path d="M2 15h2" /><path d="M2 9h2" /><path d="M20 15h2" /><path d="M20 9h2" /><path d="M9 2v2" /><path d="M9 20v2" /></>,
    eye: <><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z" /><circle cx="12" cy="12" r="3" /></>,
    chevron: <path d="M3.13523 6.15803C3.3241 5.95657 3.64052 5.94637 3.84197 6.13523L7.5 9.56464L11.158 6.13523C11.3595 5.94637 11.6759 5.95657 11.8648 6.15803C12.0536 6.35949 12.0434 6.67591 11.842 6.86477L7.84197 10.6148C7.64964 10.7951 7.35036 10.7951 7.15803 10.6148L3.15803 6.86477C2.95657 6.67591 2.94637 6.35949 3.13523 6.15803Z" fill="currentColor" fillRule="evenodd" clipRule="evenodd" />,
};

function numberValue(value) {
    return Number(value || 0);
}

function formatCurrency(value) {
    return `${(numberValue(value) / 100).toFixed(0).replace(/\B(?=(\d{3})+(?!\d))/g, ".")}${getCurrencySymbol()}`;
}

function formatTraffic(value) {
    const bytes = numberValue(value);
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(2)} KB`;
    if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(2)} MB`;
    return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
}

function unwrapPayload(response) {
    return response?.data ?? response ?? {};
}

const ORDER_RANGE_OPTIONS = [
    { value: "7d", labelKey: "overview.last7Days", days: 7 },
    { value: "30d", labelKey: "overview.last30Days", days: 30 },
    { value: "90d", labelKey: "overview.last90Days", days: 90 },
    { value: "180d", labelKey: "overview.last180Days", days: 180 },
    { value: "365d", labelKey: "overview.lastYear", days: 365 },
    { value: "custom", labelKey: "overview.customRange" },
];

const TRAFFIC_RANGE_OPTIONS = [
    { value: "today", labelKey: "trafficRank.today" },
    { value: "last7days", labelKey: "trafficRank.last7days" },
    { value: "last30days", labelKey: "trafficRank.last30days" },
    { value: "custom", labelKey: "trafficRank.customRange" },
];

const PANEL_TIMEZONE_OFFSET_MINUTES = 480;

function toPanelDate(date) {
    return new Date(date.getTime() + (PANEL_TIMEZONE_OFFSET_MINUTES + date.getTimezoneOffset()) * 60 * 1000);
}

function fromPanelDate(date) {
    return new Date(date.getTime() - (PANEL_TIMEZONE_OFFSET_MINUTES + date.getTimezoneOffset()) * 60 * 1000);
}

function addDays(date, days) {
    const nextDate = new Date(date);
    nextDate.setDate(nextDate.getDate() + days);
    return nextDate;
}

function startOfLocalDay(date) {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function getDefaultCustomRange() {
    const today = startOfLocalDay(new Date());
    return {
        from: addDays(today, -7),
        to: today,
    };
}

function formatDateForApi(date) {
    const panelDate = toPanelDate(date);
    const year = panelDate.getFullYear();
    const month = String(panelDate.getMonth() + 1).padStart(2, "0");
    const day = String(panelDate.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
}

function getOrderRange(rangeValue, customRange) {
    if (rangeValue === "custom" && customRange?.from && customRange?.to) {
        return {
            start_date: formatDateForApi(customRange.from),
            end_date: formatDateForApi(customRange.to),
        };
    }

    const range = ORDER_RANGE_OPTIONS.find((item) => item.value === rangeValue) || ORDER_RANGE_OPTIONS[1];
    const panelNow = toPanelDate(new Date());

    return {
        start_date: formatDateForApi(fromPanelDate(addDays(panelNow, -range.days))),
        end_date: formatDateForApi(fromPanelDate(panelNow)),
    };
}

function getTrafficRange(rangeValue, customRange) {
    const panelNow = toPanelDate(new Date());

    if (rangeValue === "custom" && customRange?.from && customRange?.to) {
        const panelStart = new Date(customRange.from.getFullYear(), customRange.from.getMonth(), customRange.from.getDate());
        const panelEnd = addDays(new Date(customRange.to.getFullYear(), customRange.to.getMonth(), customRange.to.getDate()), 1);

        return {
            start_time: Math.round(fromPanelDate(panelStart).getTime() / 1000),
            end_time: Math.round(fromPanelDate(panelEnd).getTime() / 1000),
        };
    }

    if (rangeValue === "last7days" || rangeValue === "last30days") {
        const days = rangeValue === "last30days" ? 30 : 7;
        return {
            start_time: Math.round(fromPanelDate(addDays(panelNow, -days)).getTime() / 1000),
            end_time: Math.round(fromPanelDate(panelNow).getTime() / 1000),
        };
    }

    const panelDayStart = new Date(panelNow.getFullYear(), panelNow.getMonth(), panelNow.getDate());
    const panelDayEnd = addDays(panelDayStart, 1);

    return {
        start_time: Math.round(fromPanelDate(panelDayStart).getTime() / 1000),
        end_time: Math.round(fromPanelDate(panelDayEnd).getTime() / 1000),
    };
}

function normalizeTrafficRank(payload) {
    if (Array.isArray(payload)) {
        return payload;
    }

    if (payload && typeof payload === "object" && Array.isArray(payload.data)) {
        return payload.data;
    }

    return [];
}

function normalizeTrafficRankItem(item, index) {
    const value = numberValue(item?.value ?? item?.currentValue ?? item?.current_value ?? item?.traffic ?? item?.total ?? 0);
    const previousValue = numberValue(item?.previousValue ?? item?.previous_value ?? item?.previous ?? 0);
    const change = item?.change != null
        ? numberValue(item.change)
        : previousValue > 0
            ? Number((((value - previousValue) / previousValue) * 100).toFixed(2))
            : 0;

    return {
        id: item?.id ?? item?.user_id ?? item?.server_id ?? item?.node_id ?? `${item?.name || "rank"}-${index}`,
        name: item?.name ?? item?.email ?? item?.user_email ?? item?.node_name ?? item?.server_name ?? `#${index + 1}`,
        value,
        previousValue,
        change,
    };
}

function normalizeTrafficRankList(payload) {
    return normalizeTrafficRank(payload).map(normalizeTrafficRankItem);
}

function formatTrafficRankValue(value) {
    return `${(numberValue(value) / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function buildStats(stats, t) {
    return [
        [t("stats.todayIncome"), formatCurrency(stats.todayIncome), t("stats.vsYesterday"), "downLine", "text-emerald-500", false, stats.dayIncomeGrowth],
        [t("stats.monthlyIncome"), formatCurrency(stats.currentMonthIncome), t("stats.vsLastMonth"), "bar", "text-blue-500", false, stats.monthIncomeGrowth],
        [t("stats.pendingTickets"), numberValue(stats.ticketPendingTotal), numberValue(stats.ticketPendingTotal) > 0 ? t("stats.hasPendingTickets") : t("stats.noPendingTickets"), "messages", "text-muted-foreground", true, null, () => navigate("/user/ticket")],
        [t("stats.pendingCommission"), numberValue(stats.commissionPendingTotal), numberValue(stats.commissionPendingTotal) > 0 ? t("stats.hasPendingCommission") : t("stats.noPendingCommission"), "bell", "text-muted-foreground", true],
        [t("stats.monthlyNewUsers"), numberValue(stats.currentMonthNewUsers), t("stats.vsLastMonth"), "users", "text-blue-500", false, stats.userGrowth],
        [t("stats.totalUsers"), numberValue(stats.totalUsers), t("stats.activeUsers", { count: numberValue(stats.activeUsers) }), "users", "text-muted-foreground", true],
        [t("stats.monthlyUpload"), formatTraffic(stats.monthTraffic?.upload), t("stats.todayTraffic", { value: formatTraffic(stats.todayTraffic?.upload) }), "upload", "text-emerald-500", true],
        [t("stats.monthlyDownload"), formatTraffic(stats.monthTraffic?.download), t("stats.todayTraffic", { value: formatTraffic(stats.todayTraffic?.download) }), "download", "text-blue-500", true],
    ];
}

function Icon({ name, className = "h-4 w-4", size = 24, strokeWidth = 2, viewBox = "0 0 24 24" }) {
    return <svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox={viewBox} fill="none" stroke="currentColor" strokeWidth={strokeWidth} strokeLinecap="round" strokeLinejoin="round" className={className}>{ICONS[name]}</svg>;
}

function LanguageFlag({ code }) {
    if (code === "en-US") {
        return <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 60 30" className="h-4 w-5 rounded-sm shadow-sm"><clipPath id="flag-en"><path d="M0 0v30h60V0z" /></clipPath><g clipPath="url(#flag-en)"><path fill="#012169" d="M0 0h60v30H0z" /><path stroke="#fff" strokeWidth="6" d="m0 0 60 30m0-30L0 30" /><path stroke="#C8102E" strokeWidth="4" d="m0 0 60 30m0-30L0 30" /><path stroke="#fff" strokeWidth="10" d="M30 0v30M0 15h60" /><path stroke="#C8102E" strokeWidth="6" d="M30 0v30M0 15h60" /></g></svg>;
    }

    if (code === "zh-CN") {
        return <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 20" className="h-4 w-5 rounded-sm shadow-sm"><path fill="#EE1C25" d="M0 0h30v20H0z" /><path fill="#FFFF00" d="m5 2 1.18 3.64H10L6.91 7.9l1.18 3.6L5 9.28 1.91 11.5l1.18-3.6L0 5.64h3.82z" /><path fill="#FFFF00" d="m12 2 .53 1.18 1.28-.13-.95.86.4 1.22L12 4.5l-1.06.63.4-1.22-.95-.86 1.28.13zM15 5l.53 1.18 1.28-.13-.95.86.4 1.22L15 7.5l-1.06.63.4-1.22-.95-.86 1.28.13zM15 10l.53 1.18 1.28-.13-.95.86.4 1.22L15 12.5l-1.06.63.4-1.22-.95-.86 1.28.13zM12 13l.53 1.18 1.28-.13-.95.86.4 1.22L12 15.5l-1.06.63.4-1.22-.95-.86 1.28.13z" /></svg>;
    }

    return <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 85.333 512 341.333" className="h-4 w-5 rounded-sm shadow-sm"><path fill="#DA251D" d="M0 85.333h512v341.334H0z" /><path fill="#FFD100" d="M256 125l32.2 99h104l-84 61 32 99-84.2-61-84 61 32.2-99-84-61h104z" /></svg>;
}

function RangeSelect({ value, onChange, options, t, ariaLabel }) {
    const [open, setOpen] = useState(false);
    const rootRef = useRef(null);
    const selected = options.find((item) => item.value === value) || options[0];

    useEffect(() => {
        if (!open) {
            return undefined;
        }

        const handlePointerDown = (event) => {
            if (!rootRef.current?.contains(event.target)) {
                setOpen(false);
            }
        };
        const handleKeyDown = (event) => {
            if (event.key === "Escape") {
                setOpen(false);
            }
        };

        document.addEventListener("pointerdown", handlePointerDown);
        document.addEventListener("keydown", handleKeyDown);
        return () => {
            document.removeEventListener("pointerdown", handlePointerDown);
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [open]);

    return (
        <div ref={rootRef} className="relative">
            <button type="button" className="inline-flex h-9 w-[120px] items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm hover:bg-accent hover:text-accent-foreground focus:outline-none focus:ring-1 focus:ring-ring" aria-label={ariaLabel} aria-haspopup="listbox" aria-expanded={open} onClick={() => setOpen((value) => !value)}>
                <span className="truncate">{t(selected.labelKey)}</span>
                <Icon name="chevron" className="ml-2 h-4 w-4 flex-shrink-0 opacity-70" viewBox="0 0 15 15" />
            </button>
            {open && (
                <div className="absolute left-0 top-10 z-50 min-w-[130px] rounded-md border bg-popover p-1 text-popover-foreground shadow-md" role="listbox">
                    {options.map((item) => (
                        <button type="button" className={classNames("flex w-full items-center justify-between rounded-sm px-3 py-2 text-left text-sm hover:bg-accent hover:text-accent-foreground", item.value === value && "bg-secondary")} role="option" aria-selected={item.value === value} onClick={() => { onChange(item.value); setOpen(false); }} key={item.value}>
                            <span>{t(item.labelKey)}</span>
                            {item.value === value && <span className="ml-4">✓</span>}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}

function SegmentedControl({ value, onChange, options }) {
    return <div className="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground">{options.map((item) => <button type="button" key={item.value} onClick={() => onChange(item.value)} className={`inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium ${value === item.value ? "bg-background text-foreground shadow" : ""}`}>{item.label}</button>)}</div>;
}

function MetricCard({ item }) {
    const [title, value, caption, icon, color, plainCaption, trendValue = 0, onClick] = item;
    const positive = numberValue(trendValue) > 0;
    return <div className={`rounded-xl border bg-card text-card-foreground shadow transition-colors ${plainCaption || onClick ? "cursor-pointer hover:bg-muted/50" : ""}`} onClick={onClick}><div className="p-6 flex flex-row items-center justify-between space-y-0 pb-2"><h3 className="tracking-tight text-sm font-medium">{title}</h3><Icon name={icon} className={`h-4 w-4 ${color}`} /></div><div className="p-6 pt-0"><div className="text-2xl font-bold">{value}</div>{plainCaption ? <p className="text-xs text-muted-foreground">{caption}</p> : <div className="flex items-center pt-1"><Icon name="trend" className={`h-4 w-4 ${positive ? "text-emerald-500" : "text-red-500"}`} /><span className={`ml-1 text-xs ${positive ? "text-emerald-500" : "text-red-500"}`}>{positive ? "+" : "-"}{Math.abs(numberValue(trendValue))}%</span><span className="ml-1 text-xs text-muted-foreground">{caption}</span></div>}</div></div>;
}

function formatISODate(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
}

function RevenueChart({ data, mode, t }) {
    const rows = Array.isArray(data) ? data : [];

    if (!rows.length) {
        return <div className="flex h-[400px] items-center justify-center text-sm text-muted-foreground">{t("common:table.noData", t("common:loading"))}</div>;
    }

    const width = 640;
    const height = 400;
    const margin = { top: 20, right: 20, bottom: 34, left: 60 };
    const chartWidth = width - margin.left - margin.right;
    const chartHeight = height - margin.top - margin.bottom;
    const keys = mode === "amount" ? ["paid_total", "commission_total"] : ["paid_count", "commission_count"];
    const maxValue = Math.max(1, ...rows.flatMap((item) => keys.map((key) => numberValue(item?.[key]))));
    const x = (index) => margin.left + (rows.length === 1 ? chartWidth / 2 : (index / (rows.length - 1)) * chartWidth);
    const y = (value) => margin.top + chartHeight - (numberValue(value) / maxValue) * chartHeight;
    const formatTick = (value) => mode === "amount" ? formatCurrency(value) : String(value);
    const dateLabel = (value) => {
        if (!value) return "";
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return String(value).slice(5);
        }
        return `${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
    };

    return <div className="h-[400px] w-full"><svg className="h-full w-full" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={t("overview.title")}><defs><linearGradient id="incomeGradient" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity="0.2" /><stop offset="100%" stopColor="transparent" stopOpacity="0.1" /></linearGradient><linearGradient id="commissionGradient" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="hsl(var(--secondary))" stopOpacity="0.25" /><stop offset="100%" stopColor="transparent" stopOpacity="0.1" /></linearGradient></defs><line strokeDasharray="3 3" stroke="hsl(var(--border))" opacity="0.4" x1={margin.left} y1={margin.top} x2={width - margin.right} y2={margin.top} /><line strokeDasharray="3 3" stroke="hsl(var(--border))" opacity="0.4" x1={margin.left} y1={margin.top + chartHeight / 2} x2={width - margin.right} y2={margin.top + chartHeight / 2} /><line strokeDasharray="3 3" stroke="hsl(var(--border))" opacity="0.4" x1={margin.left} y1={height - margin.bottom} x2={width - margin.right} y2={height - margin.bottom} /><text x="8" y={margin.top + 4} className="fill-muted-foreground text-[11px]">{formatTick(maxValue)}</text><text x="8" y={height - margin.bottom} className="fill-muted-foreground text-[11px]">{formatTick(0)}</text>{rows.map((item, index) => index % Math.ceil(rows.length / 8) === 0 ? <text key={`${item.date}-${index}`} x={x(index)} y={height - 10} textAnchor="middle" className="fill-muted-foreground text-[11px]">{dateLabel(item.date)}</text> : null)}{mode === "amount" ? <><polyline points={rows.map((item, index) => `${x(index)},${y(item.paid_total)}`).join(" ")} fill="none" stroke="hsl(var(--primary))" strokeWidth="2" /><polyline points={rows.map((item, index) => `${x(index)},${y(item.commission_total)}`).join(" ")} fill="none" stroke="hsl(var(--secondary))" strokeWidth="2" />{rows.map((item, index) => <g key={`amount-${item.date || index}`}><circle cx={x(index)} cy={y(item.paid_total)} r="3" fill="hsl(var(--primary))" /><circle cx={x(index)} cy={y(item.commission_total)} r="3" fill="hsl(var(--secondary))" /></g>)}</> : rows.map((item, index) => {
        const groupX = x(index);
        const barWidth = Math.max(4, Math.min(18, chartWidth / Math.max(rows.length, 1) / 3));
        const paidHeight = height - margin.bottom - y(item.paid_count);
        const commissionHeight = height - margin.bottom - y(item.commission_count);
        return <g key={`count-${item.date || index}`}><rect x={groupX - barWidth - 1} y={y(item.paid_count)} width={barWidth} height={paidHeight} rx="4" fill="hsl(var(--primary))" /><rect x={groupX + 1} y={y(item.commission_count)} width={barWidth} height={commissionHeight} rx="4" fill="hsl(var(--secondary))" /></g>;
    })}</svg></div>;
}

function RevenueOverview({ t, stats, range, onRangeChange, customDateRange, onCustomDateRangeChange, mode, onModeChange }) {
    const data = stats || {};
    const summary = data.summary || {};
    const selectedRange = getOrderRange(range, customDateRange);
    const startDate = summary.start_date || selectedRange.start_date;
    const endDate = summary.end_date || selectedRange.end_date;

    return <div className="rounded-xl border bg-card text-card-foreground shadow"><div className="flex flex-col space-y-1.5 p-6"><div className="flex flex-wrap items-center justify-between gap-3"><div><h3 className="font-semibold leading-none tracking-tight">{t("overview.title")}</h3><p className="text-sm text-muted-foreground">{startDate} {t("overview.to")} {endDate}</p></div><div className="flex flex-wrap items-center gap-2"><div className="flex min-w-0 flex-wrap items-center gap-1"><RangeSelect value={range} onChange={onRangeChange} options={ORDER_RANGE_OPTIONS} t={t} ariaLabel={t("overview.selectTimeRange")} />{range === "custom" && <DateRangePicker value={customDateRange} onChange={onCustomDateRangeChange} placeholder={t("overview.selectDate")} />}</div><SegmentedControl value={mode} onChange={onModeChange} options={[{ value: "amount", label: t("overview.amount") }, { value: "count", label: t("overview.count") }]} /></div></div></div><div className="p-6 pt-0"><div className="grid grid-cols-2 gap-4"><div className="space-y-1"><div className="text-sm text-muted-foreground">{t("overview.totalIncome")}</div><div className="text-2xl font-bold">{formatCurrency(summary.paid_total)}</div><div className="text-xs text-muted-foreground">{t("overview.totalTransactions", { count: numberValue(summary.paid_count) })}</div><div className="text-xs text-muted-foreground">{t("overview.avgOrderAmount")}{formatCurrency(summary.avg_paid_amount)}</div></div><div className="space-y-1"><div className="text-sm text-muted-foreground">{t("overview.totalCommission")}</div><div className="text-2xl font-bold">{formatCurrency(summary.commission_total)}</div><div className="text-xs text-muted-foreground">{t("overview.totalTransactions", { count: numberValue(summary.commission_count) })}</div><div className="text-xs text-muted-foreground">{t("overview.commissionRate")} {numberValue(summary.commission_rate).toFixed(2)}%</div></div></div><RevenueChart data={data.list || []} mode={mode} t={t} /></div></div>;
}

function TrafficRankCard({ title, icon, t, items, range, onRangeChange, customDateRange, onCustomDateRangeChange }) {
    const rows = Array.isArray(items) ? items : [];
    const maxValue = rows[0]?.value || 1;

    return <div className="rounded-xl border bg-card text-card-foreground shadow"><div className="flex flex-col space-y-1.5 p-6 flex-none pb-2"><div className="flex flex-wrap items-center justify-between gap-2"><h3 className="tracking-tight flex items-center text-base font-medium"><Icon name={icon} className="mr-2 h-4 w-4" />{title}</h3><div className="flex min-w-0 flex-wrap items-center gap-1"><RangeSelect value={range} onChange={onRangeChange} options={TRAFFIC_RANGE_OPTIONS} t={t} ariaLabel={t("trafficRank.selectTimeRange")} />{range === "custom" && <DateRangePicker value={customDateRange} onChange={onCustomDateRangeChange} placeholder={t("trafficRank.selectDateRange")} />}<Icon name="activity" className="h-4 w-4 flex-shrink-0 text-muted-foreground" /></div></div></div><div className="p-6 pt-0 flex-1">{items === null ? <div className="flex h-[400px] items-center justify-center"><div className="animate-pulse">{t("common:loading")}</div></div> : rows.length ? <div className="h-[400px] overflow-auto pr-4"><div className="space-y-3">{rows.map((item) => <div key={item.id} className="flex items-center justify-between space-x-2 rounded-lg bg-muted/50 p-2 transition-colors hover:bg-muted/70"><div className="min-w-0 flex-1"><div className="flex items-center justify-between"><span className="truncate text-sm font-medium">{item.name}</span><span className={`ml-2 flex items-center text-xs font-medium ${item.change >= 0 ? "text-green-600" : "text-red-600"}`}>{item.change >= 0 ? "+" : "-"}{Math.abs(item.change)}%</span></div><div className="mt-1 flex items-center gap-2"><div className="h-2 flex-1 overflow-hidden rounded-full bg-muted"><div className="h-full bg-primary transition-all" style={{ width: `${Math.min(100, (item.value / maxValue) * 100)}%` }} /></div><span className="text-xs text-muted-foreground">{formatTrafficRankValue(item.value)}</span></div><div className="mt-1 grid grid-cols-2 gap-2 text-[11px] text-muted-foreground"><span>{t("trafficRank.currentTraffic")}: {formatTrafficRankValue(item.value)}</span><span>{t("trafficRank.previousTraffic")}: {formatTrafficRankValue(item.previousValue)}</span></div></div></div>)}</div></div> : <div className="flex h-[400px] items-center justify-center text-sm text-muted-foreground">{t("common:table.noData", t("common:loading"))}</div>}</div></div>;
}

function QueueStatus({ stats, onRefresh, t }) {
    const statusOk = Boolean(stats?.status);
    const wait = numberValue(stats?.wait?.default ?? stats?.wait ?? stats?.waitSeconds ?? stats?.wait_seconds ?? 0);
    const recent = numberValue(stats?.recentJobs ?? stats?.recent_jobs ?? 0);
    const recentPeriod = numberValue(stats?.periods?.recentJobs ?? 1);
    const perMinute = numberValue(stats?.jobsPerMinute ?? stats?.jobs_per_minute ?? 0);
    const maxThroughput = numberValue(stats?.queueWithMaxThroughput?.throughput ?? stats?.maxThroughput ?? stats?.max_throughput ?? 1);
    const failed = numberValue(stats?.failedJobs ?? stats?.failed_jobs ?? stats?.failed ?? 0);
    const failedPeriod = numberValue(stats?.periods?.failedJobs ?? 0);
    const longest = numberValue(stats?.queueWithMaxRuntime?.runtime ?? stats?.longestWait ?? stats?.longest_wait ?? 0);
    const longestName = stats?.queueWithMaxRuntime?.name || "N/A";
    const running = numberValue(stats?.processes ?? stats?.runningProcesses ?? stats?.running_processes ?? 0);
    const paused = numberValue(stats?.pausedMasters ?? stats?.paused_masters ?? 0);
    const totalProcesses = running + paused;

    return <div className="space-y-4"><div className="grid gap-4 md:grid-cols-2"><div className="rounded-xl border bg-card text-card-foreground shadow"><div className="p-6 flex flex-row items-center justify-between space-y-0 pb-2"><div className="space-y-1"><h3 className="font-semibold leading-none tracking-tight flex items-center gap-2"><Icon name="timer" className="h-5 w-5" />{t("queue.title")}</h3><p className="text-sm text-muted-foreground">{t("queue.status.description")}</p></div><button className="inline-flex items-center justify-center gap-2 rounded-md border border-input bg-background shadow-sm hover:bg-accent hover:text-accent-foreground h-9 w-9" onClick={onRefresh} aria-label={t("common.refresh")}><Icon name="refresh" /></button></div><div className="p-6 pt-0"><div className="space-y-4"><div className="space-y-2 rounded-lg bg-muted/50 p-3"><div className="flex items-center justify-between"><div className="flex items-center gap-2"><Icon name="alert" className={`h-5 w-5 ${statusOk ? "text-emerald-500" : "text-red-500"}`} /><span className="font-medium">{t("queue.status.running")}</span></div><div className={`inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold border-transparent shadow ${statusOk ? "bg-secondary text-secondary-foreground" : "bg-destructive text-destructive-foreground"}`}>{statusOk ? t("queue.status.normal") : t("queue.status.abnormal")}</div></div><div className="text-sm text-muted-foreground">{t("queue.status.waitTime", { seconds: wait })}</div></div><div className="grid grid-cols-2 gap-4"><QueueMini title={t("queue.details.recentJobs")} value={recent} progress={(recent / (recentPeriod || 1)) * 100} /><QueueMini title={t("queue.details.jobsPerMinute")} value={perMinute} progress={(perMinute / (maxThroughput || 1)) * 100} /></div></div></div></div><div className="rounded-xl border bg-card text-card-foreground shadow"><div className="flex flex-col space-y-1.5 p-6"><h3 className="font-semibold leading-none tracking-tight flex items-center gap-2"><Icon name="cpu" className="h-5 w-5" />{t("queue.jobDetails")}</h3><p className="text-sm text-muted-foreground">{t("queue.details.description")}</p></div><div className="p-6 pt-0"><div className="space-y-4"><div className="grid grid-cols-2 gap-4"><div className="space-y-2 rounded-lg bg-muted/50 p-3"><p className="text-sm text-muted-foreground">{t("queue.details.failedJobs7Days")}</p><div className="flex items-center gap-2"><span className="cursor-pointer text-2xl font-bold text-destructive hover:underline">{failed}</span><Icon name="eye" className="h-4 w-4 cursor-pointer text-muted-foreground hover:text-destructive" /></div><div className="text-xs text-muted-foreground">{t("queue.details.retentionPeriod", { hours: failedPeriod })}</div></div><div className="space-y-2 rounded-lg bg-muted/50 p-3"><p className="text-sm text-muted-foreground">{t("queue.details.longestRunningQueue")}</p><p className="text-2xl font-bold">{longest}s</p><div className="truncate text-xs text-muted-foreground">{longestName}</div></div></div><div className="rounded-lg bg-muted/50 p-3"><div className="flex items-center justify-between"><span className="text-sm text-muted-foreground">{t("queue.details.activeProcesses")}</span><span className="font-medium">{running} / {totalProcesses}</span></div><Progress value={totalProcesses ? (running / totalProcesses) * 100 : 0} /></div></div></div></div></div></div>;
}

function QueueMini({ title, value, progress = 0 }) {
    return <div className="space-y-2 rounded-lg bg-muted/50 p-3"><p className="text-sm text-muted-foreground">{title}</p><p className="text-2xl font-bold">{value}</p><Progress value={progress} /></div>;
}

function Progress({ value = 0 }) {
    const nextValue = Math.max(0, Math.min(100, numberValue(value)));
    return <div role="progressbar" aria-valuenow={nextValue} aria-valuemin="0" aria-valuemax="100" className="relative w-full overflow-hidden rounded-full bg-primary/20 h-1"><div className="h-full w-full flex-1 bg-primary transition-all" style={{ transform: `translateX(-${100 - nextValue}%)` }} /></div>;
}

const menuItems = SIDEBAR_NAV_GROUPS.flatMap((group) => group.items.map((item) => ({
    ...item,
    groupKey: group.titleKey,
})));

export function HeaderControls({ auth }) {
    const { language, languages, changeLanguage, t: tCommon } = useTranslation("common");
    const { t: tNav } = useTranslation("nav");
    const { t: tSearch } = useTranslation("search");
    const [searchOpen, setSearchOpen] = useState(false);
    const [languageOpen, setLanguageOpen] = useState(false);
    const [avatarOpen, setAvatarOpen] = useState(false);
    const [query, setQuery] = useState("");
    const [dark, setDark] = useState(() => resolveTheme() === "dark");
    const user = auth.userInfo || {};
    const localizedMenuItems = useMemo(() => menuItems.map((item) => ({
        ...item,
        title: tNav(item.titleKey),
        group: item.groupKey ? tNav(item.groupKey) : undefined,
    })), [tNav]);
    const filteredItems = localizedMenuItems.filter((item) => item.title.toLowerCase().includes(query.trim().toLowerCase()));

    useEffect(() => {
        persistTheme(dark ? "dark" : "light");
    }, [dark]);

    useEffect(() => {
        document.body.classList.toggle("admin-command-open", searchOpen);
        return () => {
            document.body.classList.remove("admin-command-open");
        };
    }, [searchOpen]);

    const go = (path) => {
        navigate(path);
        setSearchOpen(false);
        setQuery("");
    };

    return <div className="ml-auto flex items-center space-x-4">
        <button className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring border border-input bg-background shadow-sm hover:bg-accent hover:text-accent-foreground relative h-9 w-9 p-0 xl:h-10 xl:w-60 xl:justify-start xl:px-3 xl:py-2" onClick={() => setSearchOpen(true)}>
            <Icon name="search" className="h-4 w-4 xl:mr-2" />
            <span className="hidden xl:inline-flex">{tSearch("placeholder")}</span>
            <span className="sr-only">{tSearch("shortcut.label", tSearch("placeholder"))}</span>
            <kbd className="pointer-events-none absolute right-1.5 top-2 hidden h-6 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium opacity-100 xl:flex">⌘K</kbd>
        </button>
        {searchOpen && createPortal(<div className="admin-command-overlay" onClick={() => setSearchOpen(false)}>
            <div className="admin-command-dialog" onClick={(event) => event.stopPropagation()}>
                <div className="admin-command-input-row">
                    <Icon name="search" className="h-5 w-5 text-muted-foreground" />
                    <input autoFocus className="admin-command-input placeholder:text-muted-foreground" placeholder={tSearch("placeholder")} value={query} onChange={(event) => setQuery(event.target.value)} />
                    <button className="rounded-sm px-2 text-2xl leading-none text-muted-foreground hover:text-foreground" onClick={() => setSearchOpen(false)}>×</button>
                </div>
                <div className="admin-command-list">
                    <div className="admin-command-title">{tSearch("title")}</div>
                    {filteredItems.map((item) => <button key={item.path} className="admin-command-item" onClick={() => go(item.path)}>
                        <SidebarIcon name={item.icon} className="h-4 w-4" />
                        <span className="admin-command-item-title">{item.title}</span>
                        {item.group && <span className="admin-command-item-group">{item.group}</span>}
                    </button>)}
                    {filteredItems.length === 0 ? <div className="px-3 py-6 text-center text-sm text-muted-foreground">{tSearch("noResults")}</div> : null}
                </div>
            </div>
        </div>, document.body)}
        <button className="inline-flex items-center justify-center whitespace-nowrap text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground h-9 w-9 rounded-full" onClick={() => setDark((value) => !value)} aria-label={dark ? tCommon("theme.light") : tCommon("theme.dark")}><Icon name={dark ? "sun" : "moon"} size={20} /></button>
        <div className="relative"><button className="inline-flex items-center justify-center whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground rounded-md text-xs h-8 px-2 gap-1" type="button" onClick={() => setLanguageOpen((value) => !value)}><LanguageFlag code={language} /><span className="text-sm font-medium">{languages.find((item) => item.code === language)?.shortName || "VN"}</span></button>{languageOpen && <div className="absolute right-0 top-10 z-50 w-40 rounded-md border bg-popover p-1 text-popover-foreground shadow-md">{languages.map((item) => <button key={item.code} className="flex w-full items-center justify-between rounded-sm px-3 py-2 text-sm hover:bg-accent" onClick={() => { changeLanguage(item.code); setLanguageOpen(false); }}><span className="flex items-center gap-2"><LanguageFlag code={item.code} />{item.name}</span><span className="text-muted-foreground">{item.shortName}</span></button>)}</div>}</div>
        <div className="relative"><button className="inline-flex items-center justify-center whitespace-nowrap text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground px-4 py-2 relative h-8 w-8 rounded-full" type="button" onClick={() => setAvatarOpen((value) => !value)}><span className="relative flex shrink-0 overflow-hidden rounded-full h-8 w-8"><img className="aspect-square h-full w-full" alt={user.email || "admin"} src="https://cdn.v2ex.com/gravatar/e4fe42b0d0ef68938bd143a3be16bafa?s=64&d=identicon" /></span></button>{avatarOpen && <div className="absolute right-0 top-10 z-50 w-80 rounded-md border bg-popover text-popover-foreground shadow-md"><div className="px-4 py-3"><div className="text-base font-medium">{user.name || "admin"}</div><div className="text-sm text-muted-foreground">{user.email || "admin@cc.com"}</div></div><button className="flex w-full items-center justify-between border-t px-4 py-3 text-left text-sm hover:bg-accent" onClick={() => { setAvatarOpen(false); navigate("/config/system"); }}><span>{tCommon("settings")}</span><span className="text-muted-foreground">⌘S</span></button><button className="flex w-full items-center justify-between border-t px-4 py-3 text-left text-sm hover:bg-accent" onClick={() => { clearAuthSession(); navigate("/sign-in", { replace: true }); }}><span>{tCommon("logout")}</span><span className="text-muted-foreground">⇧⌘Q</span></button></div>}</div>
    </div>;
}

export function DashboardView() {
    const auth = useAuthState();
    useCurrencySymbol();
    const { t } = useTranslation("dashboard");
    const [dashboardStats, setDashboardStats] = useState({});
    const [orderRange, setOrderRange] = useState("30d");
    const [orderCustomRange, setOrderCustomRange] = useState(getDefaultCustomRange);
    const [orderMode, setOrderMode] = useState("amount");
    const [orderStats, setOrderStats] = useState(null);
    const [nodeRange, setNodeRange] = useState("today");
    const [nodeCustomRange, setNodeCustomRange] = useState(getDefaultCustomRange);
    const [userRange, setUserRange] = useState("today");
    const [userCustomRange, setUserCustomRange] = useState(getDefaultCustomRange);
    const [nodeTrafficRank, setNodeTrafficRank] = useState(null);
    const [userTrafficRank, setUserTrafficRank] = useState(null);
    const [queueStats, setQueueStats] = useState(null);

    useEffect(() => {
        let active = true;
        fetchDashboardStats()
            .then((response) => active && setDashboardStats(unwrapPayload(response)))
            .catch(() => active && setDashboardStats({}));
        return () => {
            active = false;
        };
    }, [auth.token]);

    useEffect(() => {
        let active = true;
        const params = getOrderRange(orderRange, orderCustomRange);
        const load = () => fetchOrderStats(params)
            .then((response) => active && setOrderStats(unwrapPayload(response)))
            .catch(() => active && setOrderStats({}));

        setOrderStats(null);
        load();
        const timer = window.setInterval(load, 30000);
        return () => {
            active = false;
            window.clearInterval(timer);
        };
    }, [auth.token, orderRange, orderCustomRange]);

    useEffect(() => {
        let active = true;
        const params = { type: "node", ...getTrafficRange(nodeRange, nodeCustomRange) };
        const load = () => fetchTrafficRank(params)
            .then((response) => active && setNodeTrafficRank(normalizeTrafficRankList(unwrapPayload(response))))
            .catch(() => active && setNodeTrafficRank([]));

        setNodeTrafficRank(null);
        load();
        const timer = window.setInterval(load, 30000);
        return () => {
            active = false;
            window.clearInterval(timer);
        };
    }, [auth.token, nodeRange, nodeCustomRange]);

    useEffect(() => {
        let active = true;
        const params = { type: "user", ...getTrafficRange(userRange, userCustomRange) };
        const load = () => fetchTrafficRank(params)
            .then((response) => active && setUserTrafficRank(normalizeTrafficRankList(unwrapPayload(response))))
            .catch(() => active && setUserTrafficRank([]));

        setUserTrafficRank(null);
        load();
        const timer = window.setInterval(load, 30000);
        return () => {
            active = false;
            window.clearInterval(timer);
        };
    }, [auth.token, userRange, userCustomRange]);

    useEffect(() => {
        let active = true;
        const load = () => fetchQueueStats()
            .then((response) => active && setQueueStats(unwrapPayload(response)))
            .catch(() => active && setQueueStats({}));

        setQueueStats(null);
        load();
        const timer = window.setInterval(load, 30000);
        return () => {
            active = false;
            window.clearInterval(timer);
        };
    }, [auth.token]);

    const metricCards = useMemo(() => buildStats(dashboardStats, t), [dashboardStats, t]);
    const refreshQueueStats = () => fetchQueueStats()
        .then((response) => setQueueStats(unwrapPayload(response)))
        .catch(() => setQueueStats({}));

    return <div className="relative flex h-full w-full flex-col"><div className="flex h-[var(--header-height)] flex-none items-center gap-4 bg-background p-4 md:px-8"><div className="flex items-center"><h1 className="text-2xl font-bold tracking-tight md:text-3xl">{t("title")}</h1></div><HeaderControls auth={auth} /></div><div className="flex-1 overflow-hidden px-4 py-6 md:px-8"><div className="space-y-6"><div className="grid gap-6"><div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">{metricCards.map((item) => <MetricCard item={item} key={item[0]} />)}</div><RevenueOverview t={t} stats={orderStats} range={orderRange} onRangeChange={setOrderRange} customDateRange={orderCustomRange} onCustomDateRangeChange={setOrderCustomRange} mode={orderMode} onModeChange={setOrderMode} /><div className="grid gap-4 md:grid-cols-2"><TrafficRankCard title={t("trafficRank.nodeTrafficRank")} icon="network" t={t} items={nodeTrafficRank} range={nodeRange} onRangeChange={setNodeRange} customDateRange={nodeCustomRange} onCustomDateRangeChange={setNodeCustomRange} /><TrafficRankCard title={t("trafficRank.userTrafficRank")} icon="users" t={t} items={userTrafficRank} range={userRange} onRangeChange={setUserRange} customDateRange={userCustomRange} onCustomDateRangeChange={setUserCustomRange} /></div><QueueStatus stats={queueStats || {}} onRefresh={refreshQueueStats} t={t} /></div></div></div></div>;
}
