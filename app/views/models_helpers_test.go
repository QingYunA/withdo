package views

import (
	"path/filepath"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/providers"
)

// fakeModelsIn is Settings > Models over a provider file in dir and keys in memory.
func fakeModelsIn(dir string) *data.FakeModels {
	mgr, err := providers.NewManager(providers.NewStore(filepath.Join(dir, providers.ConfigFile)), providers.NewMemoryKeyring(), nil)
	if err != nil {
		panic(err)
	}
	return &data.FakeModels{ProviderSettings: data.ProviderSettings{Provider: mgr}}
}
