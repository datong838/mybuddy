package embeddedmodels

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hybridgroup/yzma/pkg/download"
)

// windowsCudaVersions lists CUDA runtime versions to try for Windows prebuilt
// llama.cpp binaries, newest first (ggml-org/llama.cpp release asset names).
var windowsCudaVersions = []string{"13.3", "12.4"}

func windowsCudaLlamaZip(version, cudaVer string) string {
	return fmt.Sprintf("llama-%s-bin-win-cuda-%s-x64.zip", version, cudaVer)
}

func windowsCudaCudartZip(cudaVer string) string {
	return fmt.Sprintf("cudart-llama-bin-win-cuda-%s-x64.zip", cudaVer)
}

// getLlamaLibraries downloads llama.cpp prebuilt binaries, routing github.com URLs through a proxy when configured.
// Logic mirrors yzma/pkg/download v1.18.0 GetWithContext with proxy URL rewriting.
func getLlamaLibraries(architecture, operatingSystem, processor, version, dest string, env func(string) string) error {
	return getLlamaLibrariesWithContext(context.Background(), architecture, operatingSystem, processor, version, dest, env)
}

func getLlamaLibrariesWithContext(ctx context.Context, architecture, operatingSystem, processor, version, dest string, env func(string) string) error {
	autoVersion := false
	if version == "" || version == "latest" {
		autoVersion = true
		var err error
		version, err = download.LlamaLatestVersion()
		if err != nil {
			return err
		}
	}

	arch, err := download.ParseArch(architecture)
	if err != nil {
		return download.ErrUnknownArch
	}

	os, err := download.ParseOS(operatingSystem)
	if err != nil {
		return download.ErrUnknownOS
	}

	prcssr, err := download.ParseProcessor(processor)
	if err != nil {
		return download.ErrUnknownProcessor
	}

	if err := download.VersionIsValid(version); err != nil {
		return download.ErrInvalidVersion
	}

	location, filename, err := llamaReleaseURL(arch, os, prcssr, version)
	if err != nil {
		return err
	}

	if operatingSystem == "windows" && processor == "cuda" && arch.Equal(download.AMD64) {
		err := downloadWindowsCudaLlama(ctx, location, version, dest, env)
		if err == nil {
			return nil
		}
		if autoVersion && errors.Is(err, download.ErrFileNotFound) {
			prevVersion, prevErr := download.LlamaPreviousVersion()
			if prevErr != nil {
				return err
			}
			prevLocation := fmt.Sprintf("https://github.com/ggml-org/llama.cpp/releases/download/%s", prevVersion)
			return downloadWindowsCudaLlama(ctx, prevLocation, prevVersion, dest, env)
		}
		return err
	}

	rawURL := fmt.Sprintf("%s/%s", location, filename)
	err = fetchLlamaArtifact(ctx, rawURL, dest, env)
	if err != nil && autoVersion && errors.Is(err, download.ErrFileNotFound) {
		prevVersion, prevErr := download.LlamaPreviousVersion()
		if prevErr != nil {
			return err
		}
		location, filename, prevErr = llamaReleaseURL(arch, os, prcssr, prevVersion)
		if prevErr != nil {
			return err
		}
		rawURL = fmt.Sprintf("%s/%s", location, filename)
		return fetchLlamaArtifact(ctx, rawURL, dest, env)
	}
	return err
}

