package consoleprinter

import (
	"fmt"
	"io"

	"github.com/andreyvit/diff"
	"github.com/getsyncer/syncer/sharedapi/files"
	"github.com/getsyncer/syncer/sharedapi/files/fileprinter"
)

type consolePrinterImpl struct{}

func (y *consolePrinterImpl) PrettyPrintDiffs(into io.Writer, toPrint *files.System[*files.DiffWithChangeReason]) error {
	paths := toPrint.Paths()
	for _, path := range paths {
		file := toPrint.Get(path)
		if _, err := fmt.Fprintf(into, "----- File: %s\n", path); err != nil {
			return fmt.Errorf("failed to write path: %w", err)
		}
		switch file.Diff.DiffResult.DiffAction {
		case files.DiffActionDelete:
			if _, err := fmt.Fprintf(into, "  Delete\n"); err != nil {
				return fmt.Errorf("failed to write delete: %w", err)
			}
		case files.DiffActionNoChange:
			if _, err := fmt.Fprintf(into, "  No Change\n"); err != nil {
				return fmt.Errorf("failed to write no change: %w", err)
			}
		case files.DiffActionCreate:
			fallthrough
		case files.DiffActionUpdate:
			if _, err := fmt.Fprintf(into, "%s\n", file.Diff.DiffResult.DiffAction); err != nil {
				return fmt.Errorf("failed to write create: %w", err)
			}
			if file.Diff.DiffResult.ModeToChangeTo != nil {
				if _, err := fmt.Fprintf(into, "Mode: %s\n", file.Diff.DiffResult.ModeToChangeTo.String()); err != nil {
					return fmt.Errorf("failed to write mode: %w", err)
				}
			}
			if file.Diff.DiffResult.ContentsToChangeTo != nil {
				// https://github.com/sergi/go-diff/issues/69#issuecomment-688602689
				x2 := diff.LineDiff(string(file.Diff.OldFileState.Contents), string(file.Diff.NewFileState.Contents))
				if _, err := fmt.Fprintf(into, "Contents Diff\n%s\n", x2); err != nil {
					return fmt.Errorf("failed to write contents: %w", err)
				}
			}
		case files.DiffActionUnset:
			panic("unreachable")
		}
	}
	return nil
}

var _ fileprinter.Printer = &consolePrinterImpl{}

func NewConsolePrinter() fileprinter.Printer {
	return &consolePrinterImpl{}
}
