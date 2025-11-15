package gear

import (
	"game_main/common"
)

func (c CommonItemProperties) CreateWithQuality(q common.QualityType) CommonItemProperties {

	props := CommonItemProperties{}
	if q == common.LowQuality {
		props.Name = ""
		props.Duration = common.RandomInt(3) + 1
		props.Quality = common.LowQuality

	} else if q == common.NormalQuality {
		props.Name = ""
		props.Duration = common.RandomInt(3) + 1
		props.Quality = common.NormalQuality

	} else if q == common.HighQuality {
		props.Name = ""
		props.Duration = common.RandomInt(6) + 1
		props.Quality = common.HighQuality

	}

	return props

}

func (s *Sticky) CreateWithQuality(q common.QualityType) {

	s.MainProps = s.MainProps.CreateWithQuality(q)
	s.MainProps.Name = STICKY_NAME
	if q == common.LowQuality {

		s.Spread = common.RandomInt(2) + 1

	} else if q == common.NormalQuality {

		s.Spread = common.RandomInt(4) + 1

	} else if q == common.HighQuality {

		s.Spread = common.RandomInt(6) + 1

	}

}

func (b *Burning) CreateWithQuality(q common.QualityType) {

	b.MainProps = b.MainProps.CreateWithQuality(q)
	b.MainProps.Name = BURNING_NAME
	if q == common.LowQuality {

		b.Temperature = common.RandomInt(3) + 1
	} else if q == common.NormalQuality {

		b.Temperature = common.RandomInt(5) + 1
	} else if q == common.HighQuality {

		b.Temperature = common.RandomInt(7) + 1
	}

}

func (f *Freezing) CreateWithQuality(q common.QualityType) {

	f.MainProps = f.MainProps.CreateWithQuality(q)
	f.MainProps.Name = FREEZING_NAME
	if q == common.LowQuality {

		f.Thickness = common.RandomInt(3) + 1
	} else if q == common.NormalQuality {

		f.Thickness = common.RandomInt(5) + 1

	} else if q == common.HighQuality {

		f.Thickness = common.RandomInt(7) + 1

	}

}
