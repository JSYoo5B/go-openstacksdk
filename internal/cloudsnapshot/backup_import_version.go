package cloudsnapshot

import (
	"gophercloudsdk/internal/cinderrequest"
	"math/big"
)

// Keep the existing import parser contract tests bound to the shared policy.
type backupImportVersion []*big.Int

func parseBackupImportVersion(text string) (backupImportVersion, error) {
	v, err := cinderrequest.ParseVersion(text)
	return backupImportVersion(v), err
}
func (v backupImportVersion) compare(other backupImportVersion) int {
	return cinderrequest.Version(v).Compare(cinderrequest.Version(other))
}
func (v backupImportVersion) header() (string, error) { return cinderrequest.Version(v).Header() }
func selectBackupImportVersion(maximum, minimum string) (string, error) {
	return cinderrequest.SelectVersion(maximum, minimum, "3.64")
}
