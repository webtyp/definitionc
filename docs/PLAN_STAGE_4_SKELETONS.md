← [Stage 3](PLAN_STAGE_3_MODULE_FILE.md) | Next → [Stage 5](PLAN_STAGE_5_DOCS.md)

# Stage 4 — one rule skeleton per `AddMyBusinessRule` action

The skeleton is created **once** and never rewritten. It declares the output variables and leaves them unused, so the Go compiler itself stops the module (`declared and not used: decision`, `declared and not used: rejected`, `missing return`) with the marker comment right above: an agent that forgot to complete it sees the error in that file. (Verified with a stand-alone file on 2026-11-08: those three errors, and nothing else, for this shape.)
A completed rule that "fixes" it by returning the empty `crud.RuleAnswer{}` is refused at run time by `crud` (`ErrNoAnswer`); returning empty conditions or rows from a `Search`/`Read` compiles and yields an unfiltered list inside the tenant, never another tenant's rows (the store seals the tenant).

## 4.1 `rule_skeleton.go` (new)

```go
package definitionc

import (
	"os"
	"path/filepath"

	"webtyp.com/fmt"
	"webtyp.com/model"
)

// The words of create, update and delete are model's (one owner of the action vocabulary); search and read are rule kinds, not actions.
var (
	verbCreate = model.Create.Verb()
	verbUpdate = model.Update.Verb()
	verbDelete = model.Delete.Verb()
)

const (
	verbSearch = "search" // the Read rule of a table
	verbRead   = "read"   // the Read rule of a model without a table
	rulePrefix = "rule_"
)

// ruleVerbs is every verb a rule file can have, in the order they are written.
var ruleVerbs = [...]string{verbCreate, verbUpdate, verbDelete, verbSearch, verbRead}

// ruleFile is the file of the rule of entity (its model Name, in snake case) for verb: rule_patient_create.go.
func ruleFile(entity, verb string) string { return rulePrefix + entity + "_" + verb + ".go" }

// declaredDomain reports whether entity declared model.AddMyBusinessRule for the action behind verb.
func declaredDomain(e StructInfo, verb string) bool {
	d := uint8(model.AddMyBusinessRule)
	switch verb {
	case verbCreate:
		return isStoredEntity(e) && e.Rules.Create == d
	case verbUpdate:
		return isStoredEntity(e) && e.Rules.Update == d
	case verbDelete:
		return isStoredEntity(e) && e.Rules.Delete == d
	case verbSearch:
		return isStoredEntity(e) && e.Rules.Read == d
	case verbRead:
		return isReport(e) && e.Rules.Read == d
	}
	return false
}

// actionOf is the Rules field a verb belongs to, for messages.
func actionOf(verb string) string {
	switch verb {
	case verbCreate:
		return "Create"
	case verbUpdate:
		return "Update"
	case verbDelete:
		return "Delete"
	}
	return "Read"
}

// writeRuleSkeletons creates the missing rule file of every AddMyBusinessRule action and refuses a rule file whose action is not AddMyBusinessRule.
// An existing file is never touched: it is the developer's.
func writeRuleSkeletons(entities []StructInfo, dir string) error {
	for _, e := range entities {
		for _, verb := range ruleVerbs {
			name := ruleFile(e.ModelName, verb)
			path := filepath.Join(dir, name)
			_, statErr := os.Stat(path)
			exists := statErr == nil
			want := declaredDomain(e, verb)
			switch {
			case want && !exists:
				if err := os.WriteFile(path, []byte(skeleton(e, verb)), 0o644); err != nil {
					return err
				}
			case !want && exists:
				return fmt.Errf("%s exists but %sModel does not declare Rules.%s: model.AddMyBusinessRule; declare it, or delete the file", name, e.Name, actionOf(verb))
			}
		}
	}
	return nil
}
```

## 4.2 The five skeletons (same file)

`skeleton(e, verb)` returns the source below with `%[1]s` = `e.PackageName`, `%[2]s` = `e.Name` (the Go struct), `%[3]s` = `e.ModelName`. Use `fmt.Sprintf` with indexed verbs (if `webtyp.com/fmt.Sprintf` does not support `%[n]s`, write the tokens `{{pkg}}`, `{{type}}`, `{{entity}}` in the constants instead and fill them with `strings.NewReplacer`; the produced text must be identical); the text is the contract (tests assert the signature line and the marker line).

