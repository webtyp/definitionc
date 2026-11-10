package fixturecrud

import "webtyp.com/crud"

// Module is the fixture module: the generated methods hang on it.
type Module struct{ store crud.Store }

// StorePatient is the one line the generated MountOperations reads for the patient table.
func (m *Module) StorePatient() crud.Store { return m.store }
