package fileprinter

import (
	"io"

	"github.com/cresta/syncer/sharedapi/files"
)

type Printer interface {
	PrettyPrintDiffs(into io.Writer, toPrint *files.System[*files.DiffWithChangeReason]) error
}
