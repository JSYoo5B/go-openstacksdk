package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const deleteVolumeContractBase = "/reverse/cinder/v3/project/"
const deleteVolumeContractPath = deleteVolumeContractBase + "volumes/volume-1"
const deleteVolumeContractActionPath = deleteVolumeContractPath + "/action"
const deleteVolumeContractListPath = deleteVolumeContractBase + "volumes/detail"

func deleteVolumeContractClient(cloud *testcloud.Cloud, version string) *gophercloud.ServiceClient {
	client := cloud.Client("volumev3", "/unused/cinder/")
	client.ResourceBase = cloud.Server.URL + deleteVolumeContractBase
	client.Microversion = version
	client.MoreHeaders = map[string]string{"x-source": "entry"}
	return client
}
func deleteVolumeContractBody(statusJSON string) string {
	status := ""
	if statusJSON != "" {
		status = `,"status":` + statusJSON
	}
	return `{"volume":{"id":"volume-1","name":null` + status + `,"size":2,"metadata":{"number":9007199254740993},"attachments":[{"server_id":"server","device":null,"vendor":9007199254740993}],"vendor":{"number":9007199254740993}}}`
}
func deleteVolumeContractWire(t *testing.T, r *http.Request, method, path, query, version, token string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path || r.URL.RawQuery != query || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Auth-Token") != token {
		t.Errorf("method=%s URL=%s headers=%v", r.Method, r.URL, r.Header)
	}
	expected := ""
	if version != "" {
		expected = "volume " + version
	}
	if r.Header.Get("OpenStack-API-Version") != expected {
		t.Error("configured microversion not preserved", r.Header)
	}
	if method != http.MethodPost && r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Errorf("bodyless phase body=%q error=%v", body, err)
		}
	}
}
func deleteVolumeContractLocated(t *testing.T, result *blockstorage.DeleteVolumeResult, body string) {
	t.Helper()
	if result == nil || result.VolumeID != "volume-1" || !result.Found || result.Located == nil || result.Located.Volume == nil || result.Located.StatusCode != 200 || string(result.Located.Body) != body || result.Located.Header.Get("X-Proof") != "located" || result.Located.Volume.ID == nil || *result.Located.Volume.ID != "volume-1" {
		t.Fatalf("located volume proof missing: %+v", result)
	}
}
func deleteVolumeContractOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "DeleteVolume" || operation.Cause == nil {
		t.Fatalf("delete operation context missing: %v", err)
	}
}

func TestDeleteVolumeContractsDefaultFreshWaitAndIndependentPhaseEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	client := deleteVolumeContractClient(cloud, "3.23")
	located, ready := deleteVolumeContractBody(`"in-use"`), deleteVolumeContractBody(`"DeLeTeD"`)
	opaque := []byte{0, 0xff, 'x'}
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", "3.23", "test-token")
			w.Header().Set("X-Proof", "located")
			testcloud.JSON(w, 200, located)
		case 2:
			deleteVolumeContractWire(t, r, http.MethodDelete, deleteVolumeContractPath, "cascade=false&force=false", "3.23", "test-token")
			w.Header().Set("X-Proof", "deletion")
			w.Header().Set("Location", "https://foreign.invalid/ignored")
			w.WriteHeader(202)
			_, _ = w.Write(opaque)
		case 3:
			deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", "3.23", "test-token")
			w.Header().Set("X-Proof", "ready")
			testcloud.JSON(w, 200, ready)
		default:
			t.Error("cached terminal shortcut, reroute or second mutation", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")})
	deleteVolumeContractLocated(t, result, located)
	if err != nil || !result.Deleted || result.Deletion == nil || result.Deletion.StatusCode != 202 || !bytes.Equal(result.Deletion.Body, opaque) || result.Absent != nil || result.LastAccepted == nil || result.LastAccepted.Volume == nil || string(result.LastAccepted.Body) != ready || result.Ready == nil || result.Ready == result.LastAccepted.Volume || calls.Load() != 3 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	result.Located.Body[0] = '!'
	*result.Located.Volume.ID = "caller"
	result.Located.Header.Set("X-Proof", "caller")
	result.Ready.Header.Set("X-Proof", "caller")
	*result.Ready.Status = "caller"
	result.Ready.MetadataFields["number"][0] = '!'
	result.Ready.Body["vendor"][0] = '!'
	result.Ready.Attachments[0].Body["vendor"][0] = '!'
	if string(result.LastAccepted.Body) != ready || result.LastAccepted.Header.Get("X-Proof") != "ready" || result.LastAccepted.Volume.Header.Get("X-Proof") != "ready" || result.Located.Volume.Header.Get("X-Proof") != "located" || string(result.Located.Volume.Body["id"]) != `"volume-1"` || *result.LastAccepted.Volume.Status != "DeLeTeD" || string(result.LastAccepted.Volume.MetadataFields["number"]) != "9007199254740993" || string(result.LastAccepted.Volume.Body["vendor"]) != `{"number":9007199254740993}` || string(result.LastAccepted.Volume.Attachments[0].Body["vendor"]) != "9007199254740993" || !bytes.Equal(result.Deletion.Body, opaque) {
		t.Fatal("owned phase/model evidence aliases", result)
	}
}

