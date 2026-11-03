package jobs

import (
	"regexp"
	"strings"
)

// ParseCountryFromName 对应 ServerService::parseCountryFromName：从节点名推断国家码。
// 顺序与 PHP 一致：旗帜 emoji → 中文名关键词 → 英文名/国家码（单词边界）。
func ParseCountryFromName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	// 1) 旗帜：两个连续的 regional indicator（U+1F1E6..U+1F1FF）
	var ri []rune
	for _, r := range name {
		if r >= 0x1F1E6 && r <= 0x1F1FF {
			ri = append(ri, r)
		}
	}
	if len(ri) >= 2 {
		a, b := ri[0]-0x1F1E6, ri[1]-0x1F1E6
		if a < 26 && b < 26 {
			return string([]rune{'A' + a, 'A' + b})
		}
	}
	// 2) 中文名关键词（有序，首个命中生效）
	for _, p := range countryNameAliases {
		for _, kw := range p.kws {
			if strings.Contains(name, kw) {
				return p.cc
			}
		}
	}
	// 3) 英文名 / 国家码，按单词边界匹配
	upper := strings.ToUpper(name)
	for _, p := range countryEnKeywords {
		for _, kw := range p.kws {
			if countryEnRes[kw].MatchString(upper) {
				return p.cc
			}
		}
	}
	return ""
}

type ccKeywords struct {
	cc  string
	kws []string
}

var countryNameAliases = []ccKeywords{
	{"HK", []string{"香港", "港"}}, {"TW", []string{"台湾", "臺灣", "台"}}, {"MO", []string{"澳门", "澳門"}},
	{"JP", []string{"日本", "东京", "大阪"}}, {"KR", []string{"韩国", "韓國", "首尔"}},
	{"SG", []string{"新加坡", "狮城"}}, {"MY", []string{"马来", "馬來"}}, {"TH", []string{"泰国", "泰國"}},
	{"VN", []string{"越南"}}, {"PH", []string{"菲律宾", "菲律賓"}}, {"ID", []string{"印尼", "印度尼西亚"}},
	{"IN", []string{"印度"}}, {"TR", []string{"土耳其"}}, {"AE", []string{"阿联酋", "迪拜"}}, {"IL", []string{"以色列"}},
	{"US", []string{"美国", "美國", "洛杉矶", "硅谷", "圣何塞", "纽约"}}, {"CA", []string{"加拿大"}},
	{"MX", []string{"墨西哥"}}, {"BR", []string{"巴西"}}, {"AR", []string{"阿根廷"}}, {"GB", []string{"英国", "英國", "伦敦"}},
	{"DE", []string{"德国", "德國", "法兰克福"}}, {"FR", []string{"法国", "法國", "巴黎"}},
	{"NL", []string{"荷兰", "荷蘭", "阿姆斯特丹"}}, {"RU", []string{"俄罗斯", "俄羅斯", "莫斯科"}},
	{"UA", []string{"乌克兰", "烏克蘭"}}, {"IT", []string{"意大利"}}, {"ES", []string{"西班牙"}}, {"PT", []string{"葡萄牙"}},
	{"SE", []string{"瑞典"}}, {"NO", []string{"挪威"}}, {"FI", []string{"芬兰", "芬蘭"}}, {"DK", []string{"丹麦", "丹麥"}},
	{"PL", []string{"波兰", "波蘭"}}, {"CH", []string{"瑞士"}}, {"AT", []string{"奥地利", "奧地利"}},
	{"IE", []string{"爱尔兰", "愛爾蘭"}}, {"BE", []string{"比利时", "比利時"}}, {"CZ", []string{"捷克"}},
	{"RO", []string{"罗马尼亚", "羅馬尼亞"}}, {"AU", []string{"澳大利亚", "澳洲", "悉尼", "墨尔本"}},
	{"NZ", []string{"新西兰", "紐西蘭"}}, {"ZA", []string{"南非"}}, {"CN", []string{"中国", "中國", "大陆", "回国"}},
}

var countryEnKeywords = []ccKeywords{
	{"HK", []string{"HK", "HONG KONG"}}, {"TW", []string{"TW", "TAIWAN"}}, {"JP", []string{"JP", "JAPAN", "TOKYO"}},
	{"KR", []string{"KR", "KOREA", "SEOUL"}}, {"SG", []string{"SG", "SINGAPORE"}},
	{"US", []string{"US", "USA", "UNITED STATES", "LOS ANGELES"}},
	{"GB", []string{"GB", "UK", "UNITED KINGDOM", "LONDON"}}, {"DE", []string{"DE", "GERMANY"}},
	{"FR", []string{"FR", "FRANCE", "PARIS"}}, {"NL", []string{"NL", "NETHERLANDS"}}, {"RU", []string{"RU", "RUSSIA"}},
	{"CA", []string{"CA", "CANADA"}}, {"AU", []string{"AU", "AUSTRALIA"}},
	{"MY", []string{"MY", "MALAYSIA"}}, {"TH", []string{"TH", "THAILAND"}}, {"VN", []string{"VN", "VIETNAM"}},
	{"IN", []string{"IN", "INDIA"}}, {"TR", []string{"TR", "TURKEY"}}, {"AE", []string{"AE", "DUBAI"}},
	{"CN", []string{"CN", "CHINA"}},
}

var countryEnRes = buildEnRes()

func buildEnRes() map[string]*regexp.Regexp {
	m := map[string]*regexp.Regexp{}
	for _, p := range countryEnKeywords {
		for _, kw := range p.kws {
			m[kw] = regexp.MustCompile(`\b` + regexp.QuoteMeta(kw) + `\b`)
		}
	}
	return m
}
