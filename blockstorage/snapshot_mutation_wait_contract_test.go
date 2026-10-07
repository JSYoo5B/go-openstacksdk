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

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestCreateVolumeSnapshotFreshPollOverridesInitialErrorBeforeFailure(t *testing.T) {
	for _, initial := range []string{`"error"`, `null`, `"creating"`} {
		t.Run(initial, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			created := `{"snapshot":{"id":"first","status":` + initial + `,"name":"kept","metadata":{"n":9007199254740993},"size":"003"}}`
			fresh := `{"snapshot":{"status":"AVAILABLE","description":"fresh","unknown":true}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					smwRequest(t, r, http.MethodPost, smwCollection)
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 202, created)
				case 2:
					smwRequest(t, r, http.MethodGet, smwCollection+"/first")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 203, fresh)
				default:
					t.Error("workflow replay", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(-time.Second)}))
			if err != nil || result == nil || result.Ready == nil || result.ReadySnapshot == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			value := smwFields(t, result.Value)
			if string(value["name"]) != `"kept"` || string(value["size"]) != "3" || string(value["metadata"]) != `{"n":9007199254740993}` || string(value["description"]) != `"fresh"` || string(value["status"]) != `"AVAILABLE"` || string(result.SnapshotID) != `"first"` {
				t.Fatal(string(result.Value))
			}
			if _, fabricated := result.Snapshot.Body["id"]; fabricated {
				t.Fatal("prior ID fabricated in fresh object")
			}
			if _, fabricated := result.Snapshot.Body["metadata"]; fabricated {
				t.Fatal("prior metadata fabricated in fresh object")
			}
			smwPage(t, result.Created, 202, created, "created")
			smwPage(t, result.LastAccepted, 203, fresh, "fresh")
			original := bytes.Clone(result.Value)
			result.CreatedValue[0] = '!'
			result.CreatedSnapshot.Body["id"][0] = '!'
			result.LastAccepted.Body[0] = '!'
			result.LastAccepted.Header.Set("X-Proof", "changed")
			result.Snapshot.Body["status"][0] = '!'
			if !bytes.Equal(result.Value, original) || string(result.ReadySnapshot.Body["status"]) != `"AVAILABLE"` || result.ReadySnapshot.Header.Get("X-Proof") != "fresh" {
				t.Fatal("merged/fresh phase ownership", result)
			}
		})
	}
}

func TestCreateVolumeSnapshotFreshErrorOrInvalidStatusKeepsCreatedAndLastAccepted(t *testing.T) {
	for _, freshStatus := range []string{`"ErRoR"`, `false`, `[]`} {
		t.Run(freshStatus, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			created := `{"snapshot":{"id":"first","status":"error"}}`
			fresh := `{"snapshot":{"status":` + freshStatus + `}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 202, created)
				} else {
					smwRequest(t, r, http.MethodGet, smwCollection+"/first")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 200, fresh)
				}
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(-time.Second)}))
			if result == nil || err == nil || result.Value != nil || result.Snapshot != nil || result.Ready != nil || result.CreatedValue == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			smwPage(t, result.Created, 202, created, "created")
			smwPage(t, result.LastAccepted, 200, fresh, "fresh")
			var failed *resource.FailedStateError
			var physical *resource.ResponseError
			if freshStatus == `"ErRoR"` {
				if !errors.As(err, &failed) || failed.ID != "first" || failed.Status != "ErRoR" {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if errors.As(err, &physical) {
				t.Fatal("local wait-state error borrowed HTTP proof", err)
			}
		})
	}
}

