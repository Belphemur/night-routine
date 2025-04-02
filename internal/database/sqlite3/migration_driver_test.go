package sqlite3

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/golang-migrate/migrate/v4"
	dt "github.com/golang-migrate/migrate/v4/database/testing"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs"
)

func Test(t *testing.T) {
	p := &Sqlite{}
	addr := "file::memory:?mode=memory"
	t.Logf("Using in-memory database: %s", addr)
	d, err := p.Open(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close() // Ensure driver connection is closed
	dt.Test(t, d, []byte("CREATE TABLE t (Qty int, Name string);"))
}

func TestMigrate(t *testing.T) {
	addr := "file::memory:?mode=memory"
	t.Logf("Using in-memory database: %s", addr)

	db, err := sql.Open("sqlite3", addr)
	if err != nil {
		t.Fatal(err)
	}
	// Defer closing db first, then migrator (LIFO order means m closes first)
	defer db.Close()
	driver, err := WithInstance(db, &Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./examples/migrations",
		"ql", driver)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	dt.TestMigrate(t, m)
}

func TestMigrationTable(t *testing.T) {
	addr := "file::memory:?mode=memory"
	t.Logf("Using in-memory database: %s", addr)

	db, err := sql.Open("sqlite3", addr)
	if err != nil {
		t.Fatal(err)
	}
	// Defer closing db first, then migrator (LIFO order means m closes first)
	defer db.Close()

	config := &Config{
		MigrationsTable: "my_migration_table",
	}
	driver, err := WithInstance(db, config)
	if err != nil {
		t.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://./examples/migrations",
		"ql", driver)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	t.Log("UP")
	err = m.Up()
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Query(fmt.Sprintf("SELECT * FROM %s", config.MigrationsTable))
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoTxWrap(t *testing.T) {
	p := &Sqlite{}
	addr := "file::memory:?mode=memory&x-no-tx-wrap=true"
	t.Logf("Using in-memory database with no tx wrap: %s", addr)
	d, err := p.Open(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close() // Ensure driver connection is closed
	// An explicit BEGIN statement would ordinarily fail without x-no-tx-wrap.
	// (Transactions in sqlite may not be nested.)
	dt.Test(t, d, []byte("BEGIN; CREATE TABLE t (Qty int, Name string); COMMIT;"))
}

func TestNoTxWrapInvalidValue(t *testing.T) {
	p := &Sqlite{}
	addr := "file::memory:?mode=memory&x-no-tx-wrap=yeppers"
	t.Logf("Using in-memory database with invalid no tx wrap: %s", addr)
	_, err := p.Open(addr)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "x-no-tx-wrap")
		assert.Contains(t, err.Error(), "invalid syntax")
	}
}

func TestMigrateWithDirectoryNameContainsWhitespaces(t *testing.T) {
	p := &Sqlite{}
	addr := "file::memory:?mode=memory"
	t.Logf("Using in-memory database: %s", addr)
	d, err := p.Open(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close() // Ensure driver connection is closed
	dt.Test(t, d, []byte("CREATE TABLE t (Qty int, Name string);"))
}
