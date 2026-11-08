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
				fmt.Printf("Allow found: %T\n", kv.Value)
				parseAllow(kv.Value)
			}
		}
		return true
	})
}

func parseAllow(expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		fmt.Println("SelectorExpr:", t.Sel.Name)
	case *ast.BinaryExpr:
		fmt.Println("BinaryExpr:", t.Op)
		parseAllow(t.X)
		parseAllow(t.Y)
	default:
		fmt.Printf("Other: %T\n", expr)
	}
}
