package api

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// 插件配置读取。Telegram 插件的默认文案随 config.json 一并嵌入（与 plugins-core/Telegram/config.json 同步）。

//go:embed telegram_plugin.json
var telegramPluginManifest []byte

// pluginManifestDefaults 返回内置插件（telegram）的配置默认值。
func pluginManifestDefaults(code string) map[string]any {
	out := map[string]any{}
	if code != "telegram" {
		return out
	}
	var m struct {
		Config map[string]map[string]any `json:"config"`
	}
	if json.Unmarshal(telegramPluginManifest, &m) != nil {
		return out
	}
	for k, spec := range m.Config {
		out[k] = spec["default"]
	}
	return out
}

// pluginEnabled 对应插件是否已安装且启用（PHP 中只有启用的插件才会加载其功能）。
func (a *App) pluginEnabled(code string) bool {
	r, ok := a.pluginRowOf(code)
	return ok && r.IsEnabled
}

// pluginConfigValue 读取插件配置项：已保存的值优先，其次为内置默认值，最后为 def。
func (a *App) pluginConfigValue(code, key string, def any) any {
	if r, ok := a.pluginRowOf(code); ok && r.Config != "" {
		var saved map[string]any
		if json.Unmarshal([]byte(r.Config), &saved) == nil {
			if v, ok := saved[key]; ok && v != nil && fmt.Sprint(v) != "" {
				return v
			}
		}
	}
	if v, ok := pluginManifestDefaults(code)[key]; ok && v != nil {
		return v
	}
	return def
}

// pluginConfigText 同 pluginConfigValue，返回字符串。
func (a *App) pluginConfigText(code, key, def string) string {
	return fmt.Sprint(a.pluginConfigValue(code, key, def))
}

// truthy 把配置值按 PHP 语义转为布尔（0、""、"0"、false、nil 为假）。
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0
	case int:
		return x != 0
	case int64:
		return x != 0
	case string:
		return x != "" && x != "0"
	}
	return true
}