func TestDeleteVolumeContractsInitialMissingHasFalseResultAndOnlyActualGET404Proof(t *testing.T) {
	for _, name := range []bool{false, true} {
		t.Run(fmt.Sprint(name), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.23")
			var calls atomic.Int32
			opaque := []byte{0, 0xff, 'x'}
			ref := resource.ID("volume-1")
			if name {
				ref = resource.Name("worker")
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				step := calls.Add(1)
				if name && step == 1 {
					deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractListPath, "name=worker", "3.23", "test-token")
					testcloud.JSON(w, 200, `{"volumes":[{"id":"volume-1","name":"worker"}]}`)
					return
				}
				deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", "3.23", "test-token")
				w.Header().Set("X-Proof", "initial-absent")
				w.WriteHeader(404)
				_, _ = w.Write(opaque)
			})
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: ref}, blockstorage.WithDeleteVolumeWait(false))
			wantCalls := int32(1)
			if name {
				wantCalls = 2
			}
			if err != nil || result == nil || result.VolumeID != "volume-1" || result.Found || result.Deleted || result.Deletion != nil || result.LastAccepted != nil || result.Ready != nil || result.Located == nil || result.Located.Volume != nil || result.Located.StatusCode != 404 || result.Absent == nil || result.Absent.StatusCode != 404 || !bytes.Equal(result.Absent.Body, opaque) || calls.Load() != wantCalls {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			result.Absent.Body[0] = '!'
			result.Absent.Header.Set("X-Proof", "caller")
			if !bytes.Equal(result.Located.Body, opaque) || result.Located.Header.Get("X-Proof") != "initial-absent" {
				t.Fatal("absence proof aliases located response", result)
			}
		})
	}
	t.Run("clean Name absence invents no HTTP 404", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := deleteVolumeContractClient(cloud, "")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractListPath, "name=worker", "", "test-token")
			testcloud.JSON(w, 200, `{"volumes":[]}`)
		})
		result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.Name("worker")})
		if err != nil || result == nil || result.VolumeID != "" || result.Found || result.Deleted || result.Located != nil || result.Deletion != nil || result.LastAccepted != nil || result.Absent != nil || result.Ready != nil || calls.Load() != 1 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
}

