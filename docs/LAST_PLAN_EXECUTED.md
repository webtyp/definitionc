---
PLAN: "feat!: definitionc generates the module's MountOperations from model.ModuleDefinition and Rules, and creates one rule skeleton per AddMyBusinessRule action"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 10740115110268957738
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit after every stage, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> This repo is backend tooling: stdlib (`go/ast`, `go/parser`, `go/format`, `os`, `strings`) is legitimate here, do NOT "fix" it to `webtyp/fmt`.
> Never edit a generated file (`definition_gen.go`) by hand: a build error there means the generator is wrong.

Stage files: [1 tests](PLAN_STAGE_1_TESTS.md) · [2 parsing](PLAN_STAGE_2_PARSING.md) · [3 module file](PLAN_STAGE_3_MODULE_FILE.md) · [4 rule skeletons](PLAN_STAGE_4_SKELETONS.md) · [5 docs](PLAN_STAGE_5_DOCS.md) · [6 revision](PLAN_STAGE_6_REVISION.md).

## Goal

A module is declared once, in `definition.go`:

```go
var Definition = model.ModuleDefinition{Name: "patient_directory", Entities: []model.Definition{PatientModel}}

var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Create | model.Read | model.Update,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.AddMyBusinessRule, Update: model.AddMyBusinessRule},
	Fields: …,
}
```

From it `definitionc` generates, inside the one `definition_gen.go`, the module's `ModelName()`, `Definition()` and `MountOperations(reg)`; and for every action declared `model.AddMyBusinessRule` it creates once a skeleton
`rule_<entity>_<verb>.go` that does not compile until the developer (or the LLM) completes it. An action declared `model.HaveNoBusinessRule` generates no file and no hook. The hand-written `ops.go` mounting and the
old `Mount<Entity>(reg, store, <Entity>Rules)` / `<Entity>Rules` disappear.

What each rule looks like (receiver = the record; `m` = the module, so a rule reaches its `Deps`; the signatures are fixed by decisions D31, D35, D38, D39 of the architecture master):

| Verb | File | Signature |
|---|---|---|
| create | `rule_patient_create.go` | `func (p *Patient) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` |
| update | `rule_patient_update.go` | `func (next *Patient) Update(a crud.CurrentUser, tx crud.Tx, old *Patient, fields []string, m *Module) (crud.RuleAnswer, error)` |
| delete | `rule_patient_delete.go` | `func (p *Patient) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error)` |
| search (table, `Rules.Read: AddMyBusinessRule`) | `rule_patient_search.go` | `func (p *Patient) Search(a crud.CurrentUser, text string, m *Module) ([]storage.Condition, error)` |
| read (model without a table or `Wire`, `Rules.Read: AddMyBusinessRule`) | `rule_health_read.go` | `func (p *Health) Read(a crud.CurrentUser, args crud.ReadArgs, m *Module) (model.ModelSlice, error)` |

`Search` is called on a zero record (`(&Patient{}).Search(…)`): it uses no record data. A `Read` of a table with `Rules.Read: HaveNoBusinessRule` is the default read of `crud` (id, search by the item label, page, cap, tenant).

## Out of scope (other plans / open questions)

- Effects (`crud.Then`, `Definition.Effects`, `effect_*.go`): plan A6.11, after this one.
- The typed read filter (`NewFilter`) is deleted: nothing in `veltylabs` uses it (decision D42). MFA is now `model.Definition.MFA` (model plan) and the generated `crud.StoredEntity` carries it.
- The module's `type Module`, `New` and one store method per table entity stay hand-written (`module.go`): `func (m *Module) Store<Entity>() crud.Store` (e.g. `StorePatient`), one line each. One way for every module, with one store or several (decision D42).

## Ecosystem search

Searched `ModuleDefinition`, `Rules`, `rule_`, `MountOperations`, `writeCrudFile` over `webtyp/*` and `veltylabs/*`: `definitionc/crud_gen.go` is the only generator of mounting code (**replace** it); `model.ModuleDefinition`, `model.Rules`, `Definition.CheckRules`, `ModuleDefinition.Check` come from the `model` plan (**reuse**: this repo imports `webtyp.com/model` already and calls `CheckRules` instead of re-implementing it).
`app` (the dev daemon) calls this Generator on every save of `definition.go` (`NewFileEvent`), so skeletons are created by saving the file; `app` needs no change.

## Design gate

