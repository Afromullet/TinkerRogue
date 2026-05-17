package templates

import (
	"fmt"
	"strings"
)

// FormatSpellDetail renders a spell's body fields (Mana, Damage, Target,
// Effect, Description). When includeName is true the spell's Name is
// prepended on its own line so callers that already show the name elsewhere
// (e.g. a list selection header) can skip it.
func FormatSpellDetail(def *SpellDefinition, includeName bool) string {
	var b strings.Builder
	if includeName {
		b.WriteString(def.Name)
		b.WriteString("\n\n")
	}
	b.WriteString(fmt.Sprintf("Mana: %d", def.ManaCost))
	if def.Damage > 0 {
		b.WriteString(fmt.Sprintf("\nDamage: %d", def.Damage))
	}
	b.WriteString("\nTarget: ")
	b.WriteString(string(def.TargetType))
	b.WriteString("\nEffect: ")
	b.WriteString(string(def.EffectType))
	b.WriteString("\n\n")
	b.WriteString(def.Description)
	return b.String()
}
