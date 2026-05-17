package perks

import "strings"

// FormatPerkDetail renders a perk's body fields (Tier, Category, Roles,
// Description, ExclusiveWith). When includeName is true the perk's Name is
// prepended on its own line so callers that already show the name elsewhere
// (e.g. a list selection header) can skip it.
//
// ExclusiveWith IDs are resolved through GetPerkDefinition so the rendered
// names stay accurate when the registry changes; unknown IDs fall back to
// their string form.
func FormatPerkDetail(def *PerkDefinition, includeName bool) string {
	var b strings.Builder
	if includeName {
		b.WriteString(def.Name)
		b.WriteString("\n\n")
	}
	b.WriteString("Tier: ")
	b.WriteString(def.Tier.String())
	b.WriteString("\nCategory: ")
	b.WriteString(def.Category.String())
	if len(def.Roles) > 0 {
		b.WriteString("\nRoles: ")
		b.WriteString(strings.Join(def.Roles, ", "))
	}
	b.WriteString("\n\n")
	b.WriteString(def.Description)
	if len(def.ExclusiveWith) > 0 {
		b.WriteString("\n\nExclusive with: ")
		names := make([]string, 0, len(def.ExclusiveWith))
		for _, exID := range def.ExclusiveWith {
			if exDef := GetPerkDefinition(exID); exDef != nil {
				names = append(names, exDef.Name)
			} else {
				names = append(names, string(exID))
			}
		}
		b.WriteString(strings.Join(names, ", "))
	}
	return b.String()
}
