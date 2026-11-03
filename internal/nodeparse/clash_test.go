package nodeparse

import "testing"

const clashSample = `
mixed-port: 7890
dns:
  nameserver:
    - https://doh.example/dns-query
proxies:
  - {name: "VL-Reality", type: vless, server: v.example.com, port: 443, uuid: 0f0b0c0d-1111-2222-3333-444444444444, tls: true, network: tcp, flow: xtls-rprx-vision, servername: sni.example.com, reality-opts: {public-key: PUBKEY, short-id: ab12}}
  - {name: "VL-WS", type: vless, server: w.example.com, port: 8443, uuid: 0f0b0c0d-1111-2222-3333-444444444444, tls: true, network: ws, ws-opts: {path: /ws, headers: {Host: cdn.example.com}}, servername: cdn.example.com, skip-cert-verify: true}
  - {name: "TJ", type: trojan, server: t.example.com, port: 443, password: pw1, sni: t.example.com}
  - {name: "HY2", type: hysteria2, server: h.example.com, port: 443, password: hp, sni: h.example.com, obfs: salamander, obfs-password: op, ports: "20000-30000"}
  - {name: "HY1", type: hysteria, server: h1.example.com, port: 443, auth-str: a, up: "50 Mbps", down: "100 Mbps", sni: h1.example.com}
  - {name: "TUIC", type: tuic, server: u.example.com, port: 443, uuid: 0f0b0c0d-1111-2222-3333-444444444444, password: tp, alpn: [h3], congestion-controller: bbr, sni: u.example.com}
  - {name: "ANY", type: anytls, server: a.example.com, port: 8443, password: 0f0b0c0d-1111-2222-3333-444444444444, sni: a.example.com}
  - {name: "SS", type: ss, server: 1.2.3.4, port: 8388, cipher: aes-256-gcm, password: sspw}
  - {name: "VM", type: vmess, server: m.example.com, port: 443, uuid: 0f0b0c0d-1111-2222-3333-444444444444, tls: true, network: ws, ws-opts: {path: /m, headers: {Host: m.example.com}}}
  - {name: "WG", type: wireguard, server: g.example.com, port: 51820}
proxy-groups:
  - {name: SELECT, type: select, proxies: [VL-Reality]}
`

func TestLooksLikeClashYAMLIgnoresDNSURLs(t *testing.T) {
	if !LooksLikeClashYAML(clashSample) {
		t.Fatal("yaml with https:// in dns must still be detected")
	}
	if LooksLikeClashYAML("vless://u@h:443#x\nproxies:\n") {
		t.Fatal("share-link list must not be treated as yaml")
	}
}

func TestParseClashYAMLAllTypes(t *testing.T) {
	res := ParseClashYAML(clashSample)
	if len(res.Servers) != 9 {
		t.Fatalf("servers=%d errors=%+v", len(res.Servers), res.Errors)
	}
	if len(res.Errors) != 1 || res.Errors[0].Content != "WG" {
		t.Fatalf("expected one unsupported (wireguard) error, got %+v", res.Errors)
	}
	by := map[string]Server{}
	for _, s := range res.Servers {
		by[s.Name] = s
	}
	if s := by["HY2"]; s.Type != "hysteria" || s.ProtocolSettings["version"] != 2 || s.Port != "20000-30000" {
		t.Errorf("hysteria2 port hopping: %+v", s)
	}
	if s := by["HY1"]; s.ProtocolSettings["version"] != 1 {
		t.Errorf("hysteria1: %+v", s)
	} else if bw, _ := s.ProtocolSettings["bandwidth"].(map[string]any); bw["up"] != 50 || bw["down"] != 100 {
		t.Errorf("bandwidth: %v", bw)
	}
	if s := by["VL-Reality"]; s.ProtocolSettings["reality_settings"] == nil || s.ProtocolSettings["flow"] != "xtls-rprx-vision" {
		t.Errorf("reality: %+v", s.ProtocolSettings)
	}
	if s := by["VL-WS"]; s.ProtocolSettings["network"] != "ws" {
		t.Errorf("ws: %+v", s.ProtocolSettings)
	} else if ns, _ := s.ProtocolSettings["network_settings"].(map[string]any); ns["path"] != "/ws" {
		t.Errorf("ws path: %v", ns)
	} else if ts, _ := s.ProtocolSettings["tls_settings"].(map[string]any); ts["allow_insecure"] != true {
		t.Errorf("insecure: %v", ts)
	}
	if s := by["ANY"]; s.Type != "anytls" || s.Code == nil {
		t.Errorf("anytls: %+v", s)
	}
	if s := by["SS"]; s.ProtocolSettings["cipher"] != "aes-256-gcm" || s.Host != "1.2.3.4" {
		t.Errorf("ss: %+v", s)
	}
	if s := by["VM"]; s.Type != "vmess" || s.ProtocolSettings["network"] != "ws" {
		t.Errorf("vmess: %+v", s)
	}
	if s := by["TUIC"]; s.ProtocolSettings["congestion_control"] != "bbr" {
		t.Errorf("tuic: %+v", s.ProtocolSettings)
	}
}

func TestParseBodyPicksFormat(t *testing.T) {
	if r := parseBody(clashSample); len(r.Servers) != 9 {
		t.Errorf("yaml body: %d", len(r.Servers))
	}
	if r := parseBody("anytls://pw@a.example.com:443#A\ntrojan://pw@b.example.com:443#B"); len(r.Servers) != 2 {
		t.Errorf("link body: %d", len(r.Servers))
	}
}
