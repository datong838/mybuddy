// Package session provides session usage/cost summary from transcript files.
// Mirrors src/infra/session-cost-usage.ts.
package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CostUsageTotals holds aggregated token and cost totals.
type CostUsageTotals struct {
	Input              int     `json:"input"`
	Output             int     `json:"output"`
	CacheRead          int     `json:"cacheRead"`
	CacheWrite         int     `json:"cacheWrite"`
	TotalTokens        int     `json:"totalTokens"`
	TotalCost          float64 `json:"totalCost"`
	InputCost          float64 `json:"inputCost"`
	OutputCost         float64 `json:"outputCost"`
	CacheReadCost      float64 `json:"cacheReadCost"`
	CacheWriteCost     float64 `json:"cacheWriteCost"`
	MissingCostEntries int     `json:"missingCostEntries"`
}

// SessionMessageCounts holds message counts per session.
type SessionMessageCounts struct {
	Total       int `json:"total"`
	User        int `json:"user"`
	Assistant   int `json:"assistant"`
	ToolCalls   int `json:"toolCalls"`
	ToolResults int `json:"toolResults"`
	Errors      int `json:"errors"`
}

// SessionCostSummary is the per-session usage summary (aligns with TS SessionCostSummary).
type SessionCostSummary struct {
	Input              int                         `json:"input"`
	Output             int                         `json:"output"`
	CacheRead          int                         `json:"cacheRead"`
	CacheWrite         int                         `json:"cacheWrite"`
	TotalTokens        int                         `json:"totalTokens"`
	TotalCost          float64                     `json:"totalCost"`
	InputCost          float64                     `json:"inputCost"`
	OutputCost         float64                     `json:"outputCost"`
	CacheReadCost      float64                     `json:"cacheReadCost"`
	CacheWriteCost     float64                     `json:"cacheWriteCost"`
	MissingCostEntries int                         `json:"missingCostEntries"`
	SessionID          string                      `json:"sessionId,omitempty"`
	SessionFile        string                      `json:"sessionFile,omitempty"`
	FirstActivity      *int64                      `json:"firstActivity,omitempty"`
	LastActivity       *int64                      `json:"lastActivity,omitempty"`
	DurationMs         *int64                      `json:"durationMs,omitempty"`
	ActivityDates      []string                    `json:"activityDates,omitempty"`
	DailyBreakdown     []SessionDailyUsage         `json:"dailyBreakdown,omitempty"`
	DailyMessageCounts []SessionDailyMessageCounts `json:"dailyMessageCounts,omitempty"`
	DailyLatency       []SessionDailyLatency       `json:"dailyLatency,omitempty"`
	DailyModelUsage    []SessionDailyModelUsage    `json:"dailyModelUsage,omitempty"`
	MessageCounts      *SessionMessageCounts       `json:"messageCounts,omitempty"`
	ToolUsage          *SessionToolUsage           `json:"toolUsage,omitempty"`
	ModelUsage         []SessionModelUsage         `json:"modelUsage,omitempty"`
	Latency            *SessionLatencyStats        `json:"latency,omitempty"`
}

// SessionDailyMessageCounts is per-day message counts (aligns with TS SessionDailyMessageCounts).
type SessionDailyMessageCounts struct {
	Date        string `json:"date"`
	Total       int    `json:"total"`
	User        int    `json:"user"`
	Assistant   int    `json:"assistant"`
	ToolCalls   int    `json:"toolCalls"`
	ToolResults int    `json:"toolResults"`
	Errors      int    `json:"errors"`
}

// SessionDailyLatency is per-day latency (aligns with TS SessionDailyLatency).
type SessionDailyLatency struct {
	Date  string  `json:"date"`
	Count int     `json:"count"`
	AvgMs float64 `json:"avgMs"`
	MinMs float64 `json:"minMs"`
	MaxMs float64 `json:"maxMs"`
	P95Ms float64 `json:"p95Ms"`
}

// SessionDailyModelUsage is per-day per-model usage (aligns with TS SessionDailyModelUsage).
type SessionDailyModelUsage struct {
	Date     string  `json:"date"`
	Provider string  `json:"provider,omitempty"`
	Model    string  `json:"model,omitempty"`
	Tokens   int     `json:"tokens"`
	Cost     float64 `json:"cost"`
	Count    int     `json:"count"`
}

// SessionToolUsage is tool usage summary (aligns with TS SessionToolUsage).
type SessionToolUsage struct {
	TotalCalls  int                `json:"totalCalls"`
	UniqueTools int                `json:"uniqueTools"`
	Tools       []SessionToolCount `json:"tools"`
}

// SessionToolCount is tool name + count.
type SessionToolCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// SessionModelUsage is model usage summary (aligns with TS SessionModelUsage).
type SessionModelUsage struct {
	Provider string          `json:"provider,omitempty"`
	Model    string          `json:"model,omitempty"`
	Count    int             `json:"count"`
	Totals   CostUsageTotals `json:"totals"`
}

