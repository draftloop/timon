package tasks

import (
	"timon/internal/daemon/cron/ctx"
	"timon/internal/daemon/models"
)

func FireWebhookEventTimonPing(taskContext ctx.TaskContext) error {
	models.FireWebhookEventTimonPing()
	return nil
}
