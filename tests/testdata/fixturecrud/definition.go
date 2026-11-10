package fixturecrud

import (
	"webtyp.com/input"
	"webtyp.com/model"
)

// Definition is the fixture module: it lists the one entity that crud mounts.
var Definition = model.ModuleDefinition{Name: "fixturecrud", Entities: []model.Definition{PatientModel}}

// PatientModel is a real entity: a table AND a form (input.* fields with DB). Its operations are mounted by the module.
var PatientModel = model.Definition{
	Name:  "patient",
	Allow: model.Create | model.Read | model.Update,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Update: model.HaveNoBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, DB: &model.FieldDB{}},
	},
}

// NoteModel is a table with no Allow. It gets no CRUD helpers.
var NoteModel = model.Definition{
	Name: "note",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "text", Type: model.Text(), DB: &model.FieldDB{}},
	},
}

// CodeEntryModel is a form with no table (no DB field). It gets no CRUD helpers.
var CodeEntryModel = model.Definition{
	Name: "code_entry",
	Fields: model.Fields{
		{Name: "code", Type: input.Text(), Label: "Code"},
	},
}
