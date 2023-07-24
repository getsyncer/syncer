package osfiles

import (
	"github.com/getsyncer/syncer/sharedapi/files"
	"go.uber.org/fx"
)

var Module = fx.Module("osfiles",
	fx.Provide(
		fx.Annotate(
			newOsLoader,
			fx.As(new(files.StateLoader)),
		),
		fx.Annotate(
			newOsLoader,
			fx.As(new(files.DiffExecutor)),
		),
	),
)
