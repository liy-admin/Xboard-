package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// 节点国家码解析。对应 app/Services/GeoIpService.php：
// host → IPv4 → ip-api.com 批量接口（每批 ≤100，免 key）。
// geoip_method=maxmind 在 PHP 中同样未实现并回退到 api，Go 版行为一致。

var ipAPIBatchURL = "http://ip-api.com/batch"

var geoHTTP = &http.Client{Timeout: 15 * time.Second}

// hostToIPv4 与 PHP 的 gethostbyname 对齐：IP 字面量直接返回，域名取第一个 IPv4。
func hostToIPv4(host string) (string, bool) {
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4.String(), true
		}
		return "", false
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return "", false
	}
	for _, a := range addrs {
		if v4 := a.To4(); v4 != nil {
			return v4.String(), true
		}
	}
	return "", false
}

// lookupCountriesAPI 批量查询 IP 国家码，返回 ip → 大写 alpha-2。失败的分片直接跳过。
func lookupCountriesAPI(ctx context.Context, ips []string) map[string]string {
	out := map[string]string{}
	for start := 0; start < len(ips); start += 100 {
		end := start + 100
		if end > len(ips) {
			end = len(ips)
		}
		payload := make([]map[string]string, 0, end-start)
		for _, ip := range ips[start:end] {
			payload = append(payload, map[string]string{"query": ip, "fields": "status,countryCode,query"})
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, ipAPIBatchURL, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := geoHTTP.Do(req)
		if err != nil {
			log.Printf("geoip: ip-api 查询异常: %v", err)
			continue
		}
		var rows []struct {
			Status      string `json:"status"`
			CountryCode string `json:"countryCode"`
			Query       string `json:"query"`
		}
		if resp.StatusCode == http.StatusOK && json.NewDecoder(resp.Body).Decode(&rows) == nil {
			for _, r := range rows {
				if r.Status == "success" && r.CountryCode != "" && r.Query != "" {
					out[r.Query] = r.CountryCode
				}
			}
		} else {
			log.Printf("geoip: ip-api batch 返回非 200 (%d)", resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
	return out
}

// LookupCountries 批量解析 host 列表 → host 对应的大写国家码（解析失败为空串）。
func LookupCountries(ctx context.Context, hosts []string) map[string]string {
	result := map[string]string{}
	hostIP := map[string]string{}
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, seen := hostIP[h]; seen {
			continue
		}
		if ip, ok := hostToIPv4(h); ok {
			hostIP[h] = ip
		}
		result[h] = ""
	}
	if len(hostIP) == 0 {
		return result
	}
	uniq := make([]string, 0, len(hostIP))
	seen := map[string]bool{}
	for _, ip := range hostIP {
		if !seen[ip] {
			seen[ip] = true
			uniq = append(uniq, ip)
		}
	}
	byIP := lookupCountriesAPI(ctx, uniq)
	for h, ip := range hostIP {
		result[h] = strings.ToUpper(byIP[ip])
	}
	return result
}