func TestDeleteVolumeContractsNameResolutionIsExactOnceAndErrorsNeverMutate(t *testing.T) {
	cloud := testcloud.New(t)
	client := deleteVolumeContractClient(cloud, "3.23")
	var calls atomic.Int32
	located := deleteVolumeContractBody(`null`)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractListPath, "name=worker", "3.23", "test-token")
			testcloud.JSON(w, 200, `{"volumes":[{"id":"worker","name":"worker-extra"}],"volumes_links":[{"rel":"next","href":"`+cloud.Server.URL+deleteVolumeContractListPath+`?name=worker&page=2"}]}`)
		case 2:
			deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractListPath, "name=worker&page=2", "3.23", "test-token")
			testcloud.JSON(w, 200, `{"volumes":[{"id":"volume-1","name":"worker"}]}`)
		case 3:
			deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", "3.23", "test-token")
			w.Header().Set("X-Proof", "located")
			testcloud.JSON(w, 200, located)
		case 4:
			deleteVolumeContractWire(t, r, http.MethodDelete, deleteVolumeContractPath, "cascade=false&force=false", "3.23", "test-token")
			w.WriteHeader(204)
		default:
			t.Error("Name re-resolved, ID heuristic, GET shortcut or mutation replay", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.Name("worker")}, blockstorage.WithDeleteVolumeWait(false))
	deleteVolumeContractLocated(t, result, located)
	if err != nil || !result.Deleted || calls.Load() != 4 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	for _, tc := range []struct {
		name, body string
		code       int
		cause      error
	}{{"ambiguous", `{"volumes":[{"id":"one","name":"worker"},{"id":"two","name":"worker"}]}`, 200, resource.ErrAmbiguous}, {"native list404", `{"error":"list missing"}`, 404, nil}, {"bad list", `{"volumes":false}`, 200, nil}, {"unsafe resolved ID", `{"volumes":[{"id":"../other","name":"worker"}]}`, 200, resource.ErrInvalidOption}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.23")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != deleteVolumeContractListPath {
					t.Error("name failure caused confirmation or deletion", r.URL)
				}
				testcloud.JSON(w, tc.code, tc.body)
			})
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.Name("worker")})
			if result == nil || err == nil || result.Found || result.Deleted || result.Located != nil || result.Deletion != nil || result.Absent != nil || calls.Load() != 1 || tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			if tc.code == 404 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 {
					t.Fatal("list404 became logical absence", err)
				}
			}
			deleteVolumeContractOperation(t, err)
		})
	}
	for _, tc := range []struct {
		name, body string
		code       int
		cause      error
	}{{"later page forbidden", `{"error":"later page forbidden"}`, 403, nil}, {"later page invalid", `{"volumes":false}`, 200, nil}, {"later duplicate", `{"volumes":[{"id":"volume-2","name":"worker"}]}`, 200, resource.ErrAmbiguous}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.23")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractListPath, "name=worker", "3.23", "test-token")
					testcloud.JSON(w, 200, `{"volumes":[{"id":"volume-1","name":"worker"}],"volumes_links":[{"rel":"next","href":"`+cloud.Server.URL+deleteVolumeContractListPath+`?name=worker&page=2"}]}`)
				case 2:
					deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractListPath, "name=worker&page=2", "3.23", "test-token")
					w.Header().Set("X-Proof", "later-page")
					testcloud.JSON(w, tc.code, tc.body)
				default:
					t.Error("earlier match bypassed later page failure or duplicate", r.Method, r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.Name("worker")})
			if result == nil || err == nil || result.VolumeID != "" || result.Found || result.Deleted || result.Located != nil || result.Deletion != nil || result.LastAccepted != nil || result.Absent != nil || result.Ready != nil || calls.Load() != 2 || tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			if tc.code == 403 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "later-page" || string(native.Body) != tc.body {
					t.Fatal("later page rejection lost its actual response", err)
				}
			}
			deleteVolumeContractOperation(t, err)
		})
	}
}

func TestDeleteVolumeContractsNoWaitAlwaysLooksUpAndMutation404StillReturnsTrue(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		code         int
	}{{"missing status", "", 202}, {"null status", `null`, 204}, {"error", `"error"`, 404}, {"error deleting", `"error_deleting"`, 202}, {"already deleted", `"deleted"`, 202}, {"in use", `"in-use"`, 204}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.23")
			var calls atomic.Int32
			located := deleteVolumeContractBody(tc.status)
			opaque := []byte{0, 0xff, 'x'}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", "3.23", "test-token")
					w.Header().Set("X-Proof", "located")
					testcloud.JSON(w, 200, located)
				case 2:
					deleteVolumeContractWire(t, r, http.MethodDelete, deleteVolumeContractPath, "cascade=false&force=false", "3.23", "test-token")
					w.Header().Set("X-Proof", "mutation")
					w.WriteHeader(tc.code)
					if tc.code != 204 {
						_, _ = w.Write(opaque)
					}
				default:
					t.Error("NoWait observed again or replayed deletion", r.Method, r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeWait(false))
			deleteVolumeContractLocated(t, result, located)
			if err != nil || !result.Deleted || result.Deletion == nil || result.Deletion.StatusCode != tc.code || result.LastAccepted != nil || result.Absent != nil || result.Ready != nil || calls.Load() != 2 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			if tc.code != 204 && !bytes.Equal(result.Deletion.Body, opaque) {
				t.Fatal("mutation acknowledgement should be opaque", result)
			}
		})
	}
}

