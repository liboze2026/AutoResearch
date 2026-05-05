package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.PingContext(context.Background()); err != nil {
		return nil, err
	}
	return db, nil
}

func RunMigrations(db *sql.DB, path string) error {
	files, err := migrationFiles(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			return readErr
		}
		statements := strings.Split(string(content), ";")
		for _, s := range statements {
			q := strings.TrimSpace(s)
			if q == "" {
				continue
			}
			if _, err = db.Exec(q); err != nil {
				return fmt.Errorf("migration failed: %w; file=%s; query=%s", err, file, q)
			}
		}
	}
	return nil
}

func migrationFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			continue
		}
		files = append(files, filepath.Join(path, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}
