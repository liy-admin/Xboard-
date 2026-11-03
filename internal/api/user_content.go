package api

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// ---- 抽奖（LuckyDrawController） ----

// luckyWinPercent 计算中奖概率（百分比）：目标中奖人数 / 注册用户数，夹在 [min,max] 之间。
func luckyWinPercent(target, totalUsers int64, minPct, maxPct float64) float64 {
	if totalUsers < 1 {
		totalUsers = 1
	}
	if target < 0 {
		target = 0
	}
	p := float64(target) / float64(totalUsers) * 100
	if p < minPct {
		p = minPct
	}
	if p > maxPct {
		p = maxPct
	}
	return p
}

// luckyWon 对应 rollWin：random_int(1,1000000) <= round(percent*10000)。r 取值 [1,1000000]。
func luckyWon(percent float64, r int64) bool {
	return r <= int64(percent*10000+0.5)
}

func (a *App) luckyCooldownRemaining(uid int64) int64 {
	cd := int64(a.Set.Int("lucky_draw_cooldown_seconds", 3600))
	var last model.LuckyDrawLog
	if err := a.S.DB.Where("user_id = ?", uid).Order("created_at DESC").First(&last).Error; err != nil {
		return 0
	}
	left := cd - (now() - last.CreatedAt)
	if left < 0 {
		return 0
	}
	return left
}

func (a *App) luckyTotalUsers() int64 {
	const key = "lucky_draw:total_user_count"
	ctx := bgCtx()
	if v, err := a.S.Redis.Get(ctx, key).Int64(); err == nil {
		return v
	}
	var n int64
	a.S.DB.Model(&model.User{}).Count(&n)
	a.S.Redis.Set(ctx, key, n, 300*time.Second)
	return n
}

func (a *App) luckyInfo(c *gin.Context) {
	u := cur(c)
	var today, total int64
	y, m, d := time.Now().Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	a.S.DB.Model(&model.LuckyDrawLog{}).Where("is_win = ? AND created_at BETWEEN ? AND ?", true,
		start.Unix(), start.Add(24*time.Hour-time.Second).Unix()).Count(&today)
	a.S.DB.Model(&model.LuckyDrawLog{}).Where("is_win = ?", true).
		Select("COALESCE(SUM(hours_credited),0)").Scan(&total)
	resp.OK(c, gin.H{
		"enabled":                    a.Set.Bool("lucky_draw_enabled", true),
		"reward_hours":               a.Set.Int("lucky_draw_reward_hours", 1),
		"cooldown_remaining_seconds": a.luckyCooldownRemaining(u.ID),
		"today_winner_count":         today,
		"total_hours_claimed":        total,
	})
}

func (a *App) luckyClaim(c *gin.Context) {
	u := cur(c)
	lockKey := "lucky_draw:lock:" + strconv.FormatInt(u.ID, 10)
	ok, _ := a.S.Redis.SetNX(bgCtx(), lockKey, 1, 10*time.Second).Result()
	if !ok {
		resp.Fail(c, 400, "Please try again later")
		return
	}
	defer a.S.Redis.Del(bgCtx(), lockKey)

	if !a.Set.Bool("lucky_draw_enabled", true) {
		resp.Fail(c, 400, "This feature is not available")
		return
	}
	if a.luckyCooldownRemaining(u.ID) > 0 {
		resp.Fail(c, 400, "Please try again later")
		return
	}
	rewardHours := int64(a.Set.Int("lucky_draw_reward_hours", 1))
	pct := luckyWinPercent(int64(a.Set.Int("lucky_draw_daily_target_winners", 500)), a.luckyTotalUsers(),
		a.Set.Float("lucky_draw_min_probability_percent", 1), a.Set.Float("lucky_draw_max_probability_percent", 50))
	r, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	win := luckyWon(pct, r.Int64()+1)
	var credited int64

	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, u.ID).Error; err != nil {
			return errors.New("The user does not exist")
		}
		if !win {
			ts := now()
			return tx.Create(&model.LuckyDrawLog{UserID: u.ID, IsWin: false, CreatedAt: ts, UpdatedAt: ts}).Error
		}
		credited = rewardHours
		// 永久会员（有套餐且 expired_at 为空）保持永久，不做覆盖。
		permanent := user.PlanID != nil && user.ExpiredAt == nil
		if user.PlanID == nil {
			if tryID := int64(a.Set.Int("try_out_plan_id", 0)); tryID > 0 {
				var p model.Plan
				if tx.First(&p, tryID).Error == nil {
					pid := p.ID
					user.PlanID = &pid
					user.GroupID = p.GroupID
					user.TransferEnab = shop.GbToTransfer(derefInt64(p.TransferEnable))
					user.SpeedLimit = p.SpeedLimit
					user.DeviceLimit = p.DeviceLimit
				}
			}
		}
		if !permanent {
			exp := extendExpiry(user.ExpiredAt, now(), credited*3600)
			user.ExpiredAt = &exp
		}
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		ts := now()
		return tx.Create(&model.LuckyDrawLog{UserID: u.ID, IsWin: true, HoursCredited: credited, CreatedAt: ts, UpdatedAt: ts}).Error
	})
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	resp.OK(c, gin.H{
		"win": win, "hours_credited": credited,
		"cooldown_seconds": a.Set.Int("lucky_draw_cooldown_seconds", 3600),
	})
}

