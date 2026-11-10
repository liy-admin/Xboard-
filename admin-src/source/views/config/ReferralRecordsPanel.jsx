import React, { useCallback, useEffect, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getReferralSummary, getReferralRecords } from "../../api/config.js";
import { SortLabel, useClientSort } from "../../components/SortableHead.jsx";

function fmtTime(ts) {
    if (!ts) return "-";
    return new Date(ts * 1000).toLocaleString();
}

export function ReferralRecordsPanel() {
    const { t } = useTranslation("settings");
    const [tab, setTab] = useState("summary");
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [rows, setRows] = useState([]);
    const [updatedAt, setUpdatedAt] = useState(null);

    const load = useCallback(async (which) => {
        setLoading(true);
        setError(null);
        try {
            if (which === "summary") {
                const res = await getReferralSummary();
                setRows(res?.data || []);
            } else {
                const res = await getReferralRecords({ page_size: 50 });
                setRows((res?.data && res.data.data) || []);
            }
            setUpdatedAt(new Date());
        } catch (e) {
            setError(e?.message || "error");
            setRows([]);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        load(tab);
    }, [tab, load]);

    return (
        <div>
            <h3 className="text-lg font-medium">{t("invite.referral_records.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("invite.referral_records.description")}</p>
            <div className="mt-4 flex items-center justify-between border-b">
                <div className="flex gap-4">
                    <TabButton active={tab === "summary"} onClick={() => setTab("summary")}>
                        {t("invite.referral_records.tab_summary")}
                    </TabButton>
                    <TabButton active={tab === "records"} onClick={() => setTab("records")}>
                        {t("invite.referral_records.tab_records")}
                    </TabButton>
                </div>
                <button
                    type="button"
                    className="mb-2 rounded-md border px-3 py-1 text-xs text-muted-foreground hover:bg-accent"
                    onClick={() => load(tab)}
                    disabled={loading}
                >
                    {t("invite.referral_records.refresh")}
                </button>
            </div>

            <div className="mt-4">
                {loading && <div className="py-10 text-center text-sm text-muted-foreground">...</div>}
                {!loading && error && (
                    <div className="py-10 text-center text-sm text-destructive">
                        {t("invite.referral_records.load_error")}: {error}
                    </div>
                )}
                {!loading && !error && rows.length === 0 && (
                    <div className="py-10 text-center text-sm text-muted-foreground">
                        {t("invite.referral_records.empty")}
                    </div>
                )}
                {!loading && !error && rows.length > 0 && tab === "summary" && (
                    <SummaryTable rows={rows} t={t} />
                )}
                {!loading && !error && rows.length > 0 && tab === "records" && (
                    <RecordsTable rows={rows} t={t} />
                )}
            </div>

            {updatedAt && (
                <p className="mt-2 text-xs text-muted-foreground">
                    {t("invite.referral_records.updated_at", { time: updatedAt.toLocaleTimeString() })}
                </p>
            )}
        </div>
    );
}

function TabButton({ active, onClick, children }) {
    return (
        <button
            type="button"
            onClick={onClick}
            className={
                "border-b-2 pb-2 text-sm font-medium " +
                (active ? "border-primary text-foreground" : "border-transparent text-muted-foreground")
            }
        >
            {children}
        </button>
    );
}

function SummaryTable({ rows, t }) {
    const st = useClientSort(rows, {
        inviter: (r) => r.inviter_email || null,
        invite_count: (r) => Number(r.invite_count) || 0,
        total_hours: (r) => Number(r.total_hours) || 0,
        last_at: (r) => r.last_at || null,
    });
    return (
        <div className="overflow-x-auto rounded-lg border">
            <table className="w-full text-sm">
                <thead>
                    <tr className="border-b bg-muted/50 text-left text-xs uppercase text-muted-foreground">
                        <th className="px-4 py-2"><SortLabel sortKey="inviter" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_inviter")}</SortLabel></th>
                        <th className="px-4 py-2"><SortLabel sortKey="invite_count" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_invite_count")}</SortLabel></th>
                        <th className="px-4 py-2"><SortLabel sortKey="total_hours" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_total_hours")}</SortLabel></th>
                        <th className="px-4 py-2"><SortLabel sortKey="last_at" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_last_at")}</SortLabel></th>
                    </tr>
                </thead>
                <tbody>
                    {st.sorted.map((r) => (
                        <tr key={r.inviter_id} className="border-b last:border-0">
                            <td className="px-4 py-2">
                                <div className="font-medium">{r.inviter_email || "-"}</div>
                                <div className="text-xs text-muted-foreground">UID {r.inviter_id}</div>
                            </td>
                            <td className="px-4 py-2">
                                {r.invite_count} {t("invite.referral_records.unit_person")}
                            </td>
                            <td className="px-4 py-2 font-medium text-primary">
                                {r.total_hours} {t("invite.referral_records.unit_hours")}
                            </td>
                            <td className="px-4 py-2">{fmtTime(r.last_at)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function RecordsTable({ rows, t }) {
    const st = useClientSort(rows, {
        inviter: (r) => r.inviter_email || null,
        invitee: (r) => r.invitee_email || null,
        trade_no: (r) => r.trade_no || null,
        hours_credited: (r) => Number(r.hours_credited) || 0,
        created_at: (r) => r.created_at || null,
    });
    return (
        <div className="overflow-x-auto rounded-lg border">
            <table className="w-full text-sm">
                <thead>
                    <tr className="border-b bg-muted/50 text-left text-xs uppercase text-muted-foreground">
                        <th className="px-4 py-2"><SortLabel sortKey="inviter" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_inviter")}</SortLabel></th>
                        <th className="px-4 py-2"><SortLabel sortKey="invitee" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_invitee")}</SortLabel></th>
                        <th className="px-4 py-2"><SortLabel sortKey="trade_no" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_trade_no")}</SortLabel></th>
                        <th className="px-4 py-2"><SortLabel sortKey="hours_credited" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_hours_credited")}</SortLabel></th>
                        <th className="px-4 py-2"><SortLabel sortKey="created_at" sort={st.sort} onSort={st.toggle}>{t("invite.referral_records.col_time")}</SortLabel></th>
                    </tr>
                </thead>
                <tbody>
                    {st.sorted.map((r, i) => (
                        <tr key={i} className="border-b last:border-0">
                            <td className="px-4 py-2">
                                <div className="font-medium">{r.inviter_email || "-"}</div>
                                <div className="text-xs text-muted-foreground">UID {r.inviter_id}</div>
                            </td>
                            <td className="px-4 py-2">
                                <div className="font-medium">{r.invitee_email || "-"}</div>
                                <div className="text-xs text-muted-foreground">UID {r.invitee_id}</div>
                            </td>
                            <td className="px-4 py-2 text-xs text-muted-foreground">{r.trade_no}</td>
                            <td className="px-4 py-2 font-medium text-primary">
                                {r.hours_credited} {t("invite.referral_records.unit_hours")}
                            </td>
                            <td className="px-4 py-2">{fmtTime(r.created_at)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
