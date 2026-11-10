package fixturecrud

import "webtyp.com/crud"

func (p *Patient) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	return crud.ApplyAndPublish, nil
}
