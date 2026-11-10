package definitionc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"webtyp.com/definitioncheck"
	"webtyp.com/fmt"
	"webtyp.com/model"
)

func (g *Generator) parseDefinitionsInFile(path string) ([]StructInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var infos []StructInfo
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, name := range valueSpec.Names {
				if !strings.HasSuffix(name.Name, "Model") {
					// Plan §4.2: Falla ruidoso if it doesn't end in Model but looks like a Definition?
					// For now, only process if it ends in Model as per §4.2.
					continue
				}

				if i >= len(valueSpec.Values) {
					continue
				}

				info, err := g.parseDefinition(name.Name, valueSpec.Values[i], node)
				if err != nil {
					return nil, err // Falla ruidoso
				}
				if info != nil {
					info.SourceFile = path
					info.PackageName = node.Name.Name
					infos = append(infos, *info)
				}
			}
		}
	}
	return infos, nil
}

func (g *Generator) parseDefinition(varName string, expr ast.Expr, file *ast.File) (*StructInfo, error) {
	cl, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, nil
	}

	// Check if type is model.Definition
	typeStr := exprToString(cl.Type)
	if !strings.Contains(typeStr, "Definition") {
		return nil, nil
	}

	structName := strings.TrimSuffix(varName, "Model")
	if structName == "" {
		return nil, fmt.Errf("Variable name %s must have a prefix before 'Model'", varName)
	}

	info := &StructInfo{
		Name:              structName,
		ModelNameDeclared: true,
	}

	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key := exprToString(kv.Key)
		switch key {
		case "Name":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				val, _ := strconv.Unquote(lit.Value)
				info.ModelName = val
			}
		case "Wire":
			if id, ok := kv.Value.(*ast.Ident); ok && id.Name == "true" {
				info.Wire = true
			}
		case "Allow":
			allow, err := parseAllow(varName, "Allow", kv.Value)
			if err != nil {
				return nil, err
			}
			info.Allow = allow
		case "MFA":
			mfa, err := parseAllow(varName, "MFA", kv.Value)
			if err != nil {
				return nil, err
			}
			info.MFA = mfa
		case "Rules":
			rules, err := parseRules(varName, kv.Value)
			if err != nil {
				return nil, err
			}
			info.Rules = rules
		case "Fields":
			fields, err := g.parseFields(kv.Value)
			if err != nil {
				return nil, err
			}
			info.Fields = fields
		}
	}

	if err := validateItem(varName, info.Fields); err != nil {
		return nil, err
	}

	if info.ModelName == "" {
		info.ModelName = ToSnakeCase(info.Name)
	}

	// A struct with no DB field is not a table.
	hasDB := false
	for _, f := range info.Fields {
		if f.HasDB {
			hasDB = true
		}
	}
	info.NoDB = !hasDB

	if info.NoDB && info.Allow != 0 {
		if info.Allow != uint8(model.Read) {
			return nil, fmt.Errf("%s: only model.Read can be allowed on a model without a key field; Create/Update/Delete need a primary key declared with DB (a table, or a resource served by the module's own crud.Store)", varName)
		}
	}
	def := model.Definition{Name: info.ModelName, Wire: info.Wire, Allow: model.Action(info.Allow), Rules: info.Rules.modelRules(), MFA: model.Action(info.MFA)}
	if err := definitioncheck.CheckRules(def); err != nil {
		return nil, fmt.Errf("%s: %s", varName, err.Error())
	}
	if info.NoDB && !info.Wire && info.Allow == uint8(model.Read) && info.Rules.Read != uint8(model.AddMyBusinessRule) {
		return nil, fmt.Errf("%s: a model without a table has no default read; declare Rules: model.Rules{Read: model.AddMyBusinessRule} and write its rule", varName)
	}
	if (!info.NoDB || info.Allow == uint8(model.Read)) && !info.Wire {
		for _, f := range info.Fields {
			if strings.EqualFold(f.Name, "allow") || strings.EqualFold(f.ColumnName, "allow") {
				return nil, fmt.Errf("%s: a field named allow collides with the generated Allow() method; rename the column", varName)
			}
		}
	}

	return info, nil
}

