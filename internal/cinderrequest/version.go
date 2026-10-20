package cinderrequest

import "github.com/JSYoo5B/go-openstacksdk/internal/microversions"

// Keep existing Cinder consumers bound to the shared source-audited arithmetic.
type Version = microversions.Version

func ParseVersion(text string) (Version, error) { return microversions.ParseVersion(text) }
func SelectVersion(maximum, minimum, ceiling string) (string, error) {
	return microversions.SelectVersion(maximum, minimum, ceiling)
}
