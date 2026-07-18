package embeddedmodels

import (
	"math"
	"sort"
)

const runtimeOverheadGB = 0.5

// ModelTier is the S–F compatibility grade for a model on given hardware.
type ModelTier string

const (
	TierS ModelTier = "S"
	TierA ModelTier = "A"
	TierB ModelTier = "B"
	TierC ModelTier = "C"
	TierD ModelTier = "D"
	TierF ModelTier = "F"
)

// ModelFitStatus describes whether a model can run on the hardware.
type ModelFitStatus string

const (
	FitCanRun  ModelFitStatus = "can-run"
	FitTight   ModelFitStatus = "tight"
	FitCantRun ModelFitStatus = "cant-run"
)

var tierLabels = map[ModelTier]string{
	TierS: "运行极佳",
	TierA: "运行良好",
	TierB: "尚可",
	TierC: "勉强可用",
	TierD: "非常慢",
	TierF: "无法运行",
}

// ModelRecommendation is a scored compatibility result for one catalog model.
type ModelRecommendation struct {
	ModelID      string         `json:"modelId"`
	Score        int            `json:"score"`
	Tier         ModelTier      `json:"tier"`
	TierLabel    string         `json:"tierLabel"`
	FitStatus    ModelFitStatus `json:"fitStatus"`
	VramGb       float64        `json:"vramGb"`
	TokensPerSec int            `json:"tokensPerSec"`
	MemoryPct    int            `json:"memoryPct"`
}

// RecommendationsResult bundles hardware profile with scored models.
type RecommendationsResult struct {
	Hardware        HardwareProfile       `json:"hardware"`
	Recommendations []ModelRecommendation `json:"recommendations"`
}

func catalogModelVramGb(entry CatalogEntry) float64 {
	var total int64
	for _, f := range entry.Files {
		total += f.Size
	}
	if total > 0 {
		return float64(total)/(1024*1024*1024) + runtimeOverheadGB
	}
	if entry.ParamsB > 0 {
		return entry.ParamsB*0.55 + runtimeOverheadGB
	}
	return 1
}

func availableMemoryGb(hw HardwareProfile) float64 {
	if hw.IsAppleSilicon {
		return hw.RamGb * 0.75
	}
	return hw.VramGb
}

func classifyFit(modelVram float64, hw HardwareProfile) ModelFitStatus {
	total := availableMemoryGb(hw)
	if total <= 0 {
		return FitCantRun
	}
	pct := (modelVram / total) * 100
	if hw.IsAppleSilicon {
		if pct <= 52.5 {
			return FitCanRun
		}
		if pct <= 75 {
			return FitTight
		}
		return FitCantRun
	}
	if pct <= 85 {
		return FitCanRun
	}
	if pct <= 110 {
		return FitTight
	}
	return FitCantRun
}

func speedScore(tokensPerSec float64) float64 {
	switch {
	case tokensPerSec >= 80:
		return 100
	case tokensPerSec >= 40:
		return 85
	case tokensPerSec >= 20:
		return 65
	case tokensPerSec >= 10:
		return 45
	case tokensPerSec >= 5:
		return 25
	default:
		return 10
	}
}

func memoryScore(memPct float64) float64 {
	switch {
	case memPct <= 30:
		return 100
	case memPct <= 50:
		return 80
	case memPct <= 70:
		return 55
	case memPct <= 85:
		return 30
	default:
		return 10
	}
}

func qualityBonus(paramsB float64) float64 {
	if paramsB <= 0 {
		paramsB = 0.5
	}
	return math.Min(15, math.Log2(paramsB+1)*2.5)
}

func scoreToTier(score float64, fit ModelFitStatus) ModelTier {
	if fit == FitCantRun {
		return TierF
	}
	switch {
	case score >= 85:
		return TierS
	case score >= 70:
		return TierA
	case score >= 55:
		return TierB
	case score >= 40:
		return TierC
	case score >= 20:
		return TierD
	default:
		return TierF
	}
}

// RecommendModels scores catalog entries against a hardware profile.
func RecommendModels(entries []CatalogEntry, hw HardwareProfile) []ModelRecommendation {
	totalMem := availableMemoryGb(hw)
	efficiency := 0.7
	if hw.IsAppleSilicon {
		efficiency = 0.65
	}

	out := make([]ModelRecommendation, 0, len(entries))
	for _, entry := range entries {
		vram := catalogModelVramGb(entry)
		fit := classifyFit(vram, hw)
		memPct := 0.0
		if totalMem > 0 {
			memPct = (vram / totalMem) * 100
		}
		tokensPerSec := 0.0
		if fit != FitCantRun && vram > 0 {
			tokensPerSec = (hw.BandwidthGbs / vram) * efficiency
		}

		score := speedScore(tokensPerSec)*0.55 + memoryScore(memPct)*0.35
		score += qualityBonus(entry.ParamsB)
		if fit == FitTight {
			score *= 0.65
		}
		if fit == FitCantRun {
			score = math.Min(score, 15)
		}

		tier := scoreToTier(score, fit)
		out = append(out, ModelRecommendation{
			ModelID:      entry.ID,
			Score:        int(math.Round(score)),
			Tier:         tier,
			TierLabel:    tierLabels[tier],
			FitStatus:    fit,
			VramGb:       math.Round(vram*10) / 10,
			TokensPerSec: int(math.Round(tokensPerSec)),
			MemoryPct:    int(math.Round(memPct)),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return out
}

// BuildRecommendations detects server hardware, applies optional override, and scores the catalog.
func BuildRecommendations(env func(string) string, override *HardwareProfile) RecommendationsResult {
	hw := DetectServerHardware()
	hw = ApplyHardwareOverride(hw, override)

	entries := AllCatalogEntries()
	RefreshSideloadCatalog(env)
	for id, entry := range sideloadSnapshot() {
		found := false
		for _, e := range entries {
			if e.ID == id {
				found = true
				break
			}
		}
		if !found {
			entries = append(entries, entry)
		}
	}

	return RecommendationsResult{
		Hardware:        hw,
		Recommendations: RecommendModels(entries, hw),
	}
}
