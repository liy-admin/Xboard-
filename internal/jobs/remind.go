package jobs

import (
	"context"
	"log"
	"strconv"
	"time"
)

// SendRemindMail 对应 send:remindMail：到期前一天提醒续费，流量使用超过 80% 提醒（每用户每 24 小时最多一次）。
func (e *Env) SendRemindMail() error {
	if !e.Set.Bool("remind_mail_enable", false) {
		return nil
	}
	app := e.Set.Get("app_name", "Lets VPN")
	appURL := e.Set.Get("app_url", "")
	nowTs := e.nowTs()

	type row struct {
		ID             int64
		Email          string
		ExpiredAt      *int64
		TransferEnable int64
		U, D           int64
		RemindExpire   bool
		RemindTraffic  bool
	}
	var rows []row
	if err := e.DB.Raw("SELECT id, email, expired_at, transfer_enable, u, d, remind_expire, remind_traffic FROM v2_user " +
		"WHERE (remind_expire = 1 OR remind_traffic = 1) AND banned = 0 AND email IS NOT NULL").Scan(&rows).Error; err != nil {
		return err
	}
	values := func() map[string]any {
		return map[string]any{"name": app, "url": appURL}
	}
	for _, r := range rows {
		if r.RemindExpire && r.ExpiredAt != nil {
			exp := *r.ExpiredAt
			if exp-86400 < nowTs && exp > nowTs {
				subject := replaceAppName("在 :app_name 的服务即将到期", app)
				if err := e.SendTemplateMail(r.Email, subject, "remindExpire", values()); err != nil {
					log.Printf("remind expire user %d: %v", r.ID, err)
				}
			}
		}
		if r.RemindTraffic && mailTrafficWarn(r.U, r.D, r.TransferEnable) {
			key := "LAST_SEND_EMAIL_REMIND_TRAFFIC_" + strconv.FormatInt(r.ID, 10)
			ok, err := e.Redis.SetNX(context.Background(), key, 1, 24*time.Hour)
			if err != nil || !ok {
				continue
			}
			subject := replaceAppName("在 :app_name 的已用流量已达到 80%", app)
			if err := e.SendTemplateMail(r.Email, subject, "remindTraffic", values()); err != nil {
				log.Printf("remind traffic user %d: %v", r.ID, err)
			}
		}
	}
	return nil
}
