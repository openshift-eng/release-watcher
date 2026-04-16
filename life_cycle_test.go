package main

import (
	"testing"
)

func TestParseSupportedReleases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		data         productLifeCycleResponse
		majorVersion int
		wantMin      int
		wantMax      int
		wantErr      bool
	}{
		{
			name: "typical v4 response with mix of supported and EOL",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{{
					Name: "OpenShift Container Platform",
					Versions: []productLifeCycleVersion{
						{Name: "4.17", Type: "Full support"},
						{Name: "4.16", Type: "Full support"},
						{Name: "4.15", Type: "Maintenance support"},
						{Name: "4.14", Type: "Maintenance support"},
						{Name: "4.13", Type: "End of life"},
						{Name: "4.12", Type: "End of life"},
					},
				}},
			},
			majorVersion: 4,
			wantMin:      14,
			wantMax:      17,
		},
		{
			name: "v5 entries present alongside v4",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{{
					Name: "OpenShift Container Platform",
					Versions: []productLifeCycleVersion{
						{Name: "5.2", Type: "Full support"},
						{Name: "5.1", Type: "Full support"},
						{Name: "4.18", Type: "Full support"},
						{Name: "4.17", Type: "Maintenance support"},
					},
				}},
			},
			majorVersion: 5,
			wantMin:      1,
			wantMax:      2,
		},
		{
			name: "query v4 ignores v5 entries",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{{
					Name: "OpenShift Container Platform",
					Versions: []productLifeCycleVersion{
						{Name: "5.1", Type: "Full support"},
						{Name: "4.18", Type: "Full support"},
						{Name: "4.17", Type: "Maintenance support"},
					},
				}},
			},
			majorVersion: 4,
			wantMin:      17,
			wantMax:      18,
		},
		{
			name: "multiple products in response",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{
					{
						Name: "OpenShift Container Platform",
						Versions: []productLifeCycleVersion{
							{Name: "4.17", Type: "Full support"},
							{Name: "4.16", Type: "Full support"},
						},
					},
					{
						Name: "OpenShift Container Platform 3",
						Versions: []productLifeCycleVersion{
							{Name: "3.11", Type: "End of life"},
						},
					},
				},
			},
			majorVersion: 4,
			wantMin:      16,
			wantMax:      17,
		},
		{
			name: "no supported releases for requested major version",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{{
					Name: "OpenShift Container Platform",
					Versions: []productLifeCycleVersion{
						{Name: "4.16", Type: "End of life"},
						{Name: "4.15", Type: "End of life"},
					},
				}},
			},
			majorVersion: 4,
			wantErr:      true,
		},
		{
			name: "no releases for requested major at all",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{{
					Name: "OpenShift Container Platform",
					Versions: []productLifeCycleVersion{
						{Name: "4.17", Type: "Full support"},
					},
				}},
			},
			majorVersion: 5,
			wantErr:      true,
		},
		{
			name: "empty versions list",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{{
					Name: "OpenShift Container Platform",
				}},
			},
			majorVersion: 4,
			wantErr:      true,
		},
		{
			name: "empty products list",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{},
			},
			majorVersion: 4,
			wantErr:      true,
		},
		{
			name: "malformed version names are skipped gracefully",
			data: productLifeCycleResponse{
				Data: []productLifeCycle{{
					Name: "OpenShift Container Platform",
					Versions: []productLifeCycleVersion{
						{Name: "not-a-version", Type: "Full support"},
						{Name: "4", Type: "Full support"},
						{Name: "4.x", Type: "Full support"},
						{Name: "4.16", Type: "Full support"},
					},
				}},
			},
			majorVersion: 4,
			wantMin:      16,
			wantMax:      16,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotMin, gotMax, err := parseSupportedReleases(tt.data, tt.majorVersion)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got min=%d max=%d", gotMin, gotMax)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotMin != tt.wantMin {
				t.Errorf("min: got %d, want %d", gotMin, tt.wantMin)
			}
			if gotMax != tt.wantMax {
				t.Errorf("max: got %d, want %d", gotMax, tt.wantMax)
			}
		})
	}
}
