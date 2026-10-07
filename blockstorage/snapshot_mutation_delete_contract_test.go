package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func smwDelete(t *testing.T, client *gophercloud.ServiceClient, name string, options ...blockstorage.DeleteVolumeSnapshotOption) (*blockstorage.DeleteVolumeSnapshotResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return blockstorage.DeleteVolumeSnapshot(ctx, client, blockstorage.DeleteVolumeSnapshotRequest{NameOrID: name}, options...)
}

func TestDeleteVolumeSnapshotDefaultWaitOffDeletesActualResolvedIDWithOpaqueAcknowledgement(t *testing.T) {
	for _, code := range []int{200, 203, 204, 399} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			lookup := `{"snapshot":{"id":"actual","status":"deleted","unknown":9007199254740993}}`
			ack := "\x00\xffopaque"
			if code == 204 {
				ack = ""
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					smwRequest(t, r, http.MethodGet, smwCollection+"/requested")
					w.Header().Set("X-Proof", "lookup")
					testcloud.JSON(w, 203, lookup)
				case 2:
					smwRequest(t, r, http.MethodDelete, smwCollection+"/actual")
					if r.URL.RawQuery != "" {
						t.Error("delete gained force/cascade", r.URL)
					}
					w.Header().Set("X-Proof", "ack")
					w.WriteHeader(code)
					_, _ = w.Write([]byte(ack))
				default:
					t.Error("default delete polled or repeated lookup", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := smwDelete(t, client, "requested", blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(-time.Second), PollInterval: smwDuration(-time.Second)}))
			if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Resolved == nil || result.Resolved.Snapshot == nil || result.Resolved.Observed == nil || len(result.Resolved.Pages) != 0 || string(result.SnapshotID) != `"actual"` || result.Ready != nil || result.Absent != nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			smwPage(t, result.Applied, code, ack, "ack")
			smwPage(t, result.LastAccepted, code, ack, "ack")
			result.Applied.Body = append(result.Applied.Body, '!')
			result.Applied.Header.Set("X-Proof", "changed")
			result.Resolved.Snapshot.Body["id"][0] = '!'
			result.Resolved.Observed.Body[0] = '!'
			smwPage(t, result.LastAccepted, code, ack, "ack")
			if string(result.SnapshotID) != `"actual"` || string(smwFields(t, result.Resolved.Value)["id"]) != `"actual"` {
				t.Fatal("delete phase aliases lookup", result)
			}
		})
	}
}

func TestDeleteVolumeSnapshotEmptyInputRunsOriginalsOnceWithoutConsumingServiceOrLocation(t *testing.T) {
	for _, kind := range []string{"nil client", "unusable client", "callback error", "canceled", "nil context"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			callbacks := 0
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			var ctx context.Context = context.Background()
			cause := errors.New("empty-delete original")
			var cancel context.CancelCauseFunc
			switch kind {
			case "nil client":
				client = nil
			case "unusable client":
				client.ProviderClient = nil
				client.ResourceBase = "bad"
			case "canceled":
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
			case "nil context":
				ctx = nil
			}
			option := func(o *blockstorage.DeleteVolumeSnapshotOpts) error {
				callbacks++
				o.Location = &resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage{}}}
				o.Wait = smwBool(true)
				o.WaitPolicy = blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(-time.Second), PollInterval: smwDuration(-time.Second)}
				if kind == "callback error" {
					return cause
				}
				return nil
			}
			result, err := blockstorage.DeleteVolumeSnapshot(ctx, client, blockstorage.DeleteVolumeSnapshotRequest{}, option)
			if calls.Load() != 0 {
				t.Fatal("empty delete used a service", calls.Load())
			}
			if kind == "canceled" || kind == "nil context" {
				if result != nil || err == nil || callbacks != 0 {
					t.Fatal(result, err, callbacks)
				}
				if kind == "canceled" && !errors.Is(err, cause) {
					t.Fatal(err)
				}
				return
			}
			if callbacks != 1 {
				t.Fatal("original count", callbacks)
			}
			if kind == "callback error" {
				if result != nil || !errors.Is(err, cause) {
					t.Fatal(result, err)
				}
				return
			}
			if err != nil || result == nil || result.Deleted == nil || *result.Deleted || result.Resolved != nil || result.Applied != nil || result.LastAccepted != nil || result.Absent != nil || result.SnapshotID != nil {
				t.Fatal(result, err)
			}
		})
	}
}