func TestDeleteVolumeContractsFiniteMicroversionsChooseExactNormalOrForceWireContract(t *testing.T) {
	for _, tc := range []struct {
		version string
		force   bool
		code    int
		kind    string
	}{{"", false, 202, "volumev3"}, {"3.0", false, 204, "volumev3"}, {"3.22", false, 404, "volumev3"}, {"", true, 201, "volumev3"}, {"3.22", true, 202, "volumev3"}, {"3.22", true, 404, "volumev3"}, {"3.23", false, 202, "volumev3"}, {"3.23", true, 204, "volumev3"}, {"3.100", true, 404, "volumev3"}, {"3.23", true, 202, ""}} {
		t.Run(fmt.Sprintf("%s/%v/%d/%s", tc.version, tc.force, tc.code, tc.kind), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, tc.version)
			client.Type = tc.kind
			var calls atomic.Int32
			located := deleteVolumeContractBody(`"error_deleting"`)
			modern := tc.version == "3.23" || tc.version == "3.100"
			forceAction := tc.force && !modern
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", tc.version, "test-token")
					w.Header().Set("X-Proof", "located")
					testcloud.JSON(w, 200, located)
					return
				}
				method, path, query := http.MethodDelete, deleteVolumeContractPath, "cascade=false"
				if modern {
					query += "&force=" + fmt.Sprint(tc.force)
				}
				if forceAction {
					method, path, query = http.MethodPost, deleteVolumeContractActionPath, ""
				}
				deleteVolumeContractWire(t, r, method, path, query, tc.version, "test-token")
				if forceAction {
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 || string(body["os-force_delete"]) != "null" {
						t.Errorf("legacy Python null action=%v error=%v", body, err)
					}
				}
				w.Header().Set("X-Proof", "deletion")
				w.WriteHeader(tc.code)
				if tc.code != 204 {
					_, _ = w.Write([]byte{0, 0xff})
				}
			})
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeWait(false), blockstorage.WithDeleteVolumeForce(tc.force))
			deleteVolumeContractLocated(t, result, located)
			if err != nil || !result.Deleted || result.Deletion == nil || result.Deletion.StatusCode != tc.code || result.Absent != nil || calls.Load() != 2 || client.Type != tc.kind {
				t.Fatalf("result=%+v error=%v calls=%d sourceType=%q", result, err, calls.Load(), client.Type)
			}
		})
	}
}

