package cloudsnapshot

import "encoding/json"

// A newly imported Backup has no connection, cached ID, or computed scope.
func normalizeBackupDisconnected(raw json.RawMessage) (json.RawMessage, error) {
	view, _, err := normalizeBackupView(raw, nil, false, nil)
	return view, err
}
