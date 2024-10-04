package trackers

import "game_main/gear"

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
