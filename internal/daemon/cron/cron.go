package cron

import (
	"sync/atomic"
	"time"
	configdaemon "timon/internal/config/daemon"
	"timon/internal/daemon/cron/ctx"
	"timon/internal/daemon/cron/tasks"
	"timon/internal/log"
)

type task struct {
	name       string
	interval   time.Duration
	fn         func(ctx2 ctx.TaskContext) error
	runOnStart bool
}

func Start() {
	schedule := []task{
		{"CheckRules", 5 * time.Second, tasks.CheckRules, true},
	}

	cfg := configdaemon.GetConfig()
	if cfg != nil && len(cfg.Webhooks) > 0 {
		schedule = append(schedule, task{"SendWebhookCalls", 5 * time.Second, tasks.SendWebhookCalls, true})
		if cfg.Daemon.PingIntervalDuration != nil {
			schedule = append(schedule, task{"FireWebhookEventTimonPing", *cfg.Daemon.PingIntervalDuration, tasks.FireWebhookEventTimonPing, false})
		}
	}

	run := func(t task) {
		start := time.Now()
		err := t.fn(ctx.TaskContext{Name: t.name, Start: start})
		duration := time.Since(start).Round(time.Millisecond)
		if err != nil {
			_ = log.Cron.Errorf("%s failed (%s): %s", t.name, duration, err)
		}
	}

	for _, t := range schedule {
		go func(t task) {
			ticker := time.NewTicker(t.interval)
			defer ticker.Stop()

			var running atomic.Bool

			if t.runOnStart {
				run(t)
			}

			for range ticker.C {
				if !running.CompareAndSwap(false, true) {
					continue
				}

				go func() {
					defer running.Store(false)
					run(t)
				}()
			}
		}(t)
	}
}
