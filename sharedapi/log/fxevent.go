package log

import (
	"bytes"
	"context"
	"strings"

	"github.com/cresta/zapctx"
	"go.uber.org/fx/fxevent"
)

type FxLogger struct {
	logger *zapctx.Logger
}

func (f *FxLogger) LogEvent(event fxevent.Event) {
	ctx := context.Background()
	switch e := event.(type) {
	case *fxevent.Started:
		if e.Err != nil {
			f.logger.IfErr(e.Err).Error(ctx, "Failed to start")
		} else {
			f.logger.Debug(ctx, "Started")
		}
	case *fxevent.Invoked:
		if e.Err != nil {
			f.logger.IfErr(e.Err).Error(ctx, "Failed to invoke")
		} else {
			f.logger.Debug(ctx, "Invoked")
		}
	default:
		var buf bytes.Buffer
		cl := fxevent.ConsoleLogger{W: &buf}
		cl.LogEvent(event)
		if buf.Len() > 0 {
			f.logger.Debug(ctx, strings.TrimSpace(buf.String()))
		}
	}
}

func NewFxLogger(logger *zapctx.Logger) fxevent.Logger {
	return &FxLogger{
		logger: logger,
	}
}
