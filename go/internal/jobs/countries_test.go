package jobs

import (
	"context"
	"sync"
	"testing"
	"time"

	"xboard/internal/model"
)

// memRedis 是 RedisClient 的内存实现，仅供测试。
type memRedis struct {
	mu sync.Mutex
	m  map[string]string
}

func newMemRedis() *memRedis { return &memRedis{m: map[string]string{}} }

func (r *memRedis) Get(_ context.Context, k string) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.m[k]
	return v, ok, nil
}
func (r *memRedis) Set(_ context.Context, k string, v any, _ time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[k] = v.(string)
	return nil
}
func (r *memRedis) SCard(context.Context, string) (int64, error)           { return 0, nil }
func (r *memRedis) SPopN(context.Context, string, int64) ([]string, error) { return nil, nil }
func (r *memRedis) SetNX(_ context.Context, k string, v any, _ time.Duration) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[k]; ok {
		return false, nil
	}
	r.m[k] = "1"
	return true, nil
}
func (r *memRedis) Del(_ context.Context, ks ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, k := range ks {
		delete(r.m, k)
	}
	return nil
}

func cc(s string) *string { return &s }
func i64(n int64) *int64  { return &n }

func TestSummarizeCountries(t *testing.T) {
	now := int64(1_000_000)
	nodes := []model.Server{
		{ID: 1, Name: "a", CountryCode: cc("jp"), LatencyMs: i64(300)},
		{ID: 2, Name: "b", CountryCode: cc("JP"), LatencyMs: i64(120)},
		{ID: 3, Name: "c", CountryCode: cc("US")},  // 国家有节点但都不可用 → 不列出
		{ID: 4, Name: "香港 01", CountryCode: nil},   // 国家码为空：从节点名解析
		{ID: 5, Name: "mystery", CountryCode: nil}, // 判定不出国家且可用 → 计入 unknown
		{ID: 6, Name: "d", CountryCode: cc("SG"), LatencyMs: i64(80)},
	}
	avail := map[int64]bool{1: true, 2: true, 3: false, 4: true, 5: true, 6: true}
	snap := summarizeCountries(nodes, func(n model.Server) bool { return avail[n.ID] }, now, "daily")

	got := map[string]CountryEntry{}
	for _, e := range snap.Countries {
		got[e.Code] = e
	}
	if _, ok := got["US"]; ok {
		t.Error("country with no available node must not be listed")
	}
	jp := got["JP"]
	if jp.Nodes != 2 || jp.TotalNodes != 2 || jp.BestLatencyMs == nil || *jp.BestLatencyMs != 120 {
		t.Errorf("JP: %+v", jp)
	}
	if got["HK"].Nodes != 1 {
		t.Errorf("country parsed from node name: %+v", got["HK"])
	}
	if snap.UnknownCountry != 1 {
		t.Errorf("unknown=%d", snap.UnknownCountry)
	}
	if snap.Countries[0].Code != "JP" {
		t.Errorf("sorted by available nodes desc, got %v first", snap.Countries[0].Code)
	}
	if snap.UpdatedAt != now || snap.Source != "daily" {
		t.Errorf("meta: %+v", snap)
	}
}

func TestParseCountryTags(t *testing.T) {
	got := ParseCountryTags(" hk：推荐，JP:高速\nsg\r\nUSA:太长,:空码,1A:数字, ,TW: ")
	want := map[string]string{"HK": "推荐", "JP": "高速", "SG": "推荐", "TW": "推荐"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
	if len(ParseCountryTags("")) != 0 {
		t.Error("empty setting must yield no tags")
	}
}

func TestKeepOrStoreKeepsPreviousWhenEverythingDown(t *testing.T) {
	e := &Env{Redis: newMemRedis(), Now: time.Now}
	good := CountrySnapshot{UpdatedAt: 1, Source: "daily", Countries: []CountryEntry{{Code: "JP", Name: "日本", Nodes: 2}}}
	if got := e.keepOrStore(good); len(got.Countries) != 1 || got.Kept {
		t.Fatalf("first store: %+v", got)
	}
	// 今天探测全挂（空结果）：沿用旧快照，并标记 kept，而不是把列表清空
	empty := CountrySnapshot{UpdatedAt: 2, Source: "daily", Countries: []CountryEntry{}}
	got := e.keepOrStore(empty)
	if len(got.Countries) != 1 || !got.Kept || got.UpdatedAt != 1 {
		t.Fatalf("must keep previous snapshot: %+v", got)
	}
	loaded, ok := e.LoadCountrySnapshot()
	if !ok || len(loaded.Countries) != 1 {
		t.Fatalf("stored snapshot lost: %+v", loaded)
	}
	// 有可用国家时正常覆盖
	next := CountrySnapshot{UpdatedAt: 3, Countries: []CountryEntry{{Code: "SG", Nodes: 1}}}
	if got := e.keepOrStore(next); got.Countries[0].Code != "SG" || got.Kept {
		t.Fatalf("new snapshot must replace: %+v", got)
	}
}

func TestKeepOrStoreEmptyWithoutPreviousIsStored(t *testing.T) {
	e := &Env{Redis: newMemRedis(), Now: time.Now}
	got := e.keepOrStore(CountrySnapshot{UpdatedAt: 5, Countries: []CountryEntry{}})
	if got.Kept || len(got.Countries) != 0 {
		t.Fatalf("%+v", got)
	}
	if s, ok := e.LoadCountrySnapshot(); !ok || s.Countries == nil {
		t.Fatalf("empty snapshot must round-trip with a non-nil list (JSON [] not null): %+v ok=%v", s, ok)
	}
}

func TestRefreshCountriesManualRejectsConcurrent(t *testing.T) {
	r := newMemRedis()
	e := &Env{Redis: r, Now: time.Now}
	_ = r.Set(context.Background(), countryRefreshingKey, "1", time.Minute) // 已有更新在进行
	if !e.CountriesRefreshing() {
		t.Fatal("flag must be visible")
	}
	if _, started := e.RefreshCountriesManual(false); started {
		t.Fatal("second manual refresh must not start while one is running")
	}
}

func TestNodeAvailableProbeDisabledKeepsCountries(t *testing.T) {
	e := &Env{Now: func() time.Time { return time.Unix(1_800_000_000, 0) }}
	old := int64(1_800_000_000 - 3600)
	lat := int64(90)
	reachable := model.Server{LatencyMs: &lat, LastLatencyAt: &old}
	failed := model.Server{LastLatencyAt: &old}
	if e.nodeAvailable(reachable, 600) {
		t.Fatal("probe enabled: an hour-old result is stale")
	}
	if !e.nodeAvailable(reachable, -1) || !e.nodeAvailable(model.Server{}, -1) || e.nodeAvailable(failed, -1) {
		t.Fatal("probe disabled: keep last verdict, never-probed counts as available")
	}
}
