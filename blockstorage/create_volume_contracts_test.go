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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
)

const createVolumeContractCinderBase = "/reverse/cinder/v3/project/"
const createVolumeContractGlanceBase = "/reverse/glance/v2/"
const createVolumeContractCreatePath = createVolumeContractCinderBase + "volumes"
const createVolumeContractPollPath = createVolumeContractCinderBase + "volumes/new-1"
const createVolumeContractActionPath = createVolumeContractPollPath + "/action"

func createVolumeContractClients(cloud *testcloud.Cloud) (*gophercloud.ServiceClient, *gophercloud.ServiceClient) {
	cinder, glance := cloud.Client("volumev3", "/unused/cinder/"), cloud.Client("image", "/unused/glance/")
	cinder.ResourceBase, glance.ResourceBase = cloud.Server.URL+createVolumeContractCinderBase, cloud.Server.URL+createVolumeContractGlanceBase
	cinder.Microversion = "3.60"
	cinder.MoreHeaders, glance.MoreHeaders = map[string]string{"x-source": "cinder-entry"}, map[string]string{"x-source": "glance-entry"}
	return cinder, glance
}
func createVolumeContractBody(status string) string {
	return fmt.Sprintf(`{"volume":{"id":"new-1","name":null,"status":%q,"size":2,"bootable":"false","metadata":{"large":9007199254740993},"attachments":[{"server_id":"server","device":null,"vendor":9007199254740993}],"created_at":"literal Cinder time","vendor":{"number":9007199254740993}}}`, status)
}
func createVolumeContractWire(t *testing.T, r *http.Request, method, path, token string) {
	t.Helper()
	source := "cinder-entry"
	if strings.HasPrefix(path, createVolumeContractGlanceBase) {
		source = "glance-entry"
	}
	testhelper.TestMethod(t, r, method)
	testhelper.TestHeader(t, r, "X-Source", source)
	testhelper.TestHeader(t, r, "X-Auth-Token", token)
	if r.URL.Path != path {
		t.Errorf("URL=%s, expected path=%s", r.URL, path)
	}
	if source == "cinder-entry" && (r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "volume 3.60") {
		t.Error(r.URL, r.Header)
	}
	if method == http.MethodGet && r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Errorf("GET body=%q error=%v", body, err)
		}
	}
}
func createVolumeContractFields(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		t.Error(err)
	}
	return value
}
func createVolumeContractEnvelope(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	top := createVolumeContractFields(t, r)
	var body map[string]json.RawMessage
	if err := json.Unmarshal(top["volume"], &body); err != nil {
		t.Error(err)
	}
	return body
}
func createVolumeContractCreated(t *testing.T, result *blockstorage.CreateVolumeResult, body string) {
	t.Helper()
	if result == nil || result.VolumeID != "new-1" || result.Created == nil || result.Created.Volume == nil || result.Created.StatusCode != 202 || string(result.Created.Body) != body || result.Created.Header.Get("X-Proof") != "created" || result.Created.Volume.ID == nil || *result.Created.Volume.ID != "new-1" {
		t.Fatalf("accepted creation proof missing: %+v", result)
	}
}
func createVolumeContractOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "CreateVolume" || operation.Cause == nil {
		t.Fatalf("create operation context missing: %v", err)
	}
}

