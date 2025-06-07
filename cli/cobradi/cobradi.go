package cobradi

import (
	"context"
	"reflect"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// RunE is a helper function that wraps a function to be executed within an fx application.
// It allows you to pass the command and its arguments as parameters to the function.
func RunE(fn any, modules ...fx.Option) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fnType := reflect.TypeOf(fn)
		if fnType.Kind() != reflect.Func {
			panic("RunE: fn must be a function")
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
