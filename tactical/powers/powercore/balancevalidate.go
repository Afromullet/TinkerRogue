package powercore

import (
	"fmt"
	"reflect"
	"strings"
)

// Balance range kinds, declared via struct tags on balance config fields:
//
//	CoverBonus float64 `json:"coverBonus" balance:"fraction"`
//
// Range bounds catch typos in JSON that bare "must be positive" checks would
// let through (e.g. DamageMult: 0.001 silently nerfing a perk, or
// MaxStationaryTurns: 1000 never capping). Fractions are bounded < 1.0,
// damage multipliers to [0.1, 10.0], and counts/bonuses to ≤ 100.
const (
	minDamageMult = 0.1
	maxDamageMult = 10.0
	maxCount      = 100
)

// balanceRange defines the inclusive/exclusive bounds for one balance kind.
type balanceRange struct {
	min, max         float64
	minExcl, maxExcl bool
}

func (r balanceRange) contains(v float64) bool {
	if r.minExcl && v <= r.min || !r.minExcl && v < r.min {
		return false
	}
	if r.maxExcl && v >= r.max || !r.maxExcl && v > r.max {
		return false
	}
	return true
}

func (r balanceRange) String() string {
	open, close := "[", "]"
	if r.minExcl {
		open = "("
	}
	if r.maxExcl {
		close = ")"
	}
	return fmt.Sprintf("%s%v, %v%s", open, r.min, r.max, close)
}

var balanceKinds = map[string]balanceRange{
	"fraction": {min: 0, max: 1, minExcl: true, maxExcl: true},
	"mult":     {min: minDamageMult, max: maxDamageMult},
	"count":    {min: 0, max: maxCount, minExcl: true},
	"bonus":    {min: 0, max: maxCount},
}

// ValidateBalanceRanges range-checks every numeric field of a balance config
// (a struct of per-power structs, e.g. perks.PerkBalanceConfig). Each numeric
// field must declare its range kind via a `balance` tag — a missing or unknown
// tag is itself an error, so new tuning fields cannot silently skip
// validation. Error paths use the json tags ("fortify.perTurnCoverBonus").
// Accepts a pointer or value; never panics on valid struct input.
func ValidateBalanceRanges(cfg any) []error {
	v := reflect.ValueOf(cfg)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return []error{fmt.Errorf("balance config is nil")}
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return []error{fmt.Errorf("balance config must be a struct, got %s", v.Kind())}
	}

	var errs []error
	for i := 0; i < v.NumField(); i++ {
		group := v.Type().Field(i)
		groupVal := v.Field(i)
		if groupVal.Kind() != reflect.Struct {
			errs = append(errs, fmt.Errorf("%s: balance config fields must be structs, got %s", group.Name, groupVal.Kind()))
			continue
		}
		for j := 0; j < groupVal.NumField(); j++ {
			field := group.Type.Field(j)
			path := jsonName(group) + "." + jsonName(field)

			var val float64
			switch groupVal.Field(j).Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val = float64(groupVal.Field(j).Int())
			case reflect.Float32, reflect.Float64:
				val = groupVal.Field(j).Float()
			default:
				continue // non-numeric fields carry no range
			}

			kind, ok := field.Tag.Lookup("balance")
			if !ok {
				errs = append(errs, fmt.Errorf("%s: numeric balance field missing balance tag", path))
				continue
			}
			rng, ok := balanceKinds[kind]
			if !ok {
				errs = append(errs, fmt.Errorf("%s: unknown balance kind %q", path, kind))
				continue
			}
			if !rng.contains(val) {
				errs = append(errs, fmt.Errorf("%s must be in %s, got %v", path, rng, val))
			}
		}
	}
	return errs
}

// jsonName returns the field's json tag name, falling back to the Go name.
func jsonName(f reflect.StructField) string {
	tag, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if tag != "" && tag != "-" {
		return tag
	}
	return f.Name
}
