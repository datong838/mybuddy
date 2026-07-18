//go:build !linux

package appupdate

func detectLinuxInstallFormat() PackageFormat {
	return FormatTarGz
}
