//go:build !windows

package embeddedmodels

func detectPlatformRAMGb() float64 {
	return 0
}