// parseAllow reads a set of actions (model.Create | model.Read | …) written in field of varName.
func parseAllow(varName, field string, expr ast.Expr) (uint8, error) {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		switch t.Sel.Name {
		case "Create":
			return uint8(model.Create), nil
		case "Read":
			return uint8(model.Read), nil
		case "Update":
			return uint8(model.Update), nil
		case "Delete":
			return uint8(model.Delete), nil
		}
		return 0, fmt.Errf("field %s of %s: write it as model.Create | model.Read | … (constants only)", field, varName)
	case *ast.BinaryExpr:
		if t.Op != token.OR {
			return 0, fmt.Errf("field %s of %s: write it as model.Create | model.Read | … (constants only)", field, varName)
		}
		left, err := parseAllow(varName, field, t.X)
		if err != nil {
			return 0, err
		}
		right, err := parseAllow(varName, field, t.Y)
		if err != nil {
			return 0, err
		}
		return left | right, nil
	default:
		return 0, fmt.Errf("field %s of %s: write it as model.Create | model.Read | … (constants only)", field, varName)
	}
}

const errRulesShape = "field Rules of %s: write it as model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, …} (constants only)"

// parseRules reads the Rules literal of varName: one RuleMode per action, written as a constant.
func parseRules(varName string, expr ast.Expr) (RulesInfo, error) {
	cl, ok := expr.(*ast.CompositeLit)
	if !ok {
		return RulesInfo{}, fmt.Errf(errRulesShape, varName)
	}
	var r RulesInfo
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return RulesInfo{}, fmt.Errf(errRulesShape, varName)
		}
		mode, err := parseRuleMode(varName, kv.Value)
		if err != nil {
			return RulesInfo{}, err
		}
		switch key := exprToString(kv.Key); key {
		case "Create":
			r.Create = mode
		case "Read":
			r.Read = mode
		case "Update":
			r.Update = mode
		case "Delete":
			r.Delete = mode
		default:
			return RulesInfo{}, fmt.Errf("field Rules of %s: %s is not an action; use Create, Read, Update or Delete", varName, key)
		}
	}
	return r, nil
}

func parseRuleMode(varName string, expr ast.Expr) (uint8, error) {
	switch exprToString(expr) {
	case "model.AddMyBusinessRule":
		return uint8(model.AddMyBusinessRule), nil
	case "model.HaveNoBusinessRule":
		return uint8(model.HaveNoBusinessRule), nil
	}
	return 0, fmt.Errf("field Rules of %s: write model.AddMyBusinessRule or model.HaveNoBusinessRule (constants only), got %s", varName, exprToString(expr))
}

func (g *Generator) parseFields(expr ast.Expr) ([]FieldInfo, error) {
	cl, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, nil
	}

	var fields []FieldInfo
	for _, elt := range cl.Elts {
		fieldCl, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}

		fi, err := g.parseField(fieldCl)
		if err != nil {
			return nil, err
		}
		fields = append(fields, fi)
	}
	return fields, nil
}

