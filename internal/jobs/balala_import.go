package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"xboard/internal/model"
	"xboard/internal/nodeparse"
)

// balala-sub:import：抓取 Balala 最新订阅 → 按标识整批替换节点。
// 对应 plugins/BalalaSub/Commands/ImportCommand.php；插件注册为每分钟调度，仅在插件启用时执行。

const (
	balalaLastRunKey  = "plugin:balala_sub:last_import"
	balalaStatusKey   = "plugin:balala_sub:import_status"
	balalaRequestKey  = "plugin:balala_sub:import_requested"
	balalaImportLabel = "balala_sub:import"
)

var balalaInfoKeywords = []string{
	"expire", "traffic", "sync", "reset", "剩余", "流量", "到期", "过期", "有效期",
	"重置", "套餐", "距离", "官网", "网址", "最新", "群组", "频道", "客服", "邀请",
	"教程", "公告", "续费", "购买", "订阅", "节点异常", "更新",
}

var balalaDomainRe = regexp.MustCompile(`(?i)[a-z0-9-]+\.(com|net|org|io|xyz|cc|me|top|shop|vip)\b`)

// balalaPluginRow 读取插件启用状态与原始配置。
func (e *Env) balalaPluginRow() (enabled bool, cfg map[string]any) {
	cfg = map[string]any{}
	var en bool
	var raw *string
	row := e.DB.Raw("SELECT is_enabled, config FROM v2_plugins WHERE code = ?", "balala_sub").Row()
	if row.Scan(&en, &raw) != nil {
		return false, cfg
	}
	if raw != nil && *raw != "" {
		_ = json.Unmarshal([]byte(*raw), &cfg)
	}
	return en, cfg
}

