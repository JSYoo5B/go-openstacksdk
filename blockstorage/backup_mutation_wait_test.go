package blockstorage_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCreateVolumeBackupCachedAvailableSkipsTimeoutIntervalAndUnusableIdentity(t *testing.T) {
	for _, row := range []string{`{"status":"available"}`, `{"id":null,"status":"AVAILABLE"}`, `{"id":false,"status":"available"}`, `{"id":"bad/id","status":"available"}`} {
		for _, timeout := range []time.Duration{0, -time.Second} {
			t.Run(row+timeout.String(), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
					testcloud.JSON(w, 202, `{"backup":`+row+`}`)
				})
				result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{Timeout: smwDuration(timeout), PollInterval: smwDuration(-time.Second)}))
				if err != nil || result == nil || result.Value == nil || result.Ready == nil || result.ReadyBackup == nil || calls.Load() != 1 {
					t.Fatal(result, err, calls.Load())
				}
			})
		}
	}
}

func TestCreateVolumeBackupInitialErrorIsCheckedOnlyAfterFreshFetchAndMerge(t *testing.T) {
	for _, kind := range []string{"fresh available", "fresh error", "fresh nonstring", "malformed keeps error"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			created := `{"backup":{"id":"first","status":"error","name":"prior","metadata":{"n":9007199254740993},"size":"003"}}`
			fresh := `{"backup":{"status":"AVAILABLE","description":"fresh","unknown":true}}`
			switch kind {
			case "fresh error":
				fresh = `{"backup":{"status":"ErRoR"}}`
			case "fresh nonstring":
				fresh = `{"backup":{"status":false}}`
			case "malformed keeps error":
				fresh = `not JSON`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 202, created)
				case 2:
					bmwRequest(t, r, "GET", bmwCollection+"/first", "3.60", "test-token")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 203, fresh)
				default:
					t.Error("wait replay", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{PollInterval: smwDuration(-time.Second)}))
			if result == nil || result.CreatedValue == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			bmwPage(t, result.Created, 202, created, "created")
			bmwPage(t, result.LastAccepted, 203, fresh, "fresh")
			if kind != "fresh available" {
				var physical *resource.ResponseError
				var failed *resource.FailedStateError
				if err == nil || result.Value != nil || result.Backup != nil || result.Ready != nil || errors.As(err, &physical) {
					t.Fatal(result, err)
				}
				if kind == "fresh nonstring" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.As(err, &failed) || failed.ID != "first" {
					t.Fatal(err)
				}
				return
			}
			if err != nil || result.Ready == nil || result.Backup == nil || result.ReadyBackup == nil {
				t.Fatal(result, err)
			}
			view := smwFields(t, result.Value)
			if string(view["name"]) != `"prior"` || string(view["metadata"]) != `{"n":9007199254740993}` || string(view["size"]) != "3" || string(view["description"]) != `"fresh"` || string(result.BackupID) != `"first"` {
				t.Fatal(string(result.Value))
			}
			if _, invented := result.Backup.Body["id"]; invented {
				t.Fatal("prior ID fabricated in fresh object", result.Backup)
			}
			original := bytes.Clone(result.Value)
			result.CreatedValue[0] = '!'
			result.CreatedBackup.Body["id"][0] = '!'
			result.LastAccepted.Body[0] = '!'
			result.Backup.Body["status"][0] = '!'
			if !bytes.Equal(result.Value, original) || string(result.ReadyBackup.Body["status"]) != `"AVAILABLE"` {
				t.Fatal("fresh phases alias", result)
			}
		})
	}
}

func TestCreateVolumeBackupNullablePollContinuesAndLatestIDIsUsedOnlyForNeededRoute(t *testing.T) {
	for _, terminal := range []bool{true, false} {
		t.Run(map[bool]string{true: "terminal unsafe ID", false: "nonterminal unsafe ID"}[terminal], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			fresh := `{"backup":{"id":"second","status":null,"description":"merged"}}`
			last := `{"backup":{"id":"bad/id","status":"AVAILABLE"}}`
			if !terminal {
				last = `{"backup":{"id":"bad/id","status":"creating"}}`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
					testcloud.JSON(w, 202, `{"backup":{"id":"first","status":"creating","name":"prior"}}`)
				case 2:
					bmwRequest(t, r, "GET", bmwCollection+"/first", "3.60", "test-token")
					testcloud.JSON(w, 200, fresh)
				case 3:
					bmwRequest(t, r, "GET", bmwCollection+"/second", "3.60", "test-token")
					w.Header().Set("X-Proof", "last")
					testcloud.JSON(w, 200, last)
				default:
					t.Error("unsafe terminal ID routed", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{PollInterval: smwDuration(time.Millisecond)}))
			if result == nil || calls.Load() != 3 || string(result.BackupID) != `"bad/id"` {
				t.Fatal(result, err, calls.Load())
			}
			bmwPage(t, result.LastAccepted, 200, last, "last")
			if terminal {
				if err != nil || result.Ready == nil || string(smwFields(t, result.Value)["description"]) != `"merged"` || string(smwFields(t, result.Value)["name"]) != `"prior"` {
					t.Fatal(result, err)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || result.Value != nil || result.Ready != nil || result.CreatedValue == nil {
					t.Fatal(result, err)
				}
			}
		})
	}
}