func (g *Generator) parseField(cl *ast.CompositeLit) (FieldInfo, error) {
	var fi FieldInfo
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key := exprToString(kv.Key)
		switch key {
		case "Name":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				fi.ColumnName, _ = strconv.Unquote(lit.Value)
				fi.Name = fmt.Convert(fi.ColumnName).CamelUp().String()
			}
		case "Type":
			fi.KindConstructor = exprToString(kv.Value)
			if call, ok := kv.Value.(*ast.CallExpr); ok {
				for _, arg := range call.Args {
					if ident, ok := arg.(*ast.Ident); ok {
						fi.KindArgIdents = append(fi.KindArgIdents, ident.Name)
					}
				}
			}
		case "NotNull":
			fi.NotNull = exprToBool(kv.Value)
		case "OmitEmpty":
			fi.OmitEmpty = exprToBool(kv.Value)
		case "Exclude":
			fi.Exclude = exprToBool(kv.Value)
		case "Widget":
			return FieldInfo{}, fmt.Err(fmt.Sprintf("field %s: Field.Widget was removed (Kind unification): declare the kind in Type — e.g. Type: input.Email()", fi.ColumnName))
		case "DB":
			g.parseDBField(kv.Value, &fi)
		case "Item":
			slot, ok := itemSlotOf(exprToString(kv.Value))
			if !ok {
				return FieldInfo{}, fmt.Err(fmt.Sprintf(errUnknownItemSlot, fi.ColumnName, exprToString(kv.Value)))
			}
			fi.Item = slot
		case "Row":
			return FieldInfo{}, fmt.Err(fmt.Sprintf("field %s: Field.Row was replaced by Field.Item (model.ItemLabel, model.ItemDescription, model.ItemBadge)", fi.ColumnName))
		case "Options":
			fi.Options = parseRef(kv.Value)
		case "Ref":
			fi.Ref = parseRef(kv.Value)
		case "Permitted":
			g.parsePermitted(kv.Value, &fi)
		}
	}

	if fi.KindConstructor == "" {
		return FieldInfo{}, fmt.Err(fmt.Sprintf("field %s: kind required", fi.ColumnName))
	}

	// Composition refs come from the constructor argument, not Ref:
	_, ctorName := parseConstructor(fi.KindConstructor)
	if ctorName == "Struct" || ctorName == "StructSlice" {
		// Extract arg from model.Struct(&AddressModel) / model.StructSlice(&AddressModel)
		arg := ""
		if call, ok := kvToExpr(cl, "Type").(*ast.CallExpr); ok && len(call.Args) > 0 {
			arg = exprToString(call.Args[0])
		}
		if arg == "" || arg == "nil" {
			return FieldInfo{}, fmt.Err(fmt.Sprintf("field %s: %s requires a non-nil Definition argument", fi.ColumnName, fi.KindConstructor))
		}
		if fi.Ref != "" {
			return FieldInfo{}, fmt.Err(fmt.Sprintf("field %s: contradiction: Ref: cannot be used with composition kind %s (it already takes the definition as an argument)", fi.ColumnName, fi.KindConstructor))
		}
		fi.Ref = strings.TrimPrefix(arg, "&")
		if ctorName == "StructSlice" {
			fi.Type = model.FieldStructSlice
		} else {
			fi.Type = model.FieldStruct
		}
		fi.GoType = FieldTypeToGoType(fi.Type, fi.Ref)
	}

	// Non-composition fields: fi.Type stays 0 here and is populated by
	// stage-2 resolution (resolveStorage: builtin table or dependency probe).
	// GoType for those fields is computed there once fi.Type is known.

	return fi, nil
}

func (g *Generator) parseDBField(expr ast.Expr, fi *FieldInfo) {
	fi.HasDB = true
	cl, ok := expr.(*ast.CompositeLit)
	if !ok {
		if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
			if cl2, ok := unary.X.(*ast.CompositeLit); ok {
				cl = cl2
			}
		}
	}

	if cl == nil {
		return
	}

	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key := exprToString(kv.Key)
		switch key {
		case "PK":
			fi.PK = exprToBool(kv.Value)
		case "Unique":
			fi.Unique = exprToBool(kv.Value)
		case "Revision":
			fi.Revision = exprToBool(kv.Value)
		case "AutoInc":
			fi.AutoInc = exprToBool(kv.Value)
		case "RefColumn":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				fi.RefColumn, _ = strconv.Unquote(lit.Value)
			}
		case "OnDelete":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				fi.OnDelete, _ = strconv.Unquote(lit.Value)
			}
		}
	}
}

func (g *Generator) parsePermitted(expr ast.Expr, fi *FieldInfo) {
	cl, ok := expr.(*ast.CompositeLit)
	if !ok {
		if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
			if cl2, ok := unary.X.(*ast.CompositeLit); ok {
				cl = cl2
			}
		}
	}
	if cl == nil {
		return
	}
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key := exprToString(kv.Key)
		switch key {
		case "Letters":
			fi.Letters = exprToBool(kv.Value)
		case "Tilde":
			fi.Tilde = exprToBool(kv.Value)
		case "Numbers":
			fi.Numbers = exprToBool(kv.Value)
		case "Spaces":
			fi.Spaces = exprToBool(kv.Value)
		case "Minimum":
			fi.Minimum = exprToInt(kv.Value)
		case "Maximum":
			fi.Maximum = exprToInt(kv.Value)
		}
	}
}

