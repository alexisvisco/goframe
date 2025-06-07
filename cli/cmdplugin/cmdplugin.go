package cmdplugin

import (
	"context"
	"reflect"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func RunE(fn interface{}, modules ...fx.Option) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Vérifie que fn est une fonction
		fnType := reflect.TypeOf(fn)
		if fnType.Kind() != reflect.Func {
			panic("RunE: fn doit être une fonction")
		}

		var err error

		app := fx.New(
			fx.Supply(cmd),
			fx.Supply(args),

			fx.Options(modules...),

			fx.Invoke(fn),

			fx.Invoke(func(lifecycle fx.Lifecycle) {
				lifecycle.Append(fx.Hook{
					OnStop: func(ctx context.Context) error {
						return err
					},
				})
			}),
		)

		// Démarrer l'application fx
		startCtx := cmd.Context()
		if startCtx == nil {
			startCtx = context.Background()
		}

		if err := app.Start(startCtx); err != nil {
			return err
		}

		return app.Stop(startCtx)
	}
}