func TestCreateVolumeContractsDefaultFreshWaitAndIndependentOwnedResults(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, _ := createVolumeContractClients(cloud)
	created, ready := createVolumeContractBody("available"), createVolumeContractBody("AVAILABLE")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "test-token")
			body := createVolumeContractEnvelope(t, r)
			if !reflect.DeepEqual(body, map[string]json.RawMessage{"size": json.RawMessage(`2`)}) {
				t.Error(body)
			}
			w.Header().Set("X-Proof", "created")
			w.Header().Set("Location", "https://foreign.invalid/not-followed")
			testcloud.JSON(w, 202, created)
		case 2:
			createVolumeContractWire(t, r, http.MethodGet, createVolumeContractPollPath, "test-token")
			w.Header().Set("X-Proof", "ready")
			testcloud.JSON(w, 200, ready)
		default:
			t.Error("cached create shortcut, reroute or unexpected phase", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2})
	createVolumeContractCreated(t, result, created)
	if err != nil || calls.Load() != 2 || result.LastAccepted == nil || result.LastAccepted.Volume == nil || string(result.LastAccepted.Body) != ready || result.Ready == nil || *result.Ready.Status != "AVAILABLE" || result.Ready == result.LastAccepted.Volume || result.BootableSet != nil || result.Ready.CreatedAt == nil || *result.Ready.CreatedAt != "literal Cinder time" || string(result.Ready.MetadataFields["large"]) != "9007199254740993" {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	result.Created.Body[0] = '!'
	result.Created.Header.Set("X-Proof", "caller")
	*result.Created.Volume.Status = "caller"
	result.Created.Volume.MetadataFields["large"][0] = '!'
	*result.Ready.Status = "caller"
	result.Ready.Header.Set("X-Proof", "caller")
	result.Ready.Body["vendor"][0] = '!'
	result.Ready.Attachments[0].Body["vendor"][0] = '!'
	if string(result.LastAccepted.Body) != ready || result.LastAccepted.Header.Get("X-Proof") != "ready" || *result.LastAccepted.Volume.Status != "AVAILABLE" || string(result.LastAccepted.Volume.MetadataFields["large"]) != "9007199254740993" || string(result.LastAccepted.Volume.Body["vendor"]) != `{"number":9007199254740993}` || string(result.LastAccepted.Volume.Attachments[0].Body["vendor"]) != "9007199254740993" {
		t.Fatal("phase/readiness model evidence aliases", result)
	}
}

func TestCreateVolumeContractsNoWaitSizeAndLiteralSourcesAreOwnedWithoutLookups(t *testing.T) {
	for _, size := range []int{0, -7, 2} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			image := resource.ID("selected-image")
			var calls atomic.Int32
			created := createVolumeContractBody("ERROR")
			fields := map[string]json.RawMessage{"size": json.RawMessage(`999`), "imageRef": json.RawMessage(`"raw-image"`), "snapshot_id": json.RawMessage(`"snapshot/literal?name"`), "source_volid": json.RawMessage(`"source/literal"`), "backup_id": json.RawMessage(`"backup/literal"`), "bootable": json.RawMessage(`true`)}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "test-token")
				body := createVolumeContractEnvelope(t, r)
				want := map[string]json.RawMessage{"size": json.RawMessage(fmt.Sprint(size)), "imageRef": json.RawMessage(`"selected-image"`), "snapshot_id": json.RawMessage(`"snapshot/literal?name"`), "source_volid": json.RawMessage(`"source/literal"`), "backup_id": json.RawMessage(`"backup/literal"`), "bootable": json.RawMessage(`true`)}
				if !reflect.DeepEqual(body, want) {
					t.Error(body, want)
				}
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 202, created)
			})
			result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: size, Image: &image}, blockstorage.WithCreateVolumeWait(false), blockstorage.WithCreateVolumeFields(fields))
			createVolumeContractCreated(t, result, created)
			if err != nil || calls.Load() != 1 || result.LastAccepted != nil || result.Ready != nil || result.BootableSet != nil {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}

