package api

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xboard/internal/model"
	"xboard/internal/resp"
)

// 后台「设备管理」：按设备查看绑定账号、是否已领过试用、在该设备上注册过的账号与试用领取记录，
// 并可手动重置 / 标记设备的试用资格（v2_device_binding.claimed_trial）。

// deviceRow 是设备列表 / 详情的一行。
type deviceRow struct {
	ID           int64
	DeviceID     string
	UserID       *int64
	Type         string
	ClaimedTrial bool
	CreatedAt    int64
	UpdatedAt    int64
	UserEmail    *string
	UserBanned   *bool
	AccountCount int64
	TrialCount   int64
}

const deviceRowSelect = `b.id, b.device_id, b.user_id, b.type, b.claimed_trial, b.created_at, b.updated_at,
u.email AS user_email, u.banned AS user_banned,
(SELECT COUNT(*) FROM v2_user x WHERE x.device_id = b.device_id) AS account_count,
(SELECT COUNT(*) FROM v2_device_trial_log l WHERE l.device_id = b.device_id) AS trial_count`

func (a *App) deviceBase() *gorm.DB {
	return a.S.DB.Table("v2_device_binding AS b").Joins("LEFT JOIN v2_user u ON u.id = b.user_id")
}

func (r deviceRow) view() gin.H {
	return gin.H{
		"id": r.ID, "device_id": r.DeviceID, "display_code": deviceDisplayCode(r.DeviceID),
		"user_id": r.UserID, "user_email": r.UserEmail, "user_banned": r.UserBanned,
		"type": r.Type, "claimed_trial": r.ClaimedTrial,
		"account_count": r.AccountCount, "trial_count": r.TrialCount,
		"created_at": r.CreatedAt, "updated_at": r.UpdatedAt,
	}
}

