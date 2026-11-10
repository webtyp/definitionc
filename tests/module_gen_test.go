package tests

import (
	"strings"
	"testing"
)

// moduleOutput generates src with the module.go every module needs and returns definition_gen.go, squashed and parsed.
func moduleOutput(t *testing.T, src string, extra ...file) string {
	t.Helper()
	dir, err := generate(t, src, append([]file{{"module.go", moduleType}}, extra...)...)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	got := read(t, dir, "definition_gen.go")
	mustParse(t, got)
	return squash(got)
}

// wantContains fails unless got holds every text, in whitespace-insensitive form.
func wantContains(t *testing.T, got string, texts ...string) {
	t.Helper()
	for _, s := range texts {
		if !strings.Contains(got, s) {
			t.Errorf("generated module does not contain %q", s)
		}
	}
}

func TestModule_MountsAStoredEntity(t *testing.T) {
	got := moduleOutput(t, patientDef)
	wantContains(t, got,
		`const ModelName = "patient_directory"`,
		`func (m *Module) ModelName() string { return ModelName }`,
		`func (m *Module) Definition() model.ModuleDefinition { return Definition }`,
		`func (m *Module) MountOperations(reg crud.Registry) {`,
		`reg.AddStoredEntity(crud.StoredEntity{`,
		`Model: &Patient{},`,
		`NewList: func() model.ModelSlice { return &PatientList{} },`,
		`Store: m.StorePatient(),`,
		`Rules: PatientModel.Rules,`,
		`return rec.(*Patient).Create(a, tx, m)`,
		`return next.(*Patient).Update(a, tx, old.(*Patient), fields, m)`,
		`Search: func(a crud.CurrentUser, filter model.Model, text string) ([]storage.Condition, error) {`,
		`return (&Patient{}).Search(a, filter.(*Patient), text, m)`,
	)
	if strings.Contains(got, "Delete:") {
		t.Errorf("an entity without Delete mounts a Delete hook:\n%s", got)
	}
}

