package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var errSpoolFull = errors.New("pending reports are full; existing reports retained")

type spool struct {
	cfg  config
	lock *os.File
}

func openSpool(c config) (*spool, error) {
	if err := os.MkdirAll(c.SpoolDir, 0700); err != nil {
		return nil, err
	}
	f, e := lockSpool(filepath.Join(c.SpoolDir, ".lock"))
	if e != nil {
		return nil, e
	}
	return &spool{c, f}, nil
}
func (s *spool) close() { unlockSpool(s.lock) }
func (s *spool) pending() ([]string, int64, error) {
	entries, err := os.ReadDir(s.cfg.SpoolDir)
	if err != nil {
		return nil, 0, err
	}
	var paths []string
	var size int64
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, 0, err
		}
		if !info.Mode().IsRegular() {
			return nil, 0, fmt.Errorf("non-regular pending report: %s", entry.Name())
		}
		paths = append(paths, filepath.Join(s.cfg.SpoolDir, entry.Name()))
		size += info.Size()
	}
	sort.Strings(paths)
	return paths, size, nil
}
func (s *spool) checkCapacity() error {
	paths, size, e := s.pending()
	if e != nil {
		return e
	}
	if len(paths) >= s.cfg.MaxPending || size >= s.cfg.MaxSpoolBytes {
		return errSpoolFull
	}
	return nil
}
func (s *spool) enqueue(data []byte) (string, error) {
	paths, size, e := s.pending()
	if e != nil {
		return "", e
	}
	if len(paths) >= s.cfg.MaxPending || size+int64(len(data)) > s.cfg.MaxSpoolBytes {
		return "", errSpoolFull
	}
	hash := sha256.Sum256(data)
	name := fmt.Sprintf("%020d-%s.json", time.Now().UnixNano(), hex.EncodeToString(hash[:8]))
	f, e := os.CreateTemp(s.cfg.SpoolDir, ".pending-")
	if e != nil {
		return "", e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e != nil {
		f.Close()
		return "", e
	}
	if _, e = f.Write(data); e != nil {
		f.Close()
		return "", e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return "", e
	}
	if e = f.Close(); e != nil {
		return "", e
	}
	path := filepath.Join(s.cfg.SpoolDir, name)
	if e = os.Rename(f.Name(), path); e != nil {
		return "", e
	}
	if e = syncSpoolDir(s.cfg.SpoolDir); e != nil {
		return "", e
	}
	return path, nil
}