// paramString 读取字符串参数；缺省或 nil 时返回空串。
func paramString(p map[string]any, key string) string {
	v, ok := p[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

// deviceKeywordFilter 按关键字筛设备：设备 ID、当前绑定账号邮箱、在该设备注册过的账号邮箱、用户 ID。
// 设备体验账号的邮箱是 "<9 位设备码>@device.local"，因此输入 9 位设备码也能搜到领过体验的设备。
func deviceKeywordFilter(q *gorm.DB, kw string) *gorm.DB {
	if kw == "" {
		return q
	}
	like := "%" + kw + "%"
	cond := "b.device_id LIKE ? OR u.email LIKE ? OR EXISTS (SELECT 1 FROM v2_user x WHERE x.device_id = b.device_id AND x.email LIKE ?)"
	args := []any{like, like, like}
	if id, err := strconv.ParseInt(kw, 10, 64); err == nil {
		cond += " OR b.user_id = ?"
		args = append(args, id)
	}
	return q.Where("("+cond+")", args...)
}

// deviceSort 设备列表可排序的列（列键 → SQL 表达式）；display_code 由 deviceDisplayCodePage 单独处理。
var deviceSort = map[string]string{
	"device_id":     "b.device_id",
	"current_user":  "u.email",
	"claimed_trial": "b.claimed_trial",
	"accounts":      "(SELECT COUNT(*) FROM v2_user x WHERE x.device_id = b.device_id)",
	"trials":        "(SELECT COUNT(*) FROM v2_device_trial_log l WHERE l.device_id = b.device_id)",
	"updated_at":    "b.updated_at",
	"created_at":    "b.created_at",
}

// deviceDisplayCodePage 按设备码排序时，取出全部匹配设备、按设备码排好后返回当前页的 id。
func deviceDisplayCodePage(c *gin.Context, build func() *gorm.DB, cur, size int) ([]int64, bool) {
	ss := requestSorts(c)
	if len(ss) == 0 || ss[0].ID != "display_code" {
		return nil, false
	}
	var all []struct {
		ID       int64
		DeviceID string
	}
	build().Select("b.id, b.device_id").Scan(&all)
	desc := ss[0].Desc
	sort.SliceStable(all, func(i, j int) bool {
		ci, cj := deviceDisplayCode(all[i].DeviceID), deviceDisplayCode(all[j].DeviceID)
		if desc {
			return ci > cj
		}
		return ci < cj
	})
	start := (cur - 1) * size
	if start > len(all) {
		start = len(all)
	}
	end := start + size
	if end > len(all) {
		end = len(all)
	}
	ids := make([]int64, 0, end-start)
	for _, r := range all[start:end] {
		ids = append(ids, r.ID)
	}
	return ids, true
}

func (a *App) adminDeviceFetch(c *gin.Context) {
	cur, size := pageOf(c)
	p := requestParams(c)
	kw := paramString(p, "keyword")
	claimed := paramString(p, "claimed_trial")
	build := func() *gorm.DB {
		q := deviceKeywordFilter(a.deviceBase(), kw)
		switch claimed {
		case "1":
			q = q.Where("b.claimed_trial = 1")
		case "0":
			q = q.Where("b.claimed_trial = 0")
		}
		return q
	}
	var total int64
	if err := build().Count(&total).Error; err != nil {
		resp.Fail(c, 500, "查询设备失败："+err.Error())
		return
	}
	var rows []deviceRow
	if ids, ok := deviceDisplayCodePage(c, build, cur, size); ok {
		// 设备码是 device_id 的哈希，数据库里排不了：先在内存里排出这一页的 id，再按该顺序取行。
		if len(ids) > 0 {
			if err := build().Select(deviceRowSelect).Where("b.id IN ?", ids).Scan(&rows).Error; err != nil {
				resp.Fail(c, 500, "查询设备失败："+err.Error())
				return
			}
		}
		pos := make(map[int64]int, len(ids))
		for i, id := range ids {
			pos[id] = i
		}
		sort.Slice(rows, func(i, j int) bool { return pos[rows[i].ID] < pos[rows[j].ID] })
	} else if err := adminSortExprs(c, deviceSort, build().Select(deviceRowSelect)).Order("b.updated_at DESC, b.id DESC").
		Offset((cur - 1) * size).Limit(size).Scan(&rows).Error; err != nil {
		resp.Fail(c, 500, "查询设备失败："+err.Error())
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.view())
	}
	paginateJSON(c, total, cur, size, out)
}

// adminDeviceDetail 返回设备本身、在该设备上注册过（或当前绑定）的账号，以及试用领取记录。
func (a *App) adminDeviceDetail(c *gin.Context) {
	deviceID := paramString(requestParams(c), "device_id")
	if deviceID == "" {
		resp.Fail(c, 422, "缺少 device_id")
		return
	}
	var rows []deviceRow
	a.deviceBase().Select(deviceRowSelect).Where("b.device_id = ?", deviceID).Limit(1).Scan(&rows)
	if len(rows) == 0 {
		resp.Fail(c, 404, "设备不存在")
		return
	}
	dev := rows[0]

	q := a.S.DB.Model(&model.User{}).Where("device_id = ?", deviceID)
	if dev.UserID != nil {
		q = q.Or("id = ?", *dev.UserID)
	}
	var users []model.User
	q.Order("id DESC").Limit(200).Find(&users)
	planNames := map[int64]string{}
	var planIDs []int64
	for _, u := range users {
		if u.PlanID != nil {
			planIDs = append(planIDs, *u.PlanID)
		}
	}
	if len(planIDs) > 0 {
		var ps []model.Plan
		a.S.DB.Select("id", "name").Where("id IN ?", planIDs).Find(&ps)
		for _, p := range ps {
			planNames[p.ID] = p.Name
		}
	}
	accounts := make([]gin.H, 0, len(users))
	for _, u := range users {
		var planName any
		if u.PlanID != nil {
			planName = planNames[*u.PlanID]
		}
		accounts = append(accounts, gin.H{
			"id":              u.ID,
			"email":           u.Email,
			"banned":          u.Banned,
			"is_device_trial": isTrialEmail(u.Email),
			"is_current":      dev.UserID != nil && *dev.UserID == u.ID,
			"plan_id":         u.PlanID,
			"plan_name":       planName,
			"expired_at":      u.ExpiredAt,
			"created_at":      u.CreatedAt,
		})
	}

	var logs []model.DeviceTrialLog
	a.S.DB.Where("device_id = ?", deviceID).Order("id DESC").Limit(200).Find(&logs)

	resp.OK(c, gin.H{"device": dev.view(), "accounts": accounts, "trial_logs": logs})
}

// adminDeviceSetTrialClaimed 手动设置设备是否已领取试用：置 0 即允许该设备再领一次。
func (a *App) adminDeviceSetTrialClaimed(c *gin.Context) {
	p := requestParams(c)
	deviceID := paramString(p, "device_id")
	if deviceID == "" {
		resp.Fail(c, 422, "缺少 device_id")
		return
	}
	var claimed int64
	switch v := p["claimed_trial"].(type) {
	case bool:
		if v {
			claimed = 1
		}
	default:
		n, ok := anyInt(v)
		if !ok || (n != 0 && n != 1) {
			resp.Fail(c, 422, "claimed_trial 只能是 0 或 1")
			return
		}
		claimed = n
	}
	res := a.S.DB.Exec("UPDATE v2_device_binding SET claimed_trial = ?, updated_at = ? WHERE device_id = ?", claimed, now(), deviceID)
	if res.Error != nil {
		resp.Fail(c, 500, "更新失败："+res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		resp.Fail(c, 404, "设备不存在")
		return
	}
	resp.OK(c, true)
}
