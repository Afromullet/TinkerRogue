package graphics

import (
	"game_main/common"
	"math/rand"
)

// rand.Intn(2) + 1
func (s *TileSquare) CreateWithQuality(q common.QualityType) {

	if q == common.LowQuality {
		s.Size = rand.Intn(2) + 1

	} else if q == common.NormalQuality {

		s.Size = rand.Intn(3) + 1

	} else if q == common.HighQuality {

		s.Size = rand.Intn(4) + 1

	}

}

func (l *TileLine) CreateWithQuality(q common.QualityType) {

	if q == common.LowQuality {
		l.length = rand.Intn(3) + 1

	} else if q == common.NormalQuality {
		l.length = rand.Intn(5) + 1

	} else if q == common.HighQuality {
		l.length = rand.Intn(7) + 1

	}

}

func (c *TileCone) CreateWithQuality(q common.QualityType) {

	if q == common.LowQuality {
		c.length = rand.Intn(3) + 1

	} else if q == common.NormalQuality {
		c.length = rand.Intn(5) + 1

	} else if q == common.HighQuality {
		c.length = rand.Intn(7) + 1

	}

}

func (c *TileCircle) CreateWithQuality(q common.QualityType) {

	if q == common.LowQuality {

		c.radius = rand.Intn(3)

	} else if q == common.NormalQuality {
		c.radius = rand.Intn(4)

	} else if q == common.HighQuality {
		c.radius = rand.Intn(9)

	}

}

func (r *TileRectangle) CreateWithQuality(q common.QualityType) {

	if q == common.LowQuality {

		r.height = rand.Intn(3)
		r.width = rand.Intn(5)

	} else if q == common.NormalQuality {
		r.height = rand.Intn(5)
		r.width = rand.Intn(7)

	} else if q == common.HighQuality {
		r.height = rand.Intn(7)
		r.width = rand.Intn(9)

	}

}
