package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildZReleaseRegex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		major     int
		input     string
		wantMatch bool
		wantMinor string
	}{
		{"v4 ci stream", 4, "4.14.0-0.ci", true, "14"},
		{"v4 nightly stream", 4, "4.15.0-0.nightly", true, "15"},
		{"v4 single digit minor", 4, "4.9.0-0.ci", true, "9"},
		{"v5 ci stream", 5, "5.1.0-0.ci", true, "1"},
		{"v5 nightly stream", 5, "5.2.0-0.nightly", true, "2"},
		{"v4 regex does not match v5", 4, "5.1.0-0.ci", false, ""},
		{"v5 regex does not match v4", 5, "4.14.0-0.ci", false, ""},
		{"non-z-stream stable", 4, "stable-4.14", false, ""},
		{"non-z-stream candidate", 4, "candidate-4.14", false, ""},
		{"minor zero rejected", 4, "4.0.0-0.ci", false, ""},
		{"patch release not matched", 4, "4.14.1", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			re := buildZReleaseRegex(tt.major)
			matches := re.FindStringSubmatch(tt.input)
			if tt.wantMatch {
				if matches == nil {
					t.Fatalf("expected match for %q with major=%d, got none", tt.input, tt.major)
				}
				if matches[1] != tt.wantMinor {
					t.Errorf("minor: got %q, want %q", matches[1], tt.wantMinor)
				}
			} else {
				if matches != nil {
					t.Errorf("expected no match for %q with major=%d, got %v", tt.input, tt.major, matches)
				}
			}
		})
	}
}

func TestBuildExtractMinorRegex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		major     int
		input     string
		wantMatch bool
		wantMinor string
	}{
		{"v4 full version", 4, "4.14.3", true, "14"},
		{"v4 patch zero", 4, "4.16.0", true, "16"},
		{"v5 version", 5, "5.2.0", true, "2"},
		{"v4 regex does not match v5", 4, "5.2.0", false, ""},
		{"v5 regex does not match v4", 5, "4.14.3", false, ""},
		{"v4 in payload string", 4, "4.14.0-0.ci-2024-03-15-120000", true, "14"},
		{"minor zero rejected", 4, "4.0.3", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			re := buildExtractMinorRegex(tt.major)
			matches := re.FindStringSubmatch(tt.input)
			if tt.wantMatch {
				if matches == nil {
					t.Fatalf("expected match for %q with major=%d, got none", tt.input, tt.major)
				}
				if matches[1] != tt.wantMinor {
					t.Errorf("minor: got %q, want %q", matches[1], tt.wantMinor)
				}
			} else {
				if matches != nil {
					t.Errorf("expected no match for %q with major=%d, got %v", tt.input, tt.major, matches)
				}
			}
		})
	}
}

func TestGetPayloadTimestamp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		payload   string
		wantErr   bool
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantHour  int
		wantMin   int
		wantSec   int
	}{
		{
			name:      "valid ci payload",
			payload:   "4.14.0-0.ci-2024-03-15-120000",
			wantYear:  2024,
			wantMonth: time.March,
			wantDay:   15,
			wantHour:  12,
			wantMin:   0,
			wantSec:   0,
		},
		{
			name:      "valid nightly payload",
			payload:   "4.15.0-0.nightly-2024-01-02-153045",
			wantYear:  2024,
			wantMonth: time.January,
			wantDay:   2,
			wantHour:  15,
			wantMin:   30,
			wantSec:   45,
		},
		{
			name:      "v5 ci payload",
			payload:   "5.1.0-0.ci-2026-06-01-090000",
			wantYear:  2026,
			wantMonth: time.June,
			wantDay:   1,
			wantHour:  9,
			wantMin:   0,
			wantSec:   0,
		},
		{
			name:    "no date in payload",
			payload: "4.14.0-0.ci",
			wantErr: true,
		},
		{
			name:    "empty string",
			payload: "",
			wantErr: true,
		},
		{
			name:    "malformed date",
			payload: "4.14.0-0.ci-2024-13-40-999999",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts, err := getPayloadTimestamp(tt.payload)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got %v", ts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ts.Year() != tt.wantYear {
				t.Errorf("year: got %d, want %d", ts.Year(), tt.wantYear)
			}
			if ts.Month() != tt.wantMonth {
				t.Errorf("month: got %v, want %v", ts.Month(), tt.wantMonth)
			}
			if ts.Day() != tt.wantDay {
				t.Errorf("day: got %d, want %d", ts.Day(), tt.wantDay)
			}
			if ts.Hour() != tt.wantHour {
				t.Errorf("hour: got %d, want %d", ts.Hour(), tt.wantHour)
			}
			if ts.Minute() != tt.wantMin {
				t.Errorf("minute: got %d, want %d", ts.Minute(), tt.wantMin)
			}
			if ts.Second() != tt.wantSec {
				t.Errorf("second: got %d, want %d", ts.Second(), tt.wantSec)
			}
		})
	}
}

