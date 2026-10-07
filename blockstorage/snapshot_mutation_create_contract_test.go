package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const smwBase = "/proxy/cinder/v3/snapshot-mutation-project/"
const smwCollection = smwBase + "snapshots"
const smwDetail = smwCollection + "/detail"

func smwClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("volumev3", "/catalog/cinder/")
	client.ResourceBase = cloud.Server.URL + smwBase
	client.Microversion = "3.60"
	client.MoreHeaders = map[string]string{"x-source": "original"}
	return client
}
func smwRequest(t *testing.T, r *http.Request, method, path string) []byte {
	t.Helper()
	if r.Method != method || r.URL.Path != path || r.Header.Get("X-Source") != "original" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "volume 3.60" {
		t.Error("captured snapshot request changed", r.Method, r.URL, r.Header)
	}
	var body []byte
	var err error
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
	}
	if err != nil || (method != http.MethodPost && len(body) != 0) {
		t.Error("request body", string(body), err)
	}
	return body
}
func smwFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		t.Fatal("expected object", string(raw), err)
	}
	return fields
}
func smwOperation(t *testing.T, err error, name string) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != name || operation.Resource != "volume snapshot" {
		t.Fatal("operation context", err)
	}
}
func smwDuration(value time.Duration) *time.Duration { return &value }
func smwBool(value bool) *bool                       { return &value }
func smwCreate(t *testing.T, client *gophercloud.ServiceClient, options ...blockstorage.CreateVolumeSnapshotOption) (*blockstorage.CreateVolumeSnapshotResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return blockstorage.CreateVolumeSnapshot(ctx, client, blockstorage.CreateVolumeSnapshotRequest{VolumeID: "literal/volume % identifier"}, options...)
}
func smwPage(t *testing.T, page *blockstorage.VolumeSnapshotMutationPage, code int, body, proof string) {
	t.Helper()
	if page == nil || page.StatusCode != code || string(page.Body) != body || page.Header.Get("X-Proof") != proof {
		t.Fatal("physical phase", page, code, body, proof)
	}
}

