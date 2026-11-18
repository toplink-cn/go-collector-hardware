package main

import (
	"collector/modules/collection"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testConfig(t *testing.T) config {
	t.Helper()
	return config{SpoolDir: t.TempDir(), MaxPending: 10, MaxSpoolBytes: 1 << 20, MaxDeliver: 3, HTTPTimeout: time.Second}
}
func report(id string) []byte {
	return []byte(fmt.Sprintf(`{"Memory":{"Total":1000,"Free":500},"collection":{"version":2,"sample_id":%q,"collected_at":1700000000}}`, id))
}

func TestCollectionDeadlineAndPanicDoNotEraseOtherGroups(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	tasks := []sampleTask{
		{"memory", func(context.Context) moduleResult {
			return moduleResult{name: "memory", data: map[string]int{"Total": 1000, "Free": 500}, quality: map[string]collection.Quality{"memory": {State: "success"}}}
		}},
		{"cpu", func(context.Context) moduleResult { panic("fixture") }},
		{"disks", func(context.Context) moduleResult { <-release; return moduleResult{name: "disks"} }},
	}
	data, err := collectSnapshot(ctx, tasks, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Memory     struct{ Total int }
		Collection evidence
	}
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Memory.Total != 1000 || got.Collection.Status != "partial" {
		t.Fatalf("lost available data: %s", data)
	}
	if got.Collection.Quality["cpu"].State != "failed" || got.Collection.Quality["disks"].State != "failed" {
		t.Fatalf("missing failure evidence: %s", data)
	}
	if got.Collection.Version != 2 || got.Collection.SampleID == "" || got.Collection.Units["memory"] != "bytes" {
		t.Fatalf("missing report identity/units: %s", data)
	}
}

func TestOptionalSensorsDoNotDeclareBaseCollectionFailed(t *testing.T) {
	q := map[string]collection.Quality{"cpu": {State: "success"}, "memory": {State: "success"}, "disks": {State: "success"}, "network": {State: "success"}, "temperature": {State: "unsupported"}, "ipmi": {State: "unsupported"}}
	if sampleStatus(q) != "success" {
		t.Fatal("unsupported optional sensor failed base sample")
	}
	q["cpu"] = collection.Quality{State: "failed"}
	if sampleStatus(q) != "partial" {
		t.Fatal("required failure was hidden")
	}
}

func TestSpoolIsDurableAcrossRunsAndExcludesConcurrentWriters(t *testing.T) {
	c := testConfig(t)
	s, err := openSpool(c)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate, err := openSpool(c); err == nil {
		duplicate.close()
		t.Fatal("concurrent invocation acquired lock")
	}
	path, err := s.enqueue(report("a"))
	if err != nil {
		t.Fatal(err)
	}
	s.close()
	s, err = openSpool(c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	paths, _, err := s.pending()
	if err != nil || len(paths) != 1 || paths[0] != path {
		t.Fatalf("pending data lost on restart: %v %v", paths, err)
	}
	if data, _ := os.ReadFile(path); string(data) != string(report("a")) {
		t.Fatal("pending payload mutated")
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0600 {
		t.Fatal("report is not private")
	}
}

func TestSpoolFullNeverEvictsUnacknowledgedReports(t *testing.T) {
	c := testConfig(t)
	c.MaxPending = 1
	s, err := openSpool(c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	path, err := s.enqueue(report("a"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.enqueue(report("b")); err == nil {
		t.Fatal("full queue accepted another report")
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal("old report was evicted")
	}
	if err = os.WriteFile(filepath.Join(c.SpoolDir, ".pending-interrupted"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	paths, _, _ := s.pending()
	if len(paths) != 1 {
		t.Fatal("incomplete temporary report was treated as ready")
	}
}

func TestDeliveryOnlyDeletesMatchingDurableAcknowledgements(t *testing.T) {
	for _, mode := range []string{"failure", "invalid", "wrong-id", "not-persisted", "accepted"} {
		t.Run(mode, func(t *testing.T) {
			c := testConfig(t)
			s, err := openSpool(c)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			path, err := s.enqueue(report("a"))
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "failure":
					w.WriteHeader(503)
				case "invalid":
					fmt.Fprint(w, "hello")
				case "wrong-id":
					fmt.Fprint(w, `{"success":true,"persisted":true,"sample_id":"b"}`)
				case "not-persisted":
					fmt.Fprint(w, `{"success":true,"sample_id":"a"}`)
				case "accepted":
					fmt.Fprint(w, `{"success":true,"persisted":true,"sample_id":"a"}`)
				}
			}))
			defer server.Close()
			c.Endpoint = server.URL
			err = deliver(context.Background(), c, s, path)
			_, statErr := os.Stat(path)
			if mode == "accepted" {
				if err != nil || !os.IsNotExist(statErr) {
					t.Fatalf("acknowledged report retained: %v %v", err, statErr)
				}
			} else if err == nil || statErr != nil {
				t.Fatalf("unacknowledged report removed: %v %v", err, statErr)
			}
		})
	}
}

func TestLegacyReceiverOnlyGetsCurrentReportAndBacklogIsRetained(t *testing.T) {
	c := testConfig(t)
	s, err := openSpool(c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	old, err := s.enqueue(report("old"))
	if err != nil {
		t.Fatal(err)
	}
	latest, err := s.enqueue(report("new"))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, "OK") }))
	defer server.Close()
	c.Endpoint = server.URL
	if err = deliver(context.Background(), c, s, latest); err == nil {
		t.Fatal("legacy backlog limitation was hidden")
	}
	if calls != 1 {
		t.Fatal("historical data was replayed into legacy receiver")
	}
	if _, err = os.Stat(old); err != nil {
		t.Fatal("legacy backlog deleted")
	}
	if _, err = os.Stat(latest); !os.IsNotExist(err) {
		t.Fatal("current legacy acknowledgement not honored")
	}
}

func TestModernReceiverDrainsOldestPendingSamplesWithinBudget(t *testing.T) {
	c := testConfig(t)
	c.MaxDeliver = 2
	s, err := openSpool(c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	for _, id := range []string{"a", "b", "c"} {
		if _, err = s.enqueue(report(id)); err != nil {
			t.Fatal(err)
		}
	}
	var order []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sample struct{ Collection evidence }
		if err := json.NewDecoder(r.Body).Decode(&sample); err != nil {
			t.Fatal(err)
		}
		order = append(order, sample.Collection.SampleID)
		fmt.Fprintf(w, `{"success":true,"persisted":true,"sample_id":%q}`, sample.Collection.SampleID)
	}))
	defer server.Close()
	c.Endpoint = server.URL
	if err = deliver(context.Background(), c, s, ""); err != nil {
		t.Fatal(err)
	}
	paths, _, _ := s.pending()
	if len(paths) != 1 || len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("unexpected drain order/budget: %v %v", order, paths)
	}
}