func TestGetEmptyAndStaleStreams(t *testing.T) {
	t.Parallel()
	now := time.Now()
	freshTS := now.Add(-1 * time.Hour).UTC().Format("2006-01-02-150405")
	staleTS := now.Add(-48 * time.Hour).UTC().Format("2006-01-02-150405")

	releases := map[string][]string{
		"4.14.0-0.ci":      {fmt.Sprintf("4.14.0-0.ci-%s", freshTS)},
		"4.15.0-0.nightly": {fmt.Sprintf("4.15.0-0.nightly-%s", staleTS)},
		"4.16.0-0.ci":      {},
		"4.13.0-0.ci":      {fmt.Sprintf("4.13.0-0.ci-%s", freshTS)},
		"4.17.0-0.ci":      {fmt.Sprintf("4.17.0-0.ci-%s", freshTS)},
		"stable-4.14":      {fmt.Sprintf("stable-4.14-%s", freshTS)},
	}

	zRegex := buildZReleaseRegex(4)
	empty, stale := getEmptyAndStaleStreams(releases, 24*time.Hour, 14, 16, "http://test", zRegex)

	// 4.16 is empty
	if _, ok := empty["4.16.0-0.ci"]; !ok {
		t.Error("expected 4.16.0-0.ci to be in empty set")
	}

	// 4.15 is stale (48h old, threshold 24h)
	if _, ok := stale["4.15.0-0.nightly"]; !ok {
		t.Error("expected 4.15.0-0.nightly to be in stale set")
	}

	// 4.14 is fresh — should be in neither
	if _, ok := empty["4.14.0-0.ci"]; ok {
		t.Error("4.14.0-0.ci should not be in empty set")
	}
	if _, ok := stale["4.14.0-0.ci"]; ok {
		t.Error("4.14.0-0.ci should not be in stale set")
	}

	// 4.13 is outside range (below oldestMinor) — should not appear
	if _, ok := empty["4.13.0-0.ci"]; ok {
		t.Error("4.13.0-0.ci should be excluded by oldestMinor filter")
	}
	if _, ok := stale["4.13.0-0.ci"]; ok {
		t.Error("4.13.0-0.ci should be excluded by oldestMinor filter")
	}

	// 4.17 is outside range (above newestMinor) — should not appear
	if _, ok := empty["4.17.0-0.ci"]; ok {
		t.Error("4.17.0-0.ci should be excluded by newestMinor filter")
	}
	if _, ok := stale["4.17.0-0.ci"]; ok {
		t.Error("4.17.0-0.ci should be excluded by newestMinor filter")
	}

	// stable-4.14 is a non-z-stream — should not appear
	if _, ok := empty["stable-4.14"]; ok {
		t.Error("stable-4.14 should be excluded as non-z-stream")
	}
}