func cfgString(m map[string]any, k string) string {
	if v, ok := m[k]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

// isInfoNode 判断是否为订阅里的「信息条目」（到期、流量、官网等），这类条目不是真实节点。
func isInfoNode(name string, extra string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "http") || balalaDomainRe.MatchString(name) {
		return true
	}
	kws := append([]string{}, balalaInfoKeywords...)
	for _, kw := range strings.FieldsFunc(extra, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
		if kw = strings.ToLower(strings.TrimSpace(kw)); kw != "" {
			kws = append(kws, kw)
		}
	}
	for _, kw := range kws {
		if kw != "" && strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// parseGroupIDs 解析 "1,2 3" 或数组形式的权限组。
func parseGroupIDs(raw any) []int64 {
	var out []int64
	add := func(s string) {
		if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil && n != 0 {
			out = append(out, n)
		}
	}
	switch v := raw.(type) {
	case []any:
		for _, x := range v {
			add(fmt.Sprint(x))
		}
	case string:
		for _, s := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
			add(s)
		}
	case float64:
		add(strconv.FormatInt(int64(v), 10))
	}
	return out
}

// pickBalalaURL 先按客户端名称选链接，找不到用主订阅链接。
func pickBalalaURL(r BalalaResult, client string) string {
	for _, s := range r.Subscriptions {
		c, _ := s["client"].(string)
		u, _ := s["url"].(string)
		if u != "" && strings.Contains(strings.ToLower(c), strings.ToLower(client)) {
			return u
		}
	}
	return r.SubscribeURL
}

func (e *Env) balalaStatus(ok bool, msg string, extra map[string]any) error {
	st := map[string]any{"ok": ok, "message": msg, "at": time.Now().Format(time.RFC3339)}
	for k, v := range extra {
		st[k] = v
	}
	if e.Redis != nil {
		b, _ := json.Marshal(st)
		_ = e.Redis.Set(context.Background(), balalaStatusKey, string(b), 7*24*time.Hour)
	}
	if ok {
		log.Printf("%s: %s", balalaImportLabel, msg)
		return nil
	}
	log.Printf("%s: 失败：%s", balalaImportLabel, msg)
	return fmt.Errorf("%s", msg)
}

// BalalaImport 对应 balala-sub:import；force 忽略开关与间隔。
func (e *Env) BalalaImport(force bool) error {
	enabled, cfg := e.balalaPluginRow()
	if !enabled {
		return nil
	}
	ctx := context.Background()
	if e.Redis != nil {
		if _, ok, _ := e.Redis.Get(ctx, balalaRequestKey); ok {
			_ = e.Redis.Del(ctx, balalaRequestKey)
			force = true
			log.Printf("%s: 检测到手动导入请求，立即执行一次", balalaImportLabel)
		}
	}
	if !force && !balalaTruthy(cfg["auto_import"]) {
		return nil
	}
	interval, _ := strconv.Atoi(cfgString(cfg, "auto_import_interval_hours"))
	if interval < 1 {
		interval = 24
	}
	if !force && e.Redis != nil {
		if last, ok, _ := e.Redis.Get(ctx, balalaLastRunKey); ok {
			if t, err := time.Parse(time.RFC3339, last); err == nil && time.Now().Before(t.Add(time.Duration(interval)*time.Hour)) {
				return nil
			}
		}
	}

	tag := cfgString(cfg, "auto_import_tag")
	client := cfgString(cfg, "auto_import_client")
	if client == "" {
		client = "v2rayn"
	}

	res, err := e.BalalaFetchFull(ctx, e.LoadBalalaConfig())
	if err != nil {
		return e.balalaStatus(false, "抓取失败："+err.Error(), nil)
	}
	url := pickBalalaURL(res, client)
	if url == "" {
		return e.balalaStatus(false, "未找到可用订阅链接", nil)
	}
	parsed, err := nodeparse.ParseSubscription(url)
	if err != nil {
		return e.balalaStatus(false, "订阅下载失败："+err.Error(), nil)
	}
	if len(parsed.Servers) == 0 {
		return e.balalaStatus(false, "订阅解析出 0 个节点，保留现有节点不动", nil)
	}
	extra := cfgString(cfg, "info_node_keywords")
	servers := parsed.Servers[:0:0]
	for _, s := range parsed.Servers {
		if !isInfoNode(s.Name, extra) {
			servers = append(servers, s)
		}
	}
	if len(servers) == 0 {
		return e.balalaStatus(false, "过滤信息条目后无真实节点，保留现有节点不动", nil)
	}

	deleted := int64(0)
	if tag != "" {
		deleted = e.DB.Where("JSON_CONTAINS(tags, ?)", jsonString(tag)).Delete(&model.Server{}).RowsAffected
	} else {
		log.Printf("%s: 未配置订阅标识，无法替换，只做新增（建议填写标识）", balalaImportLabel)
	}

	groupIDs := parseGroupIDs(cfg["auto_import_group_ids"])
	rate := 1.0
	if r, err := strconv.ParseFloat(cfgString(cfg, "auto_import_rate"), 64); err == nil {
		rate = r
	}
	show, _ := strconv.Atoi(cfgString(cfg, "auto_import_show"))
	if len(groupIDs) == 0 {
		log.Printf("%s: 未配置权限组(auto_import_group_ids)，导入的节点无法测速和订阅下发", balalaImportLabel)
	}

	created := 0
	for _, s := range servers {
		row := map[string]any{
			"type": s.Type, "code": s.Code, "name": s.Name, "host": s.Host, "port": s.Port,
			"server_port": s.ServerPort, "rate": rate, "protocol_settings": jsonRaw(s.ProtocolSettings),
			"show": show != 0, "created_at": time.Now(), "updated_at": time.Now(),
		}
		if len(groupIDs) > 0 {
			row["group_ids"] = jsonRaw(groupIDs)
		}
		if tag != "" {
			row["tags"] = jsonRaw([]string{tag})
		}
		if err := e.DB.Model(&model.Server{}).Create(row).Error; err != nil {
			log.Printf("%s: 单节点创建失败 %s: %v", balalaImportLabel, s.Name, err)
			continue
		}
		created++
	}
	if e.Redis != nil {
		_ = e.Redis.Set(ctx, balalaLastRunKey, time.Now().Format(time.RFC3339), 30*24*time.Hour)
	}
	if deleted > 0 || created > 0 {
		e.broadcastSubscriptionUpdated("import")
	}
	return e.balalaStatus(true, fmt.Sprintf("删除旧节点 %d 个，导入 %d 个", deleted, created),
		map[string]any{"deleted": deleted, "created": created})
}
