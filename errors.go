package definitionc

import "webtyp.com/fmt"

// ErrNoModelsFound is returned by ExportSQL when the target directory has no
// definition.go files (or none define an exported, non-NoDB model).
// Signals "nothing to export" distinctly from a successful empty schema.
var ErrNoModelsFound = fmt.Err("no", "models", "found")
