package git

import "go.uber.org/fx"

var Module = fx.Module("cli",
	fx.Provide(
		fx.Annotate(
			NewGitOs,
			fx.As(new(Git)),
		),
	),
)
