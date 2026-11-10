package tests

import (
	"strings"
	"testing"
)

func TestRevision_IsCopiedToTheGeneratedSchema(t *testing.T) {
	// An excluded field makes definitionc write the field literals of Schema(), where the flag must appear.
	src := modelOnly(`var NoteModel = model.Definition{
	Name: "note",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "revision", Type: model.Int(), DB: &model.FieldDB{Revision: true}},
		{Name: "draft", Type: model.Text(), Exclude: true},
	},
}
`)
	dir, err := generate(t, src)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	got := squash(read(t, dir, "definition_gen.go"))
	if !strings.Contains(got, `{Name: "revision", Type: model.Int(), DB: &model.FieldDB{Revision: true}},`) {
		t.Errorf("the Revision flag was not written to the generated schema:\n%s", got)
	}
}

func TestRevision_AbsentIsNotWritten(t *testing.T) {
	src := modelOnly(`var NoteModel = model.Definition{
	Name: "note",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "text", Type: model.Text(), DB: &model.FieldDB{}},
	},
}
`)
	dir, err := generate(t, src)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got := read(t, dir, "definition_gen.go"); strings.Contains(got, "Revision") {
		t.Errorf("a definition without Revision generated it:\n%s", got)
	}
}

func TestRevision_BadRevisionIsRefused(t *testing.T) {
	src := modelOnly(`var NoteModel = model.Definition{
	Name: "note",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "revision", Type: model.Text(), DB: &model.FieldDB{Revision: true}},
	},
}
`)
	_, err := generate(t, src)
	if err == nil {
		t.Fatal("expected an error for a text Revision field, got nil")
	}
	if !strings.Contains(err.Error(), "integer") {
		t.Errorf("error %q does not contain %q", err.Error(), "integer")
	}
}
