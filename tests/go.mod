// Separate test module: isolates codegen fixture deps (orm runtime for
// generated models) so the root webtyp.com/definitionc module stays
// fmt + model + modfind only.
module webtyp.com/definitionc/tests

go 1.26.8

require (
	webtyp.com/definitionc v0.1.44
	webtyp.com/json v0.5.40
	webtyp.com/model v0.2.21
	webtyp.com/orm v0.12.30
	webtyp.com/time v0.6.4
)

require (
	webtyp.com/calendar v0.1.0 // indirect
	webtyp.com/crud v0.0.15 // indirect
	webtyp.com/definitioncheck v0.0.3 // indirect
	webtyp.com/escape v0.1.0 // indirect
	webtyp.com/events v0.1.8 // indirect
	webtyp.com/fmt v1.0.0 // indirect
	webtyp.com/modfind v0.0.11 // indirect
	webtyp.com/msgtype v0.1.0 // indirect
	webtyp.com/router v0.5.0 // indirect
	webtyp.com/storage v0.2.3 // indirect
)

replace webtyp.com/definitionc => ..
