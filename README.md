# definitionc
<img src="docs/img/badges.svg">

`definitionc` is the code generator of a webtyp module. You declare the module once, in `definition.go`; it writes the
module's operations and one rule file for every business rule you declare. You write the rules, never the wiring.

## 1. Why This Exists

**Without it**, a module mounts its entities by hand in an `ops.go`:

```go
// ops.go, written by hand: every entity and every rule is a line someone must remember.
func (m *Module) MountOperations(reg crud.Registry) {
	reg.AddStoredEntity(crud.StoredEntity{Model: &Patient{}, /* … */ Create: createPatient})
	// a new entity with actions added to the module but forgotten here is never offered
}
```

A forgotten entity is silent: the API simply does not have it. A rule that exists only in a comment is silent too.

**With it**, the declaration is the only list. The entity says which actions it allows and which of them have a business
rule. The generator mounts every allowed action, and for each business rule it creates a skeleton file that **does not
compile** until someone writes the rule. The compiler is the reminder.

## 2. Step-by-Step Guide

**Step 1 — declare each model with its actions and their rules.** `Allow` is what the API offers; `Rules` says, for each
allowed action, whether the entity has a business rule of its own (`model.AddMyBusinessRule`) or crud's checks are enough
(`model.HaveNoBusinessRule`). An allowed action without a declared rule is an error.

```go
var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Create | model.Read | model.Update,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Update: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: model.Text(), Item: model.ItemLabel},
	},
}
```

**Step 2 — declare the module: its name and every entity it serves.**

```go
var Definition = model.ModuleDefinition{Name: "patient_directory", Entities: []model.Definition{PatientModel}}
```

**Step 3 — write `module.go`, the hand-written part.** The module type, and one store method per table entity:

```go
package patientdirectory

import "webtyp.com/crud"

type Module struct{ store crud.Store }

func (m *Module) StorePatient() crud.Store { return m.store }
```

**Step 4 — save `definition.go`** (or run `definitionc` in the module). The generator writes `definition_gen.go` with
`ModelName()`, `Definition()` and `MountOperations(reg)`, and creates `rule_patient_create.go` with the rule skeleton.

**Step 5 — complete the rule.** The skeleton declares the outputs and leaves them unused, so the module reports
`declared and not used` and `missing return` until you return an answer. The marker `// COMPLETA TU REGLA DE NEGOCIO AQUÍ`
shows where to write it. Once the file exists, `definitionc` never overwrites it.

## 3. The Rule Signatures

Each rule is a method on the record (`p` or `next`); `m` is the module, so a rule reaches its dependencies through `m`.
The signatures are fixed:

| Action | File | Signature |
|---|---|---|
| create | `rule_patient_create.go` | `func (p *Patient) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` |
| update | `rule_patient_update.go` | `func (next *Patient) Update(a crud.CurrentUser, tx crud.Tx, old *Patient, fields []string, m *Module) (crud.RuleAnswer, error)` |
| delete | `rule_patient_delete.go` | `func (p *Patient) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` |
| search (table, `Rules.Read: AddMyBusinessRule`) | `rule_patient_search.go` | `func (p *Patient) Search(a crud.CurrentUser, filter *Patient, text string, m *Module) ([]storage.Condition, error)` |
| read (model without a table, `Rules.Read: AddMyBusinessRule`) | `rule_health_read.go` | `func (p *Health) Read(a crud.CurrentUser, args crud.ReadArgs, m *Module) (model.ModelSlice, error)` |

- `Search` is called on a zero record (`(&Patient{}).Search(…)`): it uses no record data. `filter` carries only the fields the caller sent (the entity is its own filter), and `text` is `""` when none was sent.
- A table read declared `model.HaveNoBusinessRule` is crud's default read: id, search by the item label, page, cap and tenant.
- A rule file for an action that is not declared `model.AddMyBusinessRule` is an error, so a deleted declaration cannot
  leave a rule behind silently.
- A model with a `Wire` format may only be read, through its own rule (`rule_<name>_read.go`).

`definitionc` writes `ModelName()`, `Definition()` and `MountOperations(reg)` in `definition_gen.go`. It never writes
`Module`, `New` or the `Store<Entity>()` methods: those are the module's own code.

## 4. Documentation Map

- [docs/diagrams/MODULE_GENERATION.md](docs/diagrams/MODULE_GENERATION.md) — what is generated when `definition.go` is saved,
  with the decisions and the rule skeleton flow.
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — the module file, the rule skeletons, why a skeleton does not compile, and
  what is generated versus written by hand; plus the input contract and the storage resolution.
- [docs/TESTS.md](docs/TESTS.md) — what each test of the module declaration proves.
- [docs/DESIGN.md](docs/DESIGN.md) — decision record for the storage resolution.
- [docs/SYNC_DESIGN.md](docs/SYNC_DESIGN.md) — tool-driven development schema sync.
- [docs/WHY_GENERATED_CODE_IS_FREE.md](docs/WHY_GENERATED_CODE_IS_FREE.md) — why the generated volume costs zero bytes in WASM.
- [docs/WHY_PACKAGE_LEVEL_SCHEMA.md](docs/WHY_PACKAGE_LEVEL_SCHEMA.md) — why `Schema()` returns a package-level variable.
- [docs/diagrams/ORMC_FLOW.md](docs/diagrams/ORMC_FLOW.md) — `Definition` to generated code, error table, runtime validation.
- [docs/diagrams/DB_SYNC.md](docs/diagrams/DB_SYNC.md) — development database schema sync (tool-driven).

Run the tests with `gotest` in the module and in `tests/`.