func TestGetEmptyAndStaleStreamsV5(t *testing.T) {
	t.Parallel()
	now := time.Now()
	freshTS := now.Add(-1 * time.Hour).UTC().Format("2006-01-02-150405")

	releases := map[string][]string{
		"5.1.0-0.ci":  {fmt.Sprintf("5.1.0-0.ci-%s", freshTS)},
		"4.18.0-0.ci": {fmt.Sprintf("4.18.0-0.ci-%s", freshTS)},
	}

	// v5 regex should only see 5.1, not 4.18
	zRegex := buildZReleaseRegex(5)
	empty, stale := getEmptyAndStaleStreams(releases, 24*time.Hour, 1, 2, "http://test", zRegex)

	if _, ok := empty["4.18.0-0.ci"]; ok {
		t.Error("v5 regex should not match 4.18.0-0.ci")
	}
	if _, ok := stale["4.18.0-0.ci"]; ok {
		t.Error("v5 regex should not match 4.18.0-0.ci")
	}
	if _, ok := empty["5.1.0-0.ci"]; ok {
		t.Error("5.1.0-0.ci should not be empty (it has a payload)")
	}
	if _, ok := stale["5.1.0-0.ci"]; ok {
		t.Error("5.1.0-0.ci should not be stale (it's fresh)")
	}
}