```go
func skeleton(e StructInfo, verb string) string {
	switch verb {
	case verbCreate:
		return fmt.Sprintf(skeletonCreate, e.PackageName, e.Name, e.ModelName)
	case verbUpdate:
		return fmt.Sprintf(skeletonUpdate, e.PackageName, e.Name, e.ModelName)
	case verbDelete:
		return fmt.Sprintf(skeletonDelete, e.PackageName, e.Name, e.ModelName)
	case verbSearch:
		return fmt.Sprintf(skeletonSearch, e.PackageName, e.Name, e.ModelName)
	}
	return fmt.Sprintf(skeletonRead, e.PackageName, e.Name, e.ModelName)
}

const skeletonCreate = `package %[1]s

import "webtyp.com/crud"

// Create es la regla de negocio de crear %[3]s: valida lo que solo tu dominio sabe y responde si se guarda y si se avisa.
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (p *%[2]s) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (crear %[3]s) y termina con: return decision, rejected
	var decision crud.RuleAnswer // lo que decides: crud.ApplyAndPublish guarda y avisa; crud.ApplyWithoutPublishing guarda sin avisar
	var rejected error       // el rechazo de tu regla; nil si todo está bien
}
`

const skeletonUpdate = `package %[1]s

import "webtyp.com/crud"

// Update es la regla de negocio de actualizar %[3]s: old es la fila guardada y el receptor trae solo los campos enviados (fields).
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (next *%[2]s) Update(a crud.CurrentUser, tx crud.Tx, old *%[2]s, fields []string, m *Module) (crud.RuleAnswer, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (actualizar %[3]s) y termina con: return decision, rejected
	var decision crud.RuleAnswer // lo que decides: crud.ApplyAndPublish guarda y avisa; crud.ApplyWithoutPublishing guarda sin avisar
	var rejected error       // el rechazo de tu regla; nil si todo está bien
}
`

const skeletonDelete = `package %[1]s

import "webtyp.com/crud"

// Delete es la regla de negocio de borrar %[3]s: el receptor es la fila guardada que se va a borrar.
// crud ya validó los ids, los límites y el permiso antes de llamarte.
func (p *%[2]s) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (borrar %[3]s) y termina con: return decision, rejected
	var decision crud.RuleAnswer // lo que decides: crud.ApplyAndPublish borra y avisa; crud.ApplyWithoutPublishing borra sin avisar
	var rejected error       // el rechazo de tu regla; nil si todo está bien
}
`

const skeletonSearch = `package %[1]s

import (
	"webtyp.com/crud"
	"webtyp.com/storage"
)

// Search traduce el texto que escribió la persona en las condiciones que buscan %[3]s. Solo condiciones unidas con AND (nunca OR).
// crud ya aplicó el tope de página, el cursor y la clínica; tú solo dices qué filas corresponden al texto.
func (p *%[2]s) Search(a crud.CurrentUser, text string, m *Module) ([]storage.Condition, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (buscar %[3]s por texto) y termina con: return conditions, rejected
	var conditions []storage.Condition // las condiciones que corresponden al texto
	var rejected error                 // el rechazo de tu regla; nil si todo está bien
}
`

const skeletonRead = `package %[1]s

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

// Read responde la lectura de %[3]s, que no es una tabla: tú produces las filas.
// crud ya validó los argumentos, aplicó el tope de página y comprobó el permiso antes de llamarte.
func (p *%[2]s) Read(a crud.CurrentUser, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (leer %[3]s) y termina con: return rows, rejected
	var rows model.ModelSlice // las filas que respondes
	var rejected error        // el rechazo de tu regla; nil si todo está bien
}
`
```

Do not run these constants through `go/format` and do not "tidy" the comment alignment: the files are written exactly as above (they are not compilable on purpose, so `format.Source` would also still accept them, but a formatter must never decide the text).
`gotest` passes: every test of stage 1 is green. Commit: `feat: one rule skeleton per AddMyBusinessRule action`.