func TestVolumeBackupMutationNonpositiveTimeoutRunsMutationButPreventsFirstPoll(t *testing.T) {
	for _, entry := range []string{"create", "delete", "force delete"} {
		for _, timeout := range []time.Duration{0, -time.Second} {
			t.Run(entry+timeout.String(), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					n := calls.Add(1)
					if entry == "create" {
						bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
						w.Header().Set("X-Proof", "applied")
						testcloud.JSON(w, 202, `{"backup":{"id":false,"status":"creating"}}`)
						return
					}
					if n == 1 {
						bmwRequest(t, r, "GET", bmwCollection+"/requested", "3.60", "test-token")
						testcloud.JSON(w, 200, `{"backup":{"id":"actual","status":"deleted"}}`)
						return
					}
					method, path, version := "DELETE", bmwCollection+"/actual", "3.60"
					if entry == "force delete" {
						method, path, version = "POST", path+"/action", "3.64"
					}
					bmwRequest(t, r, method, path, version, "test-token")
					w.Header().Set("X-Proof", "applied")
					w.WriteHeader(204)
				})
				policy := blockstorage.BackupMutationWaitOpts{Timeout: smwDuration(timeout), PollInterval: smwDuration(-time.Second)}
				var err error
				if entry == "create" {
					result, failed := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWaitPolicy(policy))
					err = failed
					if result == nil || result.CreatedValue == nil || result.Value != nil || result.Ready != nil || calls.Load() != 1 {
						t.Fatal(result, err, calls.Load())
					}
					bmwPage(t, result.LastAccepted, 202, `{"backup":{"id":false,"status":"creating"}}`, "applied")
				} else {
					result, failed := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupForce(entry == "force delete"), blockstorage.WithDeleteVolumeBackupWait(true), blockstorage.WithDeleteVolumeBackupWaitPolicy(policy))
					err = failed
					if result == nil || result.Resolved == nil || result.Resolved.Value == nil || result.Deleted != nil || result.Ready != nil || result.Absent != nil || calls.Load() != 2 {
						t.Fatal(result, err, calls.Load())
					}
					bmwPage(t, result.LastAccepted, 204, "", "applied")
				}
				var expired *blockstorage.BackupWaitTimeoutError
				if !errors.As(err, &expired) || expired.Timeout != timeout || !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestVolumeBackupMutationPositiveBudgetStartsAfterMutationAndAllowsSlowTerminalHTTP(t *testing.T) {
	for _, entry := range []string{"create", "delete", "force delete"} {
		for _, slow := range []string{"mutation", "terminal GET"} {
			t.Run(entry+"/"+slow, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				var calls atomic.Int32
				mutationStep := int32(1)
				terminalStep := int32(2)
				if entry != "create" {
					mutationStep, terminalStep = 2, 3
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					n := calls.Add(1)
					if (n == mutationStep && slow == "mutation") || (n == terminalStep && slow == "terminal GET") {
						time.Sleep(350 * time.Millisecond)
					}
					if n == mutationStep {
						if entry == "create" {
							bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
							testcloud.JSON(w, 202, `{"backup":{"id":"actual","status":"creating"}}`)
						} else {
							method, path, version := "DELETE", bmwCollection+"/actual", "3.60"
							if entry == "force delete" {
								method, path, version = "POST", path+"/action", "3.64"
							}
							bmwRequest(t, r, method, path, version, "test-token")
							w.WriteHeader(204)
						}
						return
					}
					if n == terminalStep {
						bmwRequest(t, r, "GET", bmwCollection+"/actual", "3.60", "test-token")
						if r.Context().Err() != nil {
							t.Error("SDK loop budget canceled HTTP", r.Context().Err())
						}
						status := "available"
						if entry != "create" {
							status = "deleted"
						}
						testcloud.JSON(w, 200, `{"backup":{"status":"`+status+`"}}`)
						return
					}
					if n == 1 && entry != "create" {
						bmwRequest(t, r, "GET", bmwCollection+"/requested", "3.60", "test-token")
						testcloud.JSON(w, 200, `{"backup":{"id":"actual","status":"available"}}`)
						return
					}
					t.Error("budget caused workflow replay", entry, n, r.URL)
					w.WriteHeader(500)
				})
				policy := blockstorage.BackupMutationWaitOpts{Timeout: smwDuration(250 * time.Millisecond), PollInterval: smwDuration(-time.Second)}
				if entry == "create" {
					result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWaitPolicy(policy))
					if err != nil || result == nil || result.Ready == nil || calls.Load() != terminalStep {
						t.Fatal(result, err, calls.Load())
					}
				} else {
					result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupForce(entry == "force delete"), blockstorage.WithDeleteVolumeBackupWait(true), blockstorage.WithDeleteVolumeBackupWaitPolicy(policy))
					if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Ready == nil || calls.Load() != terminalStep {
						t.Fatal(result, err, calls.Load())
					}
				}
			})
		}
	}
}

