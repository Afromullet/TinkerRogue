package gear

import (
	"game_main/common"
	"math/rand"
)

func (c CommonItemProperties) CreateWithQuality(q common.QualityType) CommonItemProperties {

	props := CommonItemProperties{}
	if q == common.LowQuality {
		props.Name = ""
		props.Duration = rand.Intn(3) + 1
		props.Quality = common.LowQuality

	} else if q == common.NormalQuality {
		props.Name = ""
		props.Duration = rand.Intn(3) + 1
		props.Quality = common.NormalQuality

	} else if q == common.HighQuality {
		props.Name = ""
		props.Duration = rand.Intn(6) + 1
		props.Quality = common.HighQuality

	}

	return props

}

func (s *Sticky) CreateWithQuality(q common.QualityType) {

	s.MainProps = s.MainProps.CreateWithQuality(q)
	s.MainProps.Name = STICKY_NAME
	if q == common.LowQuality {

		s.Spread = rand.Intn(2) + 1

	} else if q == common.NormalQuality {

		s.Spread = rand.Intn(4) + 1

	} else if q == common.HighQuality {

		s.Spread = rand.Intn(6) + 1

	}

}

func (b *Burning) CreateWithQuality(q common.QualityType) {

	b.MainProps = b.MainProps.CreateWithQuality(q)
	b.MainProps.Name = BURNING_NAME
	if q == common.LowQuality {

		b.Temperature = rand.Intn(3) + 1
	} else if q == common.NormalQuality {

		b.Temperature = rand.Intn(5) + 1
	} else if q == common.HighQuality {

		b.Temperature = rand.Intn(7) + 1
	}

}

func (f *Freezing) CreateWithQuality(q common.QualityType) {

	f.MainProps = f.MainProps.CreateWithQuality(q)
	f.MainProps.Name = FREEZING_NAME
	if q == common.LowQuality {

		f.Thickness = rand.Intn(3) + 1
	} else if q == common.NormalQuality {

		f.Thickness = rand.Intn(5) + 1

	} else if q == common.HighQuality {

		f.Thickness = rand.Intn(7) + 1

	}

}
