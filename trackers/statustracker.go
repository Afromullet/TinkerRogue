package trackers

import (
	"fmt"
	"game_main/gear"
)

type StatusEffectTracker struct {
	ActiveEffects map[string]gear.StatusEffects
}

func (s *StatusEffectTracker) Add(e gear.StatusEffects) {

	// Really don't like the way this is done.
	// Since Throwable is an effect, we have to check to make sure it isn't added
	if gear.THROWABLE_NAME == e.StatusEffectName() {
		return
	}

	if s.ActiveEffects == nil {

		s.ActiveEffects = make(map[string]gear.StatusEffects)

	}

	if _, exists := s.ActiveEffects[e.StatusEffectName()]; exists {
		s.ActiveEffects[e.StatusEffectName()].StackEffect(e)

	} else {
		s.ActiveEffects[e.StatusEffectName()] = e
	}

}

// Used for displaying the active status effects to the player
func (s *StatusEffectTracker) ActiveEffectNames() string {

	result := ""

	//The key is the name
	for k, eff := range s.ActiveEffects {

		if eff.Duration() > 0 {
			result += fmt.Sprintln(eff.DisplayString())
		} else {
			result += fmt.Sprintln(k, " Is done")

		}
	}

	return result
}
