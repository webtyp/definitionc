# Tests — module declaration

> What each test of the module declaration proves. Run with `gotest` in the module and in `tests/`.
> Files are in `tests/` unless a path is given.

## Rules and the module declaration (`rules_parse_test.go`)

| Test | Proves |
|---|---|
| `TestRules_UndeclaredActionIsAnError` | an allowed action with no `Rules` entry is refused, naming the model |
| `TestRules_OrphanRuleIsAnError` | a rule for an action the model does not allow is refused |
| `TestRules_ConstantsOnly` | `Rules` must be a literal `model.Rules{…}`, not a variable |
| `TestRules_UnknownKeyIsAnError` | a key of `Rules` that is not an action is refused |
| `TestRules_ModeMustBeAConstant` | a rule mode must be `model.AddMyBusinessRule` or `model.HaveNoBusinessRule` |
| `TestRules_TableWithoutKeyReadMustBeDomain` | a model without a table can only be read through its own rule |
| `TestRules_WireReadOnlyThroughItsRule` | a `Wire` model is accepted only as a read through its rule |
| `TestRules_TableWithoutKeyOnlyRead` | a model without a key field allows only `model.Read` |
| `TestRules_WireMayOnlyRead` | a `Wire` model that allows a write is refused |
| `TestRules_AllowNonConstantIsError` | `Allow` must be written with constants |
| `TestModuleDefinition_NotMistakenForAnEntity` | the variable `Definition` is the module, never generated as an entity |

## The module (`module_gen_test.go`)

| Test | Proves |
|---|---|
| `TestModule_MountsAStoredEntity` | a table entity is mounted through `reg.AddStoredEntity` with its hooks, and has no `Delete` hook without a rule |
| `TestModule_StandardActionsHaveNoHook` | `HaveNoBusinessRule` actions mount no hook and create no rule file |
| `TestModule_MFAIsCarriedToTheStoredEntity` | the `MFA` of the model reaches the stored entity |
| `TestModule_StorePerEntity` | each table entity gets its own `Store<Entity>()` line |
| `TestModule_DeleteHook` | a `Delete` rule is mounted as the `Delete` hook |
| `TestModule_ReadOnlyModelUsesRegistryRead` | a model without a table is mounted through `reg.AddReadOnlyEntity` |
| `TestModule_WireModelUsesRegistryRead` | a `Wire` model is mounted through `reg.AddReadOnlyEntity` as well |
| `TestModule_TwoEntities` | one mount per entity, in the order of `Entities` |
| `TestModule_InternalEntityIsListedButNotMounted` | an entity that allows nothing is listed but not mounted |
| `TestModule_NoAllowNoRuleFile` | an entity that allows nothing creates no rule file and no mount |
| `TestModule_NoDefinitionNoModuleCode` | without `var Definition` there is no module code, and a legacy `definition_crud.go` is removed |
| `TestModule_EntityNotListedIsAnError` | an entity that allows actions must be listed in `Entities` |
| `TestModule_UnknownHandlerIsAnError` | a listed handler must be a declared model |
| `TestModule_HandlerMustEndInModel` | a handler is written `XModel` |
| `TestModule_ModuleTypeIsRequired` | the package must declare `type Module` |
| `TestModule_NameRequired` | the module must have a `Name` |
| `TestModule_Regenerates` | generating twice gives the same `definition_gen.go` |
| `TestGenerate_AllowMethod` | the generated `Allow()` methods return the model's `Allow` |
| `TestGenerate_CompositeItemID` | a composite key is joined with `model.JoinKey` in `Item()` |
| `TestGenerate_NoBuildTag` | the generated module file carries no build tag |

## Revision field (`revision_test.go`)

| Test | Proves |
|---|---|
| `TestRevision_IsCopiedToTheGeneratedSchema` | `Revision: true` is written into the generated schema literal of its field |
| `TestRevision_AbsentIsNotWritten` | a definition without a revision field generates none |
| `TestRevision_BadRevisionIsRefused` | a `Revision` field that is not an integer is refused |

## Rule skeletons (`rule_skeleton_test.go`)

| Test | Proves |
|---|---|
| `TestSkeleton_CreatedForEachDomainAction` | a skeleton is created for each `AddMyBusinessRule` action and no other |
| `TestSkeleton_NotCreatedForStandard` | no skeleton is created when every action is `HaveNoBusinessRule` |
| `TestSkeleton_ExactShapes` | every skeleton carries its exact signature and the completion marker |
| `TestSkeleton_DoesNotCompileUntilCompleted` | a skeleton parses and has no `return`, so the compiler stops the module |
| `TestSkeleton_NeverOverwritesACompletedRule` | a rule the developer wrote is byte-identical after regeneration |
| `TestSkeleton_DeletedFileIsRecreated` | a deleted rule file is recreated as a skeleton |
| `TestSkeleton_OrphanRuleFileIsAnError` | a rule file for an action that is not declared is an error |
| `TestSkeleton_UsesTheModelName` | the rule file takes the entity's model name, not its Go name |

## Generator, parser and scanner (root package)

Files are in the root of the module (`*_test.go`).

