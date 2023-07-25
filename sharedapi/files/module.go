package files

import "go.uber.org/fx"

var Module = fx.Module("files", fx.Provide(
	NewTracker,
))
