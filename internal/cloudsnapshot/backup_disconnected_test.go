package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBackupDisconnectedModelKeepsNullLocationAndNeverSeedsIdentity(t *testing.T) {
	for _, raw := range []string{`{}`, `{"project_id":"foreign","availability_zone":[false,0],"location":{"cloud":"wire"},"connection":{},"microversion":"3.99","_synchronized":false,"self":null}`, `{"id":false,"project_id":{"n":9007199254740993},"availability_zone":{"raw":"zone"}}`} {
		input := json.RawMessage(raw)
		view, err := normalizeBackupDisconnected(input)
		if err != nil {
			t.Fatal(string(view), err)
		}
		fields := snapshotModelContractFields(t, view)
		if len(fields) != 24 {
			t.Fatal(string(view))
		}
		snapshotModelContractRaw(t, fields, "location", "null")
		wantID := "null"
		if bytes.HasPrefix(input, []byte(`{"id":false`)) {
			wantID = "false"
		}
		snapshotModelContractRaw(t, fields, "id", wantID)
		for _, unknown := range []string{"connection", "microversion", "_synchronized", "self"} {
			if _, present := fields[unknown]; present {
				t.Fatal("runtime field leaked", unknown, string(view))
			}
		}
		before := bytes.Clone(view)
		input[0] = '!'
		if !bytes.Equal(view, before) {
			t.Fatal("disconnected result aliases source")
		}
	}
}

func TestBackupDisconnectedModelSharesNullableDescriptorConversion(t *testing.T) {
	raw := json.RawMessage(`{"force":"false","has_dependent_backups":{},"is_incremental":null,"size":6.9,"object_count":[],"links":"literal","metadata":{"n":900719925474099312345},"os-backup-project-attr:project_id":"wire","project_id":"attribute"}`)
	view, err := normalizeBackupDisconnected(raw)
	if err != nil {
		t.Fatal(string(view), err)
	}
	fields := snapshotModelContractFields(t, view)
	for key, want := range map[string]string{"force": "true", "has_dependent_backups": "false", "is_incremental": "null", "size": "6", "object_count": "0", "links": `["literal"]`, "metadata": `{"n":900719925474099312345}`, "project_id": `"attribute"`, "location": "null", "id": "null"} {
		snapshotModelContractRaw(t, fields, key, want)
	}
	for _, bad := range []json.RawMessage{json.RawMessage(`{"size":"²"}`), json.RawMessage(`{"object_count":1e400}`), json.RawMessage(`null`), json.RawMessage(`[]`), json.RawMessage(`{}` + ` trailing`), json.RawMessage{0xff}} {
		view, err := normalizeBackupDisconnected(bad)
		if err == nil || view != nil {
			t.Fatal("invalid disconnected model accepted", string(bad), string(view), err)
		}
	}
}
