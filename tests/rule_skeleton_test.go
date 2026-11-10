package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const completeMarker = "// COMPLETA TU REGLA DE NEGOCIO AQUÍ"

// fullPatient is a PatientModel that allows all four actions, each with its own business rule.
const fullPatient = `var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Create | model.Read | model.Update | model.Delete,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.AddMyBusinessRule, Update: model.AddMyBusinessRule, Delete: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), Item: model.ItemLabel},
	},
}
`

func TestSkeleton_CreatedForEachDomainAction(t *testing.T) {
	dir, err := generate(t, patientDef, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	for _, name := range []string{"rule_patient_create.go", "rule_patient_update.go", "rule_patient_search.go"} {
		if !exists(dir, name) {
			t.Errorf("%s was not created", name)
		}
	}
	if exists(dir, "rule_patient_delete.go") {
		t.Error("rule_patient_delete.go was created for an action that is not declared")
	}
}

func TestSkeleton_NotCreatedForStandard(t *testing.T) {
	src := moduleSource("PatientModel", strings.Replace(patientModel, "model.AddMyBusinessRule", "model.HaveNoBusinessRule", -1))
	dir, err := generate(t, src, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	for _, verb := range []string{"create", "update", "search", "read", "delete"} {
		if exists(dir, "rule_patient_"+verb+".go") {
			t.Errorf("rule_patient_%s.go was created for a standard action", verb)
		}
	}
}

func TestSkeleton_ExactShapes(t *testing.T) {
	src := moduleSource("PatientModel, HealthModel", fullPatient+"\n"+healthModel)
	dir, err := generate(t, src, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	shapes := map[string]string{
		"rule_patient_create.go": "func (p *Patient) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {",
		"rule_patient_update.go": "func (next *Patient) Update(a crud.CurrentUser, tx crud.Tx, old *Patient, fields []string, m *Module) (crud.RuleAnswer, error) {",
		"rule_patient_delete.go": "func (p *Patient) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {",
		"rule_patient_search.go": "func (p *Patient) Search(a crud.CurrentUser, text string, m *Module) ([]storage.Condition, error) {",
		"rule_health_read.go":    "func (p *Health) Read(a crud.CurrentUser, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {",
	}
	for name, signature := range shapes {
		got := read(t, dir, name)
		if !strings.Contains(got, signature) {
			t.Errorf("%s does not hold its signature %q:\n%s", name, signature, got)
		}
		if !strings.Contains(got, completeMarker) {
			t.Errorf("%s does not carry the completion marker", name)
		}
	}
}

func TestSkeleton_DoesNotCompileUntilCompleted(t *testing.T) {
	dir, err := generate(t, patientDef, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	src := read(t, dir, "rule_patient_create.go")
	node, err := parser.ParseFile(token.NewFileSet(), "rule_patient_create.go", src, 0)
	if err != nil {
		t.Fatalf("skeleton does not parse: %v", err)
	}
	for _, s := range []string{"var decision crud.RuleAnswer", "var rejected error"} {
		if !strings.Contains(src, s) {
			t.Errorf("skeleton does not declare %q", s)
		}
	}
	returns := 0
	ast.Inspect(node, func(n ast.Node) bool {
		if _, ok := n.(*ast.ReturnStmt); ok {
			returns++
		}
		return true
	})
	if returns != 0 {
		t.Errorf("skeleton has %d return statements; it must have none, so the compiler stops it", returns)
	}
}

func TestSkeleton_NeverOverwritesACompletedRule(t *testing.T) {
	dir, err := generate(t, patientDef, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("first Run failed: %v", err)
	}
	write(t, dir, "rule_patient_create.go", "package mod // done\n")
	if err := runGenerator(dir); err != nil {
		t.Fatalf("second Run failed: %v", err)
	}
	if got := read(t, dir, "rule_patient_create.go"); got != "package mod // done\n" {
		t.Errorf("a completed rule was overwritten:\n%s", got)
	}
}

func TestSkeleton_DeletedFileIsRecreated(t *testing.T) {
	dir, err := generate(t, patientDef, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("first Run failed: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "rule_patient_create.go")); err != nil {
		t.Fatal(err)
	}
	if err := runGenerator(dir); err != nil {
		t.Fatalf("second Run failed: %v", err)
	}
	if !strings.Contains(read(t, dir, "rule_patient_create.go"), completeMarker) {
		t.Error("a deleted rule was not recreated as a skeleton")
	}
}

func TestSkeleton_OrphanRuleFileIsAnError(t *testing.T) {
	_, err := generate(t, patientDef, file{"module.go", moduleType}, file{"rule_patient_delete.go", "package mod\n"})
	if err == nil {
		t.Fatal("expected an error for a rule file of an action that is not declared, got nil")
	}
	for _, s := range []string{"rule_patient_delete.go", "Rules.Delete"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not contain %q", err.Error(), s)
		}
	}
}

func TestSkeleton_UsesTheModelName(t *testing.T) {
	visitNote := `var VisitNoteModel = model.Definition{
	Name:  "visit_note",
	Allow: model.Create,
	Rules: model.Rules{Create: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
	},
}
`
	dir, err := generate(t, moduleSource("VisitNoteModel", visitNote), file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !exists(dir, "rule_visit_note_create.go") {
		t.Fatal("rule_visit_note_create.go was not created from the entity's model name")
	}
	if !strings.Contains(read(t, dir, "rule_visit_note_create.go"), "func (p *VisitNote) Create(") {
		t.Error("the skeleton does not name the Go struct VisitNote")
	}
}
