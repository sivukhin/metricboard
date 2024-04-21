package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Metric struct {
	PanelId    string
	Type       MetricLineType
	Group      string
	Labels     map[string]string
	Timestamps []uint64
	Values     []float32
}

type MetricResult struct {
	PanelId string
	Metric  Metric
	Err     error
}

func SubscribeToPanels(
	ctx context.Context,
	dataSource DataSource,
	panelIds []string,
	commands <-chan MetricBoardCommands,
	results chan<- MetricResult,
) {
	var (
		activePanelIds      = panelIds
		activeQuery         *MetricQuery
		previousQueries     = make(map[string]*MetricQuery)
		previousQueriesLock sync.Mutex
		refreshInterval     time.Duration
	)
	previousCtx, previousCancel := context.WithCancel(context.Background())
	previousCancel()

	workerPool := NewWorkerPool(ctx, 1, 1024)
	workerPool.Start()
	defer workerPool.Stop()
loop:
	for {
		tick := time.Tick(refreshInterval)
		select {
		case <-ctx.Done():
			Logger.Infof("context cancelled")
			break loop
		case <-tick:
			Logger.Infof("period refresh triggered")
		case command, ok := <-commands:
			if !ok {
				break loop
			}
			if command.TimeUpdate != nil {
				Logger.Infof("receive time update command: %+v", *command.TimeUpdate)
				if command.TimeUpdate.Start <= 0 || command.TimeUpdate.End < 0 || command.TimeUpdate.Resolution <= 0 {
					results <- MetricResult{Err: fmt.Errorf("invalid time parameters: %+v", *command.TimeUpdate)}
					continue
				}
				if command.TimeUpdate.End != 0 && command.TimeUpdate.Start > command.TimeUpdate.End {
					results <- MetricResult{Err: fmt.Errorf("start > time: %+v", *command.TimeUpdate)}
					continue
				}
				if command.TimeUpdate.End != 0 && (command.TimeUpdate.End-command.TimeUpdate.Start)/command.TimeUpdate.Resolution > MaxPanelDataPoints {
					results <- MetricResult{Err: fmt.Errorf("too many data points requested(%v): %+v", (command.TimeUpdate.End-command.TimeUpdate.Start)/command.TimeUpdate.Resolution, *command.TimeUpdate)}
					continue
				}
				now := time.Now()
				if command.TimeUpdate.End == 0 && (now.UnixMicro()-command.TimeUpdate.Start)/command.TimeUpdate.Resolution > MaxPanelDataPoints {
					results <- MetricResult{Err: fmt.Errorf("too many data points requested(%v): %+v", (now.UnixMicro()-command.TimeUpdate.Start)/command.TimeUpdate.Resolution, *command.TimeUpdate)}
					continue
				}
				activeQuery = &MetricQuery{
					StartTime:  time.UnixMicro(command.TimeUpdate.Start),
					EndTime:    time.UnixMicro(command.TimeUpdate.End),
					Resolution: time.Duration(command.TimeUpdate.Resolution) * time.Microsecond,
				}
			}
			if command.ConcurrencyUpdate != nil {
				Logger.Infof("receive concurrency update command: %+v", *command.ConcurrencyUpdate)
				if *command.RefreshUpdate < 0 {
					results <- MetricResult{Err: fmt.Errorf("invalid concurrency parameter: %+v", *command.ConcurrencyUpdate)}
					continue
				}
				workerPool.Resize(*command.ConcurrencyUpdate)
			}
			if command.RefreshUpdate != nil {
				Logger.Infof("receive refresh update command: %+v", *command.RefreshUpdate)
				if *command.RefreshUpdate < 0 {
					results <- MetricResult{Err: fmt.Errorf("invalid refresh parameter: %+v", *command.RefreshUpdate)}
					continue
				}
				refreshInterval = time.Duration(*command.RefreshUpdate) * time.Microsecond
			}
			if command.PanelsUpdate != nil {
				Logger.Infof("receive panels update command: %+v", *command.PanelsUpdate)
				activePanelIds = command.PanelsUpdate.ActivePanelIds
				for _, panelId := range command.PanelsUpdate.ResetPanelIds {
					previousQueriesLock.Lock()
					previousQueries[panelId] = nil
					previousQueriesLock.Unlock()
				}
			}
		}

		if activeQuery == nil {
			Logger.Infof("no active query set, skip iteration")
			continue
		}

		if previousCtx.Err() == nil {
			Logger.Warnf("previous context cancelled although it wasn't fulfilled completely")
			previousCancel()
		}

		previousCtx, previousCancel = context.WithCancel(context.Background())
		currentCtx, currentCancel := previousCtx, previousCancel

		now := time.Now()
		trigger := NewTrigger(func() { currentCancel() })
		for _, panelId := range activePanelIds {
			previousQueriesLock.Lock()
			previousQuery := previousQueries[panelId]
			previousQueriesLock.Unlock()

			fragmentQuery, fullQuery := AdjustMetricQuery(now, previousQuery, *activeQuery)
			if fragmentQuery == nil {
				Logger.Infof("redundant query requested, skipping it: previous=%v, active=%v", previousQuery, activeQuery)
				continue
			}
			Logger.Infof("put metric query '%+v' for panel %v in queue", fragmentQuery, panelId)
			trigger.Add()
			go workerPool.Exec(func(ctx context.Context) {
				defer trigger.Done()

				ctx = CombineContexts(currentCtx, ctx)
				metrics := NewStreamingWriter[Metric](ctx, 0, func(metric Metric) { results <- MetricResult{PanelId: panelId, Metric: metric} })
				err := dataSource.GetMetric(ctx, panelId, *fragmentQuery, metrics)
				close(metrics)
				if err != nil {
					Logger.Errorf("data source failed: %v", err)
					results <- MetricResult{PanelId: panelId, Err: fmt.Errorf("data source failed")}
					return
				}
				previousQueriesLock.Lock()
				previousQueries[panelId] = &fullQuery
				previousQueriesLock.Unlock()
			})
		}
		trigger.Activate()
	}
}
