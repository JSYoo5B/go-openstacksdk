package blockstorage

import (
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/cindervolume"
)

type volumeSearchConversion = cindervolume.Conversion

const (
	volumeSearchUnchanged = cindervolume.Unchanged
	volumeSearchList      = cindervolume.List
	volumeSearchDict      = cindervolume.Dict
	volumeSearchBoolStr   = cindervolume.BoolStr
	volumeSearchBoolean   = cindervolume.Boolean
	volumeSearchInteger   = cindervolume.Integer
)

type volumeSearchDescriptor struct {
	attribute  string
	wire       string
	conversion volumeSearchConversion
}

var volumeSearchDescriptors = func() [cindervolume.DescriptorCount]volumeSearchDescriptor {
	var owned [cindervolume.DescriptorCount]volumeSearchDescriptor
	for i, descriptor := range cindervolume.Descriptors() {
		owned[i] = volumeSearchDescriptor{descriptor.Attribute, descriptor.Wire, descriptor.Conversion}
	}
	return owned
}()

func volumeSearchView(row, location json.RawMessage) (json.RawMessage, error) {
	return cindervolume.View(row, location)
}
func volumeSearchConvert(raw json.RawMessage, conversion volumeSearchConversion) (json.RawMessage, error) {
	return cindervolume.Convert(raw, conversion)
}
func volumeSearchIntegerJSON(raw json.RawMessage) (json.RawMessage, error) {
	return cindervolume.IntegerJSON(raw)
}
