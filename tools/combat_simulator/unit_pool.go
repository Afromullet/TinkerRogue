package main

import (
	"game_main/tactical/squads"
)

// UnitPool provides deterministic indexed access to unit templates.
// All methods return copies (value types), not pointers.
type UnitPool struct {
	All      []squads.UnitTemplate
	ByName   map[string]squads.UnitTemplate
	ByRole   map[squads.UnitRole][]squads.UnitTemplate
	ByAttack map[squads.AttackType][]squads.UnitTemplate
}

// NewUnitPool builds a UnitPool from the global squads.Units slice.
func NewUnitPool() *UnitPool {
	pool := &UnitPool{
		All:      make([]squads.UnitTemplate, len(squads.Units)),
		ByName:   make(map[string]squads.UnitTemplate, len(squads.Units)),
		ByRole:   make(map[squads.UnitRole][]squads.UnitTemplate),
		ByAttack: make(map[squads.AttackType][]squads.UnitTemplate),
	}

	copy(pool.All, squads.Units)

	for _, u := range pool.All {
		pool.ByName[u.Name] = u
		pool.ByRole[u.Role] = append(pool.ByRole[u.Role], u)
		pool.ByAttack[u.AttackType] = append(pool.ByAttack[u.AttackType], u)
	}

	return pool
}

// Get returns a copy of the template with the given name. Panics if not found.
func (p *UnitPool) Get(name string) squads.UnitTemplate {
	t, ok := p.ByName[name]
	if !ok {
		panic("UnitPool: unknown unit name: " + name)
	}
	return t
}

// FilterByRole returns all templates with the given role.
func (p *UnitPool) FilterByRole(role squads.UnitRole) []squads.UnitTemplate {
	return append([]squads.UnitTemplate(nil), p.ByRole[role]...)
}

// FilterByAttackType returns all templates with the given attack type.
func (p *UnitPool) FilterByAttackType(at squads.AttackType) []squads.UnitTemplate {
	return append([]squads.UnitTemplate(nil), p.ByAttack[at]...)
}

// FilterByMinRange returns all templates with AttackRange >= r.
func (p *UnitPool) FilterByMinRange(r int) []squads.UnitTemplate {
	var out []squads.UnitTemplate
	for _, u := range p.All {
		if u.AttackRange >= r {
			out = append(out, u)
		}
	}
	return out
}
