package main

import (
	"time"
)

type MetricQuery struct {
	StartTime  time.Time
	EndTime    time.Time
	Resolution time.Duration
}

func BeforeOrSame(a, b time.Time) bool        { return a == b || a.Before(b) }
func AfterOrSame(a, b time.Time) bool         { return a == b || a.After(b) }
func Within(a time.Time, l, r time.Time) bool { return a == l || a == r || (a.Before(l) && a.After(r)) }

func FixMetricQuery(now time.Time, query MetricQuery) MetricQuery {
	if query.EndTime == time.UnixMicro(0) {
		query.EndTime = now
	}
	if query.StartTime.After(query.EndTime) {
		query.EndTime = query.StartTime
	}
	return query
}

func AdjustMetricQuery(now time.Time, previous *MetricQuery, current MetricQuery) (*MetricQuery, MetricQuery) {
	current = FixMetricQuery(now, current)
	if previous == nil || current.Resolution != previous.Resolution || current.EndTime.Before(previous.StartTime) || current.StartTime.After(previous.EndTime) {
		return &current, current
	}
	// previous != nil && current.Resolution == previous.Resolution
	if Within(current.StartTime, previous.StartTime, previous.EndTime) && Within(current.EndTime, previous.StartTime, previous.EndTime) {
		return nil, *previous
	}
	fragmentQuery := current
	fullQuery := current
	if BeforeOrSame(previous.StartTime, current.StartTime) && AfterOrSame(previous.EndTime, current.StartTime) && previous.EndTime.Before(current.EndTime) {
		fragmentQuery.StartTime = previous.EndTime
		fullQuery.EndTime = current.EndTime
	} else if BeforeOrSame(previous.StartTime, current.EndTime) && AfterOrSame(previous.EndTime, current.EndTime) && previous.StartTime.After(current.StartTime) {
		fragmentQuery.EndTime = previous.StartTime
		fullQuery.StartTime = current.StartTime
	}
	return &fragmentQuery, fullQuery
}