| Test | Proves |
|---|---|
| [TestExportSQL_NoModels_ReturnsErrNoModelsFound](../export_sql_test.go) | exporting a root with no models returns `ErrNoModelsFound` |
| [TestModels_SameOrderEveryTimeAndKeepsForeignKeys](../export_sql_test.go) | `Models` returns the same order on every run and keeps foreign keys |
| [TestModels_NoModels_ReturnsErrNoModelsFound](../export_sql_test.go) | `Models` on a root with no models returns `ErrNoModelsFound` |
| [TestModels_KeepsOptionsAndInstantKinds](../export_sql_test.go) | `Models` keeps a select's options and an instant's kind |
| [TestScanModules](../scan_test.go) | the scan finds every definition of the module |
| [TestScanModules_NoSyncer](../scan_test.go) | the scan works without a schema syncer |
| [TestScanModules_EmptyReadonly](../scan_test.go) | the scan of a read-only root creates no file |
| [TestField_WidgetRemoved_HardError](../storage_test.go) | the removed `Widget` key is a hard error naming `Type:` |
| [TestField_MissingType_HardError](../storage_test.go) | a field without `Type:` is a hard error |
| [TestComposition_NilArgument_HardError](../storage_test.go) | `Struct(nil)` is a hard error |
| [TestComposition_MissingArgument_HardError](../storage_test.go) | a composition kind without its argument is a hard error |
| [TestComposition_RefContradiction_HardError](../storage_test.go) | `Ref:` next to a composition kind is a contradiction |
| [TestKind_NonSelfContainedArgument_HardError](../storage_test.go) | a kind argument that names a scanned identifier is a hard error |
| [TestKind_UnknownPackageAlias_HardError](../storage_test.go) | a kind from an unknown package alias is a hard error |
| [TestKind_LocalKind_HardError](../storage_test.go) | a kind declared in the scanned package is a hard error |
| [TestProbe_Failure_SurfacesOutputVerbatim](../storage_test.go) | a failing probe reports the compiler output unchanged |
| [TestProbe_CompositionKindStorage_HardError](../storage_test.go) | a probed kind with composition storage is a hard error |
| [TestProbe_GeneratedSource_And_Resolution](../storage_test.go) | the probe's generated source resolves the real `Storage()` |
| [TestGenerate_ProbedKind_EmitsPackageImport](../storage_test.go) | a probed kind's package is imported by the generated file |
| [TestProbe_Cache_HitAndInvalidate](../storage_test.go) | the probe cache hits on the same inputs and is invalidated by a change |
| [TestParseDefinition_Basic](../generator_test.go) | a definition parses into its name, fields and kinds |
| [TestParseDefinition_Exclude](../generator_test.go) | an excluded field is parsed and left out of the struct |
| [TestGenerate_E2E](../generator_test.go) | a definition generates a compiling file end to end |
| [TestGenerate_RawField](../generator_test.go) | a raw field generates its codec lines |
| [TestGenerate_OmitEmpty](../generator_test.go) | an `OmitEmpty` field is written only when set |
| [TestGenerate_ListHasNoSchema](../generator_test.go) | a list type carries no schema of its own |
| [TestGenerate_FK_SchemaExt](../generator_test.go) | a foreign key appears in `SchemaExt()` |
| [TestGenerate_Exclude_Parallelism](../generator_test.go) | excluded fields keep the parallel order of the schema |
| [TestGenerate_AlwaysOnHelper](../generator_test.go) | the validation helper is always generated |
| [TestGenerate_UnconditionalValidate](../generator_test.go) | `Validate` is generated unconditionally |
| [TestParseDefinition_ItemSlots](../generator_test.go) | the item slots (label, description, badge) are parsed |
| [TestParseDefinition_Options](../generator_test.go) | a select's `Options` table is parsed |
| [TestParseDefinition_TwoItemLabelsFails](../generator_test.go) | two item labels are refused |
| [TestParseDefinition_DescriptionWithoutLabelFails](../generator_test.go) | a description without a label is refused |
| [TestParseDefinition_RefAndOptionsFails](../generator_test.go) | `Ref` and `Options` on one field are refused |
| [TestParseDefinition_UnknownItemSlotFails](../generator_test.go) | an unknown item slot is refused |
| [TestParseDefinition_RowKeyRejected](../generator_test.go) | the replaced `Row` key is rejected |
| [TestGenerate_ItemMethod](../generator_test.go) | `Item()` is generated from the item slots |
| [TestGenerate_ItemOneDescriptionNoJoin](../generator_test.go) | one description is used without `JoinItem` |
| [TestGenerate_ItemDateBadgeIsSeconds](../generator_test.go) | a date badge is rendered in seconds |
| [TestGenerate_NoMarksNoItem](../generator_test.go) | a model with no item marks gets no `Item()` |
| [TestGenerate_Wire](../generator_test.go) | a `Wire` model gets its `Wire()` method |
| [TestParseGenerated](../parse_generated_test.go) | a generated file is parsed back into its struct |
| [TestParseGenerated_MS2](../parse_generated_test.go) | a generated file with a second struct form is parsed |
| [TestParseGenerated_MS7](../parse_generated_test.go) | a generated file with a list form is parsed |
| [TestParseGenerated_NoDB](../parse_generated_test.go) | a generated file of a model without a table is parsed |