func TestCreateVolumeSnapshotDefaultsForwardLiteralBodyAndPreserveRequestSeed(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	var calls atomic.Int32
	body := `{"snapshot":{"status":"AvAiLaBlE","unknown":9007199254740993}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		wire := smwFields(t, smwRequest(t, r, http.MethodPost, smwCollection))
		payload := smwFields(t, wire["snapshot"])
		if len(wire) != 1 || len(payload) != 2 || string(payload["force"]) != "false" || string(payload["volume_id"]) != `"literal/volume % identifier"` || r.URL.RawQuery != "" {
			t.Error(string(wire["snapshot"]), r.URL)
		}
		w.Header().Set("X-Proof", "created")
		testcloud.JSON(w, 201, body)
	})
	result, err := smwCreate(t, client)
	if err != nil || result == nil || result.CreatedSnapshot == nil || result.Snapshot == nil || result.ReadySnapshot == nil || calls.Load() != 1 || string(result.SnapshotID) != "null" {
		t.Fatal(result, err, calls.Load())
	}
	view := smwFields(t, result.Value)
	if len(view) != 16 || string(view["volume_id"]) != `"literal/volume % identifier"` || string(view["is_forced"]) != "false" || string(view["id"]) != "null" || string(view["status"]) != `"AvAiLaBlE"` || string(view["metadata"]) != "null" {
		t.Fatal(string(result.Value))
	}
	if _, present := view["unknown"]; present || string(result.Snapshot.Body["unknown"]) != "9007199254740993" {
		t.Fatal(view, result.Snapshot)
	}
	if _, fabricated := result.Snapshot.Body["volume_id"]; fabricated {
		t.Fatal("request seed entered physical object")
	}
	smwPage(t, result.Created, 201, body, "created")
	smwPage(t, result.LastAccepted, 201, body, "created")
}

func TestCreateVolumeSnapshotCachedReadySkipsUnusedWaitAndRouteControls(t *testing.T) {
	for _, row := range []string{`{"status":"available"}`, `{"id":null,"status":"AVAILABLE"}`, `{"id":false,"status":"available"}`, `{"id":"bad/id","status":"available"}`} {
		t.Run(row, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				smwRequest(t, r, http.MethodPost, smwCollection)
				testcloud.JSON(w, 202, `{"snapshot":`+row+`}`)
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(0), PollInterval: smwDuration(-time.Second)}))
			if err != nil || result == nil || result.Value == nil || result.Ready == nil || result.ReadySnapshot == nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			fields := smwFields(t, json.RawMessage(row))
			expected := fields["id"]
			if expected == nil {
				expected = json.RawMessage("null")
			}
			if !bytes.Equal(result.SnapshotID, expected) {
				t.Fatal(string(result.SnapshotID), string(expected))
			}
		})
	}
}

func TestCreateVolumeSnapshotNoWaitKeepsArbitraryNullableIdentityAndStatus(t *testing.T) {
	for _, row := range []string{`{}`, `{"id":null,"status":null}`, `{"id":false,"status":false}`, `{"id":9007199254740993,"status":[1]}`, `{"id":{"nested":[]},"status":"error"}`, `{"id":"bad/id","status":"creating"}`} {
		t.Run(row, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				smwRequest(t, r, http.MethodPost, smwCollection)
				testcloud.JSON(w, 203, `{"snapshot":`+row+`}`)
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWait(false), blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: smwDuration(-time.Second), PollInterval: smwDuration(-time.Second)}))
			if err != nil || result == nil || result.Value == nil || result.Snapshot == nil || result.Ready != nil || result.ReadySnapshot != nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			fields := smwFields(t, json.RawMessage(row))
			id := fields["id"]
			if id == nil {
				id = json.RawMessage("null")
			}
			if !bytes.Equal(result.SnapshotID, id) {
				t.Fatal(string(result.SnapshotID), string(id))
			}
		})
	}
}

func TestCreateVolumeSnapshotAliasesUseCanonicalPresenceAndLiteralJSONValues(t *testing.T) {
	cases := []struct {
		name     string
		fields   map[string]json.RawMessage
		expected map[string]json.RawMessage
	}{
		{"canonical null suppresses display", map[string]json.RawMessage{"name": nil, "display_name": json.RawMessage(`"unused"`), "description": json.RawMessage("false"), "display_description": json.RawMessage(`"unused"`)}, map[string]json.RawMessage{}},
		{"canonical falsey suppresses display", map[string]json.RawMessage{"name": json.RawMessage("[]"), "display_name": json.RawMessage(`"unused"`), "description": json.RawMessage("{}"), "display_description": json.RawMessage(`"unused"`)}, map[string]json.RawMessage{}},
		{"truthy raw values", map[string]json.RawMessage{"display_name": json.RawMessage(`[1,false]`), "display_description": json.RawMessage(`{"n":9007199254740993}`)}, map[string]json.RawMessage{"name": json.RawMessage(`[1,false]`), "description": json.RawMessage(`{"n":9007199254740993}`)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				wire := smwFields(t, smwRequest(t, r, http.MethodPost, smwCollection))
				payload := smwFields(t, wire["snapshot"])
				expected := map[string]json.RawMessage{"volume_id": json.RawMessage(`""`), "force": json.RawMessage("true")}
				for k, v := range tc.expected {
					expected[k] = v
				}
				if !reflect.DeepEqual(payload, expected) {
					t.Error(payload, expected)
				}
				testcloud.JSON(w, 399, `{"snapshot":{"status":"available"}}`)
			})
			result, err := blockstorage.CreateVolumeSnapshot(context.Background(), client, blockstorage.CreateVolumeSnapshotRequest{}, blockstorage.WithCreateVolumeSnapshotForce(true), blockstorage.WithCreateVolumeSnapshotFields(tc.fields))
			if err != nil || result == nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			view := smwFields(t, result.Value)
			for _, k := range []string{"name", "description"} {
				expected := tc.expected[k]
				if expected == nil {
					expected = json.RawMessage("null")
				}
				if !bytes.Equal(view[k], expected) {
					t.Fatal(k, string(view[k]), string(expected))
				}
			}
		})
	}
}

func TestCreateVolumeSnapshotValidatesConsumedInputBeforePOSTButIgnoresShadowedAlias(t *testing.T) {
	for _, kind := range []string{"unknown metadata", "empty raw", "invalid utf8 raw", "invalid utf8 volume", "invalid location", "shadowed invalid alias"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 202, `{"snapshot":{"status":"available"}}`)
			})
			input := blockstorage.CreateVolumeSnapshotRequest{VolumeID: "volume"}
			var options []blockstorage.CreateVolumeSnapshotOption
			switch kind {
			case "unknown metadata":
				options = append(options, blockstorage.WithCreateVolumeSnapshotFields(map[string]json.RawMessage{"metadata": json.RawMessage("{}")}))
			case "empty raw":
				options = append(options, blockstorage.WithCreateVolumeSnapshotFields(map[string]json.RawMessage{"name": json.RawMessage{}}))
			case "invalid utf8 raw":
				options = append(options, blockstorage.WithCreateVolumeSnapshotFields(map[string]json.RawMessage{"name": json.RawMessage{'"', 0xff, '"'}}))
			case "invalid utf8 volume":
				input.VolumeID = string([]byte{0xff})
			case "invalid location":
				options = append(options, blockstorage.WithCreateVolumeSnapshotLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage{}}}))
			case "shadowed invalid alias":
				options = append(options, blockstorage.WithCreateVolumeSnapshotFields(map[string]json.RawMessage{"name": json.RawMessage("null"), "display_name": json.RawMessage{}}))
			}
			result, err := blockstorage.CreateVolumeSnapshot(context.Background(), client, input, options...)
			if kind == "shadowed invalid alias" {
				if err != nil || result == nil || calls.Load() != 1 {
					t.Fatal(result, err, calls.Load())
				}
				return
			}
			var proof *resource.ResponseError
			if result != nil || calls.Load() != 0 || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) {
				t.Fatal(result, err, calls.Load())
			}
			smwOperation(t, err, "CreateVolumeSnapshot")
		})
	}
}

func TestCreateVolumeSnapshotMergesPartialFlatEmptyAndMalformedCreationResponses(t *testing.T) {
	for _, reply := range []string{`{"snapshot":{"id":"created","size":"003","unknown":true}}`, `{"id":"flat","description":null}`, ``, `not JSON`, `{}`} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				smwRequest(t, r, http.MethodPost, smwCollection)
				w.Header().Set("X-Proof", "merge")
				testcloud.JSON(w, 202, reply)
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWait(false), blockstorage.WithCreateVolumeSnapshotName("prior name"), blockstorage.WithCreateVolumeSnapshotDescription("prior description"))
			if err != nil || result == nil || result.Value == nil || result.CreatedValue == nil || calls.Load() != 1 || result.Ready != nil {
				t.Fatal(result, err, calls.Load())
			}
			view := smwFields(t, result.Value)
			if string(view["name"]) != `"prior name"` || string(view["volume_id"]) != `"literal/volume % identifier"` || string(view["is_forced"]) != "false" {
				t.Fatal(string(result.Value))
			}
			if reply == `{"id":"flat","description":null}` {
				if string(view["description"]) != "null" {
					t.Fatal(string(result.Value))
				}
			} else if string(view["description"]) != `"prior description"` {
				t.Fatal(string(result.Value))
			}
			if reply == `{"snapshot":{"id":"created","size":"003","unknown":true}}` {
				if string(view["size"]) != "3" || string(result.Snapshot.Body["size"]) != `"003"` {
					t.Fatal(result)
				}
			}
			malformed := reply == "" || reply == "not JSON"
			if malformed != (result.CreatedSnapshot == nil) || malformed != (result.Snapshot == nil) {
				t.Fatal("physical object invented or discarded", result)
			}
			if result.Snapshot != nil {
				if _, present := result.Snapshot.Body["name"]; present {
					t.Fatal("merged name entered physical object")
				}
			}
			smwPage(t, result.Created, 202, reply, "merge")
			smwPage(t, result.LastAccepted, 202, reply, "merge")
		})
	}
}

func TestCreateVolumeSnapshotAcceptedWrongShapesAndDescriptorsKeepOnlyCreationProof(t *testing.T) {
	for _, reply := range []string{`null`, `[]`, `{"snapshot":null}`, `{"snapshot":[]}`, `{"snapshot":{"size":"²"}}`, `{"snapshot":{"force":"invalid"}}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := smwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "invalid")
				testcloud.JSON(w, 203, reply)
			})
			result, err := smwCreate(t, client, blockstorage.WithCreateVolumeSnapshotWait(false))
			var proof *resource.ResponseError
			if result == nil || err == nil || result.Value != nil || result.Snapshot != nil || result.CreatedValue != nil || result.Ready != nil || calls.Load() != 1 || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != reply {
				t.Fatal(result, err, calls.Load())
			}
			smwPage(t, result.Created, 203, reply, "invalid")
			smwPage(t, result.LastAccepted, 203, reply, "invalid")
			smwOperation(t, err, "CreateVolumeSnapshot")
			if bytes.Contains([]byte(reply), []byte("size")) || bytes.Contains([]byte(reply), []byte("force")) {
				if result.CreatedSnapshot == nil {
					t.Fatal("actual object lost after descriptor failure")
				}
			}
			proof.Body[0] = '!'
			proof.Header.Set("X-Proof", "changed")
			smwPage(t, result.Created, 203, reply, "invalid")
		})
	}
}

