package gear

import (
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

/*
func CreateStatEffWithQuality(eff StatusEffects, qual Quality) StatusEffects {

	if qual == LowQuality {

		return eff.CreateLowQual()

	} else if qual == NormalQuality {
		return eff.CreateNormQual()

	} else if qual == HighQuality {
		return eff.CreateHighQual()

	}

	return nil
}
*/

func (c CommonItemProperties) CreateWithQuality(q Quality) CommonItemProperties {

	props := CommonItemProperties{}
	if q == LowQuality {
		c.Name = ""
		c.Duration = rand.Intn(3) + 1
		c.Quality = LowQuality

	} else if q == NormalQuality {
		c.Name = ""
		c.Duration = rand.Intn(3) + 1
		c.Quality = NormalQuality

	} else if q == HighQuality {
		c.Name = ""
		c.Duration = rand.Intn(6) + 1
		c.Quality = HighQuality

	}

	return props

}

//Todo do these need pointer receivers, since we're returning something?

func (s *Sticky) CreateWithQuality(q Quality) {

	s.MainProps = s.MainProps.CreateWithQuality(q)
	if q == LowQuality {

		s.Spread = rand.Intn(2) + 1

	} else if q == NormalQuality {

		s.Spread = rand.Intn(4) + 1

	} else if q == HighQuality {

		s.Spread = rand.Intn(6) + 1

	}

}

func (b *Burning) CreateWithQuality(q Quality) {

	b.MainProps = b.MainProps.CreateWithQuality(q)
	if q == LowQuality {

		b.Temperature = rand.Intn(3) + 1
	} else if q == NormalQuality {

		b.Temperature = rand.Intn(5) + 1
	} else if q == HighQuality {

		b.Temperature = rand.Intn(7) + 1
	}

}

func (f *Freezing) CreateWithQuality(q Quality) {

	f.MainProps = f.MainProps.CreateWithQuality(q)
	if q == LowQuality {

		f.Thickness = rand.Intn(3) + 1
	} else if q == NormalQuality {

		f.Thickness = rand.Intn(5) + 1

	} else if q == HighQuality {

		f.Thickness = rand.Intn(7) + 1

	}

}

func (t *Throwable) CreateWithQuality(q Quality) {

	t.MainProps = t.MainProps.CreateWithQuality(q)
	if q == LowQuality {

		t.ThrowingRange = rand.Intn(2) + 1
	} else if q == NormalQuality {

		t.ThrowingRange = rand.Intn(5) + 1

	} else if q == HighQuality {

		t.ThrowingRange = rand.Intn(7) + 1

	}

}
