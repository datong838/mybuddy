package embeddedmodels

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const hardwareDetectCacheTTL = 60 * time.Second

var (
	hardwareDetectMu    sync.Mutex
	hardwareDetectCache HardwareProfile
	hardwareDetectAt    time.Time
)

// HardwareProfile describes the machine running embedded models (Gateway host).
type HardwareProfile struct {
	GPUName        string  `json:"gpuName"`
	VramGb         float64 `json:"vramGb"`
	RamGb          float64 `json:"ramGb"`
	CPUCores       int     `json:"cpuCores"`
	BandwidthGbs   float64 `json:"bandwidthGbs"`
	IsAppleSilicon bool    `json:"isAppleSilicon"`
	Detected       bool    `json:"detected"`
	ServerSide     bool    `json:"serverSide"`
}

type gpuSpec struct {
	vram float64
	bw   float64
}

var gpuDB = map[string]gpuSpec{
	"4090":     {24, 1008},
	"4080":     {16, 717},
	"4070 ti":  {12, 504},
	"4070":     {12, 504},
	"4060 ti":  {16, 272},
	"4060":     {8, 272},
	"3090":     {24, 936},
	"3080":     {10, 760},
	"3070":     {8, 448},
	"3060":     {12, 360},
	"7900 xtx": {24, 960},
	"7900 xt":  {20, 800},
	"7800 xt":  {16, 624},
	"7700 xt":  {12, 432},
	"7600":     {8, 288},
	"arc a770": {16, 560},
	"arc a750": {8, 512},
}

var appleDB = map[string]gpuSpec{
	"m4 max": {36, 546},
	"m4 pro": {24, 273},
	"m4":     {16, 120},
	"m3 max": {36, 400},
	"m3 pro": {18, 200},
	"m3":     {16, 100},
	"m2 max": {32, 400},
	"m2 pro": {16, 200},
	"m2":     {8, 100},
	"m1 max": {32, 400},
	"m1 pro": {16, 200},
	"m1":     {8, 68},
}

func lookupGpuSpec(gpuName string) (gpuSpec, bool, bool) {
	lower := strings.ToLower(gpuName)
	if strings.Contains(lower, "apple") || strings.Contains(lower, "m1") ||
		strings.Contains(lower, "m2") || strings.Contains(lower, "m3") || strings.Contains(lower, "m4") {
		for key, spec := range appleDB {
			compact := strings.ReplaceAll(key, " ", "")
			if strings.Contains(lower, compact) || strings.Contains(lower, key) {
				return spec, true, true
			}
		}
		ram := 16.0
		return gpuSpec{vram: ram, bw: 120}, true, false
	}
	for key, spec := range gpuDB {
		if strings.Contains(lower, key) {
			return spec, false, true
		}
	}
	return gpuSpec{}, false, false
}

// DetectServerHardware probes the Gateway host for CPU, RAM, and GPU resources.
// Results are cached briefly to avoid repeated subprocess calls (notably nvidia-smi on Windows).
func DetectServerHardware() HardwareProfile {
	hardwareDetectMu.Lock()
	defer hardwareDetectMu.Unlock()
	if !hardwareDetectAt.IsZero() && time.Since(hardwareDetectAt) < hardwareDetectCacheTTL {
		return hardwareDetectCache
	}
	hw := detectServerHardwareUncached()
	hardwareDetectCache = hw
	hardwareDetectAt = time.Now()
	return hw
}

func detectServerHardwareUncached() HardwareProfile {
	cpuCores := runtime.NumCPU()
	ramGb := detectSystemRAMGb()
	gpuName, vramFromTool := detectGPUNameAndVRAM()

	spec, isApple, matched := lookupGpuSpec(gpuName)
	if isApple {
		unified := spec.vram
		if ramGb > 0 {
			unified = ramGb
		}
		return HardwareProfile{
			GPUName:        gpuName,
			VramGb:         unified,
			RamGb:          unified,
			CPUCores:       cpuCores,
			BandwidthGbs:   spec.bw,
			IsAppleSilicon: true,
			Detected:       matched || ramGb > 0,
			ServerSide:     true,
		}
	}

	vramGb := spec.vram
	bw := spec.bw
	if vramFromTool > 0 {
		vramGb = vramFromTool
	}
	if vramGb <= 0 {
		if ramGb > 0 {
			vramGb = minFloat(ramGb, 8)
		} else {
			vramGb = 8
		}
	}
	if bw <= 0 {
		bw = 200
	}
	if ramGb <= 0 {
		ramGb = 8
	}

	return HardwareProfile{
		GPUName:        gpuName,
		VramGb:         vramGb,
		RamGb:          ramGb,
		CPUCores:       cpuCores,
		BandwidthGbs:   bw,
		IsAppleSilicon: false,
		Detected:       matched || vramFromTool > 0,
		ServerSide:     true,
	}
}

// ApplyHardwareOverride merges optional user overrides into a hardware profile.
func ApplyHardwareOverride(base HardwareProfile, override *HardwareProfile) HardwareProfile {
	if override == nil {
		return base
	}
	out := base
	if override.GPUName != "" {
		out.GPUName = override.GPUName
	}
	if override.VramGb > 0 {
		out.VramGb = override.VramGb
	}
	if override.RamGb > 0 {
		out.RamGb = override.RamGb
	}
	if override.CPUCores > 0 {
		out.CPUCores = override.CPUCores
	}
	if override.BandwidthGbs > 0 {
		out.BandwidthGbs = override.BandwidthGbs
	}
	if override.IsAppleSilicon {
		out.IsAppleSilicon = true
		if out.RamGb > 0 {
			out.VramGb = out.RamGb
		}
	}
	return out
}

func detectSystemRAMGb() float64 {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			if bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil && bytes > 0 {
				return float64(bytes) / (1024 * 1024 * 1024)
			}
		}
	case "linux":
		data, err := exec.Command("grep", "MemTotal", "/proc/meminfo").Output()
		if err == nil {
			fields := strings.Fields(string(data))
			if len(fields) >= 2 {
				if kb, err := strconv.ParseFloat(fields[1], 64); err == nil && kb > 0 {
					return kb / (1024 * 1024)
				}
			}
		}
	case "windows":
		return detectPlatformRAMGb()
	}
	return 0
}

func resolveNvidiaSMIPath() string {
	if path, err := exec.LookPath("nvidia-smi"); err == nil {
		return path
	}
	if runtime.GOOS == "windows" {
		const defaultPath = `C:\Program Files\NVIDIA Corporation\NVSMI\nvidia-smi.exe`
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}
	return ""
}

func runNvidiaSMI(args ...string) ([]byte, error) {
	path := resolveNvidiaSMIPath()
	if path == "" {
		return nil, exec.ErrNotFound
	}
	cmd := exec.Command(path, args...)
	applyExecNoWindow(cmd)
	return cmd.Output()
}

func detectGPUNameAndVRAM() (name string, vramGb float64) {
	name = runtime.GOARCH + " CPU"
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
			brand := strings.TrimSpace(string(out))
			if brand != "" {
				name = brand
			}
		}
		if strings.Contains(strings.ToLower(name), "apple") {
			return name, 0
		}
	}

	if out, err := runNvidiaSMI(
		"--query-gpu=name,memory.total",
		"--format=csv,noheader,nounits",
	); err == nil {
		line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			gpuName := strings.TrimSpace(parts[0])
			if gpuName != "" {
				name = gpuName
			}
			if mb, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil && mb > 0 {
				vramGb = mb / 1024
			}
		}
		return name, vramGb
	}

	return name, 0
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
