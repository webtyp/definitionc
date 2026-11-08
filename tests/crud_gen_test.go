package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"webtyp.com/ormc"
)

func TestCrudGen_WrittenWhenGoModRequiresCrud(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module example.com/testmod

go 1.26.8

require webtyp.com/crud v0.0.4
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	modelContent := `package testmod

import "webtyp.com/model"

var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Create | model.Read,
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text()},
	},
}
`
	modelPath := filepath.Join(dir, "model.go")
	if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	gen := ormc.New()
	gen.SetSkipTidy(true)
	gen.SetRootDir(dir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	crudPath := filepath.Join(dir, "model_crud.go")
	data, err := os.ReadFile(crudPath)
	if err != nil {
		t.Fatalf("model_crud.go not found: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "func MountPatient(reg crud.Registry, store crud.Store, rules PatientRules)") {
		t.Errorf("missing expected MountPatient function, got:\n%s", content)
	}
}

func TestCrudGen_SkippedWithoutCrud(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module example.com/testmod

go 1.26.8

require webtyp.com/model v0.2.15
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	modelContent := `package testmod

import "webtyp.com/model"

var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Read,
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
	},
}
`
	modelPath := filepath.Join(dir, "model.go")
	if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	gen := ormc.New()
	gen.SetSkipTidy(true)
	gen.SetRootDir(dir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	crudPath := filepath.Join(dir, "model_crud.go")
	if _, err := os.Stat(crudPath); !os.IsNotExist(err) {
		t.Fatalf("expected model_crud.go to not exist, err=%v", err)
	}
}

func TestCrudGen_StaleFileRemoved(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module example.com/testmod

go 1.26.8
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	modelContent := `package testmod

import "webtyp.com/model"

var PatientModel = model.Definition{
	Name: "patient",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
	},
}
`
	modelPath := filepath.Join(dir, "model.go")
	if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	crudPath := filepath.Join(dir, "model_crud.go")
	if err := os.WriteFile(crudPath, []byte("// stale"), 0644); err != nil {
		t.Fatal(err)
	}

	gen := ormc.New()
	gen.SetSkipTidy(true)
	gen.SetRootDir(dir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, err := os.Stat(crudPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale model_crud.go to be removed, err=%v", err)
	}
}

func TestCrudGen_OnlyTableEntities(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module example.com/testmod

go 1.26.8

require webtyp.com/crud v0.0.4
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	modelContent := `package testmod

import "webtyp.com/model"

var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Create,
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
	},
}

var ByStaffModel = model.Definition{
	Name: "by_staff",
	Wire: true,
	Fields: model.Fields{
		{Name: "staff_id", Type: model.Text()},
	},
}
`
	modelPath := filepath.Join(dir, "model.go")
	if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	gen := ormc.New()
	gen.SetSkipTidy(true)
	gen.SetRootDir(dir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	crudPath := filepath.Join(dir, "model_crud.go")
	data, err := os.ReadFile(crudPath)
	if err != nil {
		t.Fatalf("model_crud.go not found: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "MountPatient") {
		t.Error("expected MountPatient")
	}
	if strings.Contains(content, "MountByStaff") {
		t.Error("did not expect MountByStaff (Wire: true)")
	}
}

func TestCrudGen_Compiles(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}

	fixtureDir := filepath.Join("testdata", "fixturecrud")
	gen := ormc.New()
	gen.SetRootDir(fixtureDir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed on fixturecrud: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(fixtureDir, "model_crud.go"))
	if err != nil {
		t.Fatalf("model_crud.go not written for fixturecrud: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "func MountPatient(") {
		t.Errorf("expected func MountPatient( in model_crud.go")
	}
	if !strings.Contains(content, "type PatientRules struct") {
		t.Errorf("expected type PatientRules struct in model_crud.go")
	}
	if !strings.Contains(content, "Create    func(c crud.Caller, tx crud.Tx, p *Patient) error") {
		t.Errorf("expected Create field in PatientRules")
	}
	if !strings.Contains(content, "Update    func(c crud.Caller, tx crud.Tx, old, next *Patient, fields []string) error") {
		t.Errorf("expected Update field in PatientRules")
	}
	if strings.Contains(content, "Delete    func(c crud.Caller, tx crud.Tx, p *Patient) error") {
		t.Errorf("expected NO Delete field in PatientRules because Allow has no Delete")
	}
	if strings.Contains(content, "MountNote") {
		t.Errorf("expected no MountNote because Note has no Allow")
	}
	if strings.Contains(content, "MountCodeEntry") {
		t.Errorf("expected no MountCodeEntry: form without table")
	}

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = fixtureDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet failed in %s: %v\nOutput:\n%s", fixtureDir, err, string(out))
	}
}

func TestCrudGen_AllowMethod(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "fixturecrud")
	data, err := os.ReadFile(filepath.Join(fixtureDir, "model_orm.go"))
	if err != nil {
		t.Fatalf("model_orm.go not found: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "func (*Patient) Allow() model.Action     { return PatientModel.Allow }") {
		t.Errorf("missing func (*Patient) Allow() in model_orm.go")
	}
	if !strings.Contains(content, "func (*PatientList) Allow() model.Action { return PatientModel.Allow }") {
		t.Errorf("missing func (*PatientList) Allow() in model_orm.go")
	}
	if !strings.Contains(content, "func (*Note) Allow() model.Action     { return NoteModel.Allow }") {
		t.Errorf("missing func (*Note) Allow() in model_orm.go")
	}
}

func TestCrudGen_ReadOnlyCompiles(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module example.com/readonlymod

go 1.26.8

require (
	webtyp.com/crud v0.0.4
	webtyp.com/model v0.2.15
	webtyp.com/orm v0.12.26
	webtyp.com/storage v0.1.18
)
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	modelContent := `package readonlymod

import "webtyp.com/model"

var ReadOnlyModel = model.Definition{
	Name:  "readonly",
	Allow: model.Read,
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "text", Type: model.Text(), DB: &model.FieldDB{}},
	},
}
`
	if err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	gen := ormc.New()
	gen.SetRootDir(dir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "model_crud.go"))
	if err != nil {
		t.Fatalf("model_crud.go not found: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "rules.Create") {
		t.Errorf("expected no rules.Create in ReadOnly model_crud.go")
	}

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet failed: %v\nOutput:\n%s", err, string(out))
	}
}

func TestCrudGen_AllowOnWireIsError(t *testing.T) {
	dir := t.TempDir()
	modelContent := `package testmod

import "webtyp.com/model"

var WireModel = model.Definition{
	Name:  "wire_item",
	Wire:  true,
	Allow: model.Read,
	Fields: model.Fields{
		{Name: "id", Type: model.Text()},
	},
}
`
	if err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}
	gen := ormc.New()
	gen.SetRootDir(dir)
	err := gen.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot Allow") {
		t.Errorf("expected error containing 'cannot Allow', got: %v", err)
	}
}

func TestCrudGen_ReadModelGetsAllowNoMount(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module example.com/readmod

go 1.26.8

require (
	webtyp.com/crud v0.0.4
	webtyp.com/model v0.2.15
)
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	modelContent := `package readmod

import "webtyp.com/model"

var ReportModel = model.Definition{
	Name:  "report",
	Allow: model.Read,
	Fields: model.Fields{
		{Name: "total", Type: model.Int()},
	},
}
`
	if err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	gen := ormc.New()
	gen.SetRootDir(dir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	ormData, err := os.ReadFile(filepath.Join(dir, "model_orm.go"))
	if err != nil {
		t.Fatalf("model_orm.go not found: %v", err)
	}
	if !strings.Contains(string(ormData), "func (*Report) Allow() model.Action") {
		t.Errorf("expected Allow() method for Report")
	}

	crudPath := filepath.Join(dir, "model_crud.go")
	if _, err := os.Stat(crudPath); !os.IsNotExist(err) {
		t.Errorf("expected no model_crud.go for read model without DB fields")
	}

	// Test with Allow: model.Create -> error only model.Read
	badModelContent := `package readmod

import "webtyp.com/model"

var BadReportModel = model.Definition{
	Name:  "bad_report",
	Allow: model.Create,
	Fields: model.Fields{
		{Name: "total", Type: model.Int()},
	},
}
`
	badDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(badDir, "model.go"), []byte(badModelContent), 0644); err != nil {
		t.Fatal(err)
	}
	genBad := ormc.New()
	genBad.SetRootDir(badDir)
	errBad := genBad.Run()
	if errBad == nil {
		t.Fatal("expected error with Allow: Create on no-DB model, got nil")
	}
	if !strings.Contains(errBad.Error(), "only model.Read") {
		t.Errorf("expected error containing 'only model.Read', got: %v", errBad)
	}
}

func TestCrudGen_AllowNonConstantIsError(t *testing.T) {
	dir := t.TempDir()
	modelContent := `package testmod

import "webtyp.com/model"

var someVar = model.Read

var MyModel = model.Definition{
	Name:  "my_model",
	Allow: someVar,
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
	},
}
`
	if err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}
	gen := ormc.New()
	gen.SetRootDir(dir)
	err := gen.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "constants only") {
		t.Errorf("expected error containing 'constants only', got: %v", err)
	}
}

func TestCrudGen_NoBuildTag(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "fixturecrud")
	data, err := os.ReadFile(filepath.Join(fixtureDir, "model_crud.go"))
	if err != nil {
		t.Fatalf("model_crud.go not found: %v", err)
	}
	if strings.Contains(string(data), "//go:build") {
		t.Errorf("model_crud.go should NOT have a //go:build line")
	}
}

