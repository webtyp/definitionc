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