func TestModule_StandardActionsHaveNoHook(t *testing.T) {
	src := moduleSource("PatientModel", strings.Replace(patientModel,
		"model.Rules{Create: model.AddMyBusinessRule, Read: model.AddMyBusinessRule, Update: model.AddMyBusinessRule}",
		"model.Rules{Create: model.HaveNoBusinessRule, Read: model.HaveNoBusinessRule, Update: model.HaveNoBusinessRule}", 1))
	dir, err := generate(t, src, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	got := squash(read(t, dir, "definition_gen.go"))
	wantContains(t, got, `reg.AddStoredEntity(crud.StoredEntity{`, `Rules: PatientModel.Rules,`)
	for _, s := range []string{"Create: func", "Update: func", "Search: func", "webtyp.com/storage"} {
		if strings.Contains(got, s) {
			t.Errorf("a standard action generated %q:\n%s", s, got)
		}
	}
	for _, verb := range []string{"create", "update", "search"} {
		if exists(dir, "rule_patient_"+verb+".go") {
			t.Errorf("rule_patient_%s.go was created for a standard action", verb)
		}
	}
}

func TestModule_MFAIsCarriedToTheStoredEntity(t *testing.T) {
	src := moduleSource("PatientModel", strings.Replace(patientModel,
		"Update: model.AddMyBusinessRule},", "Update: model.AddMyBusinessRule},\n\tMFA: model.Update,", 1))
	wantContains(t, moduleOutput(t, src), `MFA: PatientModel.MFA,`)

	wantContains(t, moduleOutput(t, patientDef), `MFA: PatientModel.MFA,`)
}

func TestModule_StorePerEntity(t *testing.T) {
	src := moduleSource("PatientModel, VisitModel", patientModel+"\n"+visitModel)
	wantContains(t, moduleOutput(t, src), `Store: m.StorePatient(),`, `Store: m.StoreVisit(),`)
}

func TestModule_DeleteHook(t *testing.T) {
	src := moduleSource("PatientModel", strings.Replace(patientModel,
		"Allow: model.Create | model.Read | model.Update,", "Allow: model.Create | model.Read | model.Update | model.Delete,", 1))
	src = strings.Replace(src, "Update: model.AddMyBusinessRule},",
		"Update: model.AddMyBusinessRule, Delete: model.AddMyBusinessRule},", 1)
	wantContains(t, moduleOutput(t, src), `return rec.(*Patient).Delete(a, tx, m)`)
}

func TestModule_ReadOnlyModelUsesRegistryRead(t *testing.T) {
	src := moduleSource("HealthModel", healthModel)
	got := moduleOutput(t, src)
	wantContains(t, got,
		`reg.AddReadOnlyEntity(&Health{}, func() model.Model { return &Health{} }, func(a crud.CurrentUser, args crud.ReadArgs, filter model.Model) (model.ModelSlice, error) {`,
		`return (&Health{}).Read(a, filter.(*Health), args, m)`,
	)
	if strings.Contains(got, "crud.StoredEntity{") {
		t.Errorf("a model without a table was mounted as a stored entity:\n%s", got)
	}
}

func TestModule_WireModelUsesRegistryRead(t *testing.T) {
	wire := strings.Replace(healthModel, "Allow: model.Read,", "Wire: true,\n\tAllow: model.Read,", 1)
	wantContains(t, moduleOutput(t, moduleSource("HealthModel", wire)),
		`reg.AddReadOnlyEntity(&Health{}, func() model.Model { return &Health{} }, func(a crud.CurrentUser, args crud.ReadArgs, filter model.Model) (model.ModelSlice, error) {`,
		`return (&Health{}).Read(a, filter.(*Health), args, m)`,
	)
}

func TestModule_TwoEntities(t *testing.T) {
	got := moduleOutput(t, moduleSource("PatientModel, VisitModel", patientModel+"\n"+visitModel))
	if n := strings.Count(got, "reg.AddStoredEntity(crud.StoredEntity{"); n != 2 {
		t.Fatalf("want one AddStoredEntity per entity (2), got %d:\n%s", n, got)
	}
	if strings.Index(got, "Model: &Patient{}") > strings.Index(got, "Model: &Visit{}") {
		t.Errorf("entities are not mounted in the order of Entities:\n%s", got)
	}
}

func TestModule_InternalEntityIsListedButNotMounted(t *testing.T) {
	note := `var NoteModel = model.Definition{
	Name: "note",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "text", Type: model.Text()},
	},
}
`
	got := moduleOutput(t, moduleSource("PatientModel, NoteModel", patientModel+"\n"+note))
	if n := strings.Count(got, "reg.AddStoredEntity(crud.StoredEntity{"); n != 1 {
		t.Errorf("want only Patient mounted (1 AddStoredEntity), got %d:\n%s", n, got)
	}
	if strings.Contains(got, "Model: &Note{}") {
		t.Errorf("an internal entity was mounted:\n%s", got)
	}
}

func TestModule_NoDefinitionNoModuleCode(t *testing.T) {
	src := modelOnly(patientModel)
	dir, err := generate(t, src, file{"definition_crud.go", "package mod\n"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if exists(dir, "definition_crud.go") {
		t.Error("the legacy definition_crud.go was not removed")
	}
	if strings.Contains(read(t, dir, "definition_gen.go"), "MountOperations") {
		t.Error("a file without var Definition generated MountOperations")
	}
	if exists(dir, "rule_patient_create.go") {
		t.Error("a file without var Definition created a rule file")
	}
}

func TestModule_EntityNotListedIsAnError(t *testing.T) {
	wantRuleError(t, moduleSource("", patientModel), "PatientModel", "Entities")
}

func TestModule_UnknownHandlerIsAnError(t *testing.T) {
	wantRuleError(t, moduleSource("GhostModel", patientModel), "GhostModel")
}

func TestModule_HandlerMustEndInModel(t *testing.T) {
	wantRuleError(t, moduleSource("Patient", patientModel), "must be written XModel")
}

func TestModule_ModuleTypeIsRequired(t *testing.T) {
	_, err := generate(t, patientDef)
	if err == nil {
		t.Fatal("expected an error without type Module, got nil")
	}
	for _, s := range []string{"type Module", "package"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not contain %q", err.Error(), s)
		}
	}
}

func TestModule_NameRequired(t *testing.T) {
	src := strings.Replace(patientDef, `Name: "patient_directory"`, `Name: ""`, 1)
	wantRuleError(t, src, "Name")
}

func TestModule_Regenerates(t *testing.T) {
	dir, err := generate(t, patientDef, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("first Run failed: %v", err)
	}
	first := read(t, dir, "definition_gen.go")
	if err := runGenerator(dir); err != nil {
		t.Fatalf("second Run failed: %v", err)
	}
	if second := read(t, dir, "definition_gen.go"); second != first {
		t.Errorf("a second run changed definition_gen.go")
	}
}

func TestGenerate_AllowMethod(t *testing.T) {
	got := moduleOutput(t, patientDef)
	wantContains(t, got,
		"func (*Patient) Allow() model.Action { return PatientModel.Allow }",
		"func (*PatientList) Allow() model.Action { return PatientModel.Allow }",
	)
}

func TestGenerate_CompositeItemID(t *testing.T) {
	src := modelOnly(`var RoomEquipmentModel = model.Definition{
	Name: "room_equipment",
	Fields: model.Fields{
		{Name: "room_id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "equipment_id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), Item: model.ItemLabel},
	},
}
`)
	dir, err := generate(t, src)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(read(t, dir, "definition_gen.go"), "model.JoinKey(") {
		t.Error("expected model.JoinKey in Item() of a composite key")
	}
}

func TestGenerate_NoBuildTag(t *testing.T) {
	if got := moduleOutput(t, patientDef); strings.Contains(got, "//go:build") {
		t.Error("the generated module file must not carry a //go:build line")
	}
}

func TestModule_NoAllowNoRuleFile(t *testing.T) {
	note := "var NoteModel = model.Definition{\n\tName: \"note\",\n\tFields: model.Fields{\n\t\t" + keyFields + "\n\t},\n}\n"
	dir, err := generate(t, moduleSource("NoteModel", note), file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if strings.Contains(read(t, dir, "definition_gen.go"), "AddStoredEntity") {
		t.Error("an entity that allows nothing was mounted")
	}
	for _, verb := range []string{"create", "update", "delete", "search", "read"} {
		if exists(dir, "rule_note_"+verb+".go") {
			t.Errorf("rule_note_%s.go was created for an entity that allows nothing", verb)
		}
	}
}
