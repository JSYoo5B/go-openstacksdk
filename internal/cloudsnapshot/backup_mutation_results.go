package cloudsnapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"gophercloudsdk/resource"
	"time"
)

type BackupCreateResult struct {
	BackupID                   json.RawMessage
	Created, LastAccepted      *MutationPage
	CreatedValue, Value, Ready json.RawMessage
	CreatedBackup, Backup      *resource.RawResource
	ReadyBackup                *resource.RawResource
}

type BackupDeleteResult struct {
	BackupID                      json.RawMessage
	Deleted                       *bool
	Resolved                      *BackupResult
	Applied, LastAccepted, Absent *MutationPage
	Ready                         json.RawMessage
	ReadyBackup                   *resource.RawResource
}

type BackupWaitTimeoutError struct{ Timeout time.Duration }

func (e *BackupWaitTimeoutError) Error() string {
	return fmt.Sprintf("volume backup wait exceeded %s", e.Timeout)
}
func (e *BackupWaitTimeoutError) Unwrap() error { return context.DeadlineExceeded }
