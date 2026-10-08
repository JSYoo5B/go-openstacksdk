package cloudsnapshot

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Backup has its own raw state; it never uses Snapshot's ID/status indices.
type backupMutationState [len(backupDescriptors)]json.RawMessage

func (s *backupMutationState) overlay(raw json.RawMessage) error {
	return overlayMutationFields(s[:], backupDescriptors[:], raw)
}
func (s *backupMutationState) object() json.RawMessage {
	return mutationFieldsObject(s[:], backupDescriptors[:])
}
func (s *backupMutationState) view(location resource.CloudLocation) (json.RawMessage, error) {
	view, _, err := normalizeBackup(s.object(), nil, false, location)
	return view, err
}
func (s *backupMutationState) id() json.RawMessage { return mutationField(s[22]) }
func (s *backupMutationState) routeID() (string, error) {
	return mutationRouteID(s.id(), "backup")
}
func (s *backupMutationState) status(nullable bool) (string, error) {
	return mutationStatus(s[17], "backup", nullable)
}
