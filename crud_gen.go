package ormc

import (
	"bufio"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"webtyp.com/fmt"
	"webtyp.com/model"
)

const (
	crudImport     = "webtyp.com/crud"
	crudFileSuffix = "_crud.go"
)

// requiresCrud reports whether the nearest go.mod walking up from dir requires webtyp.com/crud.
func requiresCrud(dir string) (bool, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	curr := abs
	for {
		modPath := filepath.Join(curr, "go.mod")
		if f, err := os.Open(modPath); err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "require") {
					line = strings.TrimSpace(strings.TrimPrefix(line, "require"))
					line = strings.TrimPrefix(line, "(")
					line = strings.TrimSpace(line)
				}
				fields := strings.Fields(line)
				if len(fields) > 0 && fields[0] == crudImport {
					return true, nil
				}
			}
			return false, scanner.Err()
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return false, nil
}

// writeCrudFile writes (or removes) <source>_crud.go for the table entities of infos.
func (o *Generator) writeCrudFile(infos []StructInfo, sourceFile string) error {
	outName := fmt.Convert(sourceFile).TrimSuffix(".go").String() + crudFileSuffix

	hasCrud, err := requiresCrud(filepath.Dir(sourceFile))
	if err != nil {
		return err
	}

	var tableEntities []StructInfo
	if hasCrud {
		for _, info := range infos {
			if !info.NoDB && !info.Wire && info.Allow != 0 {
				tableEntities = append(tableEntities, info)
			}
		}
	}

	if !hasCrud || len(tableEntities) == 0 {
		err := os.Remove(outName)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	hasRead := false
	for _, info := range tableEntities {
		if info.Allow&uint8(model.Read) != 0 {
			hasRead = true
			break
		}
	}

	buf := fmt.Convert()
	buf.Write(GeneratedHeader + "\n\n")
	buf.Write(fmt.Sprintf("package %s\n\n", tableEntities[0].PackageName))
	buf.Write("import (\n")
	buf.Write(fmt.Sprintf("\t%q\n", crudImport))
	buf.Write("\t\"webtyp.com/model\"\n")
	if hasRead {
		buf.Write("\t\"webtyp.com/storage\"\n")
	}
	buf.Write(")\n\n")

	for _, info := range tableEntities {
		name := info.Name
		hasCreate := info.Allow&uint8(model.Create) != 0
		hasRead := info.Allow&uint8(model.Read) != 0
		hasUpdate := info.Allow&uint8(model.Update) != 0
		hasDelete := info.Allow&uint8(model.Delete) != 0

		buf.Write(fmt.Sprintf("// %sRules are the domain rules of %s. Every hook is optional (nil = no rule) and runs per record inside the transaction;\n", name, name))
		buf.Write(fmt.Sprintf("// Committed runs after the commit.\ntype %sRules struct {\n", name))
		if hasCreate {
			buf.Write(fmt.Sprintf("\tCreate    func(c crud.Caller, tx crud.Tx, p *%s) error\n", name))
		}
		if hasUpdate {
			buf.Write(fmt.Sprintf("\tUpdate    func(c crud.Caller, tx crud.Tx, old, next *%s, fields []string) error\n", name))
		}
		if hasDelete {
			buf.Write(fmt.Sprintf("\tDelete    func(c crud.Caller, tx crud.Tx, p *%s) error\n", name))
		}
		if hasRead {
			buf.Write("\tSearch    func(c crud.Caller, text string) ([]storage.Condition, error)\n")
			buf.Write("\tNewFilter func() model.Model\n")
		}
		buf.Write(fmt.Sprintf("\tCommitted func(c crud.Caller, action model.Action, p *%s, fields []string)\n", name))
		buf.Write("\tMFA       model.Action\n")
		buf.Write("}\n\n")

		buf.Write(fmt.Sprintf("// Mount%s offers what %sModel.Allow lists, persisted by store, with rules.\n", name, name))
		buf.Write(fmt.Sprintf("func Mount%s(reg crud.Registry, store crud.Store, rules %sRules) {\n", name, name))

		var tableFields []string
		tableFields = append(tableFields, fmt.Sprintf("Model: &%s{}", name))
		tableFields = append(tableFields, fmt.Sprintf("NewList: func() model.ModelSlice { return &%sList{} }", name))
		tableFields = append(tableFields, "Store: store")
		if hasRead {
			tableFields = append(tableFields, "Search: rules.Search")
			tableFields = append(tableFields, "NewFilter: rules.NewFilter")
		}
		tableFields = append(tableFields, "MFA: rules.MFA")

		buf.Write(fmt.Sprintf("\tt := crud.Table{%s}\n", strings.Join(tableFields, ", ")))

		if hasCreate {
			buf.Write(fmt.Sprintf("\tif rules.Create != nil {\n\t\tt.Create = func(c crud.Caller, tx crud.Tx, rec model.Model) error { return rules.Create(c, tx, rec.(*%s)) }\n\t}\n", name))
		}
		if hasUpdate {
			buf.Write(fmt.Sprintf("\tif rules.Update != nil {\n\t\tt.Update = func(c crud.Caller, tx crud.Tx, old, next model.Model, fields []string) error {\n\t\t\treturn rules.Update(c, tx, old.(*%s), next.(*%s), fields)\n\t\t}\n\t}\n", name, name))
		}
		if hasDelete {
			buf.Write(fmt.Sprintf("\tif rules.Delete != nil {\n\t\tt.Delete = func(c crud.Caller, tx crud.Tx, rec model.Model) error { return rules.Delete(c, tx, rec.(*%s)) }\n\t}\n", name))
		}
		buf.Write(fmt.Sprintf("\tif rules.Committed != nil {\n\t\tt.Committed = func(c crud.Caller, a model.Action, rec model.Model, fields []string) { rules.Committed(c, a, rec.(*%s), fields) }\n\t}\n", name))
		buf.Write("\treg.Table(t)\n}\n\n")
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	return os.WriteFile(outName, formatted, 0644)
}
