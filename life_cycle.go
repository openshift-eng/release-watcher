package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"k8s.io/klog"
)

type productLifeCycleResponse struct {
	Data []productLifeCycle `json:"data"`
}

type productLifeCycle struct {
	Name     string                    `json:"name"`
	Versions []productLifeCycleVersion `json:"versions"`
}

type productLifeCycleVersion struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func getSupportedReleases(url string, majorVersion int) (int, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("error fetching life-cycle data from %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, 0, fmt.Errorf("non-OK http response code from %s: %d", url, resp.StatusCode)
	}

	data := productLifeCycleResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, fmt.Errorf("error decoding life-cycle data from %s: %w", url, err)
	}

	return parseSupportedReleases(data, majorVersion)
}

func parseSupportedReleases(data productLifeCycleResponse, majorVersion int) (int, int, error) {
	minSupportedRelease := -1
	maxSupportedRelease := -1
	for _, product := range data.Data {
		for _, version := range product.Versions {
			if version.Type == "End of life" {
				continue
			}
			entries := strings.Split(version.Name, ".")
			if len(entries) != 2 {
				klog.V(4).Infof("expected one period in %q for parsing a minor version", version.Name)
				continue
			}
			major, err := strconv.Atoi(entries[0])
			if err != nil || major != majorVersion {
				klog.V(4).Infof("expected major version %d in %q, not %q", majorVersion, version.Name, entries[0])
				continue
			}

			minor, err := strconv.Atoi(entries[1])
			if err != nil {
				klog.V(4).Infof("expected integer minor version in %q, not %q: %v", version.Name, entries[1], err)
				continue
			}

			if minSupportedRelease == -1 || minSupportedRelease > minor {
				minSupportedRelease = minor
			}
			if maxSupportedRelease == -1 || maxSupportedRelease < minor {
				maxSupportedRelease = minor
			}
		}
	}

	if minSupportedRelease == -1 {
		return 0, 0, fmt.Errorf("life-cycle data contains no supported releases for major version %d", majorVersion)
	}

	return minSupportedRelease, maxSupportedRelease, nil
}
