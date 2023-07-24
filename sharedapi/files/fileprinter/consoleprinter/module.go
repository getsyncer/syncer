package consoleprinter

import (
	"github.com/getsyncer/syncer/sharedapi/files/fileprinter"
	"go.uber.org/fx"
)

var Module = fx.Module("fileprinter",
	fx.Provide(
		fx.Annotate(
			NewConsolePrinter,
			fx.As(new(fileprinter.Printer)),
		),
	),
)
