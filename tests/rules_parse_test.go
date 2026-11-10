package tests

import (
	"strings"
	"testing"
)

const (
	keyFields   = `{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},`
	noKeyFields = `{Name: "total", Type: model.Int()},`
)

// patientLiteral is one PatientModel with the given Allow, Rules and field lines.
func patientLiteral(allow, rules, fields string) string {
	return "var PatientModel = model.Definition{\n" +
		"\tName: \"patient\",\n" +
		"\tAllow: " + allow + ",\n" +
		"\tRules: " + rules + ",\n" +
		"\tFields: model.Fields{\n\t\t" + fields + "\n\t},\n}\n"
}

// modelOnly is a definition.go holding the given model declarations and nothing else.
func modelOnly(literal string) string {
	return "package mod\n\nimport \"webtyp.com/model\"\n\n" + literal
}

// wantRuleError generates src and fails unless it is refused with every given text in the error.
func wantRuleError(t *testing.T, src string, contains ...string) {
	t.Helper()
	_, err := generate(t, src, file{"module.go", moduleType})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	for _, s := range contains {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not contain %q", err.Error(), s)
		}
	}
}

func TestRules_UndeclaredActionIsAnError(t *testing.T) {
	src := modelOnly(patientLiteral("model.Create | model.Read", "model.Rules{Create: model.HaveNoBusinessRule}", keyFields))
	wantRuleError(t, src, "must declare its rule", "patient")
}

func TestRules_OrphanRuleIsAnError(t *testing.T) {
	src := modelOnly(patientLiteral("model.Create", "model.Rules{Create: model.HaveNoBusinessRule, Delete: model.AddMyBusinessRule}", keyFields))
	wantRuleError(t, src, "does not offer")
}

func TestRules_ConstantsOnly(t *testing.T) {
	src := modelOnly(patientLiteral("model.Create", "someVar", keyFields))
	wantRuleError(t, src, "model.Rules{", "PatientModel")
}

func TestRules_UnknownKeyIsAnError(t *testing.T) {
	src := modelOnly(patientLiteral("model.Create", "model.Rules{Archive: model.AddMyBusinessRule}", keyFields))
	wantRuleError(t, src, "Archive")
}

func TestRules_ModeMustBeAConstant(t *testing.T) {
	src := modelOnly(patientLiteral("model.Create", "model.Rules{Create: mode}", keyFields))
	wantRuleError(t, src, "model.AddMyBusinessRule or model.HaveNoBusinessRule")
}

func TestRules_TableWithoutKeyReadMustBeDomain(t *testing.T) {
	src := modelOnly(patientLiteral("model.Read", "model.Rules{Read: model.HaveNoBusinessRule}", noKeyFields))
	wantRuleError(t, src, "AddMyBusinessRule")
}

func TestRules_WireReadOnlyThroughItsRule(t *testing.T) {
	ok := modelOnly("var PatientModel = model.Definition{\n\tName: \"patient\",\n\tWire: true,\n\tAllow: model.Read,\n" +
		"\tRules: model.Rules{Read: model.AddMyBusinessRule},\n\tFields: model.Fields{\n\t\t" + noKeyFields + "\n\t},\n}\n")
	if _, err := generate(t, ok, file{"module.go", moduleType}); err != nil {
		t.Fatalf("a Wire model that reads through its rule must be accepted: %v", err)
	}

	bad := modelOnly("var PatientModel = model.Definition{\n\tName: \"patient\",\n\tWire: true,\n\tAllow: model.Create,\n" +
		"\tRules: model.Rules{Create: model.AddMyBusinessRule},\n\tFields: model.Fields{\n\t\t" + keyFields + "\n\t},\n}\n")
	wantRuleError(t, bad, "Wire")
}

func TestModuleDefinition_NotMistakenForAnEntity(t *testing.T) {
	dir, err := generate(t, patientDef, file{"module.go", moduleType})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got := read(t, dir, "definition_gen.go"); strings.Contains(got, "type Definition struct") {
		t.Errorf("the module variable Definition was generated as an entity:\n%s", got)
	}
}