func TestDeleteVolumeContractsWaitHasNoFailureStatesAndFresh404OwnsAbsenceProof(t *testing.T) {
	for _, terminal := range []string{"404", "deleted"} {
		t.Run(terminal, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.23")
			var calls, callbacks atomic.Int32
			located := deleteVolumeContractBody(`"deleted"`)
			states := []string{`null`, "", `"error"`, `"error_deleting"`, `"deleted-prefix"`}
			opaque := []byte{0, 0xff, 'x'}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				step := int(calls.Add(1))
				if step == 1 {
					w.Header().Set("X-Proof", "located")
					testcloud.JSON(w, 200, located)
					return
				}
				if step == 2 {
					deleteVolumeContractWire(t, r, http.MethodDelete, deleteVolumeContractPath, "cascade=false&force=false", "3.23", "test-token")
					w.Header().Set("X-Proof", "mutation-race404")
					w.WriteHeader(404)
					_, _ = w.Write(opaque)
					return
				}
				deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", "3.23", "test-token")
				index := step - 3
				if index < len(states) {
					testcloud.JSON(w, 200, deleteVolumeContractBody(states[index]))
					return
				}
				if index > len(states) {
					t.Error("completed wait made another request", r.URL)
					w.WriteHeader(500)
					return
				}
				w.Header().Set("X-Proof", "terminal")
				if terminal == "404" {
					w.WriteHeader(404)
					_, _ = w.Write(opaque)
				} else {
					testcloud.JSON(w, 200, deleteVolumeContractBody(`"DELETED"`))
				}
			})
			interval := time.Millisecond
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &interval, ProgressCallback: func(progress int) error {
				callbacks.Add(1)
				if progress != 0 {
					t.Error(progress)
				}
				return nil
			}}))
			deleteVolumeContractLocated(t, result, located)
			if err != nil || !result.Deleted || result.Deletion == nil || result.Deletion.StatusCode != 404 || result.LastAccepted == nil || calls.Load() != int32(3+len(states)) || callbacks.Load() != int32(len(states)) {
				t.Fatalf("result=%+v error=%v calls=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
			if terminal == "404" {
				if result.Ready != nil || result.LastAccepted.Volume != nil || result.LastAccepted.StatusCode != 404 || result.Absent == nil || !bytes.Equal(result.Absent.Body, opaque) || result.Absent.Header.Get("X-Proof") != "terminal" {
					t.Fatal(result)
				}
				result.Absent.Body[0] = '!'
				result.Absent.Header.Set("X-Proof", "caller")
				if !bytes.Equal(result.LastAccepted.Body, opaque) || result.LastAccepted.Header.Get("X-Proof") != "terminal" || result.Deletion.Header.Get("X-Proof") != "mutation-race404" {
					t.Fatal("absence stole another phase's proof", result)
				}
			} else if result.Ready == nil || *result.Ready.Status != "DELETED" || result.Absent != nil {
				t.Fatal(result)
			}
		})
	}
}

func TestDeleteVolumeContractsCanonicalIdentityAndAcceptedDecodeFailuresNeverRetarget(t *testing.T) {
	for _, phase := range []string{"initial", "poll"} {
		for _, tc := range []struct{ name, body string }{{"other ID", `{"volume":{"id":"other","status":"deleted"}}`}, {"unsafe ID", `{"volume":{"id":"../other","status":"deleted"}}`}, {"case alias ID", `{"volume":{"ID":"volume-1","status":"deleted"}}`}, {"wrong status type", `{"volume":{"id":"volume-1","status":true}}`}, {"malformed", `{"volume":`}} {
			t.Run(phase+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := deleteVolumeContractClient(cloud, "3.23")
				var calls atomic.Int32
				located := deleteVolumeContractBody(`"available"`)
				failStep := int32(1)
				if phase == "poll" {
					failStep = 3
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					step := calls.Add(1)
					if step == failStep {
						deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", "3.23", "test-token")
						w.Header().Set("X-Proof", "invalid")
						testcloud.JSON(w, 200, tc.body)
						return
					}
					if step == 1 {
						w.Header().Set("X-Proof", "located")
						testcloud.JSON(w, 200, located)
						return
					}
					if step == 2 {
						deleteVolumeContractWire(t, r, http.MethodDelete, deleteVolumeContractPath, "cascade=false&force=false", "3.23", "test-token")
						w.WriteHeader(202)
						return
					}
					t.Error("invalid identity caused new target or cleanup", r.Method, r.URL)
					w.WriteHeader(500)
				})
				result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")})
				var accepted *resource.ResponseError
				if result == nil || err == nil || result.VolumeID != "volume-1" || result.Deleted || result.Ready != nil || result.Absent != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != tc.body || calls.Load() != failStep {
					t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
				}
				if phase == "initial" {
					if result.Found || result.Located == nil || result.Located.Volume != nil || result.Deletion != nil || result.LastAccepted != nil {
						t.Fatal(result)
					}
				} else {
					deleteVolumeContractLocated(t, result, located)
					if result.Deletion == nil || result.LastAccepted == nil || string(result.LastAccepted.Body) != tc.body {
						t.Fatal(result)
					}
				}
				deleteVolumeContractOperation(t, err)
			})
		}
	}
}

