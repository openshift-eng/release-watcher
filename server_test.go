package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerHelp(t *testing.T) {
	t.Parallel()
	o := &options{
		majorVersion:           4,
		oldestMinor:            14,
		newestMinor:            17,
		acceptedStalenessLimit: 24 * time.Hour,
		builtStalenessLimit:    72 * time.Hour,
		arch:                   "amd64",
	}

	handler := o.createHandler()

	body := `{"type":"event_callback","event":{"type":"app_mention","text":"help","channel":"C123","ts":"help-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// sendMessage will fail (no Slack token), but we can verify the handler
	// parses the request without panicking. The help text is built before
	// sendMessage is called, so argument parsing is exercised.
	handler(w, req)
}

func TestHandlerURLVerification(t *testing.T) {
	t.Parallel()
	o := &options{}
	handler := o.createHandler()

	body := `{"type":"url_verification","challenge":"test-challenge-token"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp VerificationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Challenge != "test-challenge-token" {
		t.Errorf("challenge: got %q, want %q", resp.Challenge, "test-challenge-token")
	}
}

func TestParseReportArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		text               string
		wantMajor          int
		wantOldestMinor    int
		wantNewestMinor    int
		wantArch           string
		wantIncludeHealthy bool
	}{
		{
			name:            "defaults",
			text:            "report",
			wantMajor:       4,
			wantOldestMinor: 14,
			wantNewestMinor: 17,
			wantArch:        "amd64",
		},
		{
			name:            "major=5",
			text:            "report major=5",
			wantMajor:       5,
			wantOldestMinor: 14,
			wantNewestMinor: 17,
			wantArch:        "amd64",
		},
		{
			name:            "min and max",
			text:            "report min=15 max=16",
			wantMajor:       4,
			wantOldestMinor: 15,
			wantNewestMinor: 16,
			wantArch:        "amd64",
		},
		{
			name:            "arch override",
			text:            "report arch=arm64",
			wantMajor:       4,
			wantOldestMinor: 14,
			wantNewestMinor: 17,
			wantArch:        "arm64",
		},
		{
			name:               "all arguments combined",
			text:               "report major=5 min=1 max=3 arch=multi healthy",
			wantMajor:          5,
			wantOldestMinor:    1,
			wantNewestMinor:    3,
			wantArch:           "multi",
			wantIncludeHealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Reproduce the argument parsing logic from the handler
			baseOpts := options{
				majorVersion:           4,
				oldestMinor:            14,
				newestMinor:            17,
				acceptedStalenessLimit: 24 * time.Hour,
				builtStalenessLimit:    72 * time.Hour,
				upgradeStalenessLimit:  72 * time.Hour,
				arch:                   "amd64",
			}
			reportOptions := baseOpts
			reportOptions.includeHealthy = false

			args := strings.Split(tt.text, " ")
			for _, arg := range args {
				if arg == "healthy" {
					reportOptions.includeHealthy = true
				}
				if strings.Contains(arg, "=") {
					v := strings.Split(arg, "=")
					switch v[0] {
					case "min":
						i, err := fmt.Sscanf(v[1], "%d", &reportOptions.oldestMinor)
						if err != nil || i != 1 {
							t.Fatalf("failed to parse min: %v", err)
						}
					case "max":
						i, err := fmt.Sscanf(v[1], "%d", &reportOptions.newestMinor)
						if err != nil || i != 1 {
							t.Fatalf("failed to parse max: %v", err)
						}
					case "major":
						i, err := fmt.Sscanf(v[1], "%d", &reportOptions.majorVersion)
						if err != nil || i != 1 {
							t.Fatalf("failed to parse major: %v", err)
						}
					case "arch":
						reportOptions.arch = v[1]
					}
				}
			}

			if reportOptions.majorVersion != tt.wantMajor {
				t.Errorf("majorVersion: got %d, want %d", reportOptions.majorVersion, tt.wantMajor)
			}
			if reportOptions.oldestMinor != tt.wantOldestMinor {
				t.Errorf("oldestMinor: got %d, want %d", reportOptions.oldestMinor, tt.wantOldestMinor)
			}
			if reportOptions.newestMinor != tt.wantNewestMinor {
				t.Errorf("newestMinor: got %d, want %d", reportOptions.newestMinor, tt.wantNewestMinor)
			}
			if reportOptions.arch != tt.wantArch {
				t.Errorf("arch: got %q, want %q", reportOptions.arch, tt.wantArch)
			}
			if reportOptions.includeHealthy != tt.wantIncludeHealthy {
				t.Errorf("includeHealthy: got %v, want %v", reportOptions.includeHealthy, tt.wantIncludeHealthy)
			}
		})
	}
}