func TestCreateVolumeSnapshotInitialNonstringStatusIsConsumedOnlyWhenWaiting(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	var calls atomic.Int32
	body := `{"snapshot":{"id":false,"status":42}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Proof", "created")
		testcloud.JSON(w, 202, body)
	})
	result, err := smwCreate(t, client)
	var proof *resource.ResponseError
	if result == nil || !errors.Is(err, resource.ErrInvalidOption) || result.CreatedValue == nil || result.CreatedSnapshot == nil || result.Value != nil || result.Ready != nil || errors.As(err, &proof) || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	smwPage(t, result.Created, 202, body, "created")
	smwPage(t, result.LastAccepted, 202, body, "created")
}

func TestCreateVolumeSnapshotWaitRebindsLatestLogicalIDButDoesNotRouteTerminalUnsafeID(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	var calls atomic.Int32
	created := `{"snapshot":{"id":"first","status":"creating","name":"prior"}}`
	first := `{"snapshot":{"id":"second","progress":false,"status":null,"size":"03"}}`
	last := `{"snapshot":{"id":"bad/id","status":"AVAILABLE"}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			smwRequest(t, r, http.MethodPost, smwCollection)
			w.Header().Set("X-Proof", "created")
			testcloud.JSON(w, 202, created)
		case 2:
			smwRequest(t, r, http.MethodGet, smwCollection+"/first")
			w.Header().Set("X-Proof", "first")
			testcloud.JSON(w, 200, first)
		case 3:
			smwRequest(t, r, http.MethodGet, smwCollection+"/second")
			w.Header().Set("X-Proof", "terminal")
			testcloud.JSON(w, 200, last)
		default:
			t.Error("unused unsafe ID routed", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(time.Millisecond)}))
	if err != nil || result == nil || calls.Load() != 3 || string(result.SnapshotID) != `"bad/id"` {
		t.Fatal(result, err, calls.Load())
	}
	value := smwFields(t, result.Value)
	if string(value["name"]) != `"prior"` || string(value["size"]) != "3" || string(value["progress"]) != "false" {
		t.Fatal(string(result.Value))
	}
	smwPage(t, result.LastAccepted, 200, last, "terminal")
}

func TestCreateVolumeSnapshotNonterminalUnsafeIDFailsBeforeAnotherPhysicalPoll(t *testing.T) {
	for _, id := range []string{`null`, `false`, `"bad/id"`, `"%2f"`, `"white space"`} {
		t.Run(id, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			created := `{"snapshot":{"id":"first","status":"creating"}}`
			fresh := `{"snapshot":{"id":` + id + `,"status":"creating"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 202, created)
				} else {
					smwRequest(t, r, http.MethodGet, smwCollection+"/first")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 200, fresh)
				}
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(time.Millisecond)}))
			var proof *resource.ResponseError
			if result == nil || !errors.Is(err, resource.ErrInvalidOption) || result.Value != nil || result.CreatedValue == nil || string(result.SnapshotID) != id || calls.Load() != 2 || errors.As(err, &proof) {
				t.Fatal(result, err, calls.Load())
			}
			smwPage(t, result.LastAccepted, 200, fresh, "fresh")
		})
	}
}

func TestCreateVolumeSnapshotExplicitNonpositiveTimeoutIsPostCreateAndPrePoll(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			body := `{"snapshot":{"id":false,"status":"creating"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				smwRequest(t, r, http.MethodPost, smwCollection)
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 202, body)
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(timeout), PollInterval: smwDuration(-time.Second)}))
			var expired *blockstorage.SnapshotWaitTimeoutError
			if result == nil || !errors.As(err, &expired) || expired.Timeout != timeout || !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, resource.ErrInvalidOption) || result.CreatedValue == nil || result.Value != nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			smwPage(t, result.Created, 202, body, "created")
			smwPage(t, result.LastAccepted, 202, body, "created")
		})
	}
}