// SessionLatencyStats is latency statistics (aligns with TS SessionLatencyStats).
type SessionLatencyStats struct {
	Count int     `json:"count"`
	AvgMs float64 `json:"avgMs"`
	MinMs float64 `json:"minMs"`
	MaxMs float64 `json:"maxMs"`
	P95Ms float64 `json:"p95Ms"`
}

// SessionDailyUsage is per-day tokens/cost for a session.
type SessionDailyUsage struct {
	Date      string  `json:"date"`
	Tokens    int     `json:"tokens"`
	Cost      float64 `json:"cost"`
	Messages  int     `json:"messages"`
	ToolCalls int     `json:"toolCalls"`
	Errors    int     `json:"errors"`
}

// DiscoveredSession holds metadata for a discovered session file.
type DiscoveredSession struct {
	SessionID        string
	SessionFile      string
	Mtime            int64
	AgentID          string
	FirstUserMessage string
}

// DiscoverAllSessions scans the sessions directory for transcript files.
// Returns sessions modified after startMs (if set). Sorted by mtime desc.
func DiscoverAllSessions(agentID string, startMs, endMs int64, env func(string) string) ([]DiscoveredSession, error) {
	sessionsDir := ResolveAgentSessionsDir(agentID, env)
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var discovered []DiscoveredSession
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		filePath := filepath.Join(sessionsDir, e.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		mtime := info.ModTime().UnixMilli()
		if startMs > 0 && mtime < startMs {
			continue
		}
		sessionID := strings.TrimSuffix(e.Name(), ".jsonl")
		firstUser, _ := readFirstUserMessage(filePath)
		discovered = append(discovered, DiscoveredSession{
			SessionID:        sessionID,
			SessionFile:      filePath,
			Mtime:            mtime,
			AgentID:          agentID,
			FirstUserMessage: firstUser,
		})
	}
	sortByMtimeDesc(discovered)
	return discovered, nil
}

func sortByMtimeDesc(s []DiscoveredSession) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j].Mtime > s[i].Mtime {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func readFirstUserMessage(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if first {
			first = false
			var h TranscriptHeader
			if json.Unmarshal(line, &h) == nil && h.Type == "session" {
				continue
			}
		}
		var m TranscriptMessage
		if json.Unmarshal(line, &m) != nil {
			continue
		}
		if m.Role == "user" && len(m.Content) > 0 {
			t := m.Content[0].Text
			if len(t) > 100 {
				t = t[:100]
			}
			return t, nil
		}
	}
	return "", scanner.Err()
}

// transcriptEntryWrapper matches transcript line format: { "message": {...}, "timestamp": ..., "provider": ..., "model": ..., "durationMs": ... }.
type transcriptEntryWrapper struct {
	Type       string             `json:"type"`
	Message    *TranscriptMessage `json:"message"`
	Timestamp  string             `json:"timestamp"`
	Provider   string             `json:"provider,omitempty"`
	Model      string             `json:"model,omitempty"`
	DurationMs *int64             `json:"durationMs,omitempty"`
}

// extractToolCallNames extracts tool names from message content (aligns with TS extractToolCallNames).
func extractToolCallNames(msg *TranscriptMessage) []string {
	if msg == nil {
		return nil
	}
	names := make(map[string]bool)
	if msg.Content == nil {
		return nil
	}
	for _, block := range msg.Content {
		t := strings.ToLower(strings.TrimSpace(block.Type))
		if t == "tool_use" || t == "toolcall" || t == "tool_call" {
			if block.Name != "" {
				names[block.Name] = true
			}
		}
	}
	var result []string
	for n := range names {
		result = append(result, n)
	}
	return result
}

// countToolResults counts tool results from message content (aligns with TS countToolResults).
func countToolResults(msg *TranscriptMessage) (total, errors int) {
	if msg == nil || msg.Content == nil {
		return 0, 0
	}
	for _, block := range msg.Content {
		t := strings.ToLower(strings.TrimSpace(block.Type))
		if t == "tool_result" || t == "tool_result_error" {
			total++
			if block.IsError || t == "tool_result_error" {
				errors++
			}
		}
	}
	return total, errors
}

