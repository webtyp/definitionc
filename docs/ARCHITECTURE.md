# definitionc — Architecture

> WHAT definitionc is and WHY it works the way it does. The debate behind the
> storage-resolution decision lives in [DESIGN.md](DESIGN.md).

## What definitionc is

`webtyp.com/definitionc` is the **code generator** of the webtyp data
stack — nothing else. It reads hand-written `model.Definition` literals and
emits the concrete Go artifacts the runtime consumes: the plain struct, the
`Fielder` methods (`Schema()`, `Pointers()`, codec), `Validate()`, typed
query-field helpers (`User_.Email`), and DDL FK metadata (`SchemaExt()`).

It was split out of `webtyp.com/orm` (2026-07-10). The split
boundaries:

| Repo | Role | definitionc's relationship |
|---|---|---|
| `webtyp/model` | field/kind contract (`Definition`, `Field`, `Kind`) | definitionc's INPUT contract |
| `webtyp/definitionc` | this repo: generator + `cmd/definitionc` | — |
| `webtyp/ddlc` | DDL migration tool (`tui`) | not a dependency of definitionc anymore |
| `webtyp/orm` | query/scan/sync runtime | imported by generated read helpers |
| `webtyp/sqlmcp` | MCP provider over the stack | consumer, not a dependency |

definitionc runs in two modes: standalone CLI (`cmd/definitionc`) and **live watcher**
inside the dev tool (`webtyp/app` feeds file events; see
[SYNC_DESIGN.md](SYNC_DESIGN.md) and
[diagrams/DB_SYNC.md](diagrams/DB_SYNC.md)). The watcher mode is a hard
architectural constraint: generation runs on every file save, often against
code that does not compile yet.

## Input contract (authoring)

A model is a `model.Definition` composite literal. Each field declares its
kind in the single `Type:` slot as a **constructor expression**:

```go
var UserModel = model.Definition{Fields: []model.Field{
    {Name: "email",    Type: input.Email(), NotNull: true}, // form kind (validates + renders)
    {Name: "status",   Type: model.Text()},                  // base kind (validates only)
    {Name: "address",  Type: model.Struct(&AddressModel)},   // composition: ref IN the constructor
    {Name: "staff_id", Type: model.Int(), Ref: &StaffModel}, // scalar FK: Ref's ONLY meaning
}}
```

Contract rules (all violations are **hard generation errors** — definitionc never
guesses):

