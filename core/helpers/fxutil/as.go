package fxutil

import "go.uber.org/fx"

// fx.Annotate(storage.NewRepository(cfg.GetDatabase()), fx.As(new(contracts.StorageRepository))),

func As(value any, to ...any) any {
	return fx.Annotate(value, fx.As(to...))
}