func TestCreateVolumeBackupPollIntervalIsConsumedOnlyAfterNonterminalState(t *testing.T) {
	for _, status := range []string{"available", "error", "creating"} {
		t.Run(status, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			reply := `{"backup":{"status":"` + status + `"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 202, `{"backup":{"id":"actual","status":"creating"}}`)
				} else {
					bmwRequest(t, r, "GET", bmwCollection+"/actual", "3.60", "test-token")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 200, reply)
				}
			})
			result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{PollInterval: smwDuration(-time.Second)}))
			if result == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			bmwPage(t, result.LastAccepted, 200, reply, "fresh")
			switch status {
			case "available":
				if err != nil || result.Ready == nil {
					t.Fatal(result, err)
				}
			case "error":
				var e *resource.FailedStateError
				if !errors.As(err, &e) || errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "creating":
				if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "interval") {
					t.Fatal(err)
				}
			}
		})
	}
	t.Run("zero pauses rather than busy-looping", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := bmwClient(cloud)
		var calls atomic.Int32
		previous := make(chan time.Time, 1)
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				testcloud.JSON(w, 202, `{"backup":{"id":"actual","status":"creating"}}`)
			case 2:
				previous <- time.Now()
				testcloud.JSON(w, 200, `{"backup":{"status":"creating"}}`)
			case 3:
				at := <-previous
				if time.Since(at) < 75*time.Millisecond {
					t.Error("zero interval caused immediate poll", time.Since(at))
				}
				testcloud.JSON(w, 200, `{"backup":{"status":"available"}}`)
			default:
				t.Error("extra zero poll")
				w.WriteHeader(500)
			}
		})
		result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{PollInterval: smwDuration(0)}))
		if err != nil || result == nil || result.Ready == nil || calls.Load() != 3 {
			t.Fatal(result, err, calls.Load())
		}
	})
}

func TestDeleteVolumeBackupFreshFetchMergesCachedDeletedButRejectsExplicitNullStatus(t *testing.T) {
	for _, reply := range []string{`{"backup":{"status":"DeLeTeD"}}`, ``, `not JSON`, `{}`, `{"backup":{"status":null}}`, `{"backup":{"status":false}}`} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					bmwRequest(t, r, "GET", bmwCollection+"/requested", "3.60", "test-token")
					testcloud.JSON(w, 200, `{"backup":{"id":"actual","status":"deleted","name":"prior"}}`)
				case 2:
					bmwRequest(t, r, "DELETE", bmwCollection+"/actual", "3.60", "test-token")
					w.Header().Set("X-Proof", "ack")
					w.WriteHeader(204)
				case 3:
					bmwRequest(t, r, "GET", bmwCollection+"/actual", "3.60", "test-token")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 203, reply)
				default:
					t.Error("fresh delete replay")
					w.WriteHeader(500)
				}
			})
			result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupWait(true), blockstorage.WithDeleteVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{PollInterval: smwDuration(-time.Second)}))
			if result == nil || result.Resolved == nil || calls.Load() != 3 {
				t.Fatal(result, err, calls.Load())
			}
			bmwPage(t, result.Applied, 204, "", "ack")
			bmwPage(t, result.LastAccepted, 203, reply, "fresh")
			if strings.Contains(reply, "null") || strings.Contains(reply, "false") {
				var physical *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || result.Deleted != nil || result.Ready != nil || result.Absent != nil {
					t.Fatal(result, err)
				}
				return
			}
			if err != nil || result.Deleted == nil || !*result.Deleted || result.Ready == nil || result.Absent != nil {
				t.Fatal(result, err)
			}
			view := smwFields(t, result.Ready)
			if string(view["id"]) != `"actual"` || string(view["name"]) != `"prior"` {
				t.Fatal(string(result.Ready))
			}
			malformed := reply == "" || reply == "not JSON"
			if malformed != (result.ReadyBackup == nil) {
				t.Fatal("fresh physical object invented", result)
			}
		})
	}
}

func TestDeleteVolumeBackupErrorStatesContinueAndLatestIDDoesNotRouteTerminalFalse(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	var calls atomic.Int32
	last := `{"backup":{"id":false,"status":"DELETED"}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			testcloud.JSON(w, 200, `{"backup":{"id":"first","status":"error_deleting","name":"prior"}}`)
		case 2:
			bmwRequest(t, r, "DELETE", bmwCollection+"/first", "3.60", "test-token")
			w.Header().Set("X-Proof", "ack")
			w.WriteHeader(204)
		case 3:
			bmwRequest(t, r, "GET", bmwCollection+"/first", "3.60", "test-token")
			testcloud.JSON(w, 200, `{"backup":{"id":"second","status":"error","description":"fresh"}}`)
		case 4:
			bmwRequest(t, r, "GET", bmwCollection+"/second", "3.60", "test-token")
			w.Header().Set("X-Proof", "terminal")
			testcloud.JSON(w, 200, last)
		default:
			t.Error("terminal false ID routed")
			w.WriteHeader(500)
		}
	})
	result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupWait(true), blockstorage.WithDeleteVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{PollInterval: smwDuration(time.Millisecond)}))
	if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Ready == nil || string(result.BackupID) != "false" || calls.Load() != 4 {
		t.Fatal(result, err, calls.Load())
	}
	view := smwFields(t, result.Ready)
	if string(view["name"]) != `"prior"` || string(view["description"]) != `"fresh"` {
		t.Fatal(string(result.Ready))
	}
	bmwPage(t, result.LastAccepted, 200, last, "terminal")
	owned := bytes.Clone(result.Ready)
	result.Resolved.Value[0] = '!'
	result.LastAccepted.Body[0] = '!'
	result.Applied.Header.Set("X-Proof", "changed")
	if !bytes.Equal(result.Ready, owned) || string(result.ReadyBackup.Body["id"]) != "false" {
		t.Fatal("ready aliases earlier phase", result)
	}
}