func TestCreateVolumeContractsAliasesPresenceConcreteOverridesMetadataAndHints(t *testing.T) {
	for _, canonical := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		t.Run("falsey canonical "+canonical, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			var calls atomic.Int32
			fields := map[string]json.RawMessage{"name": json.RawMessage(canonical), "display_name": json.RawMessage(`"discarded name"`), "description": json.RawMessage(canonical), "display_description": json.RawMessage(`"discarded description"`)}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body := createVolumeContractEnvelope(t, r)
				if len(body) != 1 || string(body["size"]) != "2" {
					t.Error("canonical presence must suppress display fallback", body)
				}
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 202, createVolumeContractBody("creating"))
			})
			result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeWait(false), blockstorage.WithCreateVolumeFields(fields))
			if err != nil || result == nil || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
	t.Run("common fields override raw aliases and nested hints move to top level", func(t *testing.T) {
		cloud := testcloud.New(t)
		cinder, _ := createVolumeContractClients(cloud)
		var calls atomic.Int32
		fields := map[string]json.RawMessage{"display_name": json.RawMessage(`"display"`), "name": json.RawMessage(`"raw name"`), "display_description": json.RawMessage(`"display description"`), "image_id": json.RawMessage(`"alias image"`), "imageRef": json.RawMessage(`"wire image"`), "metadata": json.RawMessage(`{"raw":"discarded"}`), "scheduler_hints": json.RawMessage(`{"discarded":true}`), "volume_image_metadata": json.RawMessage(`{"number":9007199254740993}`)}
		hints := map[string]json.RawMessage{"same_host": json.RawMessage(`["server"]`), "vendor": json.RawMessage(`{"number":9007199254740993}`)}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			top := createVolumeContractFields(t, r)
			var body map[string]json.RawMessage
			_ = json.Unmarshal(top["volume"], &body)
			want := map[string]json.RawMessage{"size": json.RawMessage(`2`), "name": json.RawMessage(`"common name"`), "description": json.RawMessage(`"display description"`), "imageRef": json.RawMessage(`"wire image"`), "metadata": json.RawMessage(`{}`), "volume_image_metadata": json.RawMessage(`{"number":9007199254740993}`)}
			if !reflect.DeepEqual(body, want) || len(top) != 2 {
				t.Error(top, body, want)
			}
			var actual map[string]json.RawMessage
			_ = json.Unmarshal(top["OS-SCH-HNT:scheduler_hints"], &actual)
			if !reflect.DeepEqual(actual, hints) {
				t.Error(actual, hints)
			}
			w.Header().Set("X-Proof", "created")
			testcloud.JSON(w, 202, createVolumeContractBody("creating"))
		})
		result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeWait(false), blockstorage.WithCreateVolumeFields(fields), blockstorage.WithCreateVolumeName("common name"), blockstorage.WithCreateVolumeMetadata(map[string]string{}), blockstorage.WithCreateVolumeSchedulerHints(hints))
		if err != nil || result == nil || calls.Load() != 1 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
	for _, rawHint := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`{}`)} {
		t.Run("empty hints "+string(rawHint), func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				top := createVolumeContractFields(t, r)
				var body map[string]json.RawMessage
				_ = json.Unmarshal(top["volume"], &body)
				if len(top) != 1 || len(body) != 2 || string(body["metadata"]) != "null" {
					t.Error("null metadata or empty-hint semantics", top, body)
				}
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 202, createVolumeContractBody("creating"))
			})
			result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeWait(false), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"scheduler_hints": rawHint, "metadata": nil}))
			if err != nil || result == nil {
				t.Fatal(result, err)
			}
		})
	}
}