// extendExpiry 对应 UserService::extendSubscriptionSeconds：max(当前到期或now, now) + seconds。
func extendExpiry(expiredAt *int64, nowTs, seconds int64) int64 {
	base := nowTs
	if expiredAt != nil {
		base = *expiredAt
	}
	if base < nowTs {
		base = nowTs
	}
	return base + seconds
}

// ---- 公告（NoticeController） ----

func (a *App) noticeFetch(c *gin.Context) {
	current, _ := strconv.Atoi(c.Query("current"))
	if current < 1 {
		current = 1
	}
	const size = 5
	q := a.S.DB.Model(&model.Notice{}).Where("`show` = ?", true)
	var total int64
	q.Count(&total)
	var rows []model.Notice
	q.Order("sort ASC").Order("id DESC").Offset((current - 1) * size).Limit(size).Find(&rows)
	// 旧接口直接返回 {data,total}，不包 status/message 外壳。
	c.JSON(200, gin.H{"data": rows, "total": total})
}

// ---- 知识库（KnowledgeController） ----

var accessBlockRe = regexp.MustCompile(`(?s)<!--access start-->(.*?)<!--access end-->`)

const noAccessHTML = `<div class="v2board-no-access">您必须拥有有效的订阅才可以查看该区域的内容</div>`

// renderKnowledge 替换知识库正文占位符；无有效订阅时隐藏 access 区块。
func renderKnowledge(body string, available bool, siteName, subscribeURL string) string {
	if !available {
		body = accessBlockRe.ReplaceAllLiteralString(body, noAccessHTML)
	}
	safeB64 := strings.NewReplacer("+", "-", "/", "_", "=", "").Replace(base64.StdEncoding.EncodeToString([]byte(subscribeURL)))
	return strings.NewReplacer(
		"{{siteName}}", siteName,
		"{{subscribeUrl}}", subscribeURL,
		"{{urlEncodeSubscribeUrl}}", phpURLEncode(subscribeURL),
		"{{safeBase64SubscribeUrl}}", safeB64,
	).Replace(body)
}

// knowledgeSubscribeURL 对应 Helper::getSubscribeUrl：从 subscribe_url 列表随机选一个，否则用 app_url。
func (a *App) knowledgeSubscribeURL(token string) string { return a.subscribeURL(token, "") }

func (a *App) knowledgeFetch(c *gin.Context) {
	u := cur(c)
	avail := shop.IsAvailable(u)
	site := a.Set.Get("app_name", "Lets VPN")
	sub := a.knowledgeSubscribeURL(u.Token)

	if idStr := c.Query("id"); idStr != "" {
		var k model.Knowledge
		if err := a.S.DB.Where("`show` = ? AND id = ?", true, idStr).First(&k).Error; err != nil {
			resp.Fail(c, 500, "Article does not exist")
			return
		}
		resp.OK(c, gin.H{"id": k.ID, "category": k.Category, "title": k.Title,
			"body": renderKnowledge(k.Body, avail, site, sub), "updated_at": k.UpdatedAt})
		return
	}

	q := a.S.DB.Model(&model.Knowledge{}).Where("`show` = ? AND language = ?", true, c.Query("language"))
	if kw := c.Query("keyword"); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(title LIKE ? OR body LIKE ?)", like, like)
	}
	var rows []model.Knowledge
	q.Order("sort ASC").Find(&rows)
	grouped := &orderedGroups{vals: map[string][]gin.H{}}
	for _, k := range rows {
		grouped.add(k.Category, gin.H{
			"id": k.ID, "category": k.Category, "title": k.Title,
			"body": renderKnowledge(k.Body, avail, site, sub), "updated_at": k.UpdatedAt,
		})
	}
	resp.OK(c, grouped)
}

