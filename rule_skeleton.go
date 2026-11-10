package definitionc

import (
	"os"
	"path/filepath"
	"strings"

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

// The placeholders of a skeleton: the package, the Go struct and the model's name. Filled by strings.NewReplacer,
// because the text is the contract and webtyp/fmt has no indexed verbs.
const (
	tokPackage = "{{pkg}}"
	tokType    = "{{type}}"
	tokEntity  = "{{entity}}"
)

// skeleton returns the source of the rule file of verb for entity e.
func skeleton(e StructInfo, verb string) string {
	return strings.NewReplacer(tokPackage, e.PackageName, tokType, e.Name, tokEntity, e.ModelName).Replace(skeletonText(verb))
}

func skeletonText(verb string) string {
	switch verb {
	case verbCreate:
		return skeletonCreate
	case verbUpdate:
		return skeletonUpdate
	case verbDelete:
		return skeletonDelete
	case verbSearch:
		return skeletonSearch
	}
	return skeletonRead
}

// The five skeletons. They do not compile on purpose: the unused declarations and the missing return stop the module
// until the developer completes the rule. They are written exactly as they are here (no formatter runs on them).
const skeletonCreate = `package {{pkg}}

import "webtyp.com/crud"

// Create es la regla de negocio de crear {{entity}}: valida lo que solo tu dominio sabe y responde si se guarda y si se avisa.
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (p *{{type}}) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (crear {{entity}}) y termina con: return decision, rejected
	var decision crud.RuleAnswer // lo que decides: crud.ApplyAndPublish guarda y avisa; crud.ApplyWithoutPublishing guarda sin avisar
	var rejected error       // el rechazo de tu regla; nil si todo está bien
}
`

const skeletonUpdate = `package {{pkg}}

import "webtyp.com/crud"

// Update es la regla de negocio de actualizar {{entity}}: old es la fila guardada y el receptor trae solo los campos enviados (fields).
// crud ya validó el esquema, los campos solo del servidor, los límites y el permiso antes de llamarte.
func (next *{{type}}) Update(a crud.CurrentUser, tx crud.Tx, old *{{type}}, fields []string, m *Module) (crud.RuleAnswer, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (actualizar {{entity}}) y termina con: return decision, rejected
	var decision crud.RuleAnswer // lo que decides: crud.ApplyAndPublish guarda y avisa; crud.ApplyWithoutPublishing guarda sin avisar
	var rejected error       // el rechazo de tu regla; nil si todo está bien
}
`

const skeletonDelete = `package {{pkg}}

import "webtyp.com/crud"

// Delete es la regla de negocio de borrar {{entity}}: el receptor es la fila guardada que se va a borrar.
// crud ya validó los ids, los límites y el permiso antes de llamarte.
func (p *{{type}}) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (borrar {{entity}}) y termina con: return decision, rejected
	var decision crud.RuleAnswer // lo que decides: crud.ApplyAndPublish borra y avisa; crud.ApplyWithoutPublishing borra sin avisar
	var rejected error       // el rechazo de tu regla; nil si todo está bien
}
`

const skeletonSearch = `package {{pkg}}

import (
	"webtyp.com/crud"
	"webtyp.com/storage"
)

// Search traduce el texto que escribió la persona y los campos que envió (filter) en las condiciones que buscan {{entity}}. Solo condiciones unidas con AND (nunca OR).
// crud ya aplicó el tope de página, el cursor y la clínica; tú solo dices qué filas corresponden. text es "" cuando no se envió texto.
func (p *{{type}}) Search(a crud.CurrentUser, filter *{{type}}, text string, m *Module) ([]storage.Condition, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (buscar {{entity}} por texto) y termina con: return conditions, rejected
	var conditions []storage.Condition // las condiciones que corresponden al texto
	var rejected error                 // el rechazo de tu regla; nil si todo está bien
}
`

const skeletonRead = `package {{pkg}}

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

// Read responde la lectura de {{entity}}, que no es una tabla: tú produces las filas. filter trae los campos que envió quien llama.
// crud ya validó los argumentos, aplicó el tope de página y comprobó el permiso antes de llamarte.
func (p *{{type}}) Read(a crud.CurrentUser, filter *{{type}}, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	// COMPLETA TU REGLA DE NEGOCIO AQUÍ (leer {{entity}}) y termina con: return rows, rejected
	var rows model.ModelSlice // las filas que respondes
	var rejected error        // el rechazo de tu regla; nil si todo está bien
}
`
