package main

import (
	"collector/modules/collection"
	"collector/modules/cpu"
	"collector/modules/disk"
	"collector/modules/ipmi"
	"collector/modules/memory"
	"collector/modules/network"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var version = "2-dev"

type evidence struct {
	Version          int                           `json:"version"`
	CollectorVersion string                        `json:"collector_version"`
	SampleID         string                        `json:"sample_id"`
	StartedAt        int64                         `json:"started_at"`
	CollectedAt      int64                         `json:"collected_at"`
	Status           string                        `json:"status"`
	Quality          map[string]collection.Quality `json:"quality"`
	Units            map[string]string             `json:"units"`
	MemoryFreeKind   string                        `json:"memory_free_kind"`
}
type moduleResult struct {
	name        string
	data        interface{}
	quality     map[string]collection.Quality
	err         error
	collectedAt int64
}
type sampleTask struct {
	name    string
	collect func(context.Context) moduleResult
}

func defaultTasks() []sampleTask {
	return []sampleTask{
		{"cpu", func(ctx context.Context) moduleResult {
			r, e := cpu.Collect(ctx)
			return moduleResult{name: "cpu", data: r.Data, quality: r.Quality, err: e}
		}},
		{"memory", func(ctx context.Context) moduleResult {
			r, e := memory.Collect(ctx)
			return moduleResult{name: "memory", data: r.Data, quality: r.Quality, err: e}
		}},
		{"disks", func(ctx context.Context) moduleResult {
			r, e := disk.Collect(ctx)
			return moduleResult{name: "disks", data: r.Data, quality: r.Quality, err: e}
		}},
		{"network", func(ctx context.Context) moduleResult {
			r, e := network.Collect(ctx)
			return moduleResult{name: "network", data: r.Data, quality: r.Quality, err: e}
		}},
		{"ipmi", func(ctx context.Context) moduleResult {
			r, e := ipmi.Collect(ctx)
			return moduleResult{name: "ipmi", data: r.Data, quality: r.Quality, err: e}
		}},
	}
}

// Each module is independent. A slow sensor cannot prevent the basic metrics
// from becoming a report, and every report keeps its original collection time.
func collectSnapshot(ctx context.Context, tasks []sampleTask, now func() time.Time) ([]byte, error) {
	start := now()
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	meta := evidence{Version: 2, CollectorVersion: version, SampleID: hex.EncodeToString(id[:]), StartedAt: start.Unix(), Quality: map[string]collection.Quality{}, Units: map[string]string{"cpu": "percent", "memory": "bytes", "disk": "bytes", "network": "bytes", "temperature": "celsius"}, MemoryFreeKind: "available"}
	payload := map[string]interface{}{}
	results := make(chan moduleResult, len(tasks))
	pending := map[string]bool{}
	for _, task := range tasks {
		pending[task.name] = true
		task := task
		go func() {
			r := moduleResult{name: task.name}
			defer func() {
				if p := recover(); p != nil {
					r = moduleResult{name: task.name, err: fmt.Errorf("module panic: %v", p)}
				}
				r.collectedAt = now().Unix()
				results <- r
			}()
			r = task.collect(ctx)
		}()
	}
	for len(pending) > 0 {
		select {
		case r := <-results:
			delete(pending, r.name)
			for name, q := range r.quality {
				if q.CollectedAt == 0 && (q.State == collection.Success || q.State == collection.Partial) {
					q.CollectedAt = r.collectedAt
				}
				meta.Quality[name] = q
			}
			if _, ok := meta.Quality[r.name]; !ok {
				state, reason := "success", ""
				if r.err != nil {
					state = "failed"
					reason = r.err.Error()
				}
				q := collection.Quality{State: state, Reason: reason}
				if state == collection.Success || state == collection.Partial {
					q.CollectedAt = r.collectedAt
				}
				meta.Quality[r.name] = q
			}
			if r.data != nil {
				field := map[string]string{"cpu": "Cpus", "memory": "Memory", "disks": "Disks", "network": "Network", "ipmi": "IpmiSensors"}[r.name]
				if field != "" {
					payload[field] = r.data
				}
			}
		case <-ctx.Done():
			for name := range pending {
				meta.Quality[name] = collection.Quality{State: "failed", Reason: "collection deadline exceeded"}
			}
			// Allow context-aware command runners a bounded interval to kill/reap
			// children before main exits. Non-cooperative modules cannot stall shutdown.
			cleanup := time.NewTimer(500 * time.Millisecond)
		drain:
			for len(pending) > 0 {
				select {
				case result := <-results:
					delete(pending, result.name)
				case <-cleanup.C:
					break drain
				}
			}
			cleanup.Stop()
			pending = map[string]bool{}
		}
	}
	meta.CollectedAt = now().Unix()
	meta.Status = sampleStatus(meta.Quality)
	payload["collection"] = meta
	return json.Marshal(payload)
}

func sampleStatus(quality map[string]collection.Quality) string {
	problems, observed := false, false
	for name, q := range quality {
		if q.State == "success" || q.State == "partial" {
			observed = true
		}
		optional := name == "temperature" || name == "ipmi"
		if q.State != "success" && !(optional && q.State == "unsupported") {
			problems = true
		}
	}
	if !observed {
		return "failed"
	}
	if problems {
		return "partial"
	}
	return "success"
}

func run(ctx context.Context) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()
	store, err := openSpool(c)
	if err != nil {
		return err
	}
	defer store.close()
	if err := store.checkCapacity(); err != nil {
		if probeErr := supportsReplay(ctx, c); probeErr != nil {
			return fmt.Errorf("%v; %w", err, probeErr)
		}
		if err := deliver(ctx, c, store, ""); err != nil {
			return err
		}
		if err := store.checkCapacity(); err != nil {
			return err
		}
	}
	sampleCtx, finish := context.WithTimeout(ctx, c.CollectTimeout)
	data, err := collectSnapshot(sampleCtx, defaultTasks(), time.Now)
	finish()
	if err != nil {
		return err
	}
	path, err := queueSample(ctx, c, store, data)
	if err != nil {
		return err
	}
	return deliver(ctx, c, store, path)
}

func queueSample(ctx context.Context, c config, store *spool, data []byte) (string, error) {
	path, err := store.enqueue(data)
	if !errors.Is(err, errSpoolFull) {
		return path, err
	}
	if probeErr := supportsReplay(ctx, c); probeErr != nil {
		return "", fmt.Errorf("%v; %w", err, probeErr)
	}
	if err := deliver(ctx, c, store, ""); err != nil {
		return "", err
	}
	return store.enqueue(data)
}
