package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const bmwBase = "/proxy/cinder/v3/backup-mutation-project/"
const bmwCollection = bmwBase + "backups"
const bmwDetail = bmwCollection + "/detail"

func bmwClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := smwClient(cloud)
	client.ResourceBase = cloud.Server.URL + bmwBase
	return client
}
func bmwContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
func bmwRequest(t *testing.T, r *http.Request, method, path, version, token string) []byte {
	t.Helper()
	wantVersion := ""
	if version != "" {
		wantVersion = "volume " + version
	}
	if r.Method != method || r.URL.Path != path || r.Header.Get("X-Source") != "original" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != wantVersion {
		t.Error("backup request changed", r.Method, r.URL, r.Header, method, path, wantVersion, token)
	}
	var body []byte
	var err error
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
	}
	if err != nil || (method != http.MethodPost && len(body) != 0) {
		t.Error("bodyless request changed", string(body), err)
	}
	return body
}
func bmwPage(t *testing.T, p *blockstorage.VolumeBackupMutationPage, code int, body, proof string) {
	t.Helper()
	if p == nil || p.StatusCode != code || string(p.Body) != body || p.Header.Get("X-Proof") != proof {
		t.Fatal("backup physical phase", p, code, body, proof)
	}
}
func bmwOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var e *resource.OperationError
	if !errors.As(err, &e) || e.Operation != operation || e.Resource != "volume backup" {
		t.Fatal("backup operation", err, e)
	}
}
func bmwCreate(t *testing.T, c *gophercloud.ServiceClient, options ...blockstorage.CreateVolumeBackupOption) (*blockstorage.CreateVolumeBackupResult, error) {
	t.Helper()
	return blockstorage.CreateVolumeBackup(bmwContext(t), c, blockstorage.CreateVolumeBackupRequest{VolumeID: "literal/volume % identifier"}, options...)
}
func bmwDelete(t *testing.T, c *gophercloud.ServiceClient, options ...blockstorage.DeleteVolumeBackupOption) (*blockstorage.DeleteVolumeBackupResult, error) {
	t.Helper()
	return blockstorage.DeleteVolumeBackup(bmwContext(t), c, blockstorage.DeleteVolumeBackupRequest{NameOrID: "requested"}, options...)
}