func TestCreateVolumeSnapshotOwnsEveryLogicalAndPhysicalPhaseIndependently(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	body := `{"snapshot":{"id":"created","status":"available","metadata":{"n":9007199254740993}}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Proof", "owned")
		testcloud.JSON(w, 202, body)
	})
	result, err := smwCreate(t, client)
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	originalValue := bytes.Clone(result.Value)
	originalReady := bytes.Clone(result.Ready)
	originalID := bytes.Clone(result.SnapshotID)
	result.Created.Body[0] = '!'
	result.Created.Header.Set("X-Proof", "mutated")
	result.CreatedValue[0] = '!'
	result.CreatedSnapshot.Body["id"][0] = '!'
	result.CreatedSnapshot.Body["metadata"][0] = '!'
	result.CreatedSnapshot.Header.Set("X-Proof", "mutated")
	if !bytes.Equal(result.Value, originalValue) || !bytes.Equal(result.Ready, originalReady) || !bytes.Equal(result.SnapshotID, originalID) || string(result.Snapshot.Body["id"]) != `"created"` || string(result.ReadySnapshot.Body["metadata"]) != `{"n":9007199254740993}` {
		t.Fatal("creation phase aliases later phase", result)
	}
	smwPage(t, result.LastAccepted, 202, body, "owned")
	result.Value[0] = '!'
	result.Snapshot.Body["id"][0] = '!'
	result.Snapshot.Header.Set("X-Proof", "value mutated")
	if !bytes.Equal(result.Ready, originalReady) || string(result.ReadySnapshot.Body["id"]) != `"created"` || result.ReadySnapshot.Header.Get("X-Proof") != "owned" || !bytes.Equal(result.SnapshotID, originalID) {
		t.Fatal("final and ready phases alias", result)
	}
}

func TestCreateVolumeSnapshotOriginalOptionsAndFactoriesCannotChangeCapturedRequestAfterBoundary(t *testing.T) {
	cloud := testcloud.New(t)
	client := smwClient(cloud)
	var calls atomic.Int32
	fields := map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}
	force, wait := false, false
	factory := blockstorage.WithCreateVolumeSnapshotOptions(blockstorage.CreateVolumeSnapshotOpts{Force: &force, Wait: &wait, Attributes: blockstorage.CreateVolumeSnapshotAttributes{Fields: fields}})
	fields["name"][1] = 'X'
	force = true
	wait = true
	callbacks := []int{}
	var retained *blockstorage.CreateVolumeSnapshotOpts
	var options []blockstorage.CreateVolumeSnapshotOption
	options = []blockstorage.CreateVolumeSnapshotOption{factory, func(o *blockstorage.CreateVolumeSnapshotOpts) error {
		callbacks = append(callbacks, 1)
		retained = o
		options[2] = func(*blockstorage.CreateVolumeSnapshotOpts) error {
			t.Error("later original slot replaced")
			return errors.New("replacement")
		}
		client.MoreHeaders["x-source"] = "ordinary later change"
		return nil
	}, func(o *blockstorage.CreateVolumeSnapshotOpts) error {
		callbacks = append(callbacks, 2)
		*retained.Force = true
		*retained.Wait = true
		retained.Attributes.Fields["name"][1] = 'Y'
		if o.Force == nil || *o.Force || o.Wait == nil || *o.Wait || string(o.Attributes.Fields["name"]) != `"owned"` {
			t.Error("previous original config aliases next boundary", o)
		}
		return nil
	}}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		payload := smwFields(t, smwFields(t, smwRequest(t, r, http.MethodPost, smwCollection))["snapshot"])
		if string(payload["force"]) != "false" || string(payload["name"]) != `"owned"` {
			t.Error(payload)
		}
		testcloud.JSON(w, 202, `{"snapshot":{"id":false,"status":false}}`)
	})
	result, err := smwCreate(t, client, options...)
	if err != nil || result == nil || result.Value == nil || result.Ready != nil || calls.Load() != 1 || !reflect.DeepEqual(callbacks, []int{1, 2}) || client.MoreHeaders["x-source"] != "ordinary later change" {
		t.Fatal(result, err, calls.Load(), callbacks)
	}
}
