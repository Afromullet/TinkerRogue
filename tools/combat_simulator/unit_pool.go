package main

import (
	"game_main/tactical/unitdefs"
)

// UnitPool provides deterministic indexed access to unit templates.
// All methods return copies (value types), not pointers.
type UnitPool struct {
	All      []unitdefs.UnitTemplate
	ByName   map[string]unitdefs.UnitTemplate
	ByRole   map[unitdefs.UnitRole][]unitdefs.UnitTemplate
	ByAttack map[unitdefs.AttackType][]unitdefs.UnitTemplate
}

// NewUnitPool builds a UnitPool from the global unitdefs.Units slice.
func NewUnitPool() *UnitPool {
	pool := &UnitPool{
		All:      make([]unitdefs.UnitTemplate, len(unitdefs.Units)),
		ByName:   make(map[string]unitdefs.UnitTemplate, len(unitdefs.Units)),
		ByRole:   make(map[unitdefs.UnitRole][]unitdefs.UnitTemplate),
		ByAttack: make(map[unitdefs.AttackType][]unitdefs.UnitTemplate),
	}

	copy(pool.All, unitdefs.Units)

	for _, u := range pool.All {
		pool.ByName[u.UnitType] = u
		pool.ByRole[u.Role] = append(pool.ByRole[u.Role], u)
		pool.ByAttack[u.AttackType] = append(pool.ByAttack[u.AttackType], u)
	}

	return pool
}

// Get returns a copy of the template with the given name. Panics if not found.
func (p *UnitPool) Get(name string) unitdefs.UnitTemplate {
	t, ok := p.ByName[name]
	if !ok {
		panic("UnitPool: unknown unit name: " + name)
	}
	return t
}

// FilterByRole returns all templates with the given role.
func (p *UnitPool) FilterByRole(role unitdefs.UnitRole) []unitdefs.UnitTemplate {
	return append([]unitdefs.UnitTemplate(nil), p.ByRole[role]...)
}

// FilterByAttackType returns all templates with the given attack type.
func (p *UnitPool) FilterByAttackType(at unitdefs.AttackType) []unitdefs.UnitTemplate {
	return append([]unitdefs.UnitTemplate(nil), p.ByAttack[at]...)
}

// FilterByMinRange returns all templates with AttackRange >= r.
func (p *UnitPool) FilterByMinRange(r int) []unitdefs.UnitTemplate {
	var out []unitdefs.UnitTemplate
	for _, u := range p.All {
		if u.AttackRange >= r {
			out = append(out, u)
		}
	}
	return out
}