1. **Prior art** (unverified in this session): `protoc`/`sqlc` generate code from one declaration; Rails generators create a file once and never overwrite it.
2. **Novice-name test**: `rule_patient_create.go` = "rule: create patient".
3. **Ledger**: concepts +1 (`rule_*.go`) −2 (`<Entity>Rules`, `Mount<Entity>`); files a module writes by hand −1 (`ops.go`), +1 per AddMyBusinessRule action; ways to mount an entity 2 → 1.
4. **Where**: the generator owns generation; the types live in `model`, the engine in `crud`.
5. **Deletes**: `crud_gen.go` (`writeCrudFile`, `requiresCrud`), the `Wire` + `Allow` refusal in `parseDefinition`, `tests/crud_gen_test.go`, `tests/testdata/fixturecrud/definition_crud.go` (the file disappears: its content is part of the regenerated `definition_gen.go`).

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
```

Dependencies (all published, no `replace`): `go get webtyp.com/model@v0.2.18 webtyp.com/definitioncheck@v0.0.3 && go mod tidy`; `tests/go.mod` and `tests/testdata/fixturecrud/go.mod` get the same two lines plus `go get webtyp.com/crud@v0.0.7` (the generated code targets `crud` v0.0.7: `StoredEntity`, `AddStoredEntity`, `AddReadOnlyEntity`, `CurrentUser`, `RuleAnswer`).
`model` v0.2.18 (defines `Rules`, `RuleMode`, `AddMyBusinessRule`, `HaveNoBusinessRule`, `ModuleDefinition`, `Action.Verb`, `FieldDB.Revision`) and `definitioncheck` v0.0.3 (defines `CheckRules`, `CheckModule`) are published.

## APIs used

| symbol | exact signature | file |
|---|---|---|
| `model.RuleMode` | `type RuleMode uint8`; `AddMyBusinessRule` = 1, `HaveNoBusinessRule` = 2, zero = undeclared | `model/rules.go` (model plan) |
| `model.Rules` | `struct{ Create, Read, Update, Delete RuleMode }` | `model/rules.go` |
| `model.Definition` | `Name string; Fields Fields; Wire bool; Allow Action; Rules Rules; MFA Action` | `model/definition.go` |
| `definitioncheck.CheckRules` | `func CheckRules(def model.Definition) error` (undeclared allowed action, orphan rule, MFA outside Allow, Wire model that is not exactly `Read` + `AddMyBusinessRule`) | `model/rules.go` |
| `model.Action` | `uint8` bit set `Create, Read, Update, Delete`; `String()` → CRUD letters | `model/rbac.go:83` |
| `Generator`, `StructInfo`, `exprToString`, `parseAllow`, `ToSnakeCase`, `GeneratedHeader`, `GeneratedSuffix`, `DefinitionFile` | this repo: read `generator.go`, `parse_definition.go`, `generate.go:27-446` before editing | |
| `fmt.Errf` | `webtyp.com/fmt`: formats an error (used in `parse_definition.go`) | `fmt/` |

If a signature differs from this table, stop and name the symbol in the PR; never adapt it by guessing.

## Stages

| Stage | Files | Commit |
|---|---|---|
| 1 | `tests/module_gen_test.go`, `tests/rule_skeleton_test.go`, `tests/rules_parse_test.go` (red) | `test: module mount and rule skeletons (red)` |
| 2 | `generator.go`, `parse_definition.go`, `module_parse.go` | `feat: parse Rules and model.ModuleDefinition` |
| 3 | `module_gen.go` (replaces `crud_gen.go`), `generate.go` (module code appended to the one generated file) | `feat!: generate the module's MountOperations` |
| 4 | `rule_skeleton.go` | `feat: one rule skeleton per AddMyBusinessRule action` |
| 5 | `tests/testdata/fixturecrud/*`, `lang.json`, `README.md`, `docs/*` | `docs: module declaration` |
| 6 | `generator.go`, `parse_definition.go`, `generate.go`, `tests/revision_test.go` | `feat: Revision in the field literal` |

## Delivery check

1. `gotest` in the module and in `tests/` (a PR that does not compile, or whose tests could not run, is a failed delivery).
2. `langc sync .` → zero untranslated keys, zero reuse warnings (new texts: the errors of stages 2–4).
3. `grep -rn "Mount%s\|%sRules\|writeCrudFile\|requiresCrud\|crud\.Caller" --include=*.go .` → empty.
4. The fixture module `tests/testdata/fixturecrud` regenerates with `definitionc` and, once its three rule files are completed with the bodies of stage 5, `go build ./...` inside it passes.
5. `git diff --stat main` lists only the files named in the stages.
6. Consumers (the 18 modules, `app`) are regenerated in their own plans; do not touch them here.
