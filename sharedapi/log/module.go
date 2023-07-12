package log

import (
	"go.uber.org/fx"
)

const FxTagLogDynamicFields = `group:"dynamic_fields"`

var Module = fx.Module("logger",
	fx.Provide(
		ZapLoggerFromConfig,
		fx.Annotate(
			New,
			fx.ParamTags("", FxTagLogDynamicFields),
		),
		ZapLoggerConfigFromEnv,
	),
)