func TestReplayProbeDoesNotPostHistoricalData(t *testing.T) {
	c := testConfig(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "OPTIONS" {
			t.Error("probe sent data")
		}
		w.Header().Set("X-Collector-Protocol", "2")
		w.WriteHeader(204)
	}))
	defer server.Close()
	c.Endpoint = server.URL
	if err := supportsReplay(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}

func TestHTTPHasBoundedDeadlineAndDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(time.Second):
				return
			}
		}
		http.Redirect(w, r, "/redirect", 302)
	}))
	defer server.Close()
	client := &http.Client{Timeout: 20 * time.Millisecond, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, path := range []string{"/slow", "/redirect"} {
		if _, err := postReport(context.Background(), client, server.URL+path, report("a")); err == nil {
			t.Fatalf("accepted %s", path)
		}
	}
}

func TestNearFullSpoolDrainsAndRetriesTheSameSample(t *testing.T) {
	c := testConfig(t)
	c.MaxSpoolBytes = int64(len(report("old")) + len(report("new")) - 1)
	s, err := openSpool(c)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	old, err := s.enqueue(report("old"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.checkCapacity(); err != nil {
		t.Fatal("fixture should be below full threshold")
	}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("X-Collector-Protocol", "2")
			w.WriteHeader(204)
			return
		}
		posts++
		fmt.Fprint(w, `{"success":true,"persisted":true,"sample_id":"old"}`)
	}))
	defer server.Close()
	c.Endpoint = server.URL
	path, err := queueSample(context.Background(), c, s, report("new"))
	if err != nil {
		t.Fatal(err)
	}
	if posts != 1 {
		t.Fatal(posts)
	}
	if _, err = os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("old sample not drained")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(report("new")) {
		t.Fatal("retry mutated/lost the current sample")
	}
}

func TestSpoolProfilesAreBoundToTheReceiver(t *testing.T) {
	t.Setenv("HOST", "http://example.invalid/report/sys-collect")
	t.Setenv("COLLECTOR_SPOOL_DIR", t.TempDir())
	a, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOST", "http://other.invalid/report/sys-collect")
	b, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if a.SpoolDir == b.SpoolDir {
		t.Fatal("HOST changes reuse pending reports from another receiver")
	}
}

func TestCancellationAllowsModulesToFinishBoundedCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cleaned := make(chan struct{})
	started := make(chan struct{})
	tasks := []sampleTask{{"cpu", func(ctx context.Context) moduleResult {
		close(started)
		<-ctx.Done()
		time.Sleep(20 * time.Millisecond)
		close(cleaned)
		return moduleResult{name: "cpu", err: ctx.Err()}
	}}}
	go func() { <-started; cancel() }()
	if _, err := collectSnapshot(ctx, tasks, time.Now); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cleaned:
	default:
		t.Fatal("sampler returned before context cleanup completed")
	}
}
