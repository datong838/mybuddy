package appupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/version"
)

const latestVersionHTTPTimeout = 15 * time.Second

type LatestVersionResponse struct {
	Version string `json:"version"`
}

// FetchLatestVersion queries the platform for the newest release tag.
func FetchLatestVersion(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, latestVersionHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, LatestVersionURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("latest-version HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload LatestVersionResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	latest := strings.TrimSpace(payload.Version)
	if latest == "" {
		return "", fmt.Errorf("latest-version response missing version")
	}
	return latest, nil
}

type CheckResult struct {
	CurrentVersion       string            `json:"currentVersion"`
	LatestVersion        string            `json:"latestVersion"`
	HasUpdate            bool              `json:"hasUpdate"`
	Skipped              bool              `json:"skipped"`
	DownloadSupported    bool              `json:"downloadSupported"`
	DownloadURL          string            `json:"downloadUrl,omitempty"`
	AutoInstallSupported bool              `json:"autoInstallSupported"`
	InstallAllowed       bool              `json:"installAllowed"`
	PackageFormat        string            `json:"packageFormat,omitempty"`
	DownloadURLs         map[string]string `json:"downloadUrls,omitempty"`
	ManualInstallHint    string            `json:"manualInstallHint,omitempty"`
	CheckError           string            `json:"checkError,omitempty"`
	LastCheckAt          string            `json:"lastCheckAt,omitempty"`
	SkippedDaily         bool              `json:"skippedDaily,omitempty"`
}

// CheckOptions controls update check behavior.
type CheckOptions struct {
	Force     bool
	Record    bool
	DailyOnly bool
	Now       time.Time
}

// Check performs version comparison and optional config bookkeeping.
func Check(ctx context.Context, opts CheckOptions) CheckResult {
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}

	current := strings.TrimSpace(version.Version)
	out := CheckResult{
		CurrentVersion: current,
	}

	skippedVersions, lastCheckAt, err := loadUpdateSettings()
	if err != nil {
		out.CheckError = err.Error()
		return out
	}
	if lastCheckAt != nil {
		out.LastCheckAt = lastCheckAt.UTC().Format(time.RFC3339)
	}

	if opts.DailyOnly && !opts.Force {
		if !shouldRunDailyCheck(lastCheckAt, opts.Now) {
			out.SkippedDaily = true
			return out
		}
	}

	latest, err := FetchLatestVersion(ctx)
	if err != nil {
		out.CheckError = err.Error()
		return out
	}
	out.LatestVersion = latest

	target := ResolvePlatformTarget()
	out.InstallAllowed = InstallAllowed()
	if target.HasDownload {
		out.DownloadSupported = true
		out.DownloadURL = target.PrimaryURL
		out.AutoInstallSupported = target.AutoInstallSupported && out.InstallAllowed
		out.PackageFormat = string(target.Format)
		out.DownloadURLs = target.AllURLs
		out.ManualInstallHint = target.ManualInstallHint
	}

	newer, err := IsNewer(current, latest)
	if err != nil {
		out.CheckError = err.Error()
		return out
	}
	out.HasUpdate = newer
	out.Skipped = isVersionSkipped(latest, skippedVersions)

	if opts.Record {
		if err := recordLastCheckAt(opts.Now); err != nil && out.CheckError == "" {
			out.CheckError = err.Error()
		} else {
			out.LastCheckAt = opts.Now.UTC().Format(time.RFC3339)
		}
	}

	return out
}

// SkipVersion records a skipped release in openocta.json.
func SkipVersion(version string) error {
	return appendSkippedVersion(version)
}