func TestCrudGen_NoAllowNoFile(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module example.com/testmod

go 1.26.8

require webtyp.com/crud v0.0.4
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	modelContent := `package testmod

import "webtyp.com/model"

var OnlyTableModel = model.Definition{
	Name: "only_table",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
	},
}
`
	if err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	gen := ormc.New()
	gen.SetRootDir(dir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	crudPath := filepath.Join(dir, "model_crud.go")
	if _, err := os.Stat(crudPath); !os.IsNotExist(err) {
		t.Fatalf("expected no model_crud.go when only table has no Allow, err=%v", err)
	}
}

func TestCrudGen_CompositeItemID(t *testing.T) {
	dir := t.TempDir()
	modelContent := `package testmod

import "webtyp.com/model"

var RoomEquipmentModel = model.Definition{
	Name: "room_equipment",
	Fields: model.Fields{
		{Name: "room_id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "equipment_id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), Item: model.ItemLabel},
	},
}
`
	if err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(modelContent), 0644); err != nil {
		t.Fatal(err)
	}

	gen := ormc.New()
	gen.SetRootDir(dir)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "model_orm.go"))
	if err != nil {
		t.Fatalf("model_orm.go not found: %v", err)
	}
	if !strings.Contains(string(data), "model.JoinKey(") {
		t.Errorf("expected model.JoinKey in Item(), got:\n%s", string(data))
	}
}