func TestDeleteVolumeSnapshotFallbackRequiresCompletedExactLookupAndPreservesAbsenceOrAmbiguity(t *testing.T) {
	for _, kind := range []string{"found", "missing", "ambiguous", "late list error"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			first := `{"snapshots":[{"id":"actual","name":"requested"}],"snapshots_links":[{"rel":"next","href":"?marker=next"}]}`
			if kind == "missing" {
				first = `{"snapshots":[]}`
			}
			if kind == "ambiguous" {
				first = `{"snapshots":[{"id":"one","name":"requested"},{"id":"two","name":"requested"},{"size":"²"}],"snapshots_links":[null]}`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				switch n {
				case 1:
					smwRequest(t, r, http.MethodGet, smwCollection+"/requested")
					testcloud.JSON(w, 404, `{"error":"missing member"}`)
				case 2:
					smwRequest(t, r, http.MethodGet, smwDetail)
					if r.URL.Query().Get("name") != "requested" {
						t.Error(r.URL)
					}
					w.Header().Set("X-Proof", "list first")
					testcloud.JSON(w, 200, first)
				case 3:
					smwRequest(t, r, http.MethodGet, smwDetail)
					if r.URL.Query().Get("marker") != "next" || r.URL.Query().Get("name") != "requested" {
						t.Error(r.URL)
					}
					w.Header().Set("X-Proof", "list second")
					if kind == "late list error" {
						testcloud.JSON(w, 500, `{"error":"late list"}`)
					} else {
						testcloud.JSON(w, 200, `{"snapshots":[]}`)
					}
				case 4:
					if kind != "found" {
						t.Error("failed lookup mutated", r.URL)
					}
					smwRequest(t, r, http.MethodDelete, smwCollection+"/actual")
					w.WriteHeader(204)
				default:
					t.Error("lookup replay", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := smwDelete(t, client, "requested")
			if result == nil || result.Resolved == nil || result.Resolved.Observed != nil {
				t.Fatal(result, err)
			}
			switch kind {
			case "found":
				if err != nil || result.Deleted == nil || !*result.Deleted || calls.Load() != 4 || len(result.Resolved.Pages) != 2 || string(result.SnapshotID) != `"actual"` {
					t.Fatal(result, err, calls.Load())
				}
			case "missing":
				if err != nil || result.Deleted == nil || *result.Deleted || calls.Load() != 2 || len(result.Resolved.Pages) != 1 || result.Resolved.Value != nil || result.Applied != nil {
					t.Fatal(result, err, calls.Load())
				}
			case "ambiguous":
				if !errors.Is(err, resource.ErrAmbiguous) || result.Deleted != nil || result.Applied != nil || calls.Load() != 2 || len(result.Resolved.Pages) != 1 {
					t.Fatal(result, err, calls.Load())
				}
			case "late list error":
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 500 || result.Deleted != nil || result.Applied != nil || result.Resolved.Value != nil || calls.Load() != 3 || len(result.Resolved.Pages) != 1 {
					t.Fatal(result, err, calls.Load())
				}
			}
		})
	}
}

func TestDeleteVolumeSnapshotMemberMissingIDSeedsLookupButExplicitNullCannotRoute(t *testing.T) {
	for _, reply := range []string{`{}`, `{"snapshot":{}}`, `not JSON`, `{"snapshot":{"id":null}}`, `{"snapshot":{"id":false}}`, `{"snapshot":{"id":"bad/id"}}`} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					smwRequest(t, r, http.MethodGet, smwCollection+"/requested")
					w.Header().Set("X-Proof", "member")
					testcloud.JSON(w, 203, reply)
				} else {
					smwRequest(t, r, http.MethodDelete, smwCollection+"/requested")
					w.WriteHeader(204)
				}
			})
			result, err := smwDelete(t, client, "requested")
			if result == nil || result.Resolved == nil || result.Resolved.Observed == nil || result.Resolved.Snapshot == nil {
				t.Fatal(result, err)
			}
			seeded := reply == `{}` || reply == `{"snapshot":{}}` || reply == `not JSON`
			if seeded {
				if err != nil || result.Deleted == nil || !*result.Deleted || !result.Resolved.SeededID || result.Resolved.RequestedID != "requested" || string(result.SnapshotID) != `"requested"` || calls.Load() != 2 {
					t.Fatal(result, err, calls.Load())
				}
				if _, fabricated := result.Resolved.Snapshot.Body["id"]; fabricated {
					t.Fatal("seed fabricated physical ID")
				}
			} else {
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || result.Deleted != nil || result.Applied != nil || result.Resolved.Value == nil || result.Resolved.SeededID || errors.As(err, &proof) || calls.Load() != 1 {
					t.Fatal(result, err, calls.Load())
				}
			}
		})
	}
}