// computeLatencyStats computes latency statistics from values (aligns with TS computeLatencyStats).
func computeLatencyStats(values []float64) *SessionLatencyStats {
	if len(values) == 0 {
		return nil
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	p95Index := int(float64(len(sorted)) * 0.95)
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	if p95Index < 0 {
		p95Index = 0
	}
	return &SessionLatencyStats{
		Count: len(values),
		AvgMs: sum / float64(len(values)),
		MinMs: sorted[0],
		MaxMs: sorted[len(sorted)-1],
		P95Ms: sorted[p95Index],
	}
}

// LoadSessionCostSummary parses a transcript file line-by-line and returns full usage summary (aligns with TS loadSessionCostSummary).
func LoadSessionCostSummary(sessionFile string, startMs, endMs *int64) (*SessionCostSummary, error) {
	f, err := os.Open(sessionFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sessionID := strings.TrimSuffix(filepath.Base(sessionFile), ".jsonl")
	totals := &SessionCostSummary{
		SessionID:   sessionID,
		SessionFile: sessionFile,
	}
	msgCounts := &SessionMessageCounts{}
	dailyMap := make(map[string]*SessionDailyUsage)
	dailyMessageMap := make(map[string]*SessionDailyMessageCounts)
	dailyLatencyMap := make(map[string][]float64)
	dailyModelUsageMap := make(map[string]*SessionDailyModelUsage)
	toolUsageMap := make(map[string]int)
	callbackToolUsageMap := make(map[string]int)
	modelUsageMap := make(map[string]*SessionModelUsage)
	hasTokenUsageLines := false
	activityDatesSet := make(map[string]bool)
	var firstAct, lastAct int64
	var lastUserTimestamp int64
	latencyValues := []float64{}
	const maxLatencyMs = 12 * 60 * 60 * 1000
	first := true
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if first {
			first = false
			var h TranscriptHeader
			if json.Unmarshal(line, &h) == nil && h.Type == "session" {
				continue
			}
		}
		// Parse token_usage lines (from usage callback); these contain actual token consumption
		var tokenLine tokenUsageLine
		if json.Unmarshal(line, &tokenLine) == nil && tokenLine.Type == "token_usage" {
			hasTokenUsageLines = true
			ts := parseTimestampString(tokenLine.Timestamp)
			if ts > 0 && (startMs == nil || ts >= *startMs) && (endMs == nil || ts <= *endMs) {
				if firstAct == 0 || ts < firstAct {
					firstAct = ts
				}
				if ts > lastAct {
					lastAct = ts
				}
				dayKey := FormatDayKey(ts)
				activityDatesSet[dayKey] = true
				if u := tokenUsageToUsage(&tokenLine); u != nil {
					tokens := u.TotalTokens
					if tokens == 0 {
						tokens = u.Input + u.Output + u.CacheRead + u.CacheWrite
					}
					totals.Input += u.Input
					totals.Output += u.Output
					totals.CacheRead += u.CacheRead
					totals.CacheWrite += u.CacheWrite
					totals.TotalTokens += tokens
					totals.MissingCostEntries++ // token_usage has no cost
					if dailyMap[dayKey] == nil {
						dailyMap[dayKey] = &SessionDailyUsage{Date: dayKey}
					}
					dailyMap[dayKey].Tokens += tokens
					providerKey := strings.TrimSpace(tokenLine.Provider)
					if providerKey == "" {
						providerKey = "unknown"
					}
					modelKey := strings.TrimSpace(tokenLine.Model)
					if modelKey == "" {
						modelKey = "unknown"
					}
					dailyModelKey := dayKey + "::" + providerKey + "::" + modelKey
					if dailyModelUsageMap[dailyModelKey] == nil {
						dailyModelUsageMap[dailyModelKey] = &SessionDailyModelUsage{
							Date:     dayKey,
							Provider: providerKey,
							Model:    modelKey,
						}
					}
					dmu := dailyModelUsageMap[dailyModelKey]
					dmu.Tokens += tokens
					dmu.Count++
					globalModelKey := providerKey + "::" + modelKey
					if modelUsageMap[globalModelKey] == nil {
						modelUsageMap[globalModelKey] = &SessionModelUsage{
							Provider: providerKey,
							Model:    modelKey,
							Totals:   CostUsageTotals{},
						}
					}
					mum := modelUsageMap[globalModelKey]
					mum.Count++
					mum.Totals.Input += u.Input
					mum.Totals.Output += u.Output
					mum.Totals.CacheRead += u.CacheRead
					mum.Totals.CacheWrite += u.CacheWrite
					mum.Totals.TotalTokens += tokens
					mum.Totals.MissingCostEntries++
				}
			}
			continue
		}
		var toolLine toolCallLine
		if json.Unmarshal(line, &toolLine) == nil && toolLine.Type == "tool_call" && strings.TrimSpace(toolLine.Name) != "" {
			ts := parseTimestampString(toolLine.Timestamp)
			if ts > 0 && (startMs == nil || ts >= *startMs) && (endMs == nil || ts <= *endMs) {
				if firstAct == 0 || ts < firstAct {
					firstAct = ts
				}
				if ts > lastAct {
					lastAct = ts
				}
				dayKey := FormatDayKey(ts)
				activityDatesSet[dayKey] = true
				callbackToolUsageMap[toolLine.Name]++
				if dailyMessageMap[dayKey] == nil {
					dailyMessageMap[dayKey] = &SessionDailyMessageCounts{Date: dayKey}
				}
				dailyMessageMap[dayKey].ToolCalls++
			}
			continue
		}
		var wrapper transcriptEntryWrapper
		var msg *TranscriptMessage
		var ts int64
		var provider, model string
		var durationMs *int64
		if json.Unmarshal([]byte(line), &wrapper) == nil && wrapper.Message != nil {
			msg = wrapper.Message
			if wrapper.Timestamp != "" {
				ts = parseTimestampString(wrapper.Timestamp)
			}
			if ts == 0 {
				ts = msg.Timestamp
			}
			provider = wrapper.Provider
			if provider == "" {
				provider = msg.Provider
			}
			model = wrapper.Model
			if model == "" {
				model = msg.Model
			}
			if wrapper.DurationMs != nil {
				durationMs = wrapper.DurationMs
			} else if msg.DurationMs != nil {
				durationMs = msg.DurationMs
			}
		} else {
			var direct TranscriptMessage
			if json.Unmarshal(line, &direct) != nil {
				continue
			}
			msg = &direct
			ts = direct.Timestamp
			provider = direct.Provider
			model = direct.Model
			if direct.DurationMs != nil {
				durationMs = direct.DurationMs
			}
		}
		if ts == 0 || (startMs != nil && ts < *startMs) || (endMs != nil && ts > *endMs) {
			continue
		}
		if firstAct == 0 || ts < firstAct {
			firstAct = ts
		}
		if ts > lastAct {
			lastAct = ts
		}
		dayKey := FormatDayKey(ts)
		activityDatesSet[dayKey] = true
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		if msg.Role == "user" {
			msgCounts.User++
			msgCounts.Total++
			lastUserTimestamp = ts
		} else if msg.Role == "assistant" {
			msgCounts.Assistant++
			msgCounts.Total++
			latencyMs := 0.0
			if durationMs != nil {
				latencyMs = float64(*durationMs)
			} else if lastUserTimestamp > 0 {
				latencyMs = float64(ts - lastUserTimestamp)
			}
			if latencyMs > 0 && latencyMs <= maxLatencyMs {
				latencyValues = append(latencyValues, latencyMs)
				dailyLatencyMap[dayKey] = append(dailyLatencyMap[dayKey], latencyMs)
			}
		}
		toolNames := extractToolCallNames(msg)
		if len(toolNames) > 0 && len(callbackToolUsageMap) == 0 {
			msgCounts.ToolCalls += len(toolNames)
			for _, name := range toolNames {
				toolUsageMap[name]++
			}
		}
		toolTotal, toolErrors := countToolResults(msg)
		if toolTotal > 0 {
			msgCounts.ToolResults += toolTotal
			msgCounts.Errors += toolErrors
		}
		if msg.StopReason != "" {
			stopReason := strings.ToLower(strings.TrimSpace(msg.StopReason))
			if stopReason == "error" || stopReason == "aborted" || stopReason == "timeout" {
				msgCounts.Errors++
			}
		}
		if dailyMessageMap[dayKey] == nil {
			dailyMessageMap[dayKey] = &SessionDailyMessageCounts{Date: dayKey}
		}
		dmc := dailyMessageMap[dayKey]
		if msg.Role == "user" || msg.Role == "assistant" {
			dmc.Total++
		}
		if msg.Role == "user" {
			dmc.User++
		} else if msg.Role == "assistant" {
			dmc.Assistant++
		}
		if len(toolNames) > 0 && len(callbackToolUsageMap) == 0 {
			dmc.ToolCalls += len(toolNames)
		}
		dmc.ToolResults += toolTotal
		dmc.Errors += toolErrors
		if msg.StopReason != "" {
			stopReason := strings.ToLower(strings.TrimSpace(msg.StopReason))
			if stopReason == "error" || stopReason == "aborted" || stopReason == "timeout" {
				dmc.Errors++
			}
		}
		if msg.Usage == nil {
			continue
		}
		if hasTokenUsageLines {
			continue
		}
		u := msg.Usage
		tokens := u.TotalTokens
		if tokens == 0 {
			tokens = u.Input + u.Output + u.CacheRead + u.CacheWrite
		}
		cost := 0.0
		if u.Cost != nil {
			cost = u.Cost.Total
		}
		totals.Input += u.Input
		totals.Output += u.Output
		totals.CacheRead += u.CacheRead
		totals.CacheRead += u.CacheRead
		totals.CacheWrite += u.CacheWrite
		totals.TotalTokens += tokens
		if u.Cost != nil {
			totals.TotalCost += u.Cost.Total
			totals.InputCost += u.Cost.Input
			totals.OutputCost += u.Cost.Output
		} else {
			totals.MissingCostEntries++
		}
		if dailyMap[dayKey] == nil {
			dailyMap[dayKey] = &SessionDailyUsage{Date: dayKey}
		}
		d := dailyMap[dayKey]
		d.Tokens += tokens
		d.Cost += cost
		providerKey := provider
		if providerKey == "" {
			providerKey = "unknown"
		}
		modelKey := model
		if modelKey == "" {
			modelKey = "unknown"
		}
		dailyModelKey := dayKey + "::" + providerKey + "::" + modelKey
		if dailyModelUsageMap[dailyModelKey] == nil {
			dailyModelUsageMap[dailyModelKey] = &SessionDailyModelUsage{
				Date:     dayKey,
				Provider: providerKey,
				Model:    modelKey,
			}
		}
		dmu := dailyModelUsageMap[dailyModelKey]
		dmu.Tokens += tokens
		dmu.Cost += cost
		dmu.Count++
		globalModelKey := providerKey + "::" + modelKey
		if modelUsageMap[globalModelKey] == nil {
			modelUsageMap[globalModelKey] = &SessionModelUsage{
				Provider: providerKey,
				Model:    modelKey,
				Totals:   CostUsageTotals{},
			}
		}
		mum := modelUsageMap[globalModelKey]
		mum.Count++
		mum.Totals.Input += u.Input
		mum.Totals.Output += u.Output
		mum.Totals.CacheRead += u.CacheRead
		mum.Totals.CacheWrite += u.CacheWrite
		mum.Totals.TotalTokens += tokens
		mum.Totals.TotalCost += cost
		if u.Cost != nil {
			mum.Totals.InputCost += u.Cost.Input
			mum.Totals.OutputCost += u.Cost.Output
		} else {
			mum.Totals.MissingCostEntries++
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if firstAct > 0 {
		totals.FirstActivity = &firstAct
	}
	if lastAct > 0 {
		totals.LastActivity = &lastAct
		if firstAct > 0 {
			duration := lastAct - firstAct
			if duration < 0 {
				duration = 0
			}
			totals.DurationMs = &duration
		}
	}
	var activityDates []string
	for d := range activityDatesSet {
		activityDates = append(activityDates, d)
	}
	sort.Strings(activityDates)
	totals.ActivityDates = activityDates
	totals.MessageCounts = msgCounts
	if len(dailyMap) > 0 {
		var dates []string
		for k := range dailyMap {
			dates = append(dates, k)
		}
		sort.Strings(dates)
		totals.DailyBreakdown = make([]SessionDailyUsage, len(dates))
		for i, d := range dates {
			totals.DailyBreakdown[i] = *dailyMap[d]
		}
	}
	if len(dailyMessageMap) > 0 {
		var dates []string
		for k := range dailyMessageMap {
			dates = append(dates, k)
		}
		sort.Strings(dates)
		totals.DailyMessageCounts = make([]SessionDailyMessageCounts, len(dates))
		for i, d := range dates {
			totals.DailyMessageCounts[i] = *dailyMessageMap[d]
		}
	}
	if len(dailyLatencyMap) > 0 {
		var dates []string
		for k := range dailyLatencyMap {
			dates = append(dates, k)
		}
		sort.Strings(dates)
		var dailyLatency []SessionDailyLatency
		for _, d := range dates {
			stats := computeLatencyStats(dailyLatencyMap[d])
			if stats != nil {
				dailyLatency = append(dailyLatency, SessionDailyLatency{
					Date:  d,
					Count: stats.Count,
					AvgMs: stats.AvgMs,
					MinMs: stats.MinMs,
					MaxMs: stats.MaxMs,
					P95Ms: stats.P95Ms,
				})
			}
		}
		if len(dailyLatency) > 0 {
			totals.DailyLatency = dailyLatency
		}
	}
	if len(dailyModelUsageMap) > 0 {
		var entries []SessionDailyModelUsage
		for _, e := range dailyModelUsageMap {
			entries = append(entries, *e)
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Date != entries[j].Date {
				return entries[i].Date < entries[j].Date
			}
			return entries[j].Cost < entries[i].Cost
		})
		totals.DailyModelUsage = entries
	}
	if len(callbackToolUsageMap) > 0 {
		toolUsageMap = callbackToolUsageMap
		msgCounts.ToolCalls = 0
		for _, count := range callbackToolUsageMap {
			msgCounts.ToolCalls += count
		}
	}
	if len(toolUsageMap) > 0 {
		var tools []SessionToolCount
		totalCalls := 0
		for name, count := range toolUsageMap {
			totalCalls += count
			tools = append(tools, SessionToolCount{Name: name, Count: count})
		}
		sort.Slice(tools, func(i, j int) bool { return tools[j].Count < tools[i].Count })
		totals.ToolUsage = &SessionToolUsage{
			TotalCalls:  totalCalls,
			UniqueTools: len(toolUsageMap),
			Tools:       tools,
		}
	}
	if len(modelUsageMap) > 0 {
		var modelUsage []SessionModelUsage
		for _, e := range modelUsageMap {
			modelUsage = append(modelUsage, *e)
		}
		sort.Slice(modelUsage, func(i, j int) bool {
			if modelUsage[j].Totals.TotalCost != modelUsage[i].Totals.TotalCost {
				return modelUsage[j].Totals.TotalCost < modelUsage[i].Totals.TotalCost
			}
			return modelUsage[j].Totals.TotalTokens < modelUsage[i].Totals.TotalTokens
		})
		totals.ModelUsage = modelUsage
	}
	if len(latencyValues) > 0 {
		totals.Latency = computeLatencyStats(latencyValues)
	}
	return totals, nil
}

// parseTimestampString parses timestamp string (ISO 8601 like "2026-02-12T05:13:00.173Z" or numeric string) to int64 milliseconds.
// Returns 0 if parsing fails.
func parseTimestampString(s string) int64 {
	if s == "" {
		return 0
	}
	// Try parsing as numeric string first (int64 milliseconds)
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ms
	}
	// Try parsing as ISO 8601 timestamp string
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Try RFC3339 format without nanoseconds
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return 0
		}
	}
	return t.UnixMilli()
}

