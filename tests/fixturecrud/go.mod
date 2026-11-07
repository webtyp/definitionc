module webtyp.com/ormc/tests/fixturecrud

go 1.26.8

require (
	webtyp.com/model v0.2.13
	webtyp.com/orm v0.12.26
)

require (
	webtyp.com/fmt v1.0.0 // indirect
	webtyp.com/storage v0.1.18 // indirect
)

replace webtyp.com/crud => ../../../crud

replace webtyp.com/model => ../../../model

replace webtyp.com/router => ../../../router

replace webtyp.com/rpc => ../../../rpc
