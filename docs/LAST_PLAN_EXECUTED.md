---
PLAN: "feat!: ormc generates one Mount<Entity> with typed <Entity>Rules, only for tables that Allow actions"
EXECUTOR: antigravity
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.

# Plan: `model_crud.go` = one `Mount<Entity>` per allowed table

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go get webtyp.com/model@latest webtyp.com/crud@latest
(cd tests && go get webtyp.com/crud@latest webtyp.com/model@latest)
(cd tests/testdata/fixturecrud && go get webtyp.com/crud@latest webtyp.com/model@latest webtyp.com/storage@v0.1.18)
```
`crud@latest` must contain `crud.Table`, `crud.Tx`, `Registry.Table` (plan A6.2) and `model@latest` must contain
`Definition.Allow` (A6.1); if `go doc webtyp.com/crud Table` fails, stop. No `replace` anywhere.

## Why

Decisions D19/D20 of `webtyp/docs/ARCHITECTURE_MASTER_PLAN.md`. Today `crud_gen.go` writes four functions per table entity
(`Mount<E>Create/Read/Update/Delete`) whether the module uses them or not (`work_schedule`: 16 written, 1 used), and modules
mount all four and answer the unwanted ones with `"not supported"` stubs. Now `crud` persists the table; `ormc` writes, only
for a table whose `model.Definition` sets `Allow`, one `Mount<E>` that builds a `crud.Table` from typed hooks.

## APIs used

| Symbol | Signature |
|---|---|
| `crud.Registry.Table` | `Table(t crud.Table)` |
| `crud.Table` | `struct{ Model model.Model; NewList func() model.ModelSlice; Store crud.Store; Create func(c Caller, tx Tx, rec model.Model) error; Update func(c Caller, tx Tx, old, next model.Model, fields []string) error; Delete func(c Caller, tx Tx, rec model.Model) error; Search func(c Caller, text string) ([]storage.Condition, error); NewFilter func() model.Model; Committed func(c Caller, action model.Action, rec model.Model, fields []string); MFA model.Action }` |
| `model.Definition.Allow` | `model.Action` (`model.Create`, `model.Read`, `model.Update`, `model.Delete`, combined with `\|`) |

If a signature differs, stop and name the symbol; never adapt it by guessing.

## Stage 1 — parse `Allow` (`parse_definition.go`, `generator.go`)

`StructInfo` gains `Allow uint8` (bit set, same bits as `model.Action`: Create=1, Read=2, Update=4, Delete=8). Parse the
`Allow` key of a `model.Definition` literal: an `*ast.SelectorExpr` (`model.Read`) or an `*ast.BinaryExpr` with `token.OR`
of selectors, recursively; the selector names map `Create→1, Read→2, Update→4, Delete→8`. Any other expression →
error `field Allow of %s: write it as model.Create | model.Read | … (constants only)`. `Allow` with `Wire: true` → error
`%s: a Wire definition cannot Allow table actions`. A definition with no DB field (a read model: a report or computed view, D18) may set **only** `Allow: model.Read`: it gets
`Allow()` on the entity and the list (so its screen works with `view.NewCallerLister`) and **no** `Mount` — the module mounts
it with `crud.Registry.Read`. Any other action on it → error `%s: only model.Read can be allowed on a model without a key field; Create/Update/Delete need
a primary key declared with DB (a table, or a resource served by the module's own crud.Store)`. A resource served by its own `crud.Store`
(D21: e.g. `access.RoleRecord`, `auth.LANIdentity`) declares its key with `DB: &model.FieldDB{PK: true}`, so `ormc` treats it as a table and
generates its `Mount`; it is simply never listed in `migrate/`.

## Stage 2 — `model_orm.go`: `Allow()` for every table entity

Next to the other generated methods of each table entity (not Wire, has DB) and of each read model with `Allow: model.Read`:
```go
func (*Post) Allow() model.Action     { return PostModel.Allow }
func (*PostList) Allow() model.Action { return PostModel.Allow } // makes *PostList a model.TableSlice
```
A table field named `allow` would collide with the method: error `%s: a field named allow collides with the generated Allow() method; rename the column`.

### Composite keys in `Item()` (D22)

For a table whose schema has more than one `IsPK()` field, the generated `Item()` sets `ID: model.JoinKey(<pk fields in schema order, as text>)`
(a single key keeps today's `ID`). Test `TestCrudGen_CompositeItemID`: fixture table with PK (room_id, equipment_id) → generated `Item()` contains
`model.JoinKey(`.

## Stage 3 — `crud_gen.go`: replace the four functions by one

`requiresCrud` unchanged. `tableEntities` = table entities **with `Allow != 0`** (no `Allow` → nothing generated; a package with
none removes `model_crud.go`, existing behaviour). For each, write (only the hook fields of allowed actions; `Search`, `NewFilter`
only when Read is allowed):

```go
// PostRules are the domain rules of Post. Every hook is optional (nil = no rule) and runs per record inside the transaction;
// Committed runs after the commit.
type PostRules struct {
	Create    func(c crud.Caller, tx crud.Tx, p *Post) error                       // only when Allow has Create
	Update    func(c crud.Caller, tx crud.Tx, old, next *Post, fields []string) error // only when Allow has Update
	Delete    func(c crud.Caller, tx crud.Tx, p *Post) error                       // only when Allow has Delete
	Search    func(c crud.Caller, text string) ([]storage.Condition, error)       // only when Allow has Read
	NewFilter func() model.Model                                                  // only when Allow has Read
	Committed func(c crud.Caller, action model.Action, p *Post, fields []string)
	MFA       model.Action
}

// MountPost offers what PostModel.Allow lists, persisted by store, with rules.
func MountPost(reg crud.Registry, store crud.Store, rules PostRules) {
	t := crud.Table{Model: &Post{}, NewList: func() model.ModelSlice { return &PostList{} }, Store: store,
		Search: rules.Search, NewFilter: rules.NewFilter, MFA: rules.MFA}
	if rules.Create != nil {
		t.Create = func(c crud.Caller, tx crud.Tx, rec model.Model) error { return rules.Create(c, tx, rec.(*Post)) }
	}
	if rules.Update != nil {
		t.Update = func(c crud.Caller, tx crud.Tx, old, next model.Model, fields []string) error {
			return rules.Update(c, tx, old.(*Post), next.(*Post), fields)
		}
	}
	if rules.Delete != nil {
		t.Delete = func(c crud.Caller, tx crud.Tx, rec model.Model) error { return rules.Delete(c, tx, rec.(*Post)) }
	}
	if rules.Committed != nil {
		t.Committed = func(c crud.Caller, a model.Action, rec model.Model, fields []string) { rules.Committed(c, a, rec.(*Post), fields) }
	}
	reg.Table(t)
}
```
`model_crud.go` carries **no build tag** (as today): a module's demo (`seed.Demo`, built with `-tags browser` for WASM) mounts the module through `crud.LoopbackActingAs`, so `MountPatient` must exist in that build; in an app's browser binary nothing calls it and the linker drops it. When the table does not allow Read, `PostRules` has no `Search`/`NewFilter` fields **and** the generated `MountPost` omits `Search: rules.Search, NewFilter: rules.NewFilter` (it must compile). Same for `Create`/`Update`/`Delete`: no field, no assignment. Imports of `model_crud.go`: `webtyp.com/crud`, `webtyp.com/model`, and `webtyp.com/storage` only when some entity allows Read.
Delete the old template (`Mount%sCreate/Read/Update/Delete`): `grep -n "Mount%sCreate\|Mount%sUpdate" crud_gen.go` → empty.

## Stage 4 — tests (`tests/crud_gen_test.go`, fixture `tests/testdata/fixturecrud`, rows in `docs/TESTS.md`)

Fixture `model.go`: `PatientModel` gets `Allow: model.Create | model.Read | model.Update` (keeps its `input.Text()` field);
add `NoteModel` (table, **no** `Allow`); keep `CodeEntryModel` (form without table).

| Test | Expected |
|---|---|
| `TestCrudGen_Compiles` | after `Run`, `model_crud.go` has `func MountPatient(` and `type PatientRules struct`, `PatientRules` has `Create` and `Update` fields and **no** `Delete` field; no `MountNote`; no `MountCodeEntry`; `go vet ./...` in the fixture passes |
| `TestCrudGen_AllowMethod` | `model_orm.go` has `func (*Patient) Allow() model.Action { return PatientModel.Allow }`, `func (*PatientList) Allow()` and `func (*Note) Allow()` |
| `TestCrudGen_ReadOnlyCompiles` | a table with `Allow: model.Read` only → generated code has no `rules.Create` and the fixture `go vet` passes |
| `TestCrudGen_AllowOnWireIsError` | temp dir, definition with `Wire: true` and `Allow: model.Read` → `Run` error containing `cannot Allow` |
| `TestCrudGen_ReadModelGetsAllowNoMount` | definition without DB fields and `Allow: model.Read` → `Allow()` generated, no `Mount`; with `Allow: model.Create` → error `only model.Read` |
| `TestCrudGen_AllowNonConstantIsError` | `Allow: someVar` → `Run` error containing `constants only` |
| `TestCrudGen_NoBuildTag` | the generated `model_crud.go` contains no `//go:build` line |
| `TestCrudGen_NoAllowNoFile` | package requiring crud whose only table has no `Allow` → no `model_crud.go` |

Update the existing tests that expect `MountPatientCreate` (`TestCrudGen_WrittenWhenGoModRequiresCrud`, `TestCrudGen_OnlyTableEntities`,
`TestCrudGen_StaleFileRemoved`): their fixtures add `Allow` and they expect `MountPatient`.

## Stage 5 — documentation

`README.md` and `docs/ARCHITECTURE.md`: replace the four-function section with «`Allow` in the definition → `Mount<E>(reg, store, <E>Rules{…})`»,
the generated example above, and the rule «no `Allow`, nothing generated».

## Delivery check

```bash
gotest
(cd tests && go test ./...)
grep -n "Mount%sCreate\|Mount%sUpdate" crud_gen.go   # empty
```
A PR that does not compile, or whose tests could not run, is a failed delivery.

| Stage | Files |
|---|---|
| 1 | `parse_definition.go`, `generator.go` |
| 2 | `generate.go` |
| 3 | `crud_gen.go` |
| 4 | `tests/crud_gen_test.go`, `tests/testdata/fixturecrud/*`, `docs/TESTS.md` |
| 5 | `README.md`, `docs/ARCHITECTURE.md` |