func TestDeleteVolumeSnapshotMutation404IsTerminalAndNeverPollAbsence(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			testcloud.JSON(w, 200, `{"snapshot":{"id":"actual","status":"available"}}`)
		case 2:
			smwRequest(t, r, http.MethodDelete, smwCollection+"/actual")
			w.Header().Set("X-Proof", "rejected delete")
			testcloud.JSON(w, 404, `{"error":"mutation race"}`)
		default:
			t.Error("rejected DELETE polled", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := smwDelete(t, client, "requested", blockstorage.WithDeleteVolumeSnapshotWait(true))
	var native gophercloud.ErrUnexpectedResponseCode
	var proof *resource.ResponseError
	if result == nil || !errors.As(err, &native) || native.Actual != 404 || native.Method != http.MethodDelete || native.ResponseHeader.Get("X-Proof") != "rejected delete" || errors.As(err, &proof) || result.Deleted != nil || result.Resolved == nil || result.Resolved.Value == nil || result.Resolved.Observed == nil || result.Applied != nil || result.LastAccepted != nil || result.Absent != nil || result.Ready != nil || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
	smwOperation(t, err, "DeleteVolumeSnapshot")
}

func TestDeleteVolumeSnapshotWaitAlwaysFetchesFreshAndToleratesMalformedBodyByMergingPriorDeleted(t *testing.T) {
	for _, reply := range []string{`{"snapshot":{"status":"DeLeTeD"}}`, ``, `not JSON`, `{}`} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					smwRequest(t, r, http.MethodGet, smwCollection+"/requested")
					testcloud.JSON(w, 200, `{"snapshot":{"id":"actual","status":"deleted","name":"prior"}}`)
				case 2:
					smwRequest(t, r, http.MethodDelete, smwCollection+"/actual")
					w.Header().Set("X-Proof", "ack")
					w.WriteHeader(204)
				case 3:
					smwRequest(t, r, http.MethodGet, smwCollection+"/actual")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 203, reply)
				default:
					t.Error("extra wait", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := smwDelete(t, client, "requested", blockstorage.WithDeleteVolumeSnapshotWait(true), blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(-time.Second)}))
			if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Ready == nil || result.Absent != nil || calls.Load() != 3 {
				t.Fatal(result, err, calls.Load())
			}
			fields := smwFields(t, result.Ready)
			if string(fields["id"]) != `"actual"` || string(fields["name"]) != `"prior"` {
				t.Fatal(string(result.Ready))
			}
			malformed := reply == "" || reply == "not JSON"
			if malformed != (result.ReadySnapshot == nil) {
				t.Fatal("actual fresh object invented", result)
			}
			smwPage(t, result.Applied, 204, "", "ack")
			smwPage(t, result.LastAccepted, 203, reply, "fresh")
		})
	}
}

