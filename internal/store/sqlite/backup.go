package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/datadir"
)

const backupPrefix = "specmetry-"

// backup writes a consistent copy of the database with VACUUM INTO (never a
// raw file copy of a live WAL database) and keeps the newest `keep` backups.
func backup(ctx context.Context, db *sql.DB, dir datadir.Dir, from, to int, now time.Time, keep int) (string, error) {
	backupDir, err := dir.EnsureSub(datadir.BackupsDir)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%sv%d-to-v%d-%s.db", backupPrefix, from, to, now.UTC().Format("20060102T150405.000000000Z"))
	target := filepath.Join(backupDir, name)
	// VACUUM INTO accepts an existing empty file; creating it first gives it
	// owner-only permissions.
	f, err := datadir.CreatePrivateFile(target, true)
	if err != nil {
		return "", err
	}
	_ = f.Close()
	if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, target); err != nil {
		_ = os.Remove(target)
		return "", err
	}
	return target, pruneBackups(backupDir, keep)
}

func pruneBackups(backupDir string, keep int) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), backupPrefix) && strings.HasSuffix(e.Name(), ".db") {
			names = append(names, e.Name())
		}
	}
	// The UTC timestamp suffix sorts lexically in chronological order.
	sort.Slice(names, func(i, j int) bool { return backupTime(names[i]) < backupTime(names[j]) })
	for len(names) > keep {
		if err := os.Remove(filepath.Join(backupDir, names[0])); err != nil {
			return err
		}
		names = names[1:]
	}
	return nil
}

func backupTime(name string) string {
	i := strings.LastIndex(name, "-")
	if i < 0 {
		return name
	}
	return name[i+1:]
}