// llamaReleaseURL returns the GitHub release base URL and artifact filename for llama.cpp binaries.
// Derived from github.com/hybridgroup/yzma v1.18.0 pkg/download/download.go.
func llamaReleaseURL(arch download.Arch, os download.OS, prcssr download.Processor, version string) (location, filename string, err error) {
	location = fmt.Sprintf("https://github.com/ggml-org/llama.cpp/releases/download/%s", version)

	switch os {
	case download.Linux:
		switch prcssr {
		case download.CPU:
			if arch.Equal(download.ARM64) {
				location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-cpu-arm64.tar.gz", version)
				break
			}
			filename = fmt.Sprintf("llama-%s-bin-ubuntu-x64.tar.gz", version)
		case download.CUDA:
			location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
			if arch.Equal(download.ARM64) {
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-cuda-arm64.tar.gz", version)
			} else {
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-cuda-13-x64.tar.gz", version)
			}
		case download.Vulkan:
			if arch.Equal(download.ARM64) {
				location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-vulkan-arm64.tar.gz", version)
				break
			}
			filename = fmt.Sprintf("llama-%s-bin-ubuntu-vulkan-x64.tar.gz", version)
		case download.ROCm:
			if !arch.Equal(download.AMD64) {
				return "", "", errors.New("precompiled binaries for Linux ARM64 ROCm are not available")
			}
			filename = fmt.Sprintf("llama-%s-bin-ubuntu-rocm-7.2-x64.tar.gz", version)
		default:
			return "", "", download.ErrUnknownProcessor
		}

	case download.Bookworm:
		switch prcssr {
		case download.CPU:
			if arch.Equal(download.ARM64) {
				location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-cpu-arm64.tar.gz", version)
				break
			}
			return "", "", download.ErrUnknownProcessor
		case download.CUDA:
			location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
			if arch.Equal(download.ARM64) {
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-cuda-arm64.tar.gz", version)
				break
			}
			return "", "", download.ErrUnknownProcessor
		case download.Vulkan:
			if arch.Equal(download.ARM64) {
				location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-vulkan-arm64.tar.gz", version)
				break
			}
			return "", "", download.ErrUnknownProcessor
		default:
			return "", "", download.ErrUnknownProcessor
		}

	case download.Trixie:
		switch prcssr {
		case download.CPU:
			if arch.Equal(download.ARM64) {
				location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-trixie-cpu-arm64.tar.gz", version)
				break
			}
			filename = fmt.Sprintf("llama-%s-bin-ubuntu-x64.tar.gz", version)
		case download.CUDA:
			location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
			if arch.Equal(download.ARM64) {
				return "", "", download.ErrUnknownProcessor
			}
			filename = fmt.Sprintf("llama-%s-bin-ubuntu-cuda-13-x64.tar.gz", version)
		case download.Vulkan:
			if arch.Equal(download.ARM64) {
				location = fmt.Sprintf("https://github.com/hybridgroup/llama-cpp-builder/releases/download/%s", version)
				filename = fmt.Sprintf("llama-%s-bin-ubuntu-trixie-vulkan-arm64.tar.gz", version)
				break
			}
			filename = fmt.Sprintf("llama-%s-bin-ubuntu-vulkan-x64.tar.gz", version)
		default:
			return "", "", download.ErrUnknownProcessor
		}

	case download.Darwin:
		switch prcssr {
		case download.Metal:
			if !arch.Equal(download.ARM64) {
				return "", "", errors.New("precompiled binaries for macOS non-ARM64 CPU/Metal are not available")
			}
			filename = fmt.Sprintf("llama-%s-bin-macos-arm64.tar.gz", version)
		case download.CPU:
			if arch.Equal(download.ARM64) {
				filename = fmt.Sprintf("llama-%s-bin-macos-arm64.tar.gz", version)
			} else {
				filename = fmt.Sprintf("llama-%s-bin-macos-x64.tar.gz", version)
			}
		default:
			return "", "", download.ErrUnknownProcessor
		}

	case download.Windows:
		switch prcssr {
		case download.CPU:
			if arch.Equal(download.ARM64) {
				filename = fmt.Sprintf("llama-%s-bin-win-cpu-arm64.zip", version)
			} else {
				filename = fmt.Sprintf("llama-%s-bin-win-cpu-x64.zip", version)
			}
		case download.CUDA:
			if arch.Equal(download.ARM64) {
				return "", "", errors.New("precompiled binaries for Windows ARM64 CUDA are not available")
			}
			filename = windowsCudaLlamaZip(version, windowsCudaVersions[0])
		case download.Vulkan:
			if arch.Equal(download.ARM64) {
				return "", "", errors.New("precompiled binaries for Windows ARM64 Vulkan are not available")
			}
			filename = fmt.Sprintf("llama-%s-bin-win-vulkan-x64.zip", version)
		case download.ROCm:
			if !arch.Equal(download.AMD64) {
				return "", "", errors.New("precompiled binaries for Windows ARM64 ROCm are not available")
			}
			filename = fmt.Sprintf("llama-%s-bin-win-hip-radeon-x64.zip", version)
		default:
			return "", "", download.ErrUnknownProcessor
		}

	default:
		return "", "", download.ErrUnknownOS
	}

	return location, filename, nil
}

