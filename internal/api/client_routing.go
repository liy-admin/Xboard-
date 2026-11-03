package api

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"xboard/internal/resp"
)

// 客户端分流规则：后台「节点设置 → 客户端分流规则」配置，App 通过免登录接口拉取后
// 交给 hiddify-core（HiddifyOptions.custom-rules）生成 sing-box 路由。
// 规则字段名与 sing-box 路由规则一致；outbound 取 proxy / direct / block。
// 规则排在地区分流（.cn / geosite-cn 直连）之前，安全模式（全局）下客户端会忽略 direct 规则。

const (
	settingClientRoutingEnable = "client_routing_enable"
	settingClientRoutingRules  = "client_routing_rules"
	settingRuleSetURLTemplate  = "client_ruleset_url_template"

	// DefaultRuleSetURLTemplate：Loyalsoldier/v2ray-rules-dat 的 sing-box 移植（2dust/sing-box-rules，
	// 每日同步上游，与 v2rayN 使用的规则集相同）。{type} = geosite/geoip，{name} = 完整短名。
	DefaultRuleSetURLTemplate = "https://raw.githubusercontent.com/2dust/sing-box-rules/rule-set-{type}/{name}.srs"
)

// ValidateRuleSetURLTemplate 校验规则集地址模板；空串表示使用 App 内置的 hiddify-geo。
func ValidateRuleSetURLTemplate(tpl string) error {
	tpl = strings.TrimSpace(tpl)
	if tpl == "" {
		return nil
	}
	if !strings.HasPrefix(tpl, "https://") && !strings.HasPrefix(tpl, "http://") {
		return fmt.Errorf("规则集地址模板需以 http(s):// 开头")
	}
	if !strings.Contains(tpl, "{name}") {
		return fmt.Errorf("规则集地址模板必须包含 {name} 占位符")
	}
	return nil
}

// isRuleSetShortName 判断 geosite-xxx / geoip-xxx 形式的规则集短名。
func isRuleSetShortName(s string) bool {
	typ, rest, ok := strings.Cut(s, "-")
	return ok && rest != "" && (typ == "geosite" || typ == "geoip") && !strings.ContainsAny(s, " /?#")
}

// ClientRoutingRule 是下发给客户端的一条规则。
type ClientRoutingRule struct {
	Name          string   `json:"name,omitempty"`
	Outbound      string   `json:"outbound"`
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex   []string `json:"domain_regex,omitempty"`
	IPCIDR        []string `json:"ip_cidr,omitempty"`
	Port          []uint16 `json:"port,omitempty"`
	PortRange     []string `json:"port_range,omitempty"`
	Network       string   `json:"network,omitempty"`
	Protocol      []string `json:"protocol,omitempty"`
	ProcessName   []string `json:"process_name,omitempty"`
	PackageName   []string `json:"package_name,omitempty"`
	RuleSet       []string `json:"rule_set,omitempty"`
}

func cleanList(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func (r *ClientRoutingRule) normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Outbound = strings.ToLower(strings.TrimSpace(r.Outbound))
	if r.Outbound == "" {
		r.Outbound = "proxy"
	}
	r.Domain = cleanList(r.Domain)
	r.DomainSuffix = cleanList(r.DomainSuffix)
	r.DomainKeyword = cleanList(r.DomainKeyword)
	r.DomainRegex = cleanList(r.DomainRegex)
	r.IPCIDR = cleanList(r.IPCIDR)
	r.PortRange = cleanList(r.PortRange)
	r.Protocol = cleanList(r.Protocol)
	r.ProcessName = cleanList(r.ProcessName)
	r.PackageName = cleanList(r.PackageName)
	r.RuleSet = cleanList(r.RuleSet)
	r.Network = strings.ToLower(strings.TrimSpace(r.Network))
}

