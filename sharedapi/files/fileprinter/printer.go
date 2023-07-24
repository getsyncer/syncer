package fileprinter

import (
	"io"

	"github.com/getsyncer/syncer/sharedapi/files"
)

type Printer interface {
	PrettyPrintDiffs(into io.Writer, toPrint *files.System[*files.DiffWithChangeReason]) error
}
