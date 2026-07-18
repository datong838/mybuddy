package cron

import (
	"time"

	"github.com/robfig/cron/v3"
)

// ComputeNextRunAtMs returns the next run time in milliseconds, or 0 if none.
func ComputeNextRunAtMs(schedule CronSchedule, nowMs int64) int64 {
	switch schedule.Kind {
	case "at":
		return computeNextAt(schedule.At, nowMs)
	case "every":
		return computeNextEvery(schedule.EveryMs, schedule.AnchorMs, nowMs)
	case "cron":
		return computeNextCron(schedule.Expr, schedule.Tz, nowMs)
	default:
		return 0
	}
}

func computeNextCron(expr, tz string, nowMs int64) int64 {
	if expr == "" {
		return 0
	}
	opts := []cron.Option{}
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err == nil {
			opts = append(opts, cron.WithLocation(loc))
		}
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(expr)
	if err != nil {
		return 0
	}
	now := time.UnixMilli(nowMs)
	next := sched.Next(now)
	if next.IsZero() {
		return 0
	}
	return next.UnixMilli()
}

func computeNextAt(at string, nowMs int64) int64 {
	if at == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, at)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z07:00", at)
		if err != nil {
			return 0
		}
	}
	atMs := t.UnixMilli()
	if atMs > nowMs {
		return atMs
	}
	return 0
}

func computeNextEvery(everyMs int64, anchorMs *int64, nowMs int64) int64 {
	if everyMs < 1 {
		everyMs = 1
	}
	anchor := nowMs
	if anchorMs != nil && *anchorMs > 0 {
		anchor = *anchorMs
	}
	if nowMs < anchor {
		return anchor
	}
	elapsed := nowMs - anchor
	steps := (elapsed + everyMs - 1) / everyMs
	if steps < 1 {
		steps = 1
	}
	return anchor + steps*everyMs
}
