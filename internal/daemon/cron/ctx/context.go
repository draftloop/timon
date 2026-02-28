package ctx

import (
	"time"
	"timon/internal/log"
)

type TaskContext struct {
	Name  string
	Start time.Time
}

func (ctx TaskContext) Log(format string, args ...any) {
	log.Cron.Debugf(ctx.Name+": "+format, args...)
}

func (ctx TaskContext) LogErr(format string, args ...any) {
	_ = log.Cron.Errorf(ctx.Name+": "+format, args...)
}