- `Type:` is mandatory on every field (fail-closed; the compile-time guard
  for `model`'s always-on validation).
- `Widget:` no longer exists (Kind unification deleted it; the actionable
  error tells the author to move the constructor into `Type:`).
- Composition refs travel inside `Struct(ref)`/`StructSlice(ref)`;
  `Field.Ref` means scalar foreign key and nothing else. `Ref:` next to a
  composition constructor is a contradiction → error.
- A project-custom kind must live in its **own package**, separate from the
  Definitions that use it (see storage resolution below for why).
- Constructor arguments must be self-contained literals — they may not
  reference identifiers of the scanned package.

## Generation mechanism: AST over the user's source, always

definitionc parses the Definition literal with `go/ast`. It **never compiles or
executes the user's package**. Rationale (settled in the model repo,
`docs/ARCHITECTURE.md` §9,
<https://github.com/webtyp/model/blob/main/docs/ARCHITECTURE.md>):

1. The watcher regenerates on every save; mid-edit code frequently does not
   compile — exactly when regeneration is needed.
2. Chicken-and-egg: the generated file is what makes the package compile.
3. AST needs no build and creates no import cycle between generator and
   model.

## Storage resolution

Generated struct fields need each kind's `FieldType` at generation time,
but the new contract keeps storage behind the `Kind.Storage()` method — a
runtime value, no longer a constant in the user's source. Resolution order:

1. **Builtin table** for the closed set of `model.*` constructors
   (`model.Text` → `FieldText`, …; the two composition constructors derive
   their Go type from the captured ref argument).
2. **Cached dependency probe** for every other constructor: definitionc generates
   a throwaway `main` that imports the kinds' packages, executes each
   captured constructor, and reads the REAL `Storage()` value. Key
   distinction: the no-compile rule above protects the **user's package**;
   kind packages are ordinary, always-compilable **dependencies** of the
   scanned module. Results are cached by (module root, `go.mod` hash,
   constructor set) — watcher saves with no new constructors never re-run
   the probe. There is no annotation, registry, or marker to author, and no
   second copy of the storage fact that could drift: the probe executes the
   single source of truth. Alternatives considered and rejected:
   [DESIGN.md](DESIGN.md).
3. **Hard error** otherwise — probe failures surface the compiler output
   verbatim; a probed kind resolving to struct/structslice storage is
   rejected (custom composition kinds are unsupported by design).

## Module declaration

A module is declared once, in `definition.go`: its name, the entities it serves, and for each entity the actions it
allows and their rules. Four parts make it work.

### The module file

`var Definition = model.ModuleDefinition{Name: "...", Entities: []model.Definition{PatientModel}}` is read through the AST
like any model. Every entity with `Allow` must be listed in `Entities`, every listed handler must be declared, and the
package must have a `type Module` (written in `module.go`). Those checks run before anything is written, so a mismatch
leaves the package as it was.

The generator appends `ModelName()`, `Definition()` and `MountOperations(reg crud.Registry)` to `definition_gen.go`, the
one generated file of the package. `MountOperations` offers each entity through `reg.AddStoredEntity` (a table) or
`reg.AddReadOnlyEntity` (a model without a table, or a `Wire` format), with the `Rules` of the model. The store of each
table is the one line the developer writes: `func (m *Module) StorePatient() crud.Store`.

### Rule skeletons

For every action declared `model.AddMyBusinessRule`, `definitionc` creates `rule_<entity>_<verb>.go` once. The file is
the developer's from then on: `definitionc` never rewrites it, and a deleted file is recreated as a skeleton. A rule file
for an action that is not declared `AddMyBusinessRule` is an error. An action declared `model.HaveNoBusinessRule` creates
no file and mounts no hook. The signatures are listed in the [README](../README.md#3-the-rule-signatures).

### Why a skeleton does not compile

The skeleton declares its outputs (`var decision crud.RuleAnswer`, `var rejected error`) and returns nothing. The Go
compiler then reports `declared and not used` and `missing return` in that file, next to the marker comment that says
where to write the rule. The module is not runnable until the rule is written, which is the point: a business rule
cannot be forgotten silently. A completed rule that returns the zero `crud.RuleAnswer` is refused at run time by crud
(`ErrNoAnswer`), so the zero value never means "no decision".

### What is generated and what is written by hand

`definitionc` generates what is the same for every module: `ModelName()`, `Definition()` and `MountOperations`, which are
derived from the declaration and cannot disagree with it. It writes the rule skeletons because they are the list of
decisions the module still owes.

It never generates `Module`, `New` or `Store<Entity>()`: the type that holds the connections, the constructor that
receives them, and the store that persists each table are choices of the module (its database, its tenant, its tests).
Hand-written code stays readable and replaceable, and the generator depends on exactly one name per table, `Store<Entity>`.

## Generated-code contract

Generated files (`*_gen.go`) import:

- `webtyp/model` — always (schema types).
- `webtyp/orm` — when the model has DB role (query helpers `*orm.QB`).
- `webtyp/crud` — when the module declares a `Definition` (`MountOperations` takes a `crud.Registry`).
- `webtyp/storage` — when a table declares `Rules.Read: model.AddMyBusinessRule` (its `Search` returns `storage.Condition`).
- The kind constructors' packages (e.g. `webtyp/input`) — `Schema()` re-emits every `Type:` constructor expression **verbatim**; definitionc passes
  kinds through without understanding them.

The typed-field helpers (`User_.Email` used by `.Where(...).Eq(...)`) are
part of the stable contract and are always generated. On why the generated
volume costs nothing in WASM binaries:
[WHY_GENERATED_CODE_IS_FREE.md](WHY_GENERATED_CODE_IS_FREE.md).

Flow diagram: [diagrams/ORMC_FLOW.md](diagrams/ORMC_FLOW.md). Module flow: [diagrams/MODULE_GENERATION.md](diagrams/MODULE_GENERATION.md).

## Other generated contracts

- **List items.** `definitionc` generates `func (m *X) Item() model.Item` for every model with a field marked `Item:`
  (`model.ItemLabel`, `model.ItemDescription`, `model.ItemBadge`). Several descriptions are joined with `model.JoinItem`; a
  date marked `ItemBadge` is rendered as seconds since 1970. Invalid declarations are rejected by `model.ValidateItem`:
  at most one label and one badge, description or badge only with a label, listed models need a primary key, and only
  scalar fields can be shown. A composite key is joined with `model.JoinKey`.
- **Instants.** `model.UnixTime()` and `model.NanoTime()` generate fields typed `time.Unix` and `time.Nano`, imported
  automatically. They are stored as integers: `Pointers()` exposes them as `(*int64)(&m.Field)`, and the codec converts
  explicitly, so storage backends need no change while the compiler rejects a nanosecond instant in a seconds field.
- **Models.** `Models(root) ([]model.Model, error)` returns the database models of a root in file order, with their foreign
  keys in `SchemaExt()`, a select's `Options` table, and an instant's kind. It returns `ErrNoModelsFound` when there is
  none; `webtyp/erd` uses it.

## Failure doctrine

Every ambiguity is a loud generation error naming the field/constructor and
the fix. definitionc never silently defaults a storage type, never skips a field
it cannot parse, and never rewrites user source beyond the generated
`*_gen.go` file.
