package sub

import (
	"strings"
	"unicode/utf8"
)

var countryZH = map[string]string{
	"HK": "香港", "TW": "台湾", "MO": "澳门", "CN": "中国大陆", "JP": "日本", "KR": "韩国", "SG": "新加坡",
	"MY": "马来西亚", "TH": "泰国", "VN": "越南", "PH": "菲律宾", "ID": "印尼", "IN": "印度", "TR": "土耳其",
	"AE": "阿联酋", "IL": "以色列", "US": "美国", "CA": "加拿大", "MX": "墨西哥", "BR": "巴西", "AR": "阿根廷",
	"GB": "英国", "DE": "德国", "FR": "法国", "NL": "荷兰", "RU": "俄罗斯", "UA": "乌克兰", "IT": "意大利",
	"ES": "西班牙", "PT": "葡萄牙", "SE": "瑞典", "NO": "挪威", "FI": "芬兰", "DK": "丹麦", "PL": "波兰",
	"CH": "瑞士", "AT": "奥地利", "IE": "爱尔兰", "BE": "比利时", "CZ": "捷克", "RO": "罗马尼亚",
	"AU": "澳大利亚", "NZ": "新西兰", "ZA": "南非",
}

// CountryDisplayName 对应 ServerService::countryDisplayName（旗帜 emoji + 中文名）。
func CountryDisplayName(cc string) string {
	cc = strings.ToUpper(cc)
	name := cc
	if n, ok := countryZH[cc]; ok {
		name = n
	}
	flag := "🏳️"
	if utf8.RuneCountInString(cc) == 2 && cc[0] >= 'A' && cc[0] <= 'Z' && cc[1] >= 'A' && cc[1] <= 'Z' {
		flag = string(rune(0x1F1E6+int(cc[0]-'A'))) + string(rune(0x1F1E6+int(cc[1]-'A')))
	}
	return flag + " " + name
}

// CountryNameZH 返回国家的中文名，未收录时回退为国家码。
func CountryNameZH(cc string) string {
	cc = strings.ToUpper(cc)
	if n, ok := countryZH[cc]; ok {
		return n
	}
	return cc
}

// CountryFlag 由两位国家码计算旗帜 emoji（regional indicator），非法国家码返回白旗。
func CountryFlag(cc string) string {
	cc = strings.ToUpper(cc)
	if len(cc) == 2 && cc[0] >= 'A' && cc[0] <= 'Z' && cc[1] >= 'A' && cc[1] <= 'Z' {
		return string(rune(0x1F1E6+int(cc[0]-'A'))) + string(rune(0x1F1E6+int(cc[1]-'A')))
	}
	return "🏳️"
}