func TestDeleteVolumeContractsRejectedMutationsReturnCurrentNativeProofAndNoFallback(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprint(force), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.22")
			var calls atomic.Int32
			located := deleteVolumeContractBody(`"in-use"`)
			code := 201
			if force {
				code = 204
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.Header().Set("X-Proof", "located")
					testcloud.JSON(w, 200, located)
					return
				}
				w.Header().Set("X-Proof", "rejected-current")
				testcloud.JSON(w, code, `{"error":"current mutation rejected"}`)
			})
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeForce(force))
			deleteVolumeContractLocated(t, result, located)
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			codes := []int{202, 204, 404}
			if force {
				codes = []int{201, 202, 404}
			}
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, codes) || native.ResponseHeader.Get("X-Proof") != "rejected-current" || errors.As(err, &accepted) || result.Deleted || result.Deletion != nil || result.LastAccepted != nil || result.Absent != nil || result.Ready != nil || calls.Load() != 2 {
				t.Fatalf("result=%+v native=%+v error=%v calls=%d", result, native, err, calls.Load())
			}
			deleteVolumeContractOperation(t, err)
		})
	}
}

func TestDeleteVolumeContractsPreflightRejectsInvalidMicroversionsSourceAndReferenceBeforeCallbacks(t *testing.T) {
	for _, version := range []string{"latest", "3", "3.", "3.01", "3.023", "03.23", "2.23", "4.0", "3.-1", "3.+23", "3.２３", "3.٢٣", "3.23 ", "3.23.0", "3.9999999999999999999999999999999999999999"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, version)
			calls := 0
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, func(*blockstorage.DeleteVolumeOpts) error { calls++; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("version=%q result=%+v error=%v callbacks=%d", version, result, err, calls)
			}
		})
	}
	for _, tc := range []struct {
		name                               string
		change                             func(*gophercloud.ServiceClient)
		ref                                resource.Ref
		nilContext, nilClient, unsupported bool
	}{{name: "nil context", nilContext: true}, {name: "nil client", nilClient: true}, {name: "missing provider", change: func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }}, {name: "wrong service", unsupported: true, change: func(c *gophercloud.ServiceClient) { c.Type = "image" }}, {name: "foreign effective base", change: func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://foreign.invalid/v3/" }}, {name: "body framing", change: func(c *gophercloud.ServiceClient) { c.MoreHeaders["Content-Length"] = "1" }}, {name: "token override", change: func(c *gophercloud.ServiceClient) { c.MoreHeaders["X-Auth-Token"] = "other" }}, {name: "unsafe ID", ref: resource.ID("../other")}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.23")
			if tc.change != nil {
				tc.change(client)
			}
			if tc.nilClient {
				client = nil
			}
			ctx := context.Background()
			if tc.nilContext {
				ctx = nil
			}
			ref := resource.ID("volume-1")
			if tc.ref.String() != "" {
				ref = tc.ref
			}
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("invalid deletion reached HTTP", r.URL)
				w.WriteHeader(500)
			})
			result, err := blockstorage.DeleteVolume(ctx, client, blockstorage.DeleteVolumeRequest{Volume: ref}, func(*blockstorage.DeleteVolumeOpts) error { callbacks.Add(1); return nil })
			want := resource.ErrInvalidOption
			if tc.unsupported {
				want = resource.ErrUnsupported
			}
			if result != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatalf("result=%+v error=%v calls=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
		})
	}
	t.Run("canceled entry keeps custom cause", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := deleteVolumeContractClient(cloud, "3.23")
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("canceled before delete")
		cancel(cause)
		calls := 0
		result, err := blockstorage.DeleteVolume(ctx, client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, func(*blockstorage.DeleteVolumeOpts) error { calls++; return nil })
		if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 0 {
			t.Fatal(result, err, calls)
		}
	})
}
