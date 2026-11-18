package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

// A modern receiver must echo the immutable report identity after durable commit.
// Plain OK is retained only for sending this invocation's current report to an
// old receiver: it is not sufficient evidence to drain historical reports.
func postReport(ctx context.Context, client *http.Client, endpoint string, data []byte) (bool, error) {
	request, e := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if e != nil {
		return false, e
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "dcim-collector/"+version)
	response, e := client.Do(request)
	if e != nil {
		return false, e
	}
	defer response.Body.Close()
	body, e := io.ReadAll(io.LimitReader(response.Body, 4097))
	if e != nil {
		return false, e
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false, fmt.Errorf("receiver returned HTTP %d", response.StatusCode)
	}
	if len(body) > 4096 {
		return false, errors.New("receiver acknowledgement is too large")
	}
	if string(bytes.TrimSpace(body)) == "OK" {
		return false, nil
	}
	var ack struct {
		Success   bool   `json:"success"`
		Persisted bool   `json:"persisted"`
		SampleID  string `json:"sample_id"`
	}
	var sample struct {
		Collection struct {
			SampleID string `json:"sample_id"`
		} `json:"collection"`
	}
	if json.Unmarshal(body, &ack) != nil || json.Unmarshal(data, &sample) != nil || !ack.Success || !ack.Persisted || sample.Collection.SampleID == "" || ack.SampleID != sample.Collection.SampleID {
		return false, errors.New("receiver did not durably acknowledge this sample")
	}
	return true, nil
}
func deliver(ctx context.Context, c config, store *spool, newest string) error {
	client := &http.Client{Timeout: c.HTTPTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	paths, _, e := store.pending()
	if e != nil {
		return e
	}
	order := []string{}
	if newest != "" {
		order = append(order, newest)
	}
	for _, p := range paths {
		if p != newest {
			order = append(order, p)
		}
	}
	for index, p := range order {
		if index >= c.MaxDeliver {
			break
		}
		data, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		modern, e := postReport(ctx, client, c.Endpoint, data)
		if e != nil {
			return e
		}
		if !modern && p != newest {
			return errors.New("receiver did not durably acknowledge historical sample; report retained")
		}
		if e = os.Remove(p); e != nil {
			return e
		}
		if !modern {
			if len(paths) > 1 {
				return errors.New("legacy receiver accepted current sample; historical reports retained until receiver upgrade")
			}
			break
		}
	}
	return syncSpoolDir(c.SpoolDir)
}

// OPTIONS cannot turn an old sample into a current legacy observation.
func supportsReplay(ctx context.Context, c config) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodOptions, c.Endpoint, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: c.HTTPTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 || response.Header.Get("X-Collector-Protocol") != "2" {
		return errors.New("receiver does not advertise durable sample replay")
	}
	return nil
}
