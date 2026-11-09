← [Stage 4](PLAN_STAGE_4_SKELETONS.md) | Next → end

# Stage 5 — existing tests and fixtures, translations, documentation

## 5.1 `tests/crud_gen_test.go` (13 tests) — each one has a destination

Every definition source inside these tests that has `Allow ≠ 0` and no `Rules` is now refused (`must declare its rule`). Port or delete as follows, then delete the file:

| Old test | Destination |
|---|---|
| `TestCrudGen_WrittenWhenGoModRequiresCrud`, `TestCrudGen_Compiles`, `TestCrudGen_ReadOnlyCompiles`, `TestCrudGen_OnlyTableEntities` | replaced by `TestModule_MountsAStoredEntity`, `TestModule_TwoEntities`, `TestModule_ReadOnlyModelUsesRegistryRead` (stage 1) — delete |
| `TestCrudGen_SkippedWithoutCrud` | delete: the generator no longer reads `go.mod` to decide; a module is whatever declares `var Definition` |
| `TestCrudGen_StaleFileRemoved` | replaced by `TestModule_NoDefinitionNoModuleCode` — delete |
| `TestCrudGen_ReadModelGetsAllowNoMount` | replaced by `TestModule_ReadOnlyModelUsesRegistryRead`; keep its second half (a `bad_report` with `Allow: model.Create` and no key field) as `TestRules_TableWithoutKeyOnlyRead` in `tests/rules_parse_test.go`: error contains `only model.Read` |
| `TestCrudGen_AllowOnWireIsError` | keep as `TestRules_WireMayOnlyRead` in `rules_parse_test.go`: `Wire: true, Allow: model.Create, Rules{Create: AddMyBusinessRule}` → error (message now comes from `CheckRules`; assert it contains `Wire`) |
| `TestCrudGen_AllowNonConstantIsError` | keep unchanged, renamed `TestRules_AllowNonConstantIsError` (still `constants only`) |
| `TestCrudGen_AllowMethod`, `TestCrudGen_NoAllowNoFile`, `TestCrudGen_NoBuildTag`, `TestCrudGen_CompositeItemID` | keep, renamed `TestModule_*`/`TestGenerate_*`, adding `Rules:` (HaveNoBusinessRule for every allowed action) to every source and, where they read `definition_crud.go` from the fixture, reading it from the regenerated fixture below |

## 5.2 The fixture module `tests/testdata/fixturecrud`

Rewrite it as a real module (its own `go.mod` already requires crud, input, model, orm, storage; run `go get webtyp.com/crud@v0.0.7 webtyp.com/model@v0.2.18 webtyp.com/router@v0.4.20 webtyp.com/storage@v0.2.0 webtyp.com/events@v0.1.8` (published versions, no `replace`) and `go mod tidy`):

`definition.go` — keep `PatientModel`, `NoteModel`, `CodeEntryModel`; change `PatientModel` to

```go
var Definition = model.ModuleDefinition{Name: "fixturecrud", Entities: []model.Definition{PatientModel}}

var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Create | model.Read | model.Update,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Update: model.HaveNoBusinessRule},
	Fields: model.Fields{ …unchanged… },
}
```

`module.go` (new, hand-written):

```go
package fixturecrud

import "webtyp.com/crud"

// Module is the fixture module: the generated methods hang on it.
type Module struct{ store crud.Store }

// StorePatient is the one line the generated MountOperations reads for the patient table.
func (m *Module) StorePatient() crud.Store { return m.store }
```

Run `definitionc` in the fixture directory: it regenerates `definition_gen.go` (module code included) and creates `rule_patient_create.go`. Complete it:

```go
package fixturecrud

import "webtyp.com/crud"

func (p *Patient) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	return crud.ApplyAndPublish, nil
}
```

Then `cd tests/testdata/fixturecrud && go build ./...` must pass (this is delivery check 4). Commit these generated and written files with the fixture.

## 5.3 `tests/definition.go` and the other in-repo sources

`tests/definition.go` and `tests/fixtureuse/definitions.go`: if any `model.Definition` has `Allow ≠ 0`, add `Rules` (HaveNoBusinessRule for each allowed action) and regenerate `tests/definition_gen.go` by running `definitionc` on `tests/`. `grep -rn "Allow:" --include=*.go . | grep -v _gen.go | grep -v testdata` lists every site; there is none outside `crud_gen_test.go` and the fixture today.

## 5.4 Translations

`langc sync .`: the generator's error messages built with `fmt.Err`/`fmt.Errf` are extracted; fill every empty value for all the languages in `lang.json`, zero untranslated keys, zero reuse warnings (compose shared words at the call site instead of repeating phrases: `Definition.Entities`, `Rules.`, `model.AddMyBusinessRule` are passed as data).
The skeleton texts are generated source for the developer, not messages: they are not translated.

## 5.5 Documentation

- `docs/diagrams/MODULE_GENERATION.md` — three parts: context header (what the flow generates and for whom); the `flowchart TD` of stage 1; a numbered walkthrough, one item per node.
- `README.md` (≤ 150 lines, four ordered sections): *Why This Exists* Without/With (without: a hand-written `ops.go` that can forget an entity or a rule; with: declare once, a skeleton that does not compile until completed); *Step-by-Step Guide* with the `Definition` + `Rules` example of the Goal, the five signatures and `Module` + `Store<Entity>()`; *Documentation Map* links `MODULE_GENERATION.md`.
- `docs/ARCHITECTURE.md` (four sections): the module file, the rule skeletons, why the skeleton does not compile, why `ModelName`, `Definition` and `MountOperations` are generated and `Module`/`Store<Entity>()`/`New` are not.
- `docs/DESIGN.md`/`docs/WHY_*.md`: replace `ormc`, `_orm.go`, `Mount<Entity>`, `<Entity>Rules` where they describe the old mount. `docs/TESTS.md`: one row per new test.
- Do not link `docs/PLAN*.md` from permanent docs.

Commit: `docs: module declaration`.
