//go:build linux

package appupdate

import (
	"os"
	"os/exec"
	"strings"
)

func detectLinuxInstallFormat() PackageFormat {
	if installedLinuxPackage("deb") {
		return FormatDEB
	}
	if installedLinuxPackage("rpm") {
		return FormatRPM
	}
	if _, err := exec.LookPath("dpkg"); err == nil {
		if fileExists("/etc/debian_version") || fileExists("/etc/dpkg/origins/debian") {
			return FormatDEB
		}
	}
	if _, err := exec.LookPath("rpm"); err == nil {
		if fileExists("/etc/redhat-release") || fileExists("/etc/fedora-release") {
			return FormatRPM
		}
	}
	if _, err := exec.LookPath("dpkg"); err == nil {
		return FormatDEB
	}
	if _, err := exec.LookPath("rpm"); err == nil {
		return FormatRPM
	}
	return FormatTarGz
}

func installedLinuxPackage(kind string) bool {
	switch kind {
	case "deb":
		if _, err := exec.LookPath("dpkg-query"); err != nil {
			return false
		}
		out, err := exec.Command("dpkg-query", "-W", "-f=${Status}", "openocta").CombinedOutput()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), "install ok installed")
	case "rpm":
		if _, err := exec.LookPath("rpm"); err != nil {
			return false
		}
		err := exec.Command("rpm", "-q", "openocta").Run()
		return err == nil
	default:
		return false
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