func TestVolumeBackupMutationPositiveTimeoutIsObservedAtLoopBoundaryWithoutBorrowingProof(t *testing.T) {
	for _, entry := range []string{"create", "delete"} {
		t.Run(entry, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			pollStep := int32(2)
			status := "creating"
			if entry == "delete" {
				pollStep = 3
				status = "error_deleting"
			}
			fresh := `{"backup":{"status":"` + status + `"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if n == pollStep {
					bmwRequest(t, r, "GET", bmwCollection+"/actual", "3.60", "test-token")
					w.Header().Set("X-Proof", "nonterminal")
					testcloud.JSON(w, 200, fresh)
					return
				}
				if n == 1 {
					if entry == "create" {
						testcloud.JSON(w, 202, `{"backup":{"id":"actual","status":"creating"}}`)
					} else {
						testcloud.JSON(w, 200, `{"backup":{"id":"actual","status":"available"}}`)
					}
					return
				}
				if n == 2 && entry == "delete" {
					w.WriteHeader(204)
					return
				}
				t.Error("expired loop sent another request", r.URL)
				w.WriteHeader(500)
			})
			policy := blockstorage.BackupMutationWaitOpts{Timeout: smwDuration(250 * time.Millisecond), PollInterval: smwDuration(350 * time.Millisecond)}
			var err error
			if entry == "create" {
				result, failed := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWaitPolicy(policy))
				err = failed
				if result == nil || result.Value != nil || result.Ready != nil || result.CreatedValue == nil {
					t.Fatal(result, err)
				}
				bmwPage(t, result.LastAccepted, 200, fresh, "nonterminal")
			} else {
				result, failed := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupWait(true), blockstorage.WithDeleteVolumeBackupWaitPolicy(policy))
				err = failed
				if result == nil || result.Deleted != nil || result.Ready != nil || result.Resolved == nil || result.Applied == nil || result.Absent != nil {
					t.Fatal(result, err)
				}
				bmwPage(t, result.LastAccepted, 200, fresh, "nonterminal")
			}
			var expired *blockstorage.BackupWaitTimeoutError
			var physical *resource.ResponseError
			if !errors.As(err, &expired) || errors.As(err, &physical) || calls.Load() != pollStep {
				t.Fatal(err, calls.Load())
			}
		})
	}
}
