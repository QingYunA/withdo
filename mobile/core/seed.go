package core

import (
	"errors"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// SeedDemo writes app/data's demo workspace into dataDir/adhd-todo.db, for the debug build the
// reviewer starts with the intent extra demo=true. A database that already holds a task is left
// alone, so a second launch keeps whatever the reviewer changed.
//
// It opens the database on its own and closes it again; call it before Open.
func SeedDemo(dataDir string) error {
	if dataDir == "" {
		return errors.New("a data folder is required")
	}
	demo, err := data.OpenDemoAt(dataDir, time.Now, false)
	if err != nil {
		return errors.New("could not open the demo data")
	}
	defer func() { _ = demo.Close() }()
	snap, err := demo.Snapshot()
	if err != nil {
		return errors.New("could not read the demo data")
	}
	if len(snap.Tasks) > 0 {
		return nil
	}
	if err := demo.SeedDemo(time.Now()); err != nil {
		return errors.New("could not seed the demo data")
	}
	return nil
}
