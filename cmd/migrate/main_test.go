package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitStatements_StripsCommentsAndBlankStatements(t *testing.T) {
	sql := `-- a comment
CREATE TABLE a (x INT);

-- another comment
CREATE TABLE b (y INT);
`
	stmts := splitStatements(sql)
	if len(stmts) != 2 {
		t.Fatalf("got %d statements, want 2: %+v", len(stmts), stmts)
	}
	if stmts[0] != "CREATE TABLE a (x INT)" {
		t.Fatalf("unexpected first statement: %q", stmts[0])
	}
	if stmts[1] != "CREATE TABLE b (y INT)" {
		t.Fatalf("unexpected second statement: %q", stmts[1])
	}
}

func TestSplitStatements_AgainstRealMigrationFile(t *testing.T) {
	root := findRepoRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "db/questdb/migrations/*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no migration files found under db/questdb/migrations: %v", err)
	}
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		stmts := splitStatements(string(content))
		if len(stmts) == 0 {
			t.Fatalf("%s produced zero statements", f)
		}
		for _, s := range stmts {
			if len(s) < 10 {
				t.Fatalf("%s produced a suspiciously short statement: %q", f, s)
			}
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not find repo root (go.mod) from", dir)
	return ""
}
