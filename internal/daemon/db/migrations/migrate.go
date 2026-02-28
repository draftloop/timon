package migrations

import (
	"fmt"
	"github.com/draftloop/elm"
)

type Migration struct {
	Version int64
	Up      string
}

func Migrate(elm *elm.Elm, migrations []Migration) ([]int64, error) {
	var applied []int64

	_, err := elm.Exec(`CREATE TABLE IF NOT EXISTS _migrations ( version INTEGER PRIMARY KEY, applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP )`)
	if err != nil {
		return applied, fmt.Errorf("migrations preparation: %w", err)
	}

	for _, migration := range migrations {
		var count int64
		if err := elm.QueryRow("SELECT COUNT(*) FROM _migrations WHERE version = ?", migration.Version).Scan(&count); err != nil {
			return applied, fmt.Errorf("pre-migration %d: %w", migration.Version, err)
		} else if count > 0 {
			continue
		}

		if _, err := elm.Exec(migration.Up); err != nil {
			return applied, fmt.Errorf("migration %d: %w", migration.Version, err)
		}

		if _, err := elm.Exec("INSERT INTO _migrations (version) VALUES (?)", migration.Version); err != nil {
			return applied, fmt.Errorf("post-migration %d: %w", migration.Version, err)
		}

		applied = append(applied, migration.Version)
	}

	return applied, nil
}
