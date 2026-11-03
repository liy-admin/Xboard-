package jobs

import "testing"

func TestResolveSingBoxMissing(t *testing.T) {
	bin, _, errMsg := ResolveSingBox("xboard-no-such-singbox-binary")
	if bin != "" && errMsg == "" {
		t.Skip("sing-box is installed on this host")
	}
	if bin != "" {
		return
	}
	if errMsg == "" {
		t.Fatal("missing binary must produce an error message")
	}
}

func TestParseEcho(t *testing.T) {
	ip, cc := parseEcho([]byte(`{"status":"success","countryCode":"JP","query":"1.2.3.4"}`))
	if ip != "1.2.3.4" || cc != "JP" {
		t.Fatalf("json: %s %s", ip, cc)
	}
	ip, cc = parseEcho([]byte("8.8.4.4\n"))
	if ip != "8.8.4.4" || cc != "" {
		t.Fatalf("plain: %s %s", ip, cc)
	}
	if ip, _ = parseEcho([]byte("<html>")); ip != "" {
		t.Fatalf("garbage: %s", ip)
	}
}

func TestLatencyStats(t *testing.T) {
	mn, avg, med, mx := latencyStats([]int64{30, 10, 20})
	if mn != 10 || avg != 20 || med != 20 || mx != 30 {
		t.Fatalf("odd: %d %d %d %d", mn, avg, med, mx)
	}
	_, _, med, _ = latencyStats([]int64{10, 21})
	if med != 16 {
		t.Fatalf("even median: %d", med)
	}
}

func TestFirstPort(t *testing.T) {
	if firstPort("10000-20000") != 10000 || firstPort(" 443 ") != 443 || firstPort("x") != 0 {
		t.Fatal("firstPort")
	}
}
