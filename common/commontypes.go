package common

// Anything that displays text in the GUI implements this interface
type StringDisplay interface {
	DisplayString()
}

// Not a component, but there's no need to create a source file for just this

// Interface to create an item of a quality. Used for loot generation.
// Take a look at itemquality.go to see how it's implemented
// Implementation looks like this
// //func (t *Throwable) CreateWithQuality(q common.QualityType) {
// ...}
// Which means that we are changing a refernece. Not the best implementation, since it
// Requires us to create an object first. Todo change that in the future. Maybe use a factory?
type Quality interface {
	CreateWithQuality(q QualityType)
}

type QualityType int

var LowQualStr = "Low Quality"
var NormalQualStr = "Normal Quality"
var HighQualStr = "High Quality"

const (
	LowQuality = iota
	NormalQuality
	HighQuality
)
