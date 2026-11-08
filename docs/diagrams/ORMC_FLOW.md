# ORMC Flow — Definition → generated code

> Phase-B flow (Kind unification): the input is a hand-written
> `model.Definition` literal; roles and storage derive from the `Type:`
> constructors. The old struct-tag inference (`db:`/`input:` tags, widget
> defaults per Go type) is GONE. Contract details:
> [../ARCHITECTURE.md](../ARCHITECTURE.md); storage decision:
> [../DESIGN.md](../DESIGN.md); DB sync: [DB_SYNC.md](DB_SYNC.md).

## Generation flow

```mermaid
flowchart TD
    Def["definition.go<br/>var UserModel = model.Definition{...}"]
    Parse["go/ast parse of the literal<br/>(never compiles the user package)"]
    Field["per Field: Name, Type: constructor expr,<br/>NotNull, DB, Permitted, Ref (scalar FK)"]
    Kind{"Type: constructor?"}
    Builtin["model.* → builtin storage table<br/>Struct/StructSlice: Go type from ref arg"]
    Probe["other packages → dependency probe:<br/>temp main imports kind pkgs,<br/>go run executes real Storage(),<br/>cached by go.mod hash + constructor set"]
    Emit["emit file_gen.go:<br/>struct (storage-mapped Go types),<br/>Schema() with Type: verbatim,<br/>codec + Validate + User_.Field helpers,<br/>SchemaExt() []ddlc.FieldExt if FKs"]
    Def --> Parse
    Parse --> Field
    Field --> Kind
    Kind --> Builtin
    Kind --> Probe
    Builtin --> Emit
    Probe --> Emit
```

## Hard generation errors (never guess)

| Input | Error |
|---|---|
| field without `Type:` | kind required (fail-closed guard) |
| `Widget:` present | removed by Kind unification — declare the kind in `Type:` |
| `Ref:` next to `Struct(ref)`/`StructSlice(ref)` | contradiction: composition ref travels in the constructor |
| composition constructor with missing/nil ref | ref required |
| constructor argument referencing the scanned package | probe cannot import the user's package — literals only |
| custom kind in the same package as its Definitions | custom kinds live in their own package |
| probe compile/run failure | surfaces compiler output verbatim |
| probed kind returning struct/structslice storage | custom composition kinds unsupported |

## Runtime validation (unchanged, fail-closed)

```mermaid
flowchart TD
    Front["FRONTEND wasm<br/>form submit"]
    Back["BACKEND server<br/>orm insert/update"]
    V["model.ValidateFields<br/>(generated Validate delegates here)"]
    F["per field: NotNull → Kind.Validate → Permitted"]
    R["same result front and back"]
    Front --> V
    Back --> V
    V --> F
    F --> R
```
