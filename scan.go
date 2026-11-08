package definitionc

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"webtyp.com/modfind"
	"webtyp.com/msgtype"
)

// ScanModules syncs the DB schema of every discovered module to the injected
// SchemaSyncer. Called once at startup by the tool (app), after SetSyncer.
//   - Writable module (main / replace): regenerate <file>_gen.go from definition.go,
//     then sync each DB struct.
//   - Read-only module (cache): parse the committed definition_gen.go, then sync.
//
// No-op if no syncer is injected (CLI codegen-only path).
func (g *Generator) ScanModules(rootDir string) error {
	if g.syncer == nil {
		g.log(msgtype.Debug, "ready (no db syncer)")
		return nil
	}
	g.log("scanning modules...")
	if g.finder == nil {
		g.finder = modfind.New()
	}
	mods, err := g.finder.Discover(rootDir)
	if err != nil {
		return err
	}
	for _, m := range mods {
		if m.Writable() {
			if err := g.scanWritableModule(m.Dir); err != nil {
				g.log("definitionc scan (writable)", m.Path, ":", err) // log-and-continue
			}
		} else {
			if err := g.scanReadonlyModule(m.Dir); err != nil {
				g.log("definitionc scan (readonly)", m.Path, ":", err)
			}
		}
	}
	return nil
}

func (g *Generator) scanWritableModule(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" || info.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		fileName := info.Name()
		if fileName == DefinitionFile {
			infos, err := g.parseDefinitionsInFile(path)
			if err != nil {
				return nil // skip unparseable
			}

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return nil // skip unparseable
			}
			if err := g.resolveStorage(infos, node); err != nil {
				return err
			}

			// Merge into cache
			for _, info := range infos {
				g.cache[info.Name] = info
			}

			// Generate in-place
			if err := g.GenerateForFile(infos, path); err != nil {
				return err
			}

			// Sync
			for _, info := range infos {
				if !info.NoDB {
					if err := g.syncer.SyncSchema(info.ModelName, info.asFields()); err != nil {
						g.log("sync error for", info.ModelName, ":", err)
					}
				}
			}
		}
		return nil
	})
}

func (g *Generator) scanReadonlyModule(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" || info.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, GeneratedSuffix) {
			schemas, err := parseGenerated(path)
			if err != nil {
				g.log("failed to parse generated file", path, ":", err)
				return nil
			}
			for table, fields := range schemas {
				if err := g.syncer.SyncSchema(table, fields); err != nil {
					g.log("sync error (readonly) for", table, ":", err)
				}
			}
		}
		return nil
	})
}