func (r *ClientRoutingRule) hasMatcher() bool {
	return len(r.Domain)+len(r.DomainSuffix)+len(r.DomainKeyword)+len(r.DomainRegex)+len(r.IPCIDR)+
		len(r.Port)+len(r.PortRange)+len(r.Protocol)+len(r.ProcessName)+len(r.PackageName)+len(r.RuleSet) > 0 ||
		r.Network != ""
}

// ParseClientRoutingRules 解析并校验后台填写的规则 JSON（数组）。返回的错误带规则序号，直接展示给管理员。
// template 为当前的规则集地址模板：rule_set 用短名时必须配置了模板。
func ParseClientRoutingRules(raw, template string) ([]ClientRoutingRule, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []ClientRoutingRule{}, nil
	}
	var rules []ClientRoutingRule
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rules); err != nil {
		return nil, fmt.Errorf("分流规则不是合法的 JSON 数组：%v", err)
	}
	for i := range rules {
		r := &rules[i]
		r.normalize()
		switch r.Outbound {
		case "proxy", "direct", "block":
		default:
			return nil, fmt.Errorf("第 %d 条规则 outbound 只能是 proxy / direct / block，当前为 %q", i+1, r.Outbound)
		}
		if r.Network != "" && r.Network != "tcp" && r.Network != "udp" {
			return nil, fmt.Errorf("第 %d 条规则 network 只能是 tcp 或 udp", i+1)
		}
		if !r.hasMatcher() {
			return nil, fmt.Errorf("第 %d 条规则没有任何匹配条件", i+1)
		}
		for _, u := range r.RuleSet {
			if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
				continue
			}
			if !isRuleSetShortName(u) {
				return nil, fmt.Errorf("第 %d 条规则 rule_set 需填写 geosite-xxx / geoip-xxx 短名或 .srs 的 http(s) 地址：%q", i+1, u)
			}
			if strings.TrimSpace(template) == "" {
				return nil, fmt.Errorf("第 %d 条规则使用了规则集短名 %q，需先配置规则集地址模板", i+1, u)
			}
		}
	}
	return rules, nil
}

// clientRoutingRaw 取存储的规则文本。configSave 对字符串原样保存；若是旧数据以数组存储也兼容。
func (a *App) clientRoutingRaw() string {
	return a.Set.Get(settingClientRoutingRules, "")
}

// ruleSetURLTemplate 取规则集地址模板：从未设置过时用 Loyalsoldier 数据；显式清空则为空（App 用内置 hiddify-geo）。
func (a *App) ruleSetURLTemplate() string {
	if !a.Set.Has(settingRuleSetURLTemplate) {
		return DefaultRuleSetURLTemplate
	}
	tpl := strings.TrimSpace(a.Set.Get(settingRuleSetURLTemplate, ""))
	if ValidateRuleSetURLTemplate(tpl) != nil {
		return DefaultRuleSetURLTemplate
	}
	return tpl
}

// guestClientRouting：GET /api/v1/guest/client/routing。
// 关闭或规则无效时 rules 为空数组（客户端按无自定义规则处理）；rule_set_url_template 不受开关影响，
// 决定极速模式 geoip-cn / geosite-cn 的来源；version 为规则内容摘要，客户端据此判断是否需要重连生效。
func (a *App) guestClientRouting(c *gin.Context) {
	tpl := a.ruleSetURLTemplate()
	rules := []ClientRoutingRule{}
	if a.Set.Bool(settingClientRoutingEnable, false) {
		if parsed, err := ParseClientRoutingRules(a.clientRoutingRaw(), tpl); err == nil {
			rules = parsed
		}
	}
	dns := a.clientDirectDNS()
	b, _ := json.Marshal(gin.H{"rules": rules, "tpl": tpl, "dns": dns})
	sum := sha1.Sum(b)
	resp.OK(c, gin.H{
		"rules":                 rules,
		"rule_set_url_template": tpl,
		"direct_dns":            dns,
		"version":               hex.EncodeToString(sum[:8]),
	})
}
