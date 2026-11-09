← [Stage 1](PLAN_STAGE_1_TESTS.md) | Next → [Stage 3](PLAN_STAGE_3_MODULE_FILE.md)

# Stage 2 — parse `Rules` and `var Definition = model.ModuleDefinition{…}`

Nothing is generated yet; the tests of `tests/rules_parse_test.go` go green in this stage. Read `parse_definition.go` first (`parseDefinitionsInFile`, `parseDefinition`, `parseAllow`).

## 2.1 `generator.go` — new types (next to `StructInfo`)

Add to `StructInfo` after `Allow uint8`:

```go
	Rules             RulesInfo // Definition.Rules as written
	MFA               uint8     // Definition.MFA as written (a set of actions)
```

```go
// RulesInfo is Definition.Rules as written in definition.go: for each action the value of model.RuleMode (0 = undeclared).
type RulesInfo struct {
	Create, Read, Update, Delete uint8
}

// modelRules converts what was parsed into the model type, so definitioncheck.CheckRules is the one judge of a declaration.
func (r RulesInfo) modelRules() model.Rules {
	return model.Rules{
		Create: model.RuleMode(r.Create), Read: model.RuleMode(r.Read),
		Update: model.RuleMode(r.Update), Delete: model.RuleMode(r.Delete),
	}
}

// ModuleInfo is `var Definition = model.ModuleDefinition{…}` as written in definition.go.
type ModuleInfo struct {
	Name     string   // Definition.Name, e.g. "patient_directory"
	Package  string   // the Go package of definition.go
	Entities []string // the struct name of each listed model (PatientModel → "Patient"), in the order written
}
```

## 2.2 `parse_definition.go` — `Rules`, and one judge for a declaration

In the `switch key` of `parseDefinition`, add after `case "Allow":`

```go
		case "Rules":
			rules, err := parseRules(varName, kv.Value)
			if err != nil {
				return nil, err
			}
			info.Rules = rules
```

In the same `switch key` add a `case "MFA":` that stores `info.MFA` with `parseAllow(varName, kv.Value)` (the error text of `parseAllow` says `field Allow`; give `parseAllow` a `field string` parameter so it says `field MFA of …`, and update its other call site).

Delete the refusal `if info.Wire && info.Allow != 0 { … }` (a Wire model may now allow exactly `Read`, judged by `definitioncheck.CheckRules`). After the existing `info.NoDB && info.Allow != 0` block add:

```go
	// import "webtyp.com/definitioncheck" at the top of parse_definition.go
	def := model.Definition{Name: info.ModelName, Wire: info.Wire, Allow: model.Action(info.Allow), Rules: info.Rules.modelRules(), MFA: model.Action(info.MFA)}
	if err := definitioncheck.CheckRules(def); err != nil {
		return nil, fmt.Errf("%s: %s", varName, err.Error())
	}
	if info.NoDB && !info.Wire && info.Allow == uint8(model.Read) && info.Rules.Read != uint8(model.AddMyBusinessRule) {
		return nil, fmt.Errf("%s: a model without a table has no default read; declare Rules: model.Rules{Read: model.AddMyBusinessRule} and write its rule", varName)
	}
```

New functions in the same file:

```go
const errRulesShape = "field Rules of %s: write it as model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, …} (constants only)"

func parseRules(varName string, expr ast.Expr) (RulesInfo, error) {
	cl, ok := expr.(*ast.CompositeLit)
	if !ok {
		return RulesInfo{}, fmt.Errf(errRulesShape, varName)
	}
	var r RulesInfo
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return RulesInfo{}, fmt.Errf(errRulesShape, varName)
		}
		mode, err := parseRuleMode(varName, kv.Value)
		if err != nil {
			return RulesInfo{}, err
		}
		switch key := exprToString(kv.Key); key {
		case "Create":
			r.Create = mode
		case "Read":
			r.Read = mode
		case "Update":
			r.Update = mode
		case "Delete":
			r.Delete = mode
		default:
			return RulesInfo{}, fmt.Errf("field Rules of %s: %s is not an action; use Create, Read, Update or Delete", varName, key)
		}
	}
	return r, nil
}

func parseRuleMode(varName string, expr ast.Expr) (uint8, error) {
	switch exprToString(expr) {
	case "model.AddMyBusinessRule":
		return uint8(model.AddMyBusinessRule), nil
	case "model.HaveNoBusinessRule":
		return uint8(model.HaveNoBusinessRule), nil
	}
	return 0, fmt.Errf("field Rules of %s: write model.AddMyBusinessRule or model.HaveNoBusinessRule (constants only), got %s", varName, exprToString(expr))
}
```

(If `parseAllow` accepts an import alias of `model` that `exprToString` renders differently, copy its way of matching `model.X`; do not invent a second convention.)

## 2.3 `module_parse.go` (new file)

```go
package definitionc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"webtyp.com/fmt"
)

const (
	moduleVar          = "Definition"
	moduleDefinition   = "ModuleDefinition"
	modelSuffix        = "Model"
	errModuleNameShape = "Definition.Name is required: write Name: \"my_module\" (a string literal) in var Definition = model.ModuleDefinition{…}"
	errHandlersShape   = "Definition.Entities: write it as []model.Definition{PatientModel, VisitModel}"
)

// parseModuleInFile reads `var Definition = model.ModuleDefinition{…}` of the file at path; nil when the file declares none.
func (g *Generator) parseModuleInFile(path string) (*ModuleInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	for _, decl := range node.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if name.Name != moduleVar || i >= len(vs.Values) {
					continue
				}
				cl, ok := vs.Values[i].(*ast.CompositeLit)
				if !ok || !isModuleDefinitionType(cl.Type) {
					continue
				}
				return parseModule(node.Name.Name, cl)
			}
		}
	}
	return nil, nil
}

func isModuleDefinitionType(t ast.Expr) bool {
	s := exprToString(t)
	return s == moduleDefinition || strings.HasSuffix(s, "."+moduleDefinition)
}

func parseModule(pkg string, cl *ast.CompositeLit) (*ModuleInfo, error) {
	info := &ModuleInfo{Package: pkg}
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		switch exprToString(kv.Key) {
		case "Name":
			lit, ok := kv.Value.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return nil, fmt.Err(errModuleNameShape)
			}
			info.Name, _ = strconv.Unquote(lit.Value)
		case "Entities":
			list, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				return nil, fmt.Err(errHandlersShape)
			}
			for _, e := range list.Elts {
				id, ok := e.(*ast.Ident)
				if !ok || !strings.HasSuffix(id.Name, modelSuffix) || id.Name == modelSuffix {
					return nil, fmt.Errf("Definition.Entities: each handler must be written XModel (the variable of its model.Definition), got %s", exprToString(e))
				}
				info.Entities = append(info.Entities, strings.TrimSuffix(id.Name, modelSuffix))
			}
		}
	}
	if info.Name == "" {
		return nil, fmt.Err(errModuleNameShape)
	}
	return info, nil
}
```

`parseDefinitionsInFile` does **not** change: it keeps processing only variables whose name ends in `Model`, so `Definition` is never an entity (proved by `TestModuleDefinition_NotMistakenForAnEntity`).

`gotest -run 'TestRules_|TestModuleDefinition_'` passes. Commit: `feat: parse Rules and model.ModuleDefinition`.