func TestCheckUpgrades(t *testing.T) {
	t.Parallel()
	now := time.Now()
	freshTS := now.Add(-1 * time.Hour).UTC().Format("2006-01-02-150405")

	zRegex := buildZReleaseRegex(4)
	minorRegex := buildExtractMinorRegex(4)

	t.Run("healthy stream with both patch and minor upgrades", func(t *testing.T) {
		t.Parallel()
		payload := fmt.Sprintf("4.15.0-0.ci-%s", freshTS)
		releases := map[string][]string{
			"4.15.0-0.ci": {payload},
		}
		graph := GraphMap{
			payload: {"4.14.5", "4.15.1"},
		}

		rep := checkUpgrades(graph, releases, 24*time.Hour, 14, 16, zRegex, minorRegex)

		if rep.streams["4.15.0-0.ci"] == nil {
			t.Fatal("expected stream 4.15.0-0.ci in report")
		}
		if len(rep.streams["4.15.0-0.ci"].unhealthyMessages) != 0 {
			t.Errorf("expected no unhealthy messages, got: %v", rep.streams["4.15.0-0.ci"].unhealthyMessages)
		}
		if len(rep.streams["4.15.0-0.ci"].healthyMessages) != 2 {
			t.Errorf("expected 2 healthy messages, got %d: %v", len(rep.streams["4.15.0-0.ci"].healthyMessages), rep.streams["4.15.0-0.ci"].healthyMessages)
		}
	})

	t.Run("unhealthy stream missing minor upgrade", func(t *testing.T) {
		t.Parallel()
		payload := fmt.Sprintf("4.15.0-0.ci-%s", freshTS)
		releases := map[string][]string{
			"4.15.0-0.ci": {payload},
		}
		// Only patch upgrade, no minor
		graph := GraphMap{
			payload: {"4.15.1"},
		}

		rep := checkUpgrades(graph, releases, 24*time.Hour, 14, 16, zRegex, minorRegex)

		stream := rep.streams["4.15.0-0.ci"]
		if stream == nil {
			t.Fatal("expected stream 4.15.0-0.ci in report")
		}
		hasMinorWarning := false
		for _, msg := range stream.unhealthyMessages {
			if strings.Contains(msg, "minor level upgrade") {
				hasMinorWarning = true
			}
		}
		if !hasMinorWarning {
			t.Errorf("expected unhealthy message about missing minor upgrade, got: %v", stream.unhealthyMessages)
		}
	})

	t.Run("unhealthy stream missing patch upgrade", func(t *testing.T) {
		t.Parallel()
		payload := fmt.Sprintf("4.15.0-0.ci-%s", freshTS)
		releases := map[string][]string{
			"4.15.0-0.ci": {payload},
		}
		// Only minor upgrade, no patch
		graph := GraphMap{
			payload: {"4.14.5"},
		}

		rep := checkUpgrades(graph, releases, 24*time.Hour, 14, 16, zRegex, minorRegex)

		stream := rep.streams["4.15.0-0.ci"]
		if stream == nil {
			t.Fatal("expected stream 4.15.0-0.ci in report")
		}
		hasPatchWarning := false
		for _, msg := range stream.unhealthyMessages {
			if strings.Contains(msg, "patch level upgrade") {
				hasPatchWarning = true
			}
		}
		if !hasPatchWarning {
			t.Errorf("expected unhealthy message about missing patch upgrade, got: %v", stream.unhealthyMessages)
		}
	})

	t.Run("no upgrade edges at all", func(t *testing.T) {
		t.Parallel()
		payload := fmt.Sprintf("4.15.0-0.ci-%s", freshTS)
		releases := map[string][]string{
			"4.15.0-0.ci": {payload},
		}
		graph := GraphMap{}

		rep := checkUpgrades(graph, releases, 24*time.Hour, 14, 16, zRegex, minorRegex)

		stream := rep.streams["4.15.0-0.ci"]
		if stream == nil {
			t.Fatal("expected stream 4.15.0-0.ci in report")
		}
		if len(stream.unhealthyMessages) != 2 {
			t.Errorf("expected 2 unhealthy messages (no patch or minor), got %d: %v", len(stream.unhealthyMessages), stream.unhealthyMessages)
		}
	})

	t.Run("stream below oldestMinor excluded", func(t *testing.T) {
		t.Parallel()
		payload := fmt.Sprintf("4.13.0-0.ci-%s", freshTS)
		releases := map[string][]string{
			"4.13.0-0.ci": {payload},
		}
		graph := GraphMap{}

		rep := checkUpgrades(graph, releases, 24*time.Hour, 14, 16, zRegex, minorRegex)

		if rep.streams["4.13.0-0.ci"] != nil {
			t.Error("expected 4.13.0-0.ci to be excluded by oldestMinor")
		}
	})

	t.Run("stream above newestMinor excluded", func(t *testing.T) {
		t.Parallel()
		payload := fmt.Sprintf("4.17.0-0.ci-%s", freshTS)
		releases := map[string][]string{
			"4.17.0-0.ci": {payload},
		}
		graph := GraphMap{}

		rep := checkUpgrades(graph, releases, 24*time.Hour, 14, 16, zRegex, minorRegex)

		if rep.streams["4.17.0-0.ci"] != nil {
			t.Error("expected 4.17.0-0.ci to be excluded by newestMinor")
		}
	})

	t.Run("stale payloads only produces no upgrades", func(t *testing.T) {
		t.Parallel()
		staleTS := now.Add(-48 * time.Hour).UTC().Format("2006-01-02-150405")
		payload := fmt.Sprintf("4.15.0-0.ci-%s", staleTS)
		releases := map[string][]string{
			"4.15.0-0.ci": {payload},
		}
		graph := GraphMap{
			payload: {"4.14.5", "4.15.1"},
		}

		rep := checkUpgrades(graph, releases, 24*time.Hour, 14, 16, zRegex, minorRegex)

		stream := rep.streams["4.15.0-0.ci"]
		if stream == nil {
			t.Fatal("expected stream 4.15.0-0.ci in report")
		}
		if len(stream.unhealthyMessages) != 2 {
			t.Errorf("expected 2 unhealthy messages (no recent patch or minor), got %d: %v", len(stream.unhealthyMessages), stream.unhealthyMessages)
		}
	})
}