// FormatDayKey returns YYYY-MM-DD for a timestamp.
func FormatDayKey(ms int64) string {
	t := time.UnixMilli(ms).UTC()
	return t.Format("2006-01-02")
}

// CostUsageDailyEntry is date + full totals (aligns with TS CostUsageDailyEntry).
type CostUsageDailyEntry struct {
	Date string `json:"date"`
	CostUsageTotals
}

// CostUsageSummary is the result of loadCostUsageSummary (aligns with TS CostUsageSummary).
type CostUsageSummary struct {
	UpdatedAt int64                 `json:"updatedAt"`
	Days      int                   `json:"days"`
	Daily     []CostUsageDailyEntry `json:"daily"`
	Totals    CostUsageTotals       `json:"totals"`
}

// transcriptLine is one JSON line: either header (type "session") or message (message + optional timestamp).
type transcriptLine struct {
	Type      string             `json:"type"`
	Message   *TranscriptMessage `json:"message"`
	Timestamp string             `json:"timestamp"` // Can be ISO 8601 string or int64 (handled via parseTimestampString)
}

// tokenUsageLine is a JSON line with type "token_usage" (appended by TokenCallback / AppendTokenUsageToSession).
// Token consumption is recorded here; message.usage is often 0.
type tokenUsageLine struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"sessionId,omitempty"`
	RequestID   string `json:"requestId,omitempty"`
	Model       string `json:"model,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Input       int    `json:"input"`
	Output      int    `json:"output"`
	CacheRead   int    `json:"cacheRead"`
	CacheWrite  int    `json:"cacheWrite"`
	CacheCreate int    `json:"cacheCreation"` // alias from logging.TokenUsageSessionLine
	TotalTokens int    `json:"totalTokens"`
}

