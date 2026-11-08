package fixturecrud

import (
	"webtyp.com/input"
	"webtyp.com/model"
)

// PatientModel is a real entity: a table AND a form (input.* fields with DB). It gets CRUD helpers.
var PatientModel = model.Definition{
	Name: "patient",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, DB: &model.FieldDB{}},
	},
}

// CodeEntryModel is a form with no table (no DB field). It gets no CRUD helpers.
var CodeEntryModel = model.Definition{
	Name: "code_entry",
	Fields: model.Fields{
		{Name: "code", Type: input.Text(), Label: "Code"},
	},
}
