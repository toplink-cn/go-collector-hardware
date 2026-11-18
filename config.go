package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type config struct {
	Endpoint, SpoolDir          string
	CollectTimeout, HTTPTimeout time.Duration
	MaxPending, MaxDeliver      int
	MaxSpoolBytes               int64
}

func loadConfig() (config, error) {
	executable, err := os.Executable()
	if err != nil {
		return config{}, err
	}
	c := config{Endpoint: os.Getenv("HOST"), SpoolDir: filepath.Join(filepath.Dir(executable), "pending"), CollectTimeout: 25 * time.Second, HTTPTimeout: 5 * time.Second, MaxPending: 1440, MaxSpoolBytes: 64 << 20, MaxDeliver: 3}
	if v := os.Getenv("COLLECTOR_SPOOL_DIR"); v != "" {
		c.SpoolDir = v
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return c, fmt.Errorf("HOST must be an HTTP(S) endpoint without embedded credentials")
	}
	for key, target := range map[string]*time.Duration{"COLLECTOR_SAMPLE_TIMEOUT": &c.CollectTimeout, "COLLECTOR_HTTP_TIMEOUT": &c.HTTPTimeout} {
		if raw := os.Getenv(key); raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil || value < time.Second || value > 30*time.Second {
				return c, fmt.Errorf("%s must be between 1s and 30s", key)
			}
			*target = value
		}
	}
	if raw := os.Getenv("COLLECTOR_MAX_PENDING"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 10080 {
			return c, fmt.Errorf("COLLECTOR_MAX_PENDING must be between 1 and 10080")
		}
		c.MaxPending = value
	}
	fingerprint := sha256.Sum256([]byte(u.String()))
	c.SpoolDir = filepath.Join(c.SpoolDir, hex.EncodeToString(fingerprint[:16]))
	return c, nil
}
