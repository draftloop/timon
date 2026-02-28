package tasks

import (
	"fmt"
	"github.com/draftloop/elm"
	"time"
	"timon/internal/daemon/cron/ctx"
	database "timon/internal/daemon/db"
	"timon/internal/daemon/models"
)

func SendWebhookCalls(taskContext ctx.TaskContext) error {
	db := database.GetDB()

	var tasks []models.WebhookCall
	if err := db.Model(models.WebhookCall{}).
		Where(elm.LtEq("next_try_at", time.Now())).
		UnsafeOrderBy("id ASC").
		Scan(&tasks); err != nil {
		return fmt.Errorf("error fetching webhook calls: %w", err)
	}
	for _, task := range tasks {
		err := task.Send()
		if err != nil {
			taskContext.LogErr("error sending webhook call: %s", err)
		}
	}

	return nil
}
