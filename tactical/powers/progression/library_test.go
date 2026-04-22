package progression

import (
	"errors"
	"game_main/core/common"
	"game_main/tactical/powers/perks"
	"game_main/templates"
	testfx "game_main/testing"
	"testing"
)

func newTestManagerWithPerkData(t *testing.T) *common.EntityManager {
	t.Helper()
	manager := testfx.NewTestEntityManager()
	// Seed a perk definition for cost-check tests without requiring real JSON load.
	perks.PerkRegistry[perks.PerkBraceForImpact] = &perks.PerkDefinition{
		ID: perks.PerkBraceForImpact, Name: "Brace", UnlockCost: 2,
	}
	perks.PerkRegistry[perks.PerkOpeningSalvo] = &perks.PerkDefinition{
		ID: perks.PerkOpeningSalvo, Name: "Opening Salvo", UnlockCost: 3,
	}
	// Seed a spell definition.
	templates.SpellRegistry["spark"] = &templates.SpellDefinition{
		ID: "spark", Name: "Spark", ManaCost: 5, UnlockCost: 1,
	}
	templates.SpellRegistry["obliterate"] = &templates.SpellDefinition{
		ID: "obliterate", Name: "Obliterate", ManaCost: 30, UnlockCost: 8,
	}
	return manager
}

func TestUnlockPerkDeductsAndIsIdempotent(t *testing.T) {
	manager := newTestManagerWithPerkData(t)
	entity := manager.World.NewEntity()
	entity.AddComponent(common.PlayerComponent, &common.Player{})
	data := &ProgressionData{}
	data.SkillPoints = 5
	entity.AddComponent(ProgressionComponent, data)
	pid := entity.GetID()

	if err := UnlockPerk(pid, perks.PerkOpeningSalvo, manager); err != nil {
		t.Fatalf("unexpected unlock error: %v", err)
	}
	if data.SkillPoints != 2 {
		t.Errorf("expected skill points 2 after unlock cost 3, got %d", data.SkillPoints)
	}
	if !IsPerkUnlocked(pid, perks.PerkOpeningSalvo, manager) {
		t.Error("expected opening_salvo unlocked after UnlockPerk")
	}

	// Second call is a no-op (idempotent), no extra deduction.
	if err := UnlockPerk(pid, perks.PerkOpeningSalvo, manager); err != nil {
		t.Fatalf("second unlock should be no-op, got err: %v", err)
	}
	if data.SkillPoints != 2 {
		t.Errorf("expected skill points still 2 after idempotent unlock, got %d", data.SkillPoints)
	}
}

func TestUnlockPerkInsufficientPoints(t *testing.T) {
	manager := newTestManagerWithPerkData(t)
	entity := manager.World.NewEntity()
	entity.AddComponent(common.PlayerComponent, &common.Player{})
	data := &ProgressionData{}
	data.SkillPoints = 1
	entity.AddComponent(ProgressionComponent, data)
	pid := entity.GetID()

	err := UnlockPerk(pid, perks.PerkOpeningSalvo, manager)
	if !errors.Is(err, ErrNotEnoughPoints) {
		t.Errorf("expected ErrNotEnoughPoints, got %v", err)
	}
	if data.SkillPoints != 1 {
		t.Errorf("expected skill points unchanged, got %d", data.SkillPoints)
	}
	if IsPerkUnlocked(pid, perks.PerkOpeningSalvo, manager) {
		t.Error("expected perk to remain locked after failed unlock")
	}
}

func TestUnlockSpellDeductsAndIsIdempotent(t *testing.T) {
	manager := newTestManagerWithPerkData(t)
	entity := manager.World.NewEntity()
	entity.AddComponent(common.PlayerComponent, &common.Player{})
	data := &ProgressionData{}
	data.ArcanaPoints = 10
	entity.AddComponent(ProgressionComponent, data)
	pid := entity.GetID()

	if err := UnlockSpell(pid, "obliterate", manager); err != nil {
		t.Fatalf("unexpected unlock error: %v", err)
	}
	if data.ArcanaPoints != 2 {
		t.Errorf("expected arcana 2 after unlock cost 8, got %d", data.ArcanaPoints)
	}
	if !IsSpellUnlocked(pid, "obliterate", manager) {
		t.Error("expected obliterate unlocked")
	}

	if err := UnlockSpell(pid, "obliterate", manager); err != nil {
		t.Fatalf("second unlock should be no-op, got err: %v", err)
	}
	if data.ArcanaPoints != 2 {
		t.Errorf("expected arcana still 2 after idempotent unlock, got %d", data.ArcanaPoints)
	}
}

func TestAddPoints(t *testing.T) {
	manager := newTestManagerWithPerkData(t)
	entity := manager.World.NewEntity()
	entity.AddComponent(common.PlayerComponent, &common.Player{})
	data := &ProgressionData{}
	entity.AddComponent(ProgressionComponent, data)
	pid := entity.GetID()

	AddArcanaPoints(pid, 3, manager)
	AddSkillPoints(pid, 7, manager)
	if data.ArcanaPoints != 3 {
		t.Errorf("expected arcana 3, got %d", data.ArcanaPoints)
	}
	if data.SkillPoints != 7 {
		t.Errorf("expected skill 7, got %d", data.SkillPoints)
	}

	// Negative or zero amounts are ignored.
	AddArcanaPoints(pid, 0, manager)
	AddSkillPoints(pid, -5, manager)
	if data.ArcanaPoints != 3 || data.SkillPoints != 7 {
		t.Errorf("expected unchanged after non-positive adds, got arcana=%d skill=%d", data.ArcanaPoints, data.SkillPoints)
	}
}