func TestDeleteVolumeSnapshotFreshNullOrNonstringStatusIsAnErrorAfterAcknowledgement(t *testing.T) {
	for _, status := range []string{`null`, `false`, `42`, `[]`, `{}`} {
		t.Run(status, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			fresh := `{"snapshot":{"status":` + status + `}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					testcloud.JSON(w, 200, `{"snapshot":{"id":"actual","status":"deleted"}}`)
				case 2:
					w.Header().Set("X-Proof", "ack")
					w.WriteHeader(204)
				case 3:
					smwRequest(t, r, http.MethodGet, smwCollection+"/actual")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 200, fresh)
				default:
					t.Error("invalid status continued", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := smwDelete(t, client, "requested", blockstorage.WithDeleteVolumeSnapshotWait(true), blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(time.Millisecond)}))
			var physical *resource.ResponseError
			if result == nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || result.Deleted != nil || result.Ready != nil || result.Absent != nil || result.Resolved.Value == nil || calls.Load() != 3 {
				t.Fatal(result, err, calls.Load())
			}
			smwPage(t, result.Applied, 204, "", "ack")
			smwPage(t, result.LastAccepted, 200, fresh, "fresh")
		})
	}
}

func TestDeleteVolumeSnapshotErrorsAreNonterminalAndLatestMergedIDControlsNextPoll(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	var calls atomic.Int32
	fresh := `{"snapshot":{"id":"second","status":"error","description":"fresh"}}`
	last := `{"snapshot":{"id":false,"status":"DELETED"}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			testcloud.JSON(w, 200, `{"snapshot":{"id":"first","status":"error_deleting","name":"prior"}}`)
		case 2:
			smwRequest(t, r, http.MethodDelete, smwCollection+"/first")
			w.Header().Set("X-Proof", "ack")
			w.WriteHeader(204)
		case 3:
			smwRequest(t, r, http.MethodGet, smwCollection+"/first")
			testcloud.JSON(w, 200, fresh)
		case 4:
			smwRequest(t, r, http.MethodGet, smwCollection+"/second")
			w.Header().Set("X-Proof", "terminal")
			testcloud.JSON(w, 200, last)
		default:
			t.Error("terminal false ID routed", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := smwDelete(t, client, "requested", blockstorage.WithDeleteVolumeSnapshotWait(true), blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(time.Millisecond)}))
	if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Ready == nil || string(result.SnapshotID) != "false" || calls.Load() != 4 {
		t.Fatal(result, err, calls.Load())
	}
	fields := smwFields(t, result.Ready)
	if string(fields["name"]) != `"prior"` || string(fields["description"]) != `"fresh"` {
		t.Fatal(string(result.Ready))
	}
	smwPage(t, result.LastAccepted, 200, last, "terminal")
	priorReady := bytes.Clone(result.Ready)
	result.Resolved.Value[0] = '!'
	result.LastAccepted.Body[0] = '!'
	result.Applied.Header.Set("X-Proof", "changed")
	if !bytes.Equal(result.Ready, priorReady) || string(result.ReadySnapshot.Body["id"]) != "false" {
		t.Fatal("delete phases alias", result)
	}
}

func TestDeleteVolumeSnapshotCleanPolling404CompletesWithSeparateAbsenceProof(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	var calls atomic.Int32
	fresh := `{"snapshot":{"status":"error_deleting"}}`
	absent := `{"error":"gone"}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			testcloud.JSON(w, 200, `{"snapshot":{"id":"actual","status":"available"}}`)
		case 2:
			smwRequest(t, r, http.MethodDelete, smwCollection+"/actual")
			w.Header().Set("X-Proof", "ack")
			w.WriteHeader(204)
		case 3:
			smwRequest(t, r, http.MethodGet, smwCollection+"/actual")
			w.Header().Set("X-Proof", "last admitted")
			testcloud.JSON(w, 203, fresh)
		case 4:
			smwRequest(t, r, http.MethodGet, smwCollection+"/actual")
			w.Header().Set("X-Proof", "absent")
			testcloud.JSON(w, 404, absent)
		default:
			t.Error("absence retried", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := smwDelete(t, client, "requested", blockstorage.WithDeleteVolumeSnapshotWait(true), blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(time.Millisecond)}))
	if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Ready != nil || result.ReadySnapshot != nil || calls.Load() != 4 {
		t.Fatal(result, err, calls.Load())
	}
	smwPage(t, result.Applied, 204, "", "ack")
	smwPage(t, result.LastAccepted, 203, fresh, "last admitted")
	smwPage(t, result.Absent, 404, absent, "absent")
	result.Absent.Body[0] = '!'
	result.Absent.Header.Set("X-Proof", "changed")
	smwPage(t, result.LastAccepted, 203, fresh, "last admitted")
}

func TestDeleteVolumeSnapshotNonpositiveWaitTimeoutIsCheckedAfterDELETEBeforeEvenCachedDeletedGET(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					testcloud.JSON(w, 200, `{"snapshot":{"id":"actual","status":"deleted"}}`)
				case 2:
					smwRequest(t, r, http.MethodDelete, smwCollection+"/actual")
					w.Header().Set("X-Proof", "ack")
					w.WriteHeader(204)
				default:
					t.Error("nonpositive timeout permitted a GET", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := smwDelete(t, client, "requested", blockstorage.WithDeleteVolumeSnapshotWait(true), blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(timeout), PollInterval: smwDuration(-time.Second)}))
			var expired *blockstorage.SnapshotWaitTimeoutError
			if result == nil || !errors.As(err, &expired) || expired.Timeout != timeout || result.Deleted != nil || result.Ready != nil || result.Absent != nil || result.Resolved.Value == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			smwPage(t, result.Applied, 204, "", "ack")
			smwPage(t, result.LastAccepted, 204, "", "ack")
		})
	}
}

func TestDeleteVolumeSnapshotPositiveBudgetStartsAfterDELETEAndAcceptsSlowFreshDeleted(t *testing.T) {
	for _, slowPhase := range []string{"DELETE", "GET"} {
		t.Run(slowPhase, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if (n == 2 && slowPhase == "DELETE") || (n == 3 && slowPhase == "GET") {
					time.Sleep(350 * time.Millisecond)
				}
				switch n {
				case 1:
					testcloud.JSON(w, 200, `{"snapshot":{"id":"actual","status":"available"}}`)
				case 2:
					smwRequest(t, r, http.MethodDelete, smwCollection+"/actual")
					w.WriteHeader(204)
				case 3:
					smwRequest(t, r, http.MethodGet, smwCollection+"/actual")
					if r.Context().Err() != nil {
						t.Error("SDK budget canceled GET", r.Context().Err())
					}
					testcloud.JSON(w, 200, `{"snapshot":{"status":"deleted"}}`)
				default:
					t.Error("delete wait replay", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := smwDelete(t, client, "requested", blockstorage.WithDeleteVolumeSnapshotWait(true), blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(250 * time.Millisecond), PollInterval: smwDuration(-time.Second)}))
			if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Ready == nil || calls.Load() != 3 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}
