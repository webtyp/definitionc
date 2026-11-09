← [Stage 5](PLAN_STAGE_5_DOCS.md) | Plan: [PLAN.md](PLAN.md)

# Stage 6 — `Revision: true` in a field's `DB` literal

`model.FieldDB` gained `Revision bool` (model plan, decision D48). `definitionc` must read it from `definition.go` and write it into the generated `Schema()`, exactly as it does for `Unique`; otherwise the generated code silently drops the flag and `crud` never sees it.
Do this stage **after** stage 5 (it touches the same parser and generator).

1. `generator.go` (the struct next to `Unique bool`, line ~44): add `Revision bool` to the field info, and where `Unique: f.Unique` is copied (line ~367) copy `Revision: f.Revision` too.
2. `parse_definition.go` (the `case "Unique":` near line 313): add `case "Revision":` handled the same way (a boolean literal `true`/`false`; anything else is the same error `Unique` gives, naming `Revision`).
3. `generate.go`: both places that test `f.PK || f.Unique || f.AutoInc` (lines ~154 and ~555) also test `|| f.Revision`; where `parts = append(parts, "Unique: true")` is written, add `if f.Revision { parts = append(parts, "Revision: true") }`, and where `field.DB = &model.FieldDB{PK: f.PK, Unique: f.Unique, AutoInc: f.AutoInc}` is built add `Revision: f.Revision`.
4. Tests first (`tests/revision_test.go`, helper `generate` of stage 1): `TestRevision_IsCopiedToTheGeneratedSchema` — a `definition.go` whose `revision` field is `Type: model.Int(), DB: &model.FieldDB{Revision: true}` → `definition_gen.go` contains `Revision: true` inside that field's `DB`; and `TestRevision_AbsentIsNotWritten` — a definition without it → the generated file contains no `Revision`. Add both to `docs/TESTS.md`.
5. The judge (`definitioncheck.CheckRules`) already refuses a bad `Revision`; add `TestRevision_BadRevisionIsRefused`: a text field marked `Revision: true` → `Run` returns an error containing `integer`.

`go get webtyp.com/model@v0.2.18 webtyp.com/definitioncheck@v0.0.3`. `gotest` passes. Commit: `feat: Revision in the field literal`.
