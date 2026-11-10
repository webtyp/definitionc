package definitionc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"webtyp.com/fmt"
	"webtyp.com/model"
)

const (
	legacyModuleSuffix = "_crud.go" // the second generated file of the previous layout; removed on regeneration
	moduleTypeName     = "Module"
)

// moduleCode is what the module declaration adds to definition_gen.go.
type moduleCode struct {
	info     *ModuleInfo
	entities []StructInfo
	imports  []string // packages the module code needs, beyond what the entities need
	body     string   // declarations only: no header, no package clause, no imports
}

// isStoredEntity: an entity crud persists through a Store (it has a key and is not a transport format).
func isStoredEntity(e StructInfo) bool { return !e.NoDB && !e.Wire && e.Allow != 0 }

// isReport: an entity that only reads and is answered by its own rule (no table, or a Wire format).
func isReport(e StructInfo) bool { return (e.NoDB || e.Wire) && e.Allow == uint8(model.Read) }

// moduleCode reads `var Definition = model.ModuleDefinition{…}` of sourceFile. A file with no module yields an empty moduleCode
// (nothing is appended); a module that disagrees with its entities, or has no `type Module`, is an error.
func (o *Generator) moduleCode(infos []StructInfo, sourceFile string) (moduleCode, error) {
	mod, err := o.parseModuleInFile(sourceFile)
	if err != nil || mod == nil {
		return moduleCode{}, err
	}
	if err := checkModule(mod, infos, filepath.Dir(sourceFile)); err != nil {
		return moduleCode{}, err
	}
	entities := inEntityOrder(mod, infos)
	return moduleCode{info: mod, entities: entities, imports: moduleImports(entities), body: renderModule(mod, entities)}, nil
}

// finishModule removes the legacy second file and creates the missing rule skeletons of the module.
func (o *Generator) finishModule(code moduleCode, sourceFile string) error {
	legacy := fmt.Convert(sourceFile).TrimSuffix(".go").String() + legacyModuleSuffix
	if err := os.Remove(legacy); err != nil && !os.IsNotExist(err) {
		return err
	}
	if code.info == nil {
		return nil
	}
	if err := writeRuleSkeletons(code.entities, filepath.Dir(sourceFile)); err != nil {
		return err
	}
	return writeCallSkeletons(code.entities, filepath.Dir(sourceFile))
}

// moduleImports is every package renderModule's text names: crud always, storage when a table has a Search rule.
func moduleImports(entities []StructInfo) []string {
	out := []string{"webtyp.com/crud"}
	for _, e := range entities {
		if isStoredEntity(e) && e.Rules.Read == uint8(model.AddMyBusinessRule) {
			return append(out, "webtyp.com/storage")
		}
	}
	return out
}

// checkModule refuses a module whose listed handlers and declared entities disagree, or that has no Module type.
func checkModule(mod *ModuleInfo, infos []StructInfo, dir string) error {
	declared := make(map[string]bool, len(infos))
	for _, e := range infos {
		declared[e.Name] = true
	}
	listed := make(map[string]bool, len(mod.Entities))
	for _, h := range mod.Entities {
		if !declared[h] {
			return fmt.Errf("Definition.Entities lists %sModel, but no model.Definition named %sModel is declared in definition.go", h, h)
		}
		listed[h] = true
	}
	for _, e := range infos {
		if e.Allow != 0 && !listed[e.Name] {
			return fmt.Errf("%sModel allows actions but is not listed in Definition.Entities; add it there, or its operations are never mounted", e.Name)
		}
	}
	has, err := declaresType(dir, moduleTypeName)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errf("%s declares a module (var Definition) but the package has no `type Module`; write it in module.go: the generated methods hang on it", dir)
	}
	return nil
}

// declaresType reports whether a non-test .go file of dir declares `type name`.
func declaresType(dir, name string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, f := range files {
		fn := f.Name()
		if f.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}
		node, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, fn), nil, 0)
		if err != nil {
			return false, err
		}
		if typeDeclared(node, name) {
			return true, nil
		}
	}
	return false, nil
}

// typeDeclared reports whether the file declares a top-level `type name`.
func typeDeclared(node *ast.File, name string) bool {
	for _, decl := range node.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == name {
				return true
			}
		}
	}
	return false
}

// inEntityOrder returns the entities of the module in the order of Definition.Entities.
func inEntityOrder(mod *ModuleInfo, infos []StructInfo) []StructInfo {
	out := make([]StructInfo, 0, len(mod.Entities))
	for _, h := range mod.Entities {
		for _, e := range infos {
			if e.Name == h {
				out = append(out, e)
			}
		}
	}
	return out
}

