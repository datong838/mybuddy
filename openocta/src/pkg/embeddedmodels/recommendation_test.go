package embeddedmodels

import "testing"

func TestRecommendModelsOrdering(t *testing.T) {
	hw := HardwareProfile{
		GPUName:        "RTX 4090",
		VramGb:         24,
		RamGb:          64,
		CPUCores:       16,
		BandwidthGbs:   1008,
		IsAppleSilicon: false,
		Detected:       true,
		ServerSide:     true,
	}
	entries := []CatalogEntry{
		{ID: "tiny", Name: "Tiny", ParamsB: 0.135, Quantization: "Q4_K_M"},
		{ID: "large", Name: "Large", ParamsB: 70, Quantization: "Q4_K_M"},
	}
	recs := RecommendModels(entries, hw)
	if len(recs) != 2 {
		t.Fatalf("expected 2 recommendations, got %d", len(recs))
	}
	if recs[0].ModelID != "tiny" {
		t.Fatalf("expected tiny first, got %s", recs[0].ModelID)
	}
	if recs[1].Tier == TierS {
		t.Fatalf("expected large model to score lower than tiny")
	}
}

func TestClassifyFitAppleSilicon(t *testing.T) {
	hw := HardwareProfile{VramGb: 16, RamGb: 16, IsAppleSilicon: true}
	if classifyFit(4, hw) != FitCanRun {
		t.Fatal("expected can-run for small model on 16GB unified memory")
	}
	if classifyFit(20, hw) != FitCantRun {
		t.Fatal("expected cant-run for oversized model")
	}
}

func TestApplyHardwareOverride(t *testing.T) {
	base := DetectServerHardware()
	override := &HardwareProfile{VramGb: 48, RamGb: 128, BandwidthGbs: 900}
	merged := ApplyHardwareOverride(base, override)
	if merged.VramGb != 48 || merged.RamGb != 128 || merged.BandwidthGbs != 900 {
		t.Fatalf("override not applied: %+v", merged)
	}
}
