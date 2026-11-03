package jobs

import (
	"strconv"
	"time"
)

// StatisticsDaily 对应 xboard:statistics：统计昨天一整天的订单、佣金、注册与流量，写入 v2_stat（record_type = d）。
func (e *Env) StatisticsDaily() error {
	y, m, d := e.Now().Date()
	end := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	start := end.AddDate(0, 0, -1)
	s, en := start.Unix(), end.Unix()

	count := func(q string, args ...any) int64 {
		var n int64
		e.DB.Raw(q, args...).Scan(&n)
		return n
	}
	orderCount := count("SELECT COUNT(*) FROM v2_order WHERE created_at >= ? AND created_at < ?", s, en)
	orderTotal := count("SELECT COALESCE(SUM(total_amount), 0) FROM v2_order WHERE created_at >= ? AND created_at < ?", s, en)
	paidCount := count("SELECT COUNT(*) FROM v2_order WHERE paid_at >= ? AND paid_at < ? AND status NOT IN (0, 2)", s, en)
	paidTotal := count("SELECT COALESCE(SUM(total_amount), 0) FROM v2_order WHERE paid_at >= ? AND paid_at < ? AND status NOT IN (0, 2)", s, en)
	commCount := count("SELECT COUNT(*) FROM v2_commission_log WHERE created_at >= ? AND created_at < ?", s, en)
	commTotal := count("SELECT COALESCE(SUM(get_amount), 0) FROM v2_commission_log WHERE created_at >= ? AND created_at < ?", s, en)
	regCount := count("SELECT COUNT(*) FROM v2_user WHERE created_at >= ? AND created_at < ?", s, en)
	inviteCount := count("SELECT COUNT(*) FROM v2_user WHERE created_at >= ? AND created_at < ? AND invite_user_id IS NOT NULL", s, en)
	var transfer int64
	e.DB.Raw("SELECT COALESCE(SUM(u) + SUM(d), 0) FROM v2_stat_server WHERE created_at >= ? AND created_at < ?", s, en).Scan(&transfer)

	now := e.nowTs()
	row := map[string]any{
		"order_count": orderCount, "order_total": orderTotal,
		"paid_count": paidCount, "paid_total": paidTotal,
		"commission_count": commCount, "commission_total": commTotal,
		"register_count": regCount, "invite_count": inviteCount,
		"transfer_used_total": strconv.FormatInt(transfer, 10),
	}
	var exists int64
	e.DB.Raw("SELECT COUNT(*) FROM v2_stat WHERE record_at = ? AND record_type = 'd'", s).Scan(&exists)
	if exists > 0 {
		return e.DB.Exec("UPDATE v2_stat SET order_count=?, order_total=?, paid_count=?, paid_total=?, commission_count=?, "+
			"commission_total=?, register_count=?, invite_count=?, transfer_used_total=?, updated_at=? WHERE record_at=? AND record_type='d'",
			row["order_count"], row["order_total"], row["paid_count"], row["paid_total"], row["commission_count"],
			row["commission_total"], row["register_count"], row["invite_count"], row["transfer_used_total"], now, s).Error
	}
	return e.DB.Exec("INSERT INTO v2_stat (record_at, record_type, order_count, order_total, commission_count, commission_total, "+
		"paid_count, paid_total, register_count, invite_count, transfer_used_total, created_at, updated_at) "+
		"VALUES (?, 'd', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		s, row["order_count"], row["order_total"], row["commission_count"], row["commission_total"],
		row["paid_count"], row["paid_total"], row["register_count"], row["invite_count"],
		row["transfer_used_total"], now, now).Error
}