func TestReportString(t *testing.T) {
	t.Parallel()

	t.Run("unhealthy only", func(t *testing.T) {
		t.Parallel()
		rep := &report{
			streams: map[string]*releaseReport{
				"4.14.0-0.ci": {unhealthyMessages: []string{"test warning"}},
				"4.15.0-0.ci": {healthyMessages: []string{"all good"}},
			},
			majorVersion:  4,
			oldestMinor:   14,
			newestMinor:   15,
			releaseAPIUrl: "https://test.example.com",
		}

		output := rep.String(false)

		if !strings.Contains(output, "test warning") {
			t.Error("expected unhealthy message in output")
		}
		if strings.Contains(output, "all good") {
			t.Error("healthy messages should not appear when includeHealthy=false")
		}
		if !strings.Contains(output, "4.14.z") {
			t.Errorf("expected '4.14.z' in footer, got: %s", output)
		}
		if !strings.Contains(output, "4.15.z") {
			t.Errorf("expected '4.15.z' in footer, got: %s", output)
		}
	})

	t.Run("include healthy", func(t *testing.T) {
		t.Parallel()
		rep := &report{
			streams: map[string]*releaseReport{
				"4.14.0-0.ci": {unhealthyMessages: []string{"test warning"}},
				"4.15.0-0.ci": {healthyMessages: []string{"all good"}},
			},
			majorVersion:  4,
			oldestMinor:   14,
			newestMinor:   15,
			releaseAPIUrl: "https://test.example.com",
		}

		output := rep.String(true)

		if !strings.Contains(output, "test warning") {
			t.Error("expected unhealthy message in output")
		}
		if !strings.Contains(output, "all good") {
			t.Error("expected healthy message when includeHealthy=true")
		}
		if !strings.Contains(output, "*WARNING:*") {
			t.Error("expected WARNING prefix on unhealthy messages when includeHealthy=true")
		}
	})

	t.Run("v5 footer formatting", func(t *testing.T) {
		t.Parallel()
		rep := &report{
			streams: map[string]*releaseReport{
				"5.1.0-0.ci": {healthyMessages: []string{"ok"}},
			},
			majorVersion:  5,
			oldestMinor:   1,
			newestMinor:   2,
			releaseAPIUrl: "https://test.example.com",
		}

		output := rep.String(true)

		if !strings.Contains(output, "5.1.z") {
			t.Errorf("expected '5.1.z' in footer, got: %s", output)
		}
		if !strings.Contains(output, "5.2.z") {
			t.Errorf("expected '5.2.z' in footer, got: %s", output)
		}
		if strings.Contains(output, "4.") {
			t.Error("v5 report should not contain '4.' references")
		}
	})

	t.Run("no unhealthy streams", func(t *testing.T) {
		t.Parallel()
		rep := &report{
			streams: map[string]*releaseReport{
				"4.15.0-0.ci": {healthyMessages: []string{"all good"}},
			},
			majorVersion:  4,
			oldestMinor:   14,
			newestMinor:   15,
			releaseAPIUrl: "https://test.example.com",
		}

		output := rep.String(false)

		if !strings.Contains(output, "No unhealthy payload streams detected") {
			t.Error("expected 'No unhealthy payload streams detected' message")
		}
	})

	t.Run("streams sorted newest first", func(t *testing.T) {
		t.Parallel()
		rep := &report{
			streams: map[string]*releaseReport{
				"4.14.0-0.ci": {unhealthyMessages: []string{"warning 14"}},
				"4.16.0-0.ci": {unhealthyMessages: []string{"warning 16"}},
				"4.15.0-0.ci": {unhealthyMessages: []string{"warning 15"}},
			},
			majorVersion:  4,
			oldestMinor:   14,
			newestMinor:   16,
			releaseAPIUrl: "https://test.example.com",
		}

		output := rep.String(false)

		idx16 := strings.Index(output, "warning 16")
		idx15 := strings.Index(output, "warning 15")
		idx14 := strings.Index(output, "warning 14")
		if idx16 > idx15 || idx15 > idx14 {
			t.Errorf("streams not sorted newest-first: 16@%d 15@%d 14@%d", idx16, idx15, idx14)
		}
	})
}