func downloadWindowsCudaLlama(ctx context.Context, location, version, dest string, env func(string) string) error {
	cudaVer, err := fetchWindowsCudaCudart(ctx, location, dest, env)
	if err != nil {
		return err
	}
	rawURL := fmt.Sprintf("%s/%s", location, windowsCudaLlamaZip(version, cudaVer))
	return fetchLlamaArtifact(ctx, rawURL, dest, env)
}

func fetchWindowsCudaCudart(ctx context.Context, location, dest string, env func(string) string) (string, error) {
	var attempts []string
	for _, cudaVer := range windowsCudaVersions {
		rawURL := fmt.Sprintf("%s/%s", location, windowsCudaCudartZip(cudaVer))
		err := fetchLlamaArtifact(ctx, rawURL, dest, env)
		if err == nil {
			return cudaVer, nil
		}
		attempts = append(attempts, fmt.Sprintf("cuda-%s: %v", cudaVer, err))
		if !errors.Is(err, download.ErrFileNotFound) {
			return "", err
		}
	}
	return "", fmt.Errorf("no windows cudart package found (tried %v): %s", windowsCudaVersions, strings.Join(attempts, "; "))
}

func fetchLlamaArtifact(ctx context.Context, rawURL, dest string, env func(string) string) error {
	candidates := GitHubDownloadURLs(rawURL, env)
	if len(candidates) == 0 {
		return fmt.Errorf("no download URL for %s", rawURL)
	}
	var attempts []string
	for _, url := range candidates {
		err := fetchLlamaArtifactOnce(ctx, url, dest)
		if err == nil {
			return nil
		}
		attempts = append(attempts, fmt.Sprintf("%s: %v", url, err))
		if !isRetryableDownloadErr(err) {
			break
		}
	}
	return fmt.Errorf("github download failed (%d attempts): %s", len(attempts), strings.Join(attempts, "; "))
}

func fetchLlamaArtifactOnce(ctx context.Context, url, dest string) error {
	base := strings.Split(url, "?")[0]
	switch {
	case strings.HasSuffix(base, ".tar.gz"):
		err := extractLlamaTarGz(ctx, url, dest)
		if err != nil && strings.Contains(err.Error(), "404") {
			return fmt.Errorf("%w: %s", download.ErrFileNotFound, url)
		}
		return err
	case strings.HasSuffix(base, ".zip"):
		err := extractLlamaZip(ctx, url, dest)
		if err != nil && strings.Contains(err.Error(), "404") {
			return fmt.Errorf("%w: %s", download.ErrFileNotFound, url)
		}
		return err
	default:
		return fmt.Errorf("unsupported llama artifact: %s", url)
	}
}

func isRetryableDownloadErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "403") ||
		strings.Contains(msg, "429") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "stream error") ||
		strings.Contains(msg, "internal_error") ||
		strings.Contains(msg, "tls:") {
		return true
	}
	return false
}

func httpDownloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "openocta-embedded-models/1.0")

	client := &http.Client{
		Timeout: 2 * time.Hour,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				NextProtos: []string{"http/1.1"},
			},
			ForceAttemptHTTP2: false,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bad response code: %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func extractLlamaTarGz(ctx context.Context, url, dest string) error {
	tmp, err := os.CreateTemp(dest, "llama-archive-*.tar.gz")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	if err := httpDownloadFile(ctx, url, tmpPath); err != nil {
		return err
	}

	resp, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer resp.Close()

	gzr, err := gzip.NewReader(resp)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		name := header.Name
		if idx := strings.Index(name, "/"); idx != -1 {
			name = name[idx+1:]
		}
		if name == "" {
			continue
		}

		target := filepath.Join(dest, filepath.Clean(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil && !os.IsExist(err) {
				return fmt.Errorf("failed to create symlink: %w", err)
			}
		}
	}
	return nil
}

func extractLlamaZip(ctx context.Context, url, dest string) error {
	tmp, err := os.CreateTemp(dest, "llama-archive-*.zip")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	if err := httpDownloadFile(ctx, url, tmpPath); err != nil {
		return err
	}

	reader, err := zip.OpenReader(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		name := file.Name
		if idx := strings.Index(name, "/"); idx != -1 {
			name = name[idx+1:]
		}
		if name == "" {
			continue
		}
		target := filepath.Join(dest, filepath.Clean(name))
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry: %w", err)
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, file.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("failed to write file: %w", err)
		}
		out.Close()
		rc.Close()
	}
	return nil
}