func (a *App) knowledgeCategory(c *gin.Context) {
	var cats []string
	a.S.DB.Model(&model.Knowledge{}).Where("`show` = ? AND language = ?", true, c.Query("language")).
		Distinct().Order("sort ASC").Pluck("category", &cats)
	resp.OK(c, cats)
}

// ---- 流量统计（StatController） ----

func (a *App) trafficLog(c *gin.Context) {
	u := cur(c)
	y, m, _ := time.Now().Date()
	start := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).Unix()
	var rows []model.StatUser
	a.S.DB.Where("user_id = ? AND record_at >= ?", u.ID, start).Order("record_at DESC").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{"d": r.D, "u": r.U, "record_at": r.RecordAt, "server_rate": r.ServerRate, "user_id": r.UserID})
	}
	resp.OK(c, out)
}

// ---- 在线客服（SupportController） ----

// supportEnvelope 与加密订阅端点同构：key = sha256(enc_key)，nonce 12 字节，tag 16 字节。
type supportEnvelope struct {
	V   int    `json:"v"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	C   string `json:"c"`
	T   string `json:"t"`
}

type supportPayload struct {
	Enabled      bool   `json:"enabled"`
	BaseURL      string `json:"base_url"`
	WebsiteToken string `json:"website_token"`
	Locale       string `json:"locale"`
}

func sealSupport(encKey string, p supportPayload) (supportEnvelope, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(p); err != nil {
		return supportEnvelope{}, err
	}
	pt := bytes.TrimRight(buf.Bytes(), "\n")
	key := sha256.Sum256([]byte(encKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return supportEnvelope{}, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return supportEnvelope{}, err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return supportEnvelope{}, err
	}
	sealed := gcm.Seal(nil, nonce, pt, nil)
	ct, tag := sealed[:len(sealed)-16], sealed[len(sealed)-16:]
	return supportEnvelope{
		V: 1, Alg: "aes-256-gcm",
		N: base64.StdEncoding.EncodeToString(nonce),
		C: base64.StdEncoding.EncodeToString(ct),
		T: base64.StdEncoding.EncodeToString(tag),
	}, nil
}

func (a *App) supportConfig(c *gin.Context) {
	u := cur(c)
	p := supportPayload{
		Enabled:      a.Set.Bool("support_chatwoot_enabled", false),
		BaseURL:      strings.TrimRight(a.Set.Get("support_chatwoot_base_url", ""), "/"),
		WebsiteToken: a.Set.Get("support_chatwoot_website_token", ""),
		Locale:       a.Set.Get("support_chatwoot_locale", "zh_CN"),
	}
	key := a.encKey(u)
	if key == "" {
		resp.Fail(c, 500, "客服配置暂不可用，请稍后重试")
		return
	}
	env, err := sealSupport(key, p)
	if err != nil {
		resp.Fail(c, 500, "客服配置暂不可用，请稍后重试")
		return
	}
	resp.OK(c, env)
}

// idStrings 对应 CouponResource 的 limit_plan_ids：数字转字符串数组，空则 null。
func idStrings(raw json.RawMessage) []string {
	var ids []int64
	if len(raw) == 0 || json.Unmarshal(raw, &ids) != nil || len(ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatInt(id, 10))
	}
	return out
}

// periodLegacyNames 对应 CouponResource 的 limit_period：新周期键转旧版键名。
func periodLegacyNames(raw json.RawMessage) []string {
	var ps []string
	if len(raw) == 0 || json.Unmarshal(raw, &ps) != nil || len(ps) == 0 {
		return nil
	}
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, shop.LegacyPeriodName(p))
	}
	return out
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