func TestCreateVolumeContractsImageNameResolvesOnceAndReferenceWinsAllAttributes(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, glance := createVolumeContractClients(cloud)
	image := resource.Name("selected?literal")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			createVolumeContractWire(t, r, http.MethodGet, createVolumeContractGlanceBase+"images", "test-token")
			if r.URL.Query().Get("name") != "selected?literal" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"images":[{"id":"bound-image","name":"selected?literal"}]}`)
		case 2:
			createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "test-token")
			body := createVolumeContractEnvelope(t, r)
			if len(body) != 2 || string(body["imageRef"]) != `"bound-image"` {
				t.Error(body)
			}
			w.Header().Set("X-Proof", "created")
			testcloud.JSON(w, 202, createVolumeContractBody("creating"))
		default:
			t.Error("image resolved repeatedly or unrelated lookup", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.CreateVolume(context.Background(), cinder, glance, blockstorage.CreateVolumeRequest{Size: 2, Image: &image}, blockstorage.WithCreateVolumeWait(false), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"image_id": json.RawMessage(`"attr-image"`), "imageRef": json.RawMessage(`"wire-image"`)}))
	createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
	if err != nil || calls.Load() != 2 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	for _, tc := range []struct {
		name, rows string
		cause      error
	}{{"missing", `[]`, resource.ErrNotFound}, {"ambiguous", `[{"id":"one","name":"worker"},{"id":"two","name":"worker"}]`, resource.ErrAmbiguous}, {"unsafe resolved ID", `[{"id":"../image","name":"worker"}]`, resource.ErrInvalidOption}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, glance := createVolumeContractClients(cloud)
			name := resource.Name("worker")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != createVolumeContractGlanceBase+"images" {
					t.Error("image failure reached create", r.URL)
				}
				testcloud.JSON(w, 200, `{"images":`+tc.rows+`}`)
			})
			result, err := blockstorage.CreateVolume(context.Background(), cinder, glance, blockstorage.CreateVolumeRequest{Size: 2, Image: &name})
			if result != nil || !errors.Is(err, tc.cause) || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
	t.Run("caller image pointer cannot change captured selector in options", func(t *testing.T) {
		cloud := testcloud.New(t)
		cinder, _ := createVolumeContractClients(cloud)
		image := resource.ID("selected-image")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			body := createVolumeContractEnvelope(t, r)
			if string(body["imageRef"]) != `"selected-image"` {
				t.Error("mutable caller reference changed selected image", body)
			}
			w.Header().Set("X-Proof", "created")
			testcloud.JSON(w, 202, createVolumeContractBody("creating"))
		})
		result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2, Image: &image}, blockstorage.WithCreateVolumeWait(false), func(*blockstorage.CreateVolumeOpts) error { image = resource.ID("late-image"); return nil })
		if err != nil || result == nil || calls.Load() != 1 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
}

func TestCreateVolumeContractsAllPinnedAttributesAndWireAliasesAreSerialized(t *testing.T) {
	// Independent expected wire table derived from the pinned Python Volume
	// declarations, inherited Resource/Metadata fields and native source_replica.
	rows := []struct{ attribute, wire, value string }{
		{"attachments", "attachments", `[]`},
		{"availability_zone", "availability_zone", `"literal-availability_zone"`},
		{"backup_id", "backup_id", `"literal-backup_id"`},
		{"consistency_group_id", "consistencygroup_id", `"literal-consistency_group_id"`},
		{"consumes_quota", "consumes_quota", `false`},
		{"cluster_name", "cluster_name", `"literal-cluster_name"`},
		{"created_at", "created_at", `"literal-created_at"`},
		{"description", "description", `"literal-description"`},
		{"encryption_key_id", "encryption_key_id", `"literal-encryption_key_id"`},
		{"extended_replication_status", "os-volume-replication:extended_status", `"literal-extended_replication_status"`},
		{"group_id", "group_id", `"literal-group_id"`},
		{"host", "os-vol-host-attr:host", `"literal-host"`},
		{"image_id", "imageRef", `"literal-image_id"`},
		{"is_bootable", "bootable", `false`},
		{"is_encrypted", "encrypted", `true`},
		{"is_multiattach", "multiattach", `false`},
		{"migration_id", "os-vol-mig-status-attr:name_id", `"literal-migration_id"`},
		{"migration_status", "os-vol-mig-status-attr:migstat", `"literal-migration_status"`},
		{"project_id", "os-vol-tenant-attr:tenant_id", `"literal-project_id"`},
		{"replication_driver_data", "os-volume-replication:driver_data", `"literal-replication_driver_data"`},
		{"provider_id", "provider_id", `"literal-provider_id"`},
		{"replication_status", "replication_status", `"literal-replication_status"`},
		{"scheduler_hints", "OS-SCH-HNT:scheduler_hints", `{"same_host":["server"]}`},
		{"service_uuid", "service_uuid", `"literal-service_uuid"`},
		{"shared_targets", "shared_targets", `false`},
		{"size", "size", `99`},
		{"snapshot_id", "snapshot_id", `"literal-snapshot_id"`},
		{"source_volume_id", "source_volid", `"literal-source_volume_id"`},
		{"status", "status", `"literal-status"`},
		{"updated_at", "updated_at", `"literal-updated_at"`},
		{"user_id", "user_id", `"literal-user_id"`},
		{"volume_image_metadata", "volume_image_metadata", `{"number":9007199254740993}`},
		{"volume_type", "volume_type", `"literal-volume_type"`},
		{"volume_type_id", "volume_type_id", `"literal-volume_type_id"`},
		{"id", "id", `"literal-id"`},
		{"name", "name", `"literal-name"`},
		{"metadata", "metadata", `{"owner":"worker"}`},
		{"source_replica", "source_replica", `"native-extension"`},
	}
	for _, row := range rows {
		keys := []string{row.attribute}
		if row.wire != row.attribute {
			keys = append(keys, row.wire)
		}
		for _, key := range keys {
			t.Run(key, func(t *testing.T) {
				cloud := testcloud.New(t)
				cinder, _ := createVolumeContractClients(cloud)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "test-token")
					top := createVolumeContractFields(t, r)
					var body map[string]json.RawMessage
					_ = json.Unmarshal(top["volume"], &body)
					want := map[string]json.RawMessage{"size": json.RawMessage(`2`)}
					expectedTop := 1
					if row.wire == "OS-SCH-HNT:scheduler_hints" {
						expectedTop = 2
						if string(top[row.wire]) != row.value {
							t.Error("scheduler hints not hoisted", top)
						}
					} else if row.wire != "size" {
						want[row.wire] = json.RawMessage(row.value)
					}
					if len(top) != expectedTop || !reflect.DeepEqual(body, want) {
						t.Error("known kwargs mapping", key, top, body, want)
					}
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 202, createVolumeContractBody("creating"))
				})
				result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeWait(false), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{key: json.RawMessage(row.value)}))
				if err != nil || result == nil || result.VolumeID != "new-1" || calls.Load() != 1 {
					t.Fatalf("field=%s result=%+v error=%v calls=%d", key, result, err, calls.Load())
				}
			})
		}
	}
}

func TestCreateVolumeContractsEveryTypedCommonAttributeOverridesRawWireValues(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, _ := createVolumeContractClients(cloud)
	var calls atomic.Int32
	name, description, zone, consistency := "worker", "description", "zone/literal", "consistency/literal"
	snapshot, source, replica, image := "snapshot/literal", "source/literal", "replica/literal", "image/literal"
	backup, volumeType, group, typeID := "backup/literal", "fast", "group/literal", "type/literal"
	multiattach := false
	fields := map[string]json.RawMessage{}
	want := map[string]json.RawMessage{"size": json.RawMessage(`2`), "name": json.RawMessage(`"worker"`), "description": json.RawMessage(`"description"`), "availability_zone": json.RawMessage(`"zone/literal"`), "consistencygroup_id": json.RawMessage(`"consistency/literal"`), "snapshot_id": json.RawMessage(`"snapshot/literal"`), "source_volid": json.RawMessage(`"source/literal"`), "source_replica": json.RawMessage(`"replica/literal"`), "imageRef": json.RawMessage(`"image/literal"`), "backup_id": json.RawMessage(`"backup/literal"`), "volume_type": json.RawMessage(`"fast"`), "group_id": json.RawMessage(`"group/literal"`), "volume_type_id": json.RawMessage(`"type/literal"`), "multiattach": json.RawMessage(`false`), "metadata": json.RawMessage(`{"owner":"worker"}`)}
	for key := range want {
		if key != "size" {
			fields[key] = json.RawMessage(`"discarded"`)
		}
	}
	attributes := blockstorage.CreateVolumeAttributes{Name: &name, Description: &description, AvailabilityZone: &zone, ConsistencyGroupID: &consistency, SnapshotID: &snapshot, SourceVolumeID: &source, SourceReplica: &replica, ImageID: &image, BackupID: &backup, VolumeType: &volumeType, GroupID: &group, VolumeTypeID: &typeID, Multiattach: &multiattach, Metadata: map[string]string{"owner": "worker"}, Fields: fields}
	option := blockstorage.WithCreateVolumeAttributes(attributes)
	name, description, zone, consistency, snapshot, source, replica, image, backup, volumeType, group, typeID = "changed", "changed", "changed", "changed", "changed", "changed", "changed", "changed", "changed", "changed", "changed", "changed"
	multiattach = true
	attributes.Metadata["owner"] = "changed"
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "test-token")
		body := createVolumeContractEnvelope(t, r)
		if !reflect.DeepEqual(body, want) {
			t.Error("typed common attributes lost or raw wire value won", body, want)
		}
		w.Header().Set("X-Proof", "created")
		testcloud.JSON(w, 202, createVolumeContractBody("creating"))
	})
	result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeWait(false), option)
	createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
	if err != nil || calls.Load() != 1 || result.LastAccepted != nil || result.Ready != nil {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
}

func TestCreateVolumeContractsBootableTriStateAlwaysWaitsButOnlyTrueSendsAction(t *testing.T) {
	for _, tc := range []struct {
		name     string
		bootable *bool
		calls    int32
	}{{"nil", nil, 1}, {"false", new(bool), 2}, {"true", func() *bool { v := true; return &v }(), 3}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			var calls atomic.Int32
			created, ready := createVolumeContractBody("creating"), createVolumeContractBody("available")
			opaque := []byte{0, 0xff, 'x'}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "test-token")
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 202, created)
				case 2:
					createVolumeContractWire(t, r, http.MethodGet, createVolumeContractPollPath, "test-token")
					testcloud.JSON(w, 200, ready)
				case 3:
					createVolumeContractWire(t, r, http.MethodPost, createVolumeContractActionPath, "test-token")
					fields := createVolumeContractFields(t, r)
					if len(fields) != 1 || string(fields["os-set_bootable"]) != `{"bootable":true}` {
						t.Error(fields)
					}
					w.Header().Set("X-Proof", "bootable")
					w.WriteHeader(200)
					_, _ = w.Write(opaque)
				default:
					t.Error("extra verification GET or mutation", r.Method, r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeOptions(blockstorage.CreateVolumeOpts{Bootable: tc.bootable}), blockstorage.WithCreateVolumeWait(false))
			createVolumeContractCreated(t, result, created)
			if err != nil || calls.Load() != tc.calls || tc.bootable == nil && (result.Ready != nil || result.LastAccepted != nil) || tc.bootable != nil && (result.Ready == nil || result.LastAccepted == nil || result.Ready.IsBootable == nil || *result.Ready.IsBootable) {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			if tc.calls == 3 {
				if result.BootableSet == nil || result.BootableSet.StatusCode != 200 || !bytes.Equal(result.BootableSet.Body, opaque) || result.BootableSet.Header.Get("X-Proof") != "bootable" || string(result.Ready.Body["bootable"]) != `"false"` {
					t.Fatal("action proof or cache boundary wrong", result)
				}
			} else if result.BootableSet != nil {
				t.Fatal("false/nil sent bootable action", result)
			}
		})
	}
}

func TestCreateVolumeContractsInitialLiteralErrorAndExactFreshWaitStates(t *testing.T) {
	stopped := errors.New("progress stopped create")
	for _, tc := range []struct {
		name, initial string
		wait          bool
		states, polls []string
		callbackCause error
		failed        bool
		callbacks     int32
	}{
		{name: "literal initial error no wait", initial: "error", failed: true},
		{name: "upper initial error no wait", initial: "ERROR"},
		{name: "target before configured failure", initial: "available", wait: true, states: []string{"available"}, polls: []string{"AVAILABLE"}},
		{name: "exact wait failure", initial: "creating", wait: true, polls: []string{"ErRoR"}, failed: true},
		{name: "prefix error is nonterminal", initial: "creating", wait: true, polls: []string{"error_deleting", "available"}, callbacks: 1},
		{name: "empty failures disable default", initial: "creating", wait: true, states: []string{}, polls: []string{"error", "available"}, callbacks: 1},
		{name: "callback stops before bootable", initial: "creating", wait: true, polls: []string{"creating"}, callbackCause: stopped, callbacks: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			var calls, callbacks atomic.Int32
			created := createVolumeContractBody(tc.initial)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				step := int(calls.Add(1))
				if step == 1 {
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 202, created)
					return
				}
				index := step - 2
				if index >= len(tc.polls) {
					t.Error("error/callback caused action or cleanup", r.Method, r.URL)
					w.WriteHeader(500)
					return
				}
				createVolumeContractWire(t, r, http.MethodGet, createVolumeContractPollPath, "test-token")
				testcloud.JSON(w, 200, createVolumeContractBody(tc.polls[index]))
			})
			interval := time.Millisecond
			options := []blockstorage.CreateVolumeOption{blockstorage.WithCreateVolumeWait(tc.wait), blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{PollInterval: &interval, FailureStates: tc.states, ProgressCallback: func(progress int) error {
				callbacks.Add(1)
				if progress != 0 {
					t.Error(progress)
				}
				return tc.callbackCause
			}})}
			if tc.callbackCause != nil {
				options = append(options, blockstorage.WithCreateVolumeBootable(true))
			}
			result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, options...)
			createVolumeContractCreated(t, result, created)
			if calls.Load() != int32(1+len(tc.polls)) || callbacks.Load() != tc.callbacks || errors.Is(err, resource.ErrFailedState) != tc.failed || tc.callbackCause != nil && !errors.Is(err, tc.callbackCause) || !tc.failed && tc.callbackCause == nil && err != nil || result.BootableSet != nil {
				t.Fatalf("result=%+v error=%v calls=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
			if err != nil {
				createVolumeContractOperation(t, err)
			}
		})
	}
}

func TestCreateVolumeContractsAcceptedCreateAndActionFailuresPreservePhaseEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
	}{
		{"unexpected create code", createVolumeContractBody("creating"), 201},
		{"malformed JSON", `{"volume":`, 202},
		{"null envelope", `{"volume":null}`, 202},
		{"missing canonical ID", `{"volume":{"ID":"new-1","status":"creating"}}`, 202},
		{"unsafe ID", `{"volume":{"id":"../other","status":"creating"}}`, 202},
		{"missing status", `{"volume":{"id":"new-1"}}`, 202},
		{"bad known size", `{"volume":{"id":"new-1","status":"creating","size":false}}`, 202},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "test-token")
				w.Header().Set("X-Proof", "created-failure")
				testcloud.JSON(w, tc.code, tc.body)
			})
			result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2})
			var accepted *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || err == nil || result.Ready != nil || result.LastAccepted != nil || result.BootableSet != nil || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			if tc.code == 202 {
				if result.Created == nil || result.Created.StatusCode != 202 || string(result.Created.Body) != tc.body || !errors.As(err, &accepted) || accepted.StatusCode != 202 {
					t.Fatal(result, err)
				}
			} else if result.Created != nil || !errors.As(err, &native) || native.Actual != tc.code || !reflect.DeepEqual(native.Expected, []int{202}) {
				t.Fatal(result, err)
			}
			createVolumeContractOperation(t, err)
		})
	}
	t.Run("bootable rejected retains ready but no accepted action proof", func(t *testing.T) {
		cloud := testcloud.New(t)
		cinder, _ := createVolumeContractClients(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 202, createVolumeContractBody("creating"))
			case 2:
				testcloud.JSON(w, 200, createVolumeContractBody("available"))
			case 3:
				w.Header().Set("X-Proof", "rejected-action")
				testcloud.JSON(w, 202, `{"error":"unexpected action code"}`)
			default:
				t.Error("action rejection caused cleanup", r.Method, r.URL)
				w.WriteHeader(500)
			}
		})
		result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeBootable(true))
		createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
		var native gophercloud.ErrUnexpectedResponseCode
		var accepted *resource.ResponseError
		if !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{200}) || native.ResponseHeader.Get("X-Proof") != "rejected-action" || errors.As(err, &accepted) || result.Ready == nil || result.LastAccepted == nil || result.BootableSet != nil || calls.Load() != 3 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
}

func TestCreateVolumeContractsPreflightValidatesEveryProvidedSourceBeforeCallbacks(t *testing.T) {
	for _, tc := range []struct {
		name                                          string
		change                                        func(*gophercloud.ServiceClient, *gophercloud.ServiceClient, *blockstorage.CreateVolumeRequest)
		nilContext, nilCinder, nilGlance, unsupported bool
		option                                        blockstorage.CreateVolumeOption
		callbacks                                     int32
	}{
		{name: "nil context", nilContext: true}, {name: "nil Cinder", nilCinder: true},
		{name: "missing Glance before image Name", nilGlance: true, callbacks: 1, change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.CreateVolumeRequest) {
			v := resource.Name("image")
			i.Image = &v
		}},
		{name: "unused Glance provider invalid", change: func(_, g *gophercloud.ServiceClient, _ *blockstorage.CreateVolumeRequest) { g.ProviderClient = nil }},
		{name: "unused Glance service invalid", unsupported: true, change: func(_, g *gophercloud.ServiceClient, _ *blockstorage.CreateVolumeRequest) { g.Type = "compute" }},
		{name: "Cinder wrong service", unsupported: true, change: func(c, _ *gophercloud.ServiceClient, _ *blockstorage.CreateVolumeRequest) { c.Type = "network" }},
		{name: "Glance foreign effective base", change: func(_, g *gophercloud.ServiceClient, _ *blockstorage.CreateVolumeRequest) {
			g.ResourceBase = "https://foreign.invalid/v2/"
		}},
		{name: "Cinder body framing override", change: func(c, _ *gophercloud.ServiceClient, _ *blockstorage.CreateVolumeRequest) {
			c.MoreHeaders["Content-Length"] = "1"
		}},
		{name: "Glance token override", change: func(_, g *gophercloud.ServiceClient, _ *blockstorage.CreateVolumeRequest) {
			g.MoreHeaders["X-Auth-Token"] = "other"
		}},
		{name: "unsafe image reference", change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.CreateVolumeRequest) {
			v := resource.ID("../image")
			i.Image = &v
		}},
		{name: "unknown kwargs before Name resolution", callbacks: 1, change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.CreateVolumeRequest) {
			v := resource.Name("image")
			i.Image = &v
		}, option: blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"unknown": json.RawMessage(`true`)})},
		{name: "nonobject hints before create", callbacks: 1, option: blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"scheduler_hints": json.RawMessage(`[]`)})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, glance := createVolumeContractClients(cloud)
			input := blockstorage.CreateVolumeRequest{Size: 2}
			if tc.change != nil {
				tc.change(cinder, glance, &input)
			}
			if tc.nilCinder {
				cinder = nil
			}
			if tc.nilGlance {
				glance = nil
			}
			ctx := context.Background()
			if tc.nilContext {
				ctx = nil
			}
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("invalid create reached name lookup/mutation", r.Method, r.URL)
				w.WriteHeader(500)
			})
			options := []blockstorage.CreateVolumeOption{func(*blockstorage.CreateVolumeOpts) error { callbacks.Add(1); return nil }, blockstorage.WithCreateVolumeWait(false)}
			if tc.option != nil {
				options = append(options, tc.option)
			}
			result, err := blockstorage.CreateVolume(ctx, cinder, glance, input, options...)
			want := resource.ErrInvalidOption
			if tc.unsupported {
				want = resource.ErrUnsupported
			}
			if result != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != tc.callbacks {
				t.Fatalf("result=%+v error=%v HTTP=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
			createVolumeContractOperation(t, err)
		})
	}
	t.Run("entry canceled custom cause", func(t *testing.T) {
		cloud := testcloud.New(t)
		cinder, _ := createVolumeContractClients(cloud)
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("caller canceled create")
		cancel(cause)
		calls := 0
		result, err := blockstorage.CreateVolume(ctx, cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, func(*blockstorage.CreateVolumeOpts) error { calls++; return nil })
		if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 0 {
			t.Fatalf("result=%+v error=%v callbacks=%d", result, err, calls)
		}
	})
}
