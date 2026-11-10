# Module generation — what `definitionc` writes for a module

> When a developer saves `definition.go`, `definitionc` writes the module's operations into `definition_gen.go`
> and creates one rule file per business rule the developer declared. The developer completes those rule files;
> `definitionc` never overwrites them.

## Flow

```mermaid
flowchart TD
    A["definition.go saved or definitionc run"] --> B["parse every XModel (Rules included) and var Definition = model.ModuleDefinition"]
    B -->|"Rules rejected by CheckRules, constants only, bad Entities"| X["error: nothing written"]
    B --> C{"var Definition declared?"}
    C -->|"no"| D["no module code in definition_gen.go; a legacy definition_crud.go removed; no rule files"]
    C -->|"yes"| E{"type Module in the package?"}
    E -->|"no"| X
    E -->|"yes"| F["check: every entity with Allow ≠ 0 is in Entities, every handler exists"]
    F -->|"mismatch"| X
    F --> G["append ModelName, Definition, MountOperations to definition_gen.go"]
    G --> H{"each AddMyBusinessRule action"}
    H -->|"rule_<entity>_<verb>.go missing"| I["create the skeleton (does not compile until completed)"]
    H -->|"exists"| J["leave it untouched"]
    G --> K{"rule file exists for a non-AddMyBusinessRule action"}
    K -->|"yes"| X
```
