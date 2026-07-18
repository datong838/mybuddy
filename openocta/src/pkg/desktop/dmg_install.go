//go:build !darwin

package desktop

// MaybePromptInstallFromDMG is a no-op on non-macOS platforms.
func MaybePromptInstallFromDMG() {}
