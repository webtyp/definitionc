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

require webtyp.com/crud v0.0.3
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

	expected := []string{
		"func MountPatientCreate(reg crud.Registry, create func(c crud.Caller, list PatientList) error) crud.Route",
		"func MountPatientRead(reg crud.Registry, newFilter func() model.Model, read func(c crud.Caller, args crud.ReadArgs, filter model.Model) (PatientList, error)) crud.Route",
		"func MountPatientUpdate(reg crud.Registry, update func(c crud.Caller, ids, fields []string, rec Patient) error) crud.Route",
		"func MountPatientDelete(reg crud.Registry, remove func(c crud.Caller, ids []string) error) crud.Route",
	}
	for _, exp := range expected {
		if !strings.Contains(content, exp) {
			t.Errorf("missing expected function: %s\ngot:\n%s", exp, content)
		}
	}
}

func TestCrudGen_SkippedWithoutCrud(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module example.com/testmod

go 1.26.8

require webtyp.com/model v0.2.12
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

require webtyp.com/crud v0.0.3
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

	if !strings.Contains(content, "MountPatientCreate") {
		t.Error("expected MountPatientCreate")
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

	// A table entity with input.* fields gets its helpers; a form without a table does not.
	data, err := os.ReadFile(filepath.Join(fixtureDir, "model_crud.go"))
	if err != nil {
		t.Fatalf("model_crud.go not written for fixturecrud (does its go.mod still require webtyp.com/crud?): %v", err)
	}
	for _, want := range []string{"MountPatientCreate", "MountPatientRead", "MountPatientUpdate", "MountPatientDelete"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("model_crud.go lacks %s: a table entity that is also a form must get CRUD helpers", want)
		}
	}
	if strings.Contains(string(data), "MountCodeEntry") {
		t.Error("model_crud.go has MountCodeEntry: a form without a table must get no CRUD helpers")
	}

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = fixtureDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet failed in %s: %v\nOutput:\n%s", fixtureDir, err, string(out))
	}
}
