// Package jobs 承载定时任务与 CLI 子命令的业务逻辑（对应 app/Console/Commands）。
package jobs

import (
	"log"
	"math"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xboard/internal/model"
	"xboard/internal/setting"
)

const (
	orderStatusCompleted = 3
	commissionPending    = 0 // 待确认
	commissionChecked    = 1 // 已确认，待发放
	commissionPaid       = 2 // 已发放
)

// CommissionShares 返回各级分销比例（百分比）。未开启分销时只有直接推荐人一级 100%。
func CommissionShares(distEnabled bool, l1, l2, l3 int) []int {
	if !distEnabled {
		return []int{100}
	}
	return []int{l1, l2, l3}
}

// ReferralExpiry 计算推荐时长奖励后的到期时间：max(当前到期, now) + seconds。
// 永久会员（plan_id 非空且 expired_at 为空）不做覆盖，调用方需先判断。
func ReferralExpiry(expiredAt *int64, nowTs, seconds int64) int64 {
	base := nowTs
	if expiredAt != nil && *expiredAt > base {
		base = *expiredAt
	}
	return base + seconds
}

// roundCents 对应 MySQL 写入 INT 列时的四舍五入（半数远离零）。
func roundCents(f float64) int64 { return int64(math.Round(f)) }

// CheckCommission 对应 check:commission：先自动确认超过 3 天的佣金，再按分销比例发放。
func CheckCommission(db *gorm.DB, set *setting.Service, nowTs int64) error {
	if set.Bool("commission_auto_check_enable", true) {
		if err := db.Model(&model.Order{}).
			Where("commission_status = ? AND invite_user_id IS NOT NULL AND status = ? AND updated_at <= ?",
				commissionPending, orderStatusCompleted, nowTs-3*86400).
			Update("commission_status", commissionChecked).Error; err != nil {
			return err
		}
	}
	var orders []model.Order
	if err := db.Where("commission_status = ? AND invite_user_id IS NOT NULL", commissionChecked).
		Find(&orders).Error; err != nil {
		return err
	}
	for i := range orders {
		o := &orders[i]
		err := db.Transaction(func(tx *gorm.DB) error {
			ok, err := payCommission(tx, set, *o.InviteUserID, o, nowTs)
			if err != nil || !ok {
				return err
			}
			return tx.Model(&model.Order{}).Where("id = ?", o.ID).Update("commission_status", commissionPaid).Error
		})
		if err != nil {
			// PHP 在异常时直接抛出中断整个命令；这里记录后继续处理其余订单，避免单笔坏数据卡住全部发放。
			log.Printf("check:commission order %d: %v", o.ID, err)
		}
	}
	return nil
}

// payCommission 对应 CheckCommission::payHandle，返回 false 表示需要回滚该订单。
func payCommission(tx *gorm.DB, set *setting.Service, inviteID int64, o *model.Order, nowTs int64) (bool, error) {
	levels := CommissionShares(set.Bool("commission_distribution_enable", false),
		set.Int("commission_distribution_l1", 0), set.Int("commission_distribution_l2", 0), set.Int("commission_distribution_l3", 0))
	withdrawClosed := set.Bool("withdraw_close_enable", false)

	for l := 0; l < 3; l++ {
		var inviter model.User
		if inviteID == 0 || tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&inviter, inviteID).Error != nil {
			continue
		}
		if l >= len(levels) {
			continue
		}
		amount := float64(o.CommissionBalance) * (float64(levels[l]) / 100)
		if amount != 0 {
			credit := roundCents(amount)
			col := "commission_balance"
			if withdrawClosed {
				col = "balance"
			}
			if err := tx.Model(&model.User{}).Where("id = ?", inviter.ID).
				UpdateColumn(col, gormExprAdd(col, credit)).Error; err != nil {
				return false, err
			}
			if err := tx.Create(&model.CommissionLog{
				InviteUserID: inviter.ID, UserID: o.UserID, TradeNo: o.TradeNo,
				OrderAmount: o.TotalAmount, GetAmount: credit, CreatedAt: nowTs, UpdatedAt: nowTs,
			}).Error; err != nil {
				return false, err
			}
			if err := tx.Model(&model.Order{}).Where("id = ?", o.ID).
				UpdateColumn("actual_commission_balance", gormExprAdd("actual_commission_balance", credit)).Error; err != nil {
				return false, err
			}
		}

		// 推荐时长奖励只发给直接推荐人（l == 0），不做多级分销。
		if l == 0 && o.ReferralDurationSeconds != nil && *o.ReferralDurationSeconds > 0 {
			permanent := inviter.PlanID != nil && inviter.ExpiredAt == nil
			if !permanent {
				exp := ReferralExpiry(inviter.ExpiredAt, nowTs, *o.ReferralDurationSeconds)
				if err := tx.Model(&model.User{}).Where("id = ?", inviter.ID).Update("expired_at", exp).Error; err != nil {
					return false, err
				}
				if err := tx.Create(&model.ReferralDurationLog{
					InviterID: inviter.ID, InviteeID: o.UserID, TradeNo: o.TradeNo,
					SecondsCredited: *o.ReferralDurationSeconds, CreatedAt: nowTs, UpdatedAt: nowTs,
				}).Error; err != nil {
					return false, err
				}
			}
		}
		if inviter.InviteUserID == nil {
			break
		}
		inviteID = *inviter.InviteUserID
	}
	return true, nil
}

// gormExprAdd 生成 `col = col + n` 的原子更新表达式。
func gormExprAdd(col string, n int64) any {
	return gorm.Expr(col+" + ?", n)
}
