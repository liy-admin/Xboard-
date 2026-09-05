import React, { useEffect, useMemo, useRef, useState } from "react";
import { classNames } from "../utils/classNames.js";

const WEEKDAYS = ["CN", "T2", "T3", "T4", "T5", "T6", "T7"];

function Icon({ name, className = "h-4 w-4" }) {
    if (name === "calendar") {
        return <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}><path d="M8 2v4" /><path d="M16 2v4" /><rect width="18" height="18" x="3" y="4" rx="2" /><path d="M3 10h18" /></svg>;
    }

    return <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}><path d="m9 18 6-6-6-6" /></svg>;
}

export function addDays(date, days) {
    const nextDate = new Date(date);
    nextDate.setDate(nextDate.getDate() + days);
    return nextDate;
}

export function startOfDay(date) {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

export function formatDisplayDate(date) {
    if (!date) {
        return "";
    }

    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const year = date.getFullYear();
    return `${day}-${month}-${year}`;
}

function sameDay(left, right) {
    return Boolean(left && right && left.getFullYear() === right.getFullYear() && left.getMonth() === right.getMonth() && left.getDate() === right.getDate());
}

function monthStart(date) {
    return new Date(date.getFullYear(), date.getMonth(), 1);
}

function addMonths(date, months) {
    return new Date(date.getFullYear(), date.getMonth() + months, 1);
}

function isBetween(day, from, to) {
    if (!from || !to) {
        return false;
    }

    const time = day.getTime();
    return time > from.getTime() && time < to.getTime();
}

function buildMonthDays(monthDate) {
    const first = monthStart(monthDate);
    const gridStart = addDays(first, -first.getDay());
    return Array.from({ length: 42 }, (_, index) => addDays(gridStart, index));
}

function normalizeRange(range) {
    return {
        from: range?.from ? startOfDay(range.from) : undefined,
        to: range?.to ? startOfDay(range.to) : undefined,
    };
}

function MonthView({ monthDate, draft, onPick, onMonthChange }) {
    const days = useMemo(() => buildMonthDays(monthDate), [monthDate]);
    const years = useMemo(() => {
        const current = new Date().getFullYear();
        return Array.from({ length: 11 }, (_, index) => current - 5 + index);
    }, []);

    return (
        <div className="admin-date-month">
            <div className="admin-date-month-controls">
                <select className="admin-date-select" value={monthDate.getMonth()} onChange={(event) => onMonthChange(new Date(monthDate.getFullYear(), Number(event.target.value), 1))} aria-label="Month">
                    {Array.from({ length: 12 }, (_, index) => <option value={index} key={index}>Tháng {index + 1}</option>)}
                </select>
                <select className="admin-date-select" value={monthDate.getFullYear()} onChange={(event) => onMonthChange(new Date(Number(event.target.value), monthDate.getMonth(), 1))} aria-label="Year">
                    {years.map((year) => <option value={year} key={year}>{year}</option>)}
                </select>
            </div>
            <div className="admin-date-weekdays">
                {WEEKDAYS.map((day) => <div key={day}>{day}</div>)}
            </div>
            <div className="admin-date-grid">
                {days.map((day) => {
                    const inMonth = day.getMonth() === monthDate.getMonth();
                    const selectedStart = sameDay(day, draft.from);
                    const selectedEnd = sameDay(day, draft.to);
                    const selected = selectedStart || selectedEnd;
                    const between = isBetween(day, draft.from, draft.to);

                    return (
                        <button type="button" className={classNames("admin-date-day", !inMonth && "admin-date-day-muted", between && "admin-date-day-range", selected && "admin-date-day-selected")} onClick={() => onPick(day)} key={day.toISOString()}>
                            {day.getDate()}
                        </button>
                    );
                })}
            </div>
        </div>
    );
}

export function DateRangePicker({ value, onChange, placeholder = "Chọn ngày", align = "end" }) {
    const rootRef = useRef(null);
    const [open, setOpen] = useState(false);
    const [draft, setDraft] = useState(() => normalizeRange(value));
    const [viewMonth, setViewMonth] = useState(() => monthStart(value?.from || new Date()));

    useEffect(() => {
        setDraft(normalizeRange(value));
        if (value?.from) {
            setViewMonth(monthStart(value.from));
        }
    }, [value]);

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

    const label = value?.from
        ? value.to
            ? `${formatDisplayDate(value.from)} - ${formatDisplayDate(value.to)}`
            : formatDisplayDate(value.from)
        : placeholder;

    const pickDay = (day) => {
        const nextDay = startOfDay(day);
        if (!draft.from || draft.to) {
            setDraft({ from: nextDay, to: undefined });
            return;
        }

        const nextRange = nextDay < draft.from
            ? { from: nextDay, to: draft.from }
            : { from: draft.from, to: nextDay };
        setDraft(nextRange);
        onChange?.(nextRange);
        setOpen(false);
    };

    return (
        <div ref={rootRef} className="admin-date-range">
            <button type="button" className={classNames("admin-date-trigger", !value?.from && "text-muted-foreground")} onClick={() => setOpen((nextOpen) => !nextOpen)}>
                <Icon name="calendar" className="mr-2 h-4 w-4 flex-shrink-0" />
                <span className="truncate">{label}</span>
            </button>
            {open && (
                <div className={classNames("admin-date-popover", align === "end" && "admin-date-popover-end")}>
                    <div className="admin-date-calendar">
                        <button type="button" className="admin-date-nav admin-date-nav-prev" onClick={() => setViewMonth((month) => addMonths(month, -1))} aria-label="Previous month"><Icon name="chevron" className="h-4 w-4 rotate-180" /></button>
                        <button type="button" className="admin-date-nav admin-date-nav-next" onClick={() => setViewMonth((month) => addMonths(month, 1))} aria-label="Next month"><Icon name="chevron" className="h-4 w-4" /></button>
                        <MonthView monthDate={viewMonth} draft={draft} onPick={pickDay} onMonthChange={setViewMonth} />
                        <MonthView monthDate={addMonths(viewMonth, 1)} draft={draft} onPick={pickDay} onMonthChange={(month) => setViewMonth(addMonths(month, -1))} />
                    </div>
                </div>
            )}
        </div>
    );
}