// toolCallLine is a lightweight tool invocation record from the Eino usage callback.
type toolCallLine struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	SessionID string `json:"sessionId,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	Name      string `json:"name"`
}

// applyUsageTotals adds usage to bucket (aligns with TS applyUsageTotals).
func applyUsageTotals(t *CostUsageTotals, u *Usage) {
	if u == nil {
		return
	}
	t.Input += u.Input
	t.Output += u.Output
	t.CacheRead += u.CacheRead
	t.CacheWrite += u.CacheWrite
	tokens := u.TotalTokens
	if tokens == 0 {
		tokens = u.Input + u.Output + u.CacheRead + u.CacheWrite
	}
	t.TotalTokens += tokens
}

// applyCostToTotals adds cost to bucket: cost breakdown if present, else missingCostEntries (aligns with TS applyCostBreakdown/applyCostTotal).
func applyCostToTotals(t *CostUsageTotals, u *Usage) {
	if u == nil {
		return
	}
	if u.Cost != nil {
		t.TotalCost += u.Cost.Total
		t.InputCost += u.Cost.Input
		t.OutputCost += u.Cost.Output
		t.CacheReadCost += u.Cost.CacheRead
		t.CacheWriteCost += u.Cost.CacheWrite
	} else {
		t.MissingCostEntries++
	}
}

// tokenUsageToUsage converts tokenUsageLine to Usage (no cost; cost is computed separately).
func tokenUsageToUsage(t *tokenUsageLine) *Usage {
	if t == nil || (t.Input == 0 && t.Output == 0 && t.CacheRead == 0 && t.CacheWrite == 0 && t.CacheCreate == 0 && t.TotalTokens == 0) {
		return nil
	}
	cacheWrite := t.CacheWrite
	if cacheWrite == 0 {
		cacheWrite = t.CacheCreate
	}
	tokens := t.TotalTokens
	if tokens == 0 {
		tokens = t.Input + t.Output + t.CacheRead + cacheWrite
	}
	return &Usage{
		Input:       t.Input,
		Output:      t.Output,
		CacheRead:   t.CacheRead,
		CacheWrite:  cacheWrite,
		TotalTokens: tokens,
	}
}

// scanTranscriptForCostUsage reads file line by line, parses JSON, and for each message with usage or token_usage in [startMs,endMs] calls onEntry.
// token_usage lines (from TokenCallback) contain actual token consumption; message.usage is often 0.
func scanTranscriptForCostUsage(filePath string, startMs, endMs int64, onEntry func(ts int64, u *Usage)) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		// Check for token_usage lines first (type "token_usage" with input/output/cacheRead/totalTokens at top level)
		var tokenLine tokenUsageLine
		if json.Unmarshal(line, &tokenLine) == nil && tokenLine.Type == "token_usage" {
			ts := parseTimestampString(tokenLine.Timestamp)
			if ts > 0 && ts >= startMs && ts <= endMs {
				if u := tokenUsageToUsage(&tokenLine); u != nil {
					onEntry(ts, u)
				}
			}
			continue
		}
		var wrapper transcriptLine
		if err := json.Unmarshal(line, &wrapper); err != nil {
			continue
		}
		if first {
			first = false
			if wrapper.Type == "session" {
				continue
			}
		}
		var msg *TranscriptMessage
		var ts int64
		if wrapper.Message != nil {
			msg = wrapper.Message
			if wrapper.Timestamp != "" {
				ts = parseTimestampString(wrapper.Timestamp)
			}
			if ts == 0 {
				ts = msg.Timestamp
			}
		} else {
			var direct TranscriptMessage
			if json.Unmarshal(line, &direct) != nil {
				continue
			}
			msg = &direct
			ts = direct.Timestamp
		}
		if msg == nil || msg.Usage == nil {
			continue
		}
		if ts < startMs || ts > endMs {
			continue
		}
		onEntry(ts, msg.Usage)
	}
	return scanner.Err()
}

// LoadCostUsageSummary reads each transcript file line by line, parses JSON per line, and aggregates usage in [startMs, endMs].
// Aligns with TS loadCostUsageSummary: list .jsonl with mtime >= sinceTime, scanUsageFile (line-by-line JSON), filter by timestamp, applyUsageTotals + applyCost per entry.
func LoadCostUsageSummary(agentIDs []string, startMs, endMs int64, env func(string) string) (*CostUsageSummary, error) {
	if len(agentIDs) == 0 {
		agentIDs = []string{DefaultAgentID}
	}
	totals := &CostUsageTotals{}
	dailyMap := make(map[string]*CostUsageTotals)
	seenFiles := make(map[string]bool)
	for _, agentID := range agentIDs {
		discovered, err := DiscoverAllSessions(agentID, startMs, 0, env)
		if err != nil {
			continue
		}
		for _, d := range discovered {
			if seenFiles[d.SessionFile] {
				continue
			}
			seenFiles[d.SessionFile] = true
			_ = scanTranscriptForCostUsage(d.SessionFile, startMs, endMs, func(ts int64, u *Usage) {
				dayKey := FormatDayKey(ts)
				bucket := dailyMap[dayKey]
				if bucket == nil {
					bucket = &CostUsageTotals{}
					dailyMap[dayKey] = bucket
				}
				applyUsageTotals(bucket, u)
				applyCostToTotals(bucket, u)
				applyUsageTotals(totals, u)
				applyCostToTotals(totals, u)
			})
		}
	}
	var daily []CostUsageDailyEntry
	for date, bucket := range dailyMap {
		daily = append(daily, CostUsageDailyEntry{Date: date, CostUsageTotals: *bucket})
	}
	sort.Slice(daily, func(i, j int) bool { return daily[i].Date < daily[j].Date })
	days := 1
	if endMs > startMs {
		days = int(math.Ceil(float64(endMs-startMs)/(24*60*60*1000))) + 1
	}
	return &CostUsageSummary{
		UpdatedAt: time.Now().UnixMilli(),
		Days:      days,
		Daily:     daily,
		Totals:    *totals,
	}, nil
}

// UsageTimePoint is one point in the usage timeseries (aligns with TS SessionUsageTimePoint).
type UsageTimePoint struct {
	Timestamp        int64   `json:"timestamp"`
	Input            int     `json:"input"`
	Output           int     `json:"output"`
	CacheRead        int     `json:"cacheRead"`
	CacheWrite       int     `json:"cacheWrite"`
	TotalTokens      int     `json:"totalTokens"`
	Cost             float64 `json:"cost"`
	CumulativeTokens int     `json:"cumulativeTokens"`
	CumulativeCost   float64 `json:"cumulativeCost"`
}

// LoadSessionUsageTimeSeries reads a transcript and returns usage points (aligns with TS loadSessionUsageTimeSeries).
// Includes both message.usage and token_usage lines.
func LoadSessionUsageTimeSeries(sessionFile string, maxPoints int) ([]UsageTimePoint, error) {
	f, err := os.Open(sessionFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var points []UsageTimePoint
	var cumulativeTokens int
	var cumulativeCost float64
	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if first {
			first = false
			var h TranscriptHeader
			if json.Unmarshal(line, &h) == nil && h.Type == "session" {
				continue
			}
		}
		var ts int64
		var tokens int
		var cost float64
		var input, output, cacheRead, cacheWrite int
		// Try token_usage first
		var tokenLine tokenUsageLine
		if json.Unmarshal(line, &tokenLine) == nil && tokenLine.Type == "token_usage" {
			ts = parseTimestampString(tokenLine.Timestamp)
			if ts == 0 {
				continue
			}
			u := tokenUsageToUsage(&tokenLine)
			if u == nil {
				continue
			}
			input = u.Input
			output = u.Output
			cacheRead = u.CacheRead
			cacheWrite = u.CacheWrite
			tokens = u.TotalTokens
			if tokens == 0 {
				tokens = input + output + cacheRead + cacheWrite
			}
		} else {
			var m TranscriptMessage
			if json.Unmarshal(line, &m) != nil {
				continue
			}
			ts = m.Timestamp
			if ts == 0 {
				continue
			}
			if m.Usage != nil {
				input = m.Usage.Input
				output = m.Usage.Output
				cacheRead = m.Usage.CacheRead
				cacheWrite = m.Usage.CacheWrite
				tokens = m.Usage.TotalTokens
				if tokens == 0 {
					tokens = input + output + cacheRead + cacheWrite
				}
				if m.Usage.Cost != nil {
					cost = m.Usage.Cost.Total
				}
			}
		}
		cumulativeTokens += tokens
		cumulativeCost += cost
		points = append(points, UsageTimePoint{
			Timestamp:        ts,
			Input:            input,
			Output:           output,
			CacheRead:        cacheRead,
			CacheWrite:       cacheWrite,
			TotalTokens:      tokens,
			Cost:             cost,
			CumulativeTokens: cumulativeTokens,
			CumulativeCost:   cumulativeCost,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if maxPoints <= 0 {
		maxPoints = 200
	}
	if len(points) <= maxPoints {
		return points, nil
	}
	step := (len(points) + maxPoints - 1) / maxPoints
	downsampled := make([]UsageTimePoint, 0, maxPoints+1)
	for i := 0; i < len(points); i += step {
		downsampled = append(downsampled, points[i])
	}
	if len(downsampled) > 0 && len(points) > 0 && downsampled[len(downsampled)-1].Timestamp != points[len(points)-1].Timestamp {
		downsampled = append(downsampled, points[len(points)-1])
	}
	return downsampled, nil
}

// LogEntry is one log entry (aligns with TS SessionLogEntry).
type LogEntry struct {
	Timestamp int64    `json:"timestamp"`
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	Tokens    *int     `json:"tokens,omitempty"`
	Cost      *float64 `json:"cost,omitempty"`
}

// LoadSessionLogs reads a transcript and returns the last limit log entries (aligns with TS loadSessionLogs).
func LoadSessionLogs(sessionFile string, limit int) ([]LogEntry, error) {
	f, err := os.Open(sessionFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var all []LogEntry
	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if first {
			first = false
			var h TranscriptHeader
			if json.Unmarshal(line, &h) == nil && h.Type == "session" {
				continue
			}
		}
		var m TranscriptMessage
		if json.Unmarshal(line, &m) != nil {
			continue
		}
		content := ""
		for _, b := range m.Content {
			if b.Type == "text" && b.Text != "" {
				content = b.Text
				break
			}
		}
		var tokens *int
		var cost *float64
		if m.Usage != nil {
			t := m.Usage.TotalTokens
			if t == 0 {
				t = m.Usage.Input + m.Usage.Output + m.Usage.CacheRead + m.Usage.CacheWrite
			}
			tokens = &t
			if m.Usage.Cost != nil {
				c := m.Usage.Cost.Total
				cost = &c
			}
		}
		role := m.Role
		if role == "" {
			role = "user"
		}
		all = append(all, LogEntry{Timestamp: m.Timestamp, Role: role, Content: content, Tokens: tokens, Cost: cost})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}
	if len(all) <= limit {
		return all, nil
	}
	return all[len(all)-limit:], nil
}
