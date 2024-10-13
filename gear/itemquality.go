package gear

import (
	"fmt"
	"math/rand"
)

type Quality int

var LowQualStr = "Low Quality"
var NormalQualStr = "Normal Quality"
var HighQualStr = "High Quality"

const (
	LowQuality = iota
	NormalQuality
	HighQuality
)

// Any item that implements the interface defines the random ranges of the item stats

type ItemQuality[T any] interface {
	CreateLowQual() T
	CreateNormQual() T
	CreateHighQual() T
	Quality() string
}

func CreateLowQualityItem[T any](iq ItemQuality[T]) T {
	return iq.CreateLowQual()
}

func CreateNormQualityItem[T any](iq ItemQuality[T]) T {
	return iq.CreateNormQual()
}

func CreateHighQualityItem[T any](iq ItemQuality[T]) T {
	return iq.CreateHighQual()
}

func CreateThrowableWithQual[T any](effect StatusEffects, iq Quality) (StatusEffects, error) {
	qualityItem, ok := effect.(ItemQuality[T])
	if !ok {
		return nil, fmt.Errorf("effect does not implement ItemQuality for the given type")
	}

	var item T
	switch iq {
	case LowQuality:
		item = qualityItem.CreateLowQual()
	case NormalQuality:
		item = qualityItem.CreateNormQual()
	case HighQuality:
		item = qualityItem.CreateHighQual()
	default:
		return nil, fmt.Errorf("unknown quality: %v", iq)
	}

	// Type assertion to ensure the created item implements StatusEffects
	statusEffect, ok := any(item).(StatusEffects)
	if !ok {
		return nil, fmt.Errorf("created item does not implement StatusEffects")
	}

	return statusEffect, nil
}

/*
func CreateThrowableWithQual(effect StatusEffects, iq Quality) {

	switch v := effect.(type) {
	case ItemQuality[Sticky]:

		if iq == LowQuality {
			it := v.CreateLowQual()
		} else if iq == NormalQuality {
			it := v.CreateNormQual()
		} else if iq == HighQuality {
			it := v.CreateHighQual()
		}

		// Add more cases for other types that implement both interfaces
	case ItemQuality[Burning]:
		if iq == LowQuality {
			it := v.CreateLowQual()
		} else if iq == NormalQuality {
			it := v.CreateNormQual()
		} else if iq == HighQuality {
			it := v.CreateHighQual()
		}

	case ItemQuality[Throwable]:
		if iq == LowQuality {
			it := v.CreateLowQual()
		} else if iq == NormalQuality {
			it := v.CreateNormQual()
		} else if iq == HighQuality {
			it := v.CreateHighQual()
		}
	default:
		fmt.Println("This effect doesn't implement ItemQuality")
	}

}
*/

func (c CommonItemProperties) CreateLowQual() CommonItemProperties {

	c.Name = ""
	c.Duration = rand.Intn(3) + 1
	c.Quality = LowQuality
	return c

}

func (c CommonItemProperties) CreateMedQual() CommonItemProperties {

	c.Name = ""
	c.Duration = rand.Intn(3) + 1
	c.Quality = NormalQuality
	return c

}

func (c CommonItemProperties) CreateHighQual() CommonItemProperties {

	c.Name = ""
	c.Duration = rand.Intn(6) + 1
	return c

}

func (s Sticky) CreateLowQual() Sticky {

	s.MainProps = s.MainProps.CreateLowQual()
	s.Spread = rand.Intn(2) + 1
	return s

}

func (s Sticky) CreateMedQual() Sticky {
	s.MainProps = s.MainProps.CreateMedQual()
	s.Spread = rand.Intn(4) + 1
	return s

}

func (s Sticky) CreateHighQual() Sticky {
	s.MainProps = s.MainProps.CreateHighQual()
	s.Spread = rand.Intn(6) + 1
	return s

}

func (b Burning) CreateLowQual() Burning {

	b.MainProps = b.MainProps.CreateLowQual()
	b.Temperature = rand.Intn(3) + 1
	return b

}

func (b Burning) CreateMedQual() Burning {
	b.MainProps = b.MainProps.CreateMedQual()
	b.Temperature = rand.Intn(5) + 1
	return b

}

func (b Burning) CreateHighQual() Burning {

	b.MainProps = b.MainProps.CreateHighQual()
	b.Temperature = rand.Intn(7) + 1
	return b

}

func (f Freezing) CreateLowQual() Freezing {

	f.MainProps = f.MainProps.CreateLowQual()
	f.Thickness = rand.Intn(3) + 1
	return f

}

func (f Freezing) CreateMedQual() Freezing {

	f.MainProps = f.MainProps.CreateMedQual()
	f.Thickness = rand.Intn(5) + 1
	return f

}

func (f Freezing) CreateHighQual() Freezing {

	f.MainProps = f.MainProps.CreateHighQual()
	f.Thickness = rand.Intn(7) + 1

	return f

}
