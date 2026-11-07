package ormc

import (
	"bufio"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"webtyp.com/fmt"
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
			if !info.NoDB && !info.Wire && !info.IsForm {
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

	buf := fmt.Convert()
	buf.Write(GeneratedHeader + "\n\n")
	buf.Write(fmt.Sprintf("package %s\n\n", tableEntities[0].PackageName))
	buf.Write("import (\n")
	buf.Write(fmt.Sprintf("\t%q\n", crudImport))
	buf.Write("\t\"webtyp.com/model\"\n")
	buf.Write(")\n\n")

	for _, info := range tableEntities {
		name := info.Name

		// Mount<Entity>Create
		buf.Write(fmt.Sprintf("// Mount%sCreate mounts the create operation of %s with a typed batch.\n", name, name))
		buf.Write(fmt.Sprintf("func Mount%sCreate(reg crud.Registry, create func(c crud.Caller, list %sList) error) crud.Route {\n", name, name))
		buf.Write(fmt.Sprintf("\treturn reg.Create(func() model.ModelSlice { return &%sList{} }, func(c crud.Caller, l model.ModelSlice) error {\n", name))
		buf.Write(fmt.Sprintf("\t\treturn create(c, *l.(*%sList))\n", name))
		buf.Write("\t})\n}\n\n")

		// Mount<Entity>Read
		buf.Write(fmt.Sprintf("// Mount%sRead mounts the read operation of %s; newFilter may be nil (no filter).\n", name, name))
		buf.Write(fmt.Sprintf("func Mount%sRead(reg crud.Registry, newFilter func() model.Model, read func(c crud.Caller, args crud.ReadArgs, filter model.Model) (%sList, error)) crud.Route {\n", name, name))
		buf.Write(fmt.Sprintf("\treturn reg.Read(&%s{}, newFilter, func(c crud.Caller, args crud.ReadArgs, filter model.Model) (model.ModelSlice, error) {\n", name))
		buf.Write(fmt.Sprintf("\t\tlist, err := read(c, args, filter)\n"))
		buf.Write("\t\treturn &list, err\n")
		buf.Write("\t})\n}\n\n")

		// Mount<Entity>Update
		buf.Write(fmt.Sprintf("// Mount%sUpdate mounts the update operation of %s with a typed carrier record.\n", name, name))
		buf.Write(fmt.Sprintf("func Mount%sUpdate(reg crud.Registry, update func(c crud.Caller, ids, fields []string, rec %s) error) crud.Route {\n", name, name))
		buf.Write(fmt.Sprintf("\treturn reg.Update(func() model.Model { return &%s{} }, func(c crud.Caller, ids, fields []string, rec model.Model) error {\n", name))
		buf.Write(fmt.Sprintf("\t\treturn update(c, ids, fields, *rec.(*%s))\n", name))
		buf.Write("\t})\n}\n\n")

		// Mount<Entity>Delete
		buf.Write(fmt.Sprintf("// Mount%sDelete mounts the delete operation of %s.\n", name, name))
		buf.Write(fmt.Sprintf("func Mount%sDelete(reg crud.Registry, remove func(c crud.Caller, ids []string) error) crud.Route {\n", name, name))
		buf.Write(fmt.Sprintf("\treturn reg.Delete(&%s{}, remove)\n", name))
		buf.Write("}\n\n")
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	return os.WriteFile(outName, formatted, 0644)
}