func TestCreateVolumeSnapshotPositiveWaitBudgetBeginsAfterCreationAndDoesNotAbortSlowTargetGET(t *testing.T) {
	for _, slowPhase := range []string{"POST", "GET"} {
		t.Run(slowPhase, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if (n == 1 && slowPhase == "POST") || (n == 2 && slowPhase == "GET") {
					time.Sleep(350 * time.Millisecond)
				}
				if n == 1 {
					smwRequest(t, r, http.MethodPost, smwCollection)
					testcloud.JSON(w, 202, `{"snapshot":{"id":"first","status":"creating"}}`)
				} else if n == 2 {
					smwRequest(t, r, http.MethodGet, smwCollection+"/first")
					if r.Context().Err() != nil {
						t.Error("wait budget canceled HTTP", r.Context().Err())
					}
					testcloud.JSON(w, 200, `{"snapshot":{"status":"available"}}`)
				} else {
					t.Error("replayed wait", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(250 * time.Millisecond), PollInterval: smwDuration(-time.Second)}))
			if err != nil || result == nil || result.Ready == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestCreateVolumeSnapshotPositiveTimeoutIsNoticedAtNextLoopBoundary(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	var calls atomic.Int32
	fresh := `{"snapshot":{"status":"creating","progress":87}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			testcloud.JSON(w, 202, `{"snapshot":{"id":"first","status":"creating"}}`)
		case 2:
			smwRequest(t, r, http.MethodGet, smwCollection+"/first")
			w.Header().Set("X-Proof", "nonterminal")
			testcloud.JSON(w, 200, fresh)
		default:
			t.Error("expired loop made another GET", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(250 * time.Millisecond), PollInterval: smwDuration(350 * time.Millisecond)}))
	var expired *blockstorage.SnapshotWaitTimeoutError
	if result == nil || !errors.As(err, &expired) || result.Value != nil || result.Ready != nil || result.CreatedValue == nil || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
	smwPage(t, result.LastAccepted, 200, fresh, "nonterminal")
}

func TestCreateVolumeSnapshotPollIntervalsAreConsumedOnlyAtNonterminalSleep(t *testing.T) {
	for _, status := range []string{"available", "error", "creating"} {
		t.Run(status, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			fresh := `{"snapshot":{"status":"` + status + `"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 202, `{"snapshot":{"id":"first","status":"creating"}}`)
				} else {
					smwRequest(t, r, http.MethodGet, smwCollection+"/first")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 200, fresh)
				}
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(-time.Second)}))
			if result == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			switch status {
			case "available":
				if err != nil || result.Value == nil {
					t.Fatal(result, err)
				}
			case "error":
				var failed *resource.FailedStateError
				if !errors.As(err, &failed) || errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "creating":
				if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "interval") {
					t.Fatal(err)
				}
			}
			smwPage(t, result.LastAccepted, 200, fresh, "fresh")
		})
	}
	t.Run("zero normalizes to a pause", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := smwClient(cloud)
		var calls atomic.Int32
		previous := make(chan time.Time, 1)
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				testcloud.JSON(w, 202, `{"snapshot":{"id":"first","status":"creating"}}`)
			case 2:
				smwRequest(t, r, http.MethodGet, smwCollection+"/first")
				previous <- time.Now()
				testcloud.JSON(w, 200, `{"snapshot":{"status":"creating"}}`)
			case 3:
				smwRequest(t, r, http.MethodGet, smwCollection+"/first")
				at := <-previous
				if time.Since(at) < 75*time.Millisecond {
					t.Error("explicit zero caused an immediate polling loop", time.Since(at))
				}
				testcloud.JSON(w, 200, `{"snapshot":{"status":"available"}}`)
			default:
				t.Error("zero interval replay", r.URL)
				w.WriteHeader(500)
			}
		})
		result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{PollInterval: smwDuration(0)}))
		if err != nil || result == nil || result.Ready == nil || calls.Load() != 3 {
			t.Fatal(result, err, calls.Load())
		}
	})
}

func TestCreateVolumeSnapshotToleratedPollBodiesMergePriorStateBeforeTerminalCheck(t *testing.T) {
	for _, reply := range []string{"", `not JSON`, `{}`, `{"snapshot":{"description":"partial"}}`} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 202, `{"snapshot":{"id":"first","status":"error","description":"prior"}}`)
				} else {
					smwRequest(t, r, http.MethodGet, smwCollection+"/first")
					w.Header().Set("X-Proof", "fresh")
					testcloud.JSON(w, 200, reply)
				}
			})
			result, err := smwCreate(t, client)
			var failed *resource.FailedStateError
			if result == nil || !errors.As(err, &failed) || failed.Status != "error" || result.Value != nil || result.CreatedValue == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			smwPage(t, result.LastAccepted, 200, reply, "fresh")
		})
	}
}