// renderModule returns the declarations of the module (appended to definition_gen.go, whose header, package clause and imports already exist).
func renderModule(mod *ModuleInfo, entities []StructInfo) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("// ModelName is the module identity: Definition.Name.\nconst ModelName = %q\n\n", mod.Name))
	b.WriteString("func (m *Module) ModelName() string { return ModelName }\n\n")
	b.WriteString("// Definition is the module's declaration (definition.go); crud.Mount compares it with what MountOperations mounted.\n")
	b.WriteString("func (m *Module) Definition() model.ModuleDefinition { return Definition }\n\n")
	b.WriteString("// MountOperations offers every entity of the module: the actions its model allows, with crud's validation and the rules written in rule_*.go.\n")
	b.WriteString("func (m *Module) MountOperations(reg crud.Registry) {\n")
	for _, e := range entities {
		switch {
		case isStoredEntity(e):
			renderStoredEntity(&b, e)
		case isReport(e):
			renderReport(&b, e)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func renderStoredEntity(b *strings.Builder, e StructInfo) {
	n := e.Name
	b.WriteString("\treg.AddStoredEntity(crud.StoredEntity{\n")
	b.WriteString(fmt.Sprintf("\t\tModel: &%s{},\n", n))
	b.WriteString(fmt.Sprintf("\t\tNewList: func() model.ModelSlice { return &%sList{} },\n", n))
	b.WriteString(fmt.Sprintf("\t\tStore: m.Store%s(),\n", n))
	b.WriteString(fmt.Sprintf("\t\tRules: %sModel.Rules,\n", n))
	b.WriteString(fmt.Sprintf("\t\tMFA: %sModel.MFA,\n", n))
	domain := uint8(model.AddMyBusinessRule)
	if e.Rules.Create == domain {
		b.WriteString(fmt.Sprintf("\t\tCreate: func(a crud.CurrentUser, tx crud.Tx, rec model.Model) (crud.RuleAnswer, error) {\n\t\t\treturn rec.(*%s).Create(a, tx, m)\n\t\t},\n", n))
	}
	if e.Rules.Update == domain {
		b.WriteString(fmt.Sprintf("\t\tUpdate: func(a crud.CurrentUser, tx crud.Tx, old, next model.Model, fields []string) (crud.RuleAnswer, error) {\n\t\t\treturn next.(*%s).Update(a, tx, old.(*%s), fields, m)\n\t\t},\n", n, n))
	}
	if e.Rules.Delete == domain {
		b.WriteString(fmt.Sprintf("\t\tDelete: func(a crud.CurrentUser, tx crud.Tx, rec model.Model) (crud.RuleAnswer, error) {\n\t\t\treturn rec.(*%s).Delete(a, tx, m)\n\t\t},\n", n))
	}
	if e.Rules.Read == domain {
		b.WriteString(fmt.Sprintf("\t\tSearch: func(a crud.CurrentUser, filter model.Model, text string) ([]storage.Condition, error) {\n\t\t\treturn (&%s{}).Search(a, filter.(*%s), text, m)\n\t\t},\n", n, n))
	}
	if len(e.ExternalCalls) > 0 {
		b.WriteString(fmt.Sprintf("\t\tExternalCalls: %sModel.ExternalCalls,\n", n))
		b.WriteString("\t\tCalls: map[string]crud.ExternalCallFunc{\n")
		for _, call := range e.ExternalCalls {
			methodName := fmt.Convert(call.Name).CamelUp().String()
			b.WriteString(fmt.Sprintf("\t\t\t%q: func(a crud.CurrentUser, rec model.Model, key crud.IdempotencyKey) (crud.FieldsToSave, error) {\n", call.Name))
			b.WriteString(fmt.Sprintf("\t\t\t\treturn rec.(*%s).%s(a, m, key)\n", n, methodName))
			b.WriteString("\t\t\t},\n")
		}
		b.WriteString("\t\t},\n")
	}
	b.WriteString("\t})\n")
}

func renderReport(b *strings.Builder, e StructInfo) {
	b.WriteString(fmt.Sprintf("\treg.AddReadOnlyEntity(&%s{}, func() model.Model { return &%s{} }, func(a crud.CurrentUser, args crud.ReadArgs, filter model.Model) (model.ModelSlice, error) {\n\t\treturn (&%s{}).Read(a, filter.(*%s), args, m)\n\t})\n", e.Name, e.Name, e.Name, e.Name))
}
