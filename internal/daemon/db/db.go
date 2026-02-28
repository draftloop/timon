package db

import (
	"fmt"
	"github.com/draftloop/elm"
	_ "modernc.org/sqlite"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"timon/internal/config/daemon"
	"timon/internal/daemon/db/migrations"
	"timon/internal/log"
)

var (
	staticDb *elm.Elm
	once     sync.Once
)

func GetDB() *elm.Elm {
	once.Do(func() {
		var err error
		dbFilePath := filepath.Join(configdaemon.GetConfig().Daemon.DataDir, "timon.db")

		db, err := elm.Open("sqlite", dbFilePath, elm.Config{
			Logger: func(query string, args []any, duration time.Duration, err error) {
				formatArgs := func(args []any) string {
					if len(args) == 0 {
						return ""
					}
					parts := make([]string, len(args))
					for i, a := range args {
						parts[i] = fmt.Sprintf("%#v", a)
					}
					return " [" + strings.Join(parts, ", ") + "]"
				}
				if err != nil {
					_ = log.SQL.Errorf("%s (%s) — %s", query+formatArgs(args), duration.Round(time.Millisecond), err)
				} else {
					log.SQL.Debugf("%s (%s)", query+formatArgs(args), duration.Round(time.Millisecond))
				}
			},
		})
		if err != nil {
			_ = log.Daemon.Errorf("database unavailable: %v", err)
			panic(fmt.Sprintf("database unavailable: %v", err))
		}

		if err := db.Ping(); err != nil {
			_ = log.Daemon.Errorf("database ping failed: %v", err)
			panic(fmt.Sprintf("database ping failed: %v", err))
		}

		appliedMigrationsVersions, err := migrations.Migrate(db, []migrations.Migration{
			migrations.M001Init,
		})
		if len(appliedMigrationsVersions) != 0 {
			log.Daemon.Infof("migrations applied: %v", appliedMigrationsVersions)
		}
		if err != nil {
			_ = log.Daemon.Errorf("migrations: %v", err)
		}
		if len(appliedMigrationsVersions) == 0 {
			log.Daemon.Info("migrations: nothing to apply")
		}

		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		log.Daemon.Debugf("database initialized on %s", dbFilePath)

		staticDb = db
	})

	return staticDb
}