func TestCreateVolumeBackupDefaultsSendSixLiteralFieldsWithoutResolvers(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	var calls atomic.Int32
	reply := `{"backup":{"status":"AvAiLaBlE","unknown":9007199254740993}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		outer := smwFields(t, bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token"))
		payload := smwFields(t, outer["backup"])
		expected := map[string]json.RawMessage{"name": json.RawMessage("null"), "volume_id": json.RawMessage(`"literal/volume % identifier"`), "description": json.RawMessage("null"), "force": json.RawMessage("false"), "incremental": json.RawMessage("false"), "snapshot_id": json.RawMessage("null")}
		if len(outer) != 1 || !reflect.DeepEqual(payload, expected) || r.URL.RawQuery != "" {
			t.Error(outer, payload, r.URL)
		}
		w.Header().Set("X-Proof", "created")
		testcloud.JSON(w, 201, reply)
	})
	result, err := bmwCreate(t, client)
	if err != nil || result == nil || result.CreatedBackup == nil || result.Backup == nil || result.ReadyBackup == nil || result.Ready == nil || string(result.BackupID) != "null" || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	view := smwFields(t, result.Value)
	if len(view) != 24 || string(view["id"]) != "null" || string(view["force"]) != "false" || string(view["is_incremental"]) != "false" || string(view["volume_id"]) != `"literal/volume % identifier"` || string(view["name"]) != "null" || string(view["description"]) != "null" || string(view["snapshot_id"]) != "null" {
		t.Fatal(string(result.Value))
	}
	if _, wrong := view["is_forced"]; wrong {
		t.Fatal("Snapshot descriptor leaked", view)
	}
	if _, wrong := view["incremental"]; wrong {
		t.Fatal("create wire alias entered logical view", view)
	}
	if _, fabricated := result.Backup.Body["volume_id"]; fabricated || string(result.Backup.Body["unknown"]) != "9007199254740993" {
		t.Fatal(result.Backup)
	}
	bmwPage(t, result.Created, 201, reply, "created")
	bmwPage(t, result.LastAccepted, 201, reply, "created")
}

func TestCreateVolumeBackupConcreteOptionsPreserveEmptyFalseAndLiteralSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		outer := smwFields(t, bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token"))
		payload := smwFields(t, outer["backup"])
		expected := map[string]json.RawMessage{"name": json.RawMessage(`""`), "volume_id": json.RawMessage(`""`), "description": json.RawMessage(`""`), "force": json.RawMessage("true"), "incremental": json.RawMessage("true"), "snapshot_id": json.RawMessage(`"literal/snapshot % value"`)}
		if len(outer) != 1 || !reflect.DeepEqual(payload, expected) || r.URL.RawQuery != "" {
			t.Error(outer, payload, r.URL)
		}
		testcloud.JSON(w, 399, `{"backup":{"id":false,"status":false}}`)
	})
	result, err := blockstorage.CreateVolumeBackup(bmwContext(t), client, blockstorage.CreateVolumeBackupRequest{}, blockstorage.WithCreateVolumeBackupName(""), blockstorage.WithCreateVolumeBackupDescription(""), blockstorage.WithCreateVolumeBackupForce(true), blockstorage.WithCreateVolumeBackupIncremental(true), blockstorage.WithCreateVolumeBackupSnapshotID("literal/snapshot % value"), blockstorage.WithCreateVolumeBackupWait(false), blockstorage.WithCreateVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{Timeout: smwDuration(-time.Second), PollInterval: smwDuration(-time.Second)}))
	if err != nil || result == nil || result.Value == nil || result.Ready != nil || result.ReadyBackup != nil || string(result.BackupID) != "false" || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	view := smwFields(t, result.Value)
	if string(view["name"]) != `""` || string(view["description"]) != `""` || string(view["is_incremental"]) != "true" || string(view["snapshot_id"]) != `"literal/snapshot % value"` {
		t.Fatal(string(result.Value))
	}
	for _, kind := range []string{"VolumeID", "Name", "Description", "SnapshotID", "Location"} {
		t.Run("invalid input/"+kind, func(t *testing.T) {
			invalidCloud := testcloud.New(t)
			invalidClient := bmwClient(invalidCloud)
			var requests atomic.Int32
			invalidCloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) })
			input := blockstorage.CreateVolumeBackupRequest{VolumeID: "literal"}
			var options []blockstorage.CreateVolumeBackupOption
			bad := string([]byte{0xff})
			switch kind {
			case "VolumeID":
				input.VolumeID = bad
			case "Name":
				options = append(options, blockstorage.WithCreateVolumeBackupName(bad))
			case "Description":
				options = append(options, blockstorage.WithCreateVolumeBackupDescription(bad))
			case "SnapshotID":
				options = append(options, blockstorage.WithCreateVolumeBackupSnapshotID(bad))
			case "Location":
				options = append(options, blockstorage.WithCreateVolumeBackupLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage{}}}))
			}
			rejected, err := blockstorage.CreateVolumeBackup(bmwContext(t), invalidClient, input, options...)
			var physical *resource.ResponseError
			if rejected != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || requests.Load() != 0 {
				t.Fatal(rejected, err, requests.Load())
			}
			bmwOperation(t, err, "CreateVolumeBackup")
		})
	}
}

func TestCreateVolumeBackupNoWaitPreservesArbitraryIdentityAndStatus(t *testing.T) {
	for _, row := range []string{`{}`, `{"id":null,"status":null}`, `{"id":false,"status":false}`, `{"id":9007199254740993,"status":[1]}`, `{"id":{"nested":[]},"status":"error"}`, `{"id":"bad/id","status":"creating"}`} {
		t.Run(row, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
				testcloud.JSON(w, 203, `{"backup":`+row+`}`)
			})
			result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWait(false))
			id := smwFields(t, json.RawMessage(row))["id"]
			if id == nil {
				id = json.RawMessage("null")
			}
			if err != nil || result == nil || result.Value == nil || result.Backup == nil || result.Ready != nil || result.ReadyBackup != nil || !bytes.Equal(result.BackupID, id) || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestCreateVolumeBackupMergesKnownResponseFieldsWithoutInventingPhysicalInputs(t *testing.T) {
	for _, reply := range []string{`{"backup":{"id":"created","size":"003","is_incremental":"false","unknown":true,"availability_zone":"literal zone","os-backup-project-attr:project_id":"project"}}`, `{"id":"flat","description":null}`, `{"backup":{"object_count":[]}}`, ``, `not JSON`, `{}`} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
				w.Header().Set("X-Proof", "merge")
				testcloud.JSON(w, 202, reply)
			})
			result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupName("prior name"), blockstorage.WithCreateVolumeBackupDescription("prior description"), blockstorage.WithCreateVolumeBackupWait(false))
			if err != nil || result == nil || result.Value == nil || result.CreatedValue == nil || result.Ready != nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			view := smwFields(t, result.Value)
			if string(view["name"]) != `"prior name"` || string(view["volume_id"]) != `"literal/volume % identifier"` || string(view["force"]) != "false" {
				t.Fatal(string(result.Value))
			}
			wantDesc := `"prior description"`
			if reply == `{"id":"flat","description":null}` {
				wantDesc = "null"
			}
			if string(view["description"]) != wantDesc {
				t.Fatal(string(result.Value))
			}
			if strings.Contains(reply, "size") {
				location := smwFields(t, view["location"])
				project := smwFields(t, location["project"])
				if string(view["size"]) != "3" || string(view["is_incremental"]) != "true" || string(location["zone"]) != `"literal zone"` || string(project["id"]) != `"project"` || string(result.Backup.Body["size"]) != `"003"` {
					t.Fatal(string(result.Value), result.Backup)
				}
			}
			if strings.Contains(reply, "object_count") {
				if string(view["object_count"]) != "0" || string(result.Backup.Body["object_count"]) != "[]" {
					t.Fatal("nonnumeric container descriptor must become zero without changing raw", string(result.Value), result.Backup)
				}
			}
			malformed := reply == "" || reply == "not JSON"
			if malformed != (result.CreatedBackup == nil) || malformed != (result.Backup == nil) {
				t.Fatal("physical object fabricated/discarded", result)
			}
			if result.Backup != nil {
				if _, invented := result.Backup.Body["name"]; invented {
					t.Fatal(result.Backup)
				}
			}
			bmwPage(t, result.Created, 202, reply, "merge")
			bmwPage(t, result.LastAccepted, 202, reply, "merge")
		})
	}
}

func TestCreateVolumeBackupRejectedShapesAndDescriptorsRetainCreationProof(t *testing.T) {
	for _, reply := range []string{`null`, `[]`, `{"backup":null}`, `{"backup":[]}`, `{"backup":{"size":"²"}}`, `{"backup":{"object_count":"²"}}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "invalid")
				testcloud.JSON(w, 203, reply)
			})
			result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupWait(false))
			var proof *resource.ResponseError
			if result == nil || err == nil || result.Value != nil || result.Backup != nil || result.CreatedValue != nil || result.Ready != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != reply || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if (strings.Contains(reply, "size") || strings.Contains(reply, "object_count")) && result.CreatedBackup == nil {
				t.Fatal("descriptor error discarded actual object", result)
			}
			bmwOperation(t, err, "CreateVolumeBackup")
			bmwPage(t, result.Created, 203, reply, "invalid")
			bmwPage(t, result.LastAccepted, 203, reply, "invalid")
			proof.Body[0] = '!'
			proof.Header.Set("X-Proof", "changed")
			bmwPage(t, result.Created, 203, reply, "invalid")
		})
	}
}

