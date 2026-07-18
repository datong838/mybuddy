package appupdate

import (
	"fmt"
	"runtime"
)

const pkgCDNBase = "https://openocta.com/pkg"

// LatestVersionURL is the platform API for newest release tag.
const LatestVersionURL = "https://openocta.com/api/v1/pkg/latest-version"

// PackageFormat identifies the install artifact type.
type PackageFormat string

const (
	FormatDMG   PackageFormat = "dmg"
	FormatEXE   PackageFormat = "exe"
	FormatDEB   PackageFormat = "deb"
	FormatRPM   PackageFormat = "rpm"
	FormatTarGz PackageFormat = "tar.gz"
)

// PlatformTarget describes download and install options for the current OS/arch.
type PlatformTarget struct {
	HasDownload          bool
	PrimaryURL           string
	AutoInstallURL       string
	AutoInstallSupported bool
	Format               PackageFormat
	AllURLs              map[string]string
	ManualInstallHint    string
}

func linuxArchSuffix() string {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return runtime.GOARCH
	default:
		return ""
	}
}

func linuxPackageURLs(arch string) map[string]string {
	if arch == "" {
		return nil
	}
	return map[string]string{
		string(FormatDEB):   fmt.Sprintf("%s/openocta_linux_%s.deb", pkgCDNBase, arch),
		string(FormatRPM):   fmt.Sprintf("%s/openocta_linux_%s.rpm", pkgCDNBase, arch),
		string(FormatTarGz): fmt.Sprintf("%s/openocta_linux_%s.tar.gz", pkgCDNBase, arch),
	}
}

// ResolvePlatformTarget returns download/install metadata for the current machine.
func ResolvePlatformTarget() PlatformTarget {
	switch runtime.GOOS {
	case "darwin":
		url := ""
		switch runtime.GOARCH {
		case "arm64":
			url = pkgCDNBase + "/OpenOcta-arm64.dmg"
		case "amd64":
			url = pkgCDNBase + "/OpenOcta-amd64.dmg"
		}
		if url == "" {
			return PlatformTarget{}
		}
		return PlatformTarget{
			HasDownload:          true,
			PrimaryURL:           url,
			AutoInstallURL:       url,
			AutoInstallSupported: true,
			Format:               FormatDMG,
			AllURLs:              map[string]string{string(FormatDMG): url},
		}
	case "windows":
		if runtime.GOARCH != "amd64" {
			return PlatformTarget{}
		}
		url := pkgCDNBase + "/OpenOcta-amd64-installer.exe"
		return PlatformTarget{
			HasDownload:          true,
			PrimaryURL:           url,
			AutoInstallURL:       url,
			AutoInstallSupported: true,
			Format:               FormatEXE,
			AllURLs:              map[string]string{string(FormatEXE): url},
		}
	case "linux":
		arch := linuxArchSuffix()
		urls := linuxPackageURLs(arch)
		if len(urls) == 0 {
			return PlatformTarget{}
		}
		format := detectLinuxInstallFormat()
		autoURL := ""
		autoOK := false
		switch format {
		case FormatDEB:
			autoURL = urls[string(FormatDEB)]
			autoOK = autoURL != ""
		case FormatRPM:
			autoURL = urls[string(FormatRPM)]
			autoOK = autoURL != ""
		default:
			format = FormatTarGz
		}
		primary := autoURL
		if primary == "" {
			primary = urls[string(FormatDEB)]
		}
		if primary == "" {
			primary = urls[string(FormatTarGz)]
		}
		return PlatformTarget{
			HasDownload:          true,
			PrimaryURL:           primary,
			AutoInstallURL:       autoURL,
			AutoInstallSupported: autoOK,
			Format:               format,
			AllURLs:              urls,
			ManualInstallHint:    linuxManualInstallHint(urls),
		}
	default:
		return PlatformTarget{}
	}
}

// DownloadURL returns the preferred auto-install URL, if any.
func DownloadURL() (url string, supported bool) {
	t := ResolvePlatformTarget()
	if !t.HasDownload {
		return "", false
	}
	if t.AutoInstallURL != "" {
		return t.AutoInstallURL, true
	}
	return t.PrimaryURL, t.HasDownload
}
