package jobs

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"xboard/internal/model"
	"xboard/internal/nodeparse"
)

// ImportSubscriptions 对应 server:import-subscriptions：按预设定时拉取订阅，并以"标签"为单位整批替换节点。
// balala 重抓（refetch_balala）依赖外部插件，这里回退为使用预设中保存的链接。
func (e *Env) ImportSubscriptions(onlyID string, force bool) error {
	raw := e.Set.Get("node_import_subscriptions", "")
	var presets []map[string]any
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &presets) != nil || len(presets) == 0 {
		return nil
	}
	changed := false
	for i, p := range presets {
		id, _ := p["id"].(string)
		if onlyID != "" && id != onlyID {
			continue
		}
		if onlyID == "" && !truthy(p["schedule_enabled"]) {
			continue
		}
		if !force && onlyID == "" && !presetDue(p, e.Now()) {
			continue
		}
		url, _ := p["url"].(string)
		if truthy(p["refetch_balala"]) {
			if fresh, err := e.BalalaFetch(context.Background(), e.LoadBalalaConfig()); err == nil {
				url = fresh
			} else {
				log.Printf("import-subscriptions %v: balala 重抓失败，回退预设链接: %v", p["name"], err)
			}
		}
		if url == "" {
			continue
		}
		parsed, err := nodeparse.ParseSubscription(url)
		if err != nil {
			log.Printf("import-subscriptions %v: %v", p["name"], err)
			continue
		}
		if len(parsed.Servers) == 0 {
			log.Printf("import-subscriptions %v: 解析出 0 个节点，跳过（不动已有节点）", p["name"])
			continue
		}
		tag, _ := p["tag"].(string)
		tag = strings.TrimSpace(tag)
		if tag != "" {
			e.DB.Where("JSON_CONTAINS(tags, ?)", jsonString(tag)).Delete(&model.Server{})
		}
		created := 0
		for _, s := range parsed.Servers {
			row := map[string]any{
				"type": s.Type, "code": s.Code, "name": s.Name, "host": s.Host, "port": s.Port,
				"server_port": s.ServerPort, "rate": 1, "protocol_settings": jsonRaw(s.ProtocolSettings),
				"show":       int(numberOf(p["show"])) != 0,
				"created_at": time.Now(), "updated_at": time.Now(),
			}
			if gids, ok := p["group_ids"].([]any); ok && len(gids) > 0 {
				row["group_ids"] = jsonRaw(gids)
			}
			if r, ok := p["rate"]; ok && r != nil {
				row["rate"] = r
			}
			if tag != "" {
				row["tags"] = jsonRaw([]string{tag})
			}
			if err := e.DB.Model(&model.Server{}).Create(row).Error; err != nil {
				log.Printf("import-subscriptions create %s: %v", s.Name, err)
				continue
			}
			created++
		}
		log.Printf("import-subscriptions %v: 导入新节点 %d 个", p["name"], created)
		presets[i]["last_run_at"] = time.Now().Format(time.RFC3339)
		changed = true
	}
	if changed {
		// 有订阅源实际导入过节点：通知在线客户端重新拉取。
		e.broadcastSubscriptionUpdated("import")
		b, _ := json.Marshal(presets)
		return e.DB.Exec("UPDATE v2_settings SET value = ? WHERE name = 'node_import_subscriptions'", string(b)).Error
	}
	return nil
}

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x != "" && x != "0"
	}
	return false
}

func numberOf(v any) float64 {
	f, _ := v.(float64)
	return f
}

// presetDue 对应 isDue：间隔（小时，默认 24）已过或从未运行。
func presetDue(p map[string]any, now time.Time) bool {
	interval := 24.0
	if v, ok := p["interval_hours"].(float64); ok && v > 0 {
		interval = v
	}
	last, _ := p["last_run_at"].(string)
	if last == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return true
	}
	return !now.Before(t.Add(time.Duration(interval) * time.Hour))
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// jsonRaw 把任意值序列化为 JSON 文本（存入 JSON 列）。
func jsonRaw(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