func TestCreateVolumeBackupOwnsCreatedFinalReadyAndRawIdentityIndependently(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	reply := `{"backup":{"id":"created","status":"available","metadata":{"n":9007199254740993}}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
		w.Header().Set("X-Proof", "owned")
		testcloud.JSON(w, 202, reply)
	})
	result, err := bmwCreate(t, client)
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	originalValue, originalReady, originalID := bytes.Clone(result.Value), bytes.Clone(result.Ready), bytes.Clone(result.BackupID)
	result.Created.Body[0] = '!'
	result.Created.Header.Set("X-Proof", "mutated")
	result.CreatedValue[0] = '!'
	result.CreatedBackup.Body["id"][0] = '!'
	result.CreatedBackup.Body["metadata"][0] = '!'
	result.CreatedBackup.Header.Set("X-Proof", "mutated")
	if !bytes.Equal(result.Value, originalValue) || !bytes.Equal(result.Ready, originalReady) || !bytes.Equal(result.BackupID, originalID) || string(result.Backup.Body["id"]) != `"created"` || string(result.ReadyBackup.Body["metadata"]) != `{"n":9007199254740993}` {
		t.Fatal("created phase aliases later phases", result)
	}
	bmwPage(t, result.LastAccepted, 202, reply, "owned")
	result.Value[0] = '!'
	result.Backup.Body["id"][0] = '!'
	result.Backup.Header.Set("X-Proof", "final mutated")
	if !bytes.Equal(result.Ready, originalReady) || string(result.ReadyBackup.Body["id"]) != `"created"` || result.ReadyBackup.Header.Get("X-Proof") != "owned" || !bytes.Equal(result.BackupID, originalID) {
		t.Fatal("ready phase aliases final", result)
	}
}

func TestDeleteVolumeBackupDefaultWaitOffUsesActualIDAndOpaqueAcknowledgement(t *testing.T) {
	for _, code := range []int{200, 203, 204, 399} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			ack := "\x00\xffopaque"
			if code == 204 {
				ack = ""
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					bmwRequest(t, r, "GET", bmwCollection+"/requested", "3.60", "test-token")
					w.Header().Set("X-Proof", "lookup")
					testcloud.JSON(w, 203, `{"backup":{"id":"actual","status":"deleted"}}`)
				case 2:
					bmwRequest(t, r, "DELETE", bmwCollection+"/actual", "3.60", "test-token")
					if r.URL.RawQuery != "" {
						t.Error("force/cascade query added", r.URL)
					}
					w.Header().Set("X-Proof", "ack")
					w.WriteHeader(code)
					_, _ = w.Write([]byte(ack))
				default:
					t.Error("default delete polled/repeated lookup", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{Timeout: smwDuration(-time.Second), PollInterval: smwDuration(-time.Second)}))
			if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Resolved == nil || result.Resolved.Backup == nil || result.Resolved.Observed == nil || len(result.Resolved.Pages) != 0 || string(result.BackupID) != `"actual"` || result.Ready != nil || result.Absent != nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			bmwPage(t, result.Applied, code, ack, "ack")
			bmwPage(t, result.LastAccepted, code, ack, "ack")
			result.Applied.Body = append(result.Applied.Body, '!')
			result.Applied.Header.Set("X-Proof", "changed")
			result.Resolved.Backup.Body["id"][0] = '!'
			result.Resolved.Observed.Body[0] = '!'
			bmwPage(t, result.LastAccepted, code, ack, "ack")
			if string(result.BackupID) != `"actual"` || string(smwFields(t, result.Resolved.Value)["id"]) != `"actual"` {
				t.Fatal("mutation aliases lookup", result)
			}
		})
	}
}

func TestDeleteVolumeBackupForceActionUsesNullAnd364WithoutChangingPollVersion(t *testing.T) {
	for _, version := range []string{"", "3.60", "3.80"} {
		for _, explicitHeaders := range []bool{false, true} {
			t.Run(version+"/explicit-headers-"+map[bool]string{true: "true", false: "false"}[explicitHeaders], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				client.Microversion = version
				if explicitHeaders && version != "" {
					client.MoreHeaders["OpenStack-API-Version"] = "volume " + version
					client.MoreHeaders["X-OpenStack-Volume-API-Version"] = version
				}
				originalHeaders := map[string]string{}
				for k, v := range client.MoreHeaders {
					originalHeaders[k] = v
				}
				var calls atomic.Int32
				ackCode := map[string]int{"": 204, "3.60": 302, "3.80": 399}[version]
				ack := string([]byte{0xff, 0, 'x'})
				if ackCode == 204 {
					ack = ""
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					switch calls.Add(1) {
					case 1:
						bmwRequest(t, r, "GET", bmwCollection+"/requested", version, "test-token")
						testcloud.JSON(w, 200, `{"backup":{"id":"actual","status":"deleted"}}`)
					case 2:
						raw := bmwRequest(t, r, "POST", bmwCollection+"/actual/action", "3.64", "test-token")
						if string(raw) != `{"os-force_delete":null}` || r.URL.RawQuery != "" || (r.Header.Get("X-OpenStack-Volume-API-Version") != "" && r.Header.Get("X-OpenStack-Volume-API-Version") != "3.64") {
							t.Error("force action policy", string(raw), r.URL, r.Header)
						}
						w.Header().Set("X-Proof", "force")
						w.WriteHeader(ackCode)
						_, _ = w.Write([]byte(ack))
					case 3:
						bmwRequest(t, r, "GET", bmwCollection+"/actual", version, "test-token")
						testcloud.JSON(w, 200, `{"backup":{"status":"DELETED"}}`)
					default:
						t.Error("force workflow replay", r.URL)
						w.WriteHeader(500)
					}
				})
				result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupForce(true), blockstorage.WithDeleteVolumeBackupWait(true), blockstorage.WithDeleteVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{PollInterval: smwDuration(-time.Second)}))
				if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.ReadyBackup == nil || result.Ready == nil || result.Absent != nil || calls.Load() != 3 || client.Microversion != version || !reflect.DeepEqual(client.MoreHeaders, originalHeaders) {
					t.Fatal(result, err, calls.Load(), client)
				}
				bmwPage(t, result.Applied, ackCode, ack, "force")
			})
		}
	}
}

func TestDeleteVolumeBackupEmptyInputRunsOriginalsWithoutServiceOrLocation(t *testing.T) {
	for _, kind := range []string{"nil client", "unusable client", "callback error", "canceled", "nil context"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			callbacks := 0
			cause := errors.New("empty backup original")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			var ctx context.Context = bmwContext(t)
			switch kind {
			case "nil client":
				client = nil
			case "unusable client":
				client.ProviderClient = nil
				client.ResourceBase = "bad"
			case "canceled":
				c, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = c
			case "nil context":
				ctx = nil
			}
			original := func(o *blockstorage.DeleteVolumeBackupOpts) error {
				callbacks++
				o.Location = &resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage{}}}
				o.Wait = smwBool(true)
				if kind == "callback error" {
					return cause
				}
				return nil
			}
			result, err := blockstorage.DeleteVolumeBackup(ctx, client, blockstorage.DeleteVolumeBackupRequest{}, original)
			if calls.Load() != 0 {
				t.Fatal("empty delete consumed service", calls.Load())
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
				t.Fatal(callbacks)
			}
			if kind == "callback error" {
				if result != nil || !errors.Is(err, cause) {
					t.Fatal(result, err)
				}
				return
			}
			if err != nil || result == nil || result.Deleted == nil || *result.Deleted || result.Resolved != nil || result.Applied != nil || result.LastAccepted != nil || result.Absent != nil || result.BackupID != nil {
				t.Fatal(result, err)
			}
		})
	}
}

func TestDeleteVolumeBackupLookupCompletionAndLogicalSeedControlMutation(t *testing.T) {
	for _, kind := range []string{"member missing ID", "member null ID", "fallback found", "missing", "ambiguous", "late list error"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if n == 1 {
					bmwRequest(t, r, "GET", bmwCollection+"/requested", "3.60", "test-token")
					if kind == "member missing ID" {
						testcloud.JSON(w, 200, `{"backup":{}}`)
					} else if kind == "member null ID" {
						testcloud.JSON(w, 200, `{"backup":{"id":null}}`)
					} else {
						testcloud.JSON(w, 404, `{"error":"fallback"}`)
					}
					return
				}
				if strings.HasPrefix(kind, "member") {
					bmwRequest(t, r, "DELETE", bmwCollection+"/requested", "3.60", "test-token")
					w.WriteHeader(204)
					return
				}
				if n == 2 {
					bmwRequest(t, r, "GET", bmwDetail, "3.60", "test-token")
					if r.URL.Query().Get("name") != "requested" {
						t.Error(r.URL)
					}
					rows := `[{"id":"actual","name":"requested"}]`
					if kind == "missing" {
						rows = `[]`
					}
					if kind == "ambiguous" {
						rows = `[{"id":"one","name":"requested"},{"id":"two","name":"requested"},{"size":"²"}]`
					}
					suffix := ``
					if kind == "fallback found" || kind == "late list error" {
						suffix = `,"backups_links":[{"rel":"next","href":"?marker=next"}]`
					}
					if kind == "ambiguous" {
						suffix = `,"backups_links":[null]`
					}
					testcloud.JSON(w, 200, `{"backups":`+rows+suffix+`}`)
					return
				}
				if n == 3 && (kind == "fallback found" || kind == "late list error") {
					bmwRequest(t, r, "GET", bmwDetail, "3.60", "test-token")
					if r.URL.Query().Get("marker") != "next" {
						t.Error(r.URL)
					}
					if kind == "late list error" {
						testcloud.JSON(w, 503, `{"error":"late list"}`)
					} else {
						testcloud.JSON(w, 200, `{"backups":[]}`)
					}
					return
				}
				if n == 4 && kind == "fallback found" {
					bmwRequest(t, r, "DELETE", bmwCollection+"/actual", "3.60", "test-token")
					w.WriteHeader(204)
					return
				}
				t.Error("lookup routed mutation before completion", kind, n, r.Method, r.URL)
				w.WriteHeader(500)
			})
			result, err := bmwDelete(t, client)
			if result == nil || result.Resolved == nil {
				t.Fatal(result, err)
			}
			switch kind {
			case "member missing ID":
				if err != nil || result.Deleted == nil || !*result.Deleted || !result.Resolved.SeededID || string(result.BackupID) != `"requested"` || calls.Load() != 2 {
					t.Fatal(result, err, calls.Load())
				}
				if _, invented := result.Resolved.Backup.Body["id"]; invented {
					t.Fatal("physical ID fabricated", result.Resolved)
				}
			case "member null ID":
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || result.Deleted != nil || result.Applied != nil || result.Resolved.Value == nil || result.Resolved.SeededID || calls.Load() != 1 {
					t.Fatal(result, err, calls.Load())
				}
			case "fallback found":
				if err != nil || result.Deleted == nil || !*result.Deleted || string(result.BackupID) != `"actual"` || len(result.Resolved.Pages) != 2 || result.Resolved.Observed != nil || calls.Load() != 4 {
					t.Fatal(result, err, calls.Load())
				}
			case "missing":
				if err != nil || result.Deleted == nil || *result.Deleted || result.Applied != nil || result.Resolved.Value != nil || calls.Load() != 2 {
					t.Fatal(result, err, calls.Load())
				}
			case "ambiguous":
				if !errors.Is(err, resource.ErrAmbiguous) || result.Deleted != nil || result.Applied != nil || calls.Load() != 2 {
					t.Fatal(result, err, calls.Load())
				}
			case "late list error":
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || result.Deleted != nil || result.Applied != nil || result.Resolved.Value != nil || result.Resolved.Observed != nil || len(result.Resolved.Pages) != 1 || calls.Load() != 3 {
					t.Fatal(result, err, calls.Load())
				}
			}
		})
	}
}

func TestDeleteVolumeBackupNormalAndForceMutation404NeverBecomeAbsence(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "force"}[force], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					testcloud.JSON(w, 200, `{"backup":{"id":"actual","status":"available"}}`)
				case 2:
					method, path, version := "DELETE", bmwCollection+"/actual", "3.60"
					if force {
						method, path, version = "POST", path+"/action", "3.64"
					}
					bmwRequest(t, r, method, path, version, "test-token")
					w.Header().Set("X-Proof", "rejected mutation")
					testcloud.JSON(w, 404, `{"error":"mutation race"}`)
				default:
					t.Error("mutation rejection polled", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupForce(force), blockstorage.WithDeleteVolumeBackupWait(true))
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			if result == nil || !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Proof") != "rejected mutation" || errors.As(err, &proof) || result.Deleted != nil || result.Resolved == nil || result.Resolved.Value == nil || result.Applied != nil || result.LastAccepted != nil || result.Absent != nil || result.Ready != nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			bmwOperation(t, err, "DeleteVolumeBackup")
		})
	}
}
