package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestNovaProjectQuotaTypedLimitsPreserveIntegerPrecision(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("this native int limit requires a 64-bit platform")
	}
	cloud := testcloud.New(t)
	var calls atomic.Int32
	keys := []string{"fixed_ips", "floating_ips", "injected_file_content_bytes", "injected_file_path_bytes", "injected_files", "key_pairs", "metadata_items", "ram", "security_group_rules", "security_groups", "cores", "instances", "server_groups", "server_group_members"}
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPut || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.56" {
			t.Errorf("request=%s %s version=%s", r.Method, r.URL, r.Header.Get("X-OpenStack-Nova-API-Version"))
		}
		var body struct {
			Quota map[string]json.RawMessage `json:"quota_set"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		for _, key := range keys {
			if string(body.Quota[key]) != "9007199254740993" {
				t.Errorf("typed limit %s lost precision: %s", key, body.Quota[key])
			}
		}
		if len(body.Quota) != len(keys)+2 || string(body.Quota["force"]) != "false" || string(body.Quota["vendor_counter"]) != "9007199254740993" {
			t.Errorf("force/extensions=%s", body.Quota)
		}
		w.Header().Set("X-Openstack-Request-Id", "exact-quota")
		testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project","cores":9007199254740993,"vendor_counter":9007199254740993}}`)
	})
	scope := newProjectQuotaScope(t, cloud)
	var integer int64 = 9007199254740993
	large := int(integer)
	opts := quotasets.UpdateOpts{FixedIPs: &large, FloatingIPs: &large, InjectedFileContentBytes: &large, InjectedFilePathBytes: &large, InjectedFiles: &large, KeyPairs: &large, MetadataItems: &large, RAM: &large, SecurityGroupRules: &large, SecurityGroups: &large, Cores: &large, Instances: &large, ServerGroups: &large, ServerGroupMembers: &large, Force: true}
	for _, options := range [][]quotasets.UpdateOption{
		{quotasets.WithUpdateForce(false), quotasets.WithUpdateField("vendor_counter", json.Number("9007199254740993"))},
		{quotasets.WithQuotaOptions(opts), quotasets.WithUpdateForce(true), quotasets.WithUpdateForce(false), quotasets.WithUpdateField("vendor_counter", json.Number("9007199254740993"))},
		{quotasets.WithUpdateOptions(opts), quotasets.WithUpdateForce(false), quotasets.WithUpdateField("vendor_counter", json.Number("9007199254740993"))},
	} {
		value, err := scope.Update(context.Background(), opts, options...)
		if err != nil || value == nil || value.ProjectID != "project-fixed" || value.ID != "wire-project" || value.Cores != large || string(value.Body["cores"]) != "9007199254740993" || string(value.Body["vendor_counter"]) != "9007199254740993" || value.Header.Get("X-Openstack-Request-Id") != "exact-quota" {
			t.Fatalf("quota=%+v err=%v", value, err)
		}
	}
	if calls.Load() != 3 {
		t.Fatal(calls.Load())
	}
}

func TestNovaProjectQuotaOptionsCapturePointersAndAreReusable(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Quota map[string]json.RawMessage `json:"quota_set"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Quota) != 4 || string(body.Quota["cores"]) != "7" || string(body.Quota["instances"]) != "-1" || string(body.Quota["ram"]) != "0" || string(body.Quota["force"]) != "true" {
			t.Errorf("snapshot changed or base limits leaked: %s", body.Quota)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"cores":7,"instances":-1,"ram":0}}`)
	})
	scope := newProjectQuotaScope(t, cloud)
	cores, instances, ram := 7, -1, 0
	input := quotasets.UpdateOpts{Cores: &cores, Instances: &instances, RAM: &ram, Force: true}
	option := quotasets.WithQuotaOptions(input)
	cores, instances, ram = 99, 99, 99
	input.Force = false
	baseOnly := 123
	// A later option may inspect/mutate its own configured copy. Reusing the
	// snapshot must still start with the values captured at construction.
	mutateCopy := func(config *request.Config[quotasets.UpdateOpts]) error {
		if config.Options.Cores == input.Cores || config.Options.Instances == input.Instances || config.Options.RAM == input.RAM {
			t.Error("snapshot borrowed caller pointers")
		}
		if *config.Options.Cores != 7 || *config.Options.Instances != -1 || *config.Options.RAM != 0 {
			t.Error("prior option application mutated a reusable snapshot")
		}
		*config.Options.Cores = 8
		return nil
	}
	for range 3 {
		// Reapplying replaces this mutable copy with another independent copy.
		value, err := scope.Update(context.Background(), quotasets.UpdateOpts{SecurityGroups: &baseOnly}, option, mutateCopy, option)
		if err != nil || value == nil || value.Cores != 7 || value.Instances != -1 || value.RAM != 0 {
			t.Fatalf("quota=%+v err=%v", value, err)
		}
	}
	if calls.Load() != 3 || cores != 99 || instances != 99 || ram != 99 {
		t.Fatal("snapshot affected caller values or wrong request count")
	}
}

func TestNovaProjectQuotaSnapshotPreservesPreHTTPValidation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, projectQuotaLimits)
	})
	scope := newProjectQuotaScope(t, cloud)
	invalid := -2
	snapshot := quotasets.WithQuotaOptions(quotasets.UpdateOpts{Cores: &invalid})
	invalid = 0
	for _, options := range [][]quotasets.UpdateOption{
		{snapshot},
		{quotasets.WithQuotaOptions(quotasets.UpdateOpts{}), quotasets.WithUpdateField("cores", 0)},
		{quotasets.WithQuotaOptions(quotasets.UpdateOpts{}), quotasets.WithUpdateField("force", false)},
		{quotasets.WithQuotaOptions(quotasets.UpdateOpts{}), request.WithQuery[quotasets.UpdateOpts]("user_id", "user")},
		{quotasets.WithQuotaOptions(quotasets.UpdateOpts{}), request.WithHeader[quotasets.UpdateOpts]("X-Vendor", "value")},
		{quotasets.WithQuotaOptions(quotasets.UpdateOpts{}), request.WithArgument[quotasets.UpdateOpts]("unsupported", true)},
	} {
		if value, err := scope.Update(context.Background(), quotasets.UpdateOpts{}, options...); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatalf("quota=%v calls=%d err=%v", value, calls.Load(), err)
		}
	}
}