const errUnknownItemSlot = "field %s: unknown Item slot %s (use model.ItemLabel, model.ItemDescription or model.ItemBadge)"

// itemSlotOf maps the selector written in a Definition (model.ItemLabel, or ItemLabel
// with a dot-import) to its slot.
func itemSlotOf(expr string) (model.ItemSlot, bool) {
	name := expr
	if i := strings.LastIndex(expr, "."); i >= 0 {
		name = expr[i+1:]
	}
	switch name {
	case "ItemLabel":
		return model.ItemLabel, true
	case "ItemDescription":
		return model.ItemDescription, true
	case "ItemBadge":
		return model.ItemBadge, true
	case "ItemNone":
		return model.ItemNone, true
	}
	return model.ItemNone, false
}

func parseRef(expr ast.Expr) string {
	s := exprToString(expr)
	s = strings.TrimPrefix(s, "&")
	return s
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.BasicLit:
		return t.Value
	case *ast.UnaryExpr:
		return t.Op.String() + exprToString(t.X)
	case *ast.CallExpr:
		s := exprToString(t.Fun) + "("
		for i, arg := range t.Args {
			if i > 0 {
				s += ", "
			}
			s += exprToString(arg)
		}
		s += ")"
		return s
	case *ast.CompositeLit:
		s := ""
		if t.Type != nil {
			s += exprToString(t.Type)
		}
		s += "{"
		for i, elt := range t.Elts {
			if i > 0 {
				s += ", "
			}
			s += exprToString(elt)
		}
		s += "}"
		return s
	case *ast.KeyValueExpr:
		return exprToString(t.Key) + ": " + exprToString(t.Value)
	}
	return ""
}

func exprToBool(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "true"
	}
	return false
}

func exprToInt(expr ast.Expr) int {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.INT {
		val, _ := strconv.Atoi(lit.Value)
		return val
	}
	return 0
}

func ToSnakeCase(s string) string {
	return fmt.Convert(s).SnakeLow().String()
}

func kvToExpr(cl *ast.CompositeLit, key string) ast.Expr {
	for _, elt := range cl.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if exprToString(kv.Key) == key {
				return kv.Value
			}
		}
	}
	return nil
}

func FieldTypeToGoType(ft model.FieldType, ref string) string {
	switch ft {
	case model.FieldText, model.FieldRaw:
		return "string"
	case model.FieldInt:
		return "int64"
	case model.FieldFloat:
		return "float64"
	case model.FieldBool:
		return "bool"
	case model.FieldBlob:
		return "[]byte"
	case model.FieldIntSlice:
		return "[]int"
	case model.FieldStruct:
		if ref != "" {
			return resolveStructType(ref)
		}
		return "struct{}"
	case model.FieldStructSlice:
		if ref != "" {
			return "[]" + resolveStructType(ref)
		}
		return "[]struct{}"
	}
	return "string"
}

func resolveStructType(ref string) string {
	// ref is like "UserModel" or "pkg.UserModel"
	parts := strings.Split(ref, ".")
	last := parts[len(parts)-1]
	if strings.HasSuffix(last, "Model") {
		parts[len(parts)-1] = strings.TrimSuffix(last, "Model")
	}
	return strings.Join(parts, ".")
}

// validateItem hands model.ValidateItem the facts it reads (name, item slot, PK,
// and whether Ref/Options are set) and names the Definition in the error.
func validateItem(varName string, fields []FieldInfo) error {
	mf := make([]model.Field, len(fields))
	for i, f := range fields {
		var db *model.FieldDB
		if f.PK {
			db = &model.FieldDB{PK: true}
		}
		mf[i] = model.Field{Name: f.ColumnName, Item: f.Item, DB: db}
		if f.Ref != "" && f.Type != model.FieldStruct && f.Type != model.FieldStructSlice {
			mf[i].Ref = &model.Definition{Name: f.Ref}
		}
		if f.Options != "" {
			mf[i].Options = &model.Definition{Name: f.Options}
		}
	}
	if err := model.ValidateItem(mf); err != nil {
		return fmt.Err(varName + ": " + err.Error())
	}
	return nil
}
