package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// countingDriver wraps the sqlite driver and counts the statements that reach it.
type countingDriver struct {
	inner driver.Driver
	n     *atomic.Int64
}

func (d countingDriver) Open(name string) (driver.Conn, error) {
	c, err := d.inner.Open(name)
	if err != nil {
		return nil, err
	}
	return &countingConn{Conn: c, n: d.n}, nil
}

type countingConn struct {
	driver.Conn
	n *atomic.Int64
}

func (c *countingConn) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	c.n.Add(1)
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, args)
}

func (c *countingConn) ExecContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	c.n.Add(1)
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, q, args)
}

func (c *countingConn) PrepareContext(ctx context.Context, q string) (driver.Stmt, error) {
	c.n.Add(1)
	return c.Conn.(driver.ConnPrepareContext).PrepareContext(ctx, q)
}

func (c *countingConn) BeginTx(ctx context.Context, o driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, o)
}

var countedDrivers atomic.Int32

// countedDemo is a seeded demo store whose statements are counted. fill adds rows first; then
// the same file is opened again through a driver that counts what it is asked, from zero.
func countedDemo(t *testing.T, fill func(*Real)) (*Real, *atomic.Int64) {
	t.Helper()
	plain := openDemo(t, fixedNow, true)
	if fill != nil {
		fill(plain)
	}
	var path string
	if err := plain.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name = 'main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	probe, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	inner := probe.Driver()
	_ = probe.Close()
	n := new(atomic.Int64)
	name := "sqlite-counted-" + strconv.Itoa(int(countedDrivers.Add(1)))
	sql.Register(name, countingDriver{inner: inner, n: n})
	db, err := sql.Open(name, "file:"+filepath.ToSlash(path)+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	n.Store(0)
	return NewReal(db, RealOptions{Now: fixedNow, DataFolder: filepath.Dir(path)}), n
}
