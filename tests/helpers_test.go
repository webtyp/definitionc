package tests

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"webtyp.com/definitionc"
)

// generate writes a module in a temp dir (go.mod requiring crud, the given definition.go, an optional extra file) and runs definitionc on it.
func generate(t *testing.T, definition string, extra ...file) (dir string, err error) {
	t.Helper()
	dir = t.TempDir()
	write(t, dir, "go.mod", "module example.com/mod\n\ngo 1.26.8\n\nrequire webtyp.com/crud v0.0.10\n")
	write(t, dir, "definition.go", definition)
	for _, f := range extra {
		write(t, dir, f.name, f.body)
	}
	return dir, runGenerator(dir)
}

// runGenerator runs definitionc on dir without tidying the module.
func runGenerator(dir string) error {
	gen := definitionc.New()
	gen.SetSkipTidy(true)
	gen.SetRootDir(dir)
	return gen.Run()
}

type file struct{ name, body string }

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// squash collapses every run of spaces and tabs to one space, so gofmt's alignment never decides a test.
func squash(s string) string { return strings.Join(strings.Fields(s), " ") }

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// mustParse fails the test when src is not valid Go.
func mustParse(t *testing.T, src string) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", src, 0); err != nil {
		t.Fatalf("generated file does not parse: %v\n%s", err, src)
	}
}

const moduleType = "package mod\n\ntype Module struct{}\n" // the hand-written type the generated methods hang on

// moduleSource is a definition.go whose Definition lists entities, followed by the model declarations.
func moduleSource(entities, models string) string {
	return "package mod\n\nimport \"webtyp.com/model\"\n\n" +
		"var Definition = model.ModuleDefinition{Name: \"patient_directory\", Entities: []model.Definition{" + entities + "}}\n\n" +
		models
}

// patientModel is a PatientModel that allows Create, Read and Update, each with a business rule.
const patientModel = `var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Create | model.Read | model.Update,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.AddMyBusinessRule, Update: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), Item: model.ItemLabel},
	},
}
`

// patientDef is definition.go for one patient entity with every action a business rule.
var patientDef = moduleSource("PatientModel", patientModel)

// healthModel is a model without a table: a report that is read through its own rule.
const healthModel = `var HealthModel = model.Definition{
	Name:  "health",
	Allow: model.Read,
	Rules: model.Rules{Read: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "status", Type: model.Text()},
	},
}
`

// visitModel is a second table entity with the standard actions only.
const visitModel = `var VisitModel = model.Definition{
	Name:  "visit",
	Allow: model.Create | model.Read,
	Rules: model.Rules{Create: model.HaveNoBusinessRule, Read: model.HaveNoBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "code", Type: model.Text()},
	},
}
`
