package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"fmt"
)

func main() {
	src := `package main
var PatientModel = model.Definition{
	Name: "patient",
	Allow: model.Create | model.Read | model.Update,
}
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "", src, 0)
	ast.Inspect(f, func(n ast.Node) bool {
		if kv, ok := n.(*ast.KeyValueExpr); ok {
			if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Allow" {
				val, err := ParseAllow(kv.Value)
				fmt.Printf("val: %d, err: %v\n", val, err)
			}
		}
		return true
	})
}

func ParseAllow(expr ast.Expr) (uint8, error) {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		switch t.Sel.Name {
		case "Create":
			return 1, nil
		case "Read":
			return 2, nil
		case "Update":
			return 4, nil
		case "Delete":
			return 8, nil
		}
		return 0, fmt.Errorf("field Allow: write it as model.Create | model.Read | … (constants only)")
	case *ast.BinaryExpr:
		if t.Op != token.OR {
			return 0, fmt.Errorf("field Allow: write it as model.Create | model.Read | … (constants only)")
		}
		left, err := ParseAllow(t.X)
		if err != nil {
			return 0, err
		}
		right, err := ParseAllow(t.Y)
		if err != nil {
			return 0, err
		}
		return left | right, nil
	}
	return 0, fmt.Errorf("field Allow: write it as model.Create | model.Read | … (constants only)")
}
