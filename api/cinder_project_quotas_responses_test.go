package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, defaults, usage, updates, resets atomic.Int32
	cloud.Mux.HandleFunc("/cinder/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-OpenStack-Volume-API-Version") != "3.70" || r.Header.Get("OpenStack-API-Version") != "volume 3.70" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("request headers=%s", r.Header)
		}
		switch r.Method {
		case http.MethodGet:
			if r.URL.RawQuery == "usage=true" {
				usage.Add(1)
				w.Header().Set("X-Openstack-Request-Id", "req-usage")
				testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project","volumes":{"limit":-1,"in_use":2,"reserved":3,"allocated":4,"vendor":null},"snapshots":{"limit":0,"in_use":0,"reserved":0},"volumes_SSD":{"limit":8,"in_use":1,"reserved":2,"allocated":5,"counter":9007199254740993},"optional":null}}`)
				return
			}
			if r.URL.RawQuery != "" {
				t.Error(r.URL.RawQuery)
			}
			gets.Add(1)
			w.Header().Set("X-Openstack-Request-Id", "req-get")
			testcloud.JSON(w, 200, cinderProjectQuotaLimits)
		case http.MethodPut:
			updates.Add(1)
			if r.URL.RawQuery != "" {
				t.Error(r.URL.RawQuery)
			}
			var body struct {
				Quota map[string]json.RawMessage `json:"quota_set"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if string(body.Quota["volumes"]) != "-1" || string(body.Quota["snapshots"]) != "0" || string(body.Quota["force"]) != "false" || string(body.Quota["vendor"]) != `{"tier":"requested"}` || string(body.Quota["volumes_SSD"]) != "12" || len(body.Quota) != 5 {
				t.Errorf("update body=%s", body.Quota)
			}
			w.Header().Set("X-Openstack-Request-Id", "req-update")
			testcloud.JSON(w, 200, cinderProjectQuotaLimits)
		case http.MethodDelete:
			resets.Add(1)
			if r.URL.RawQuery != "" {
				t.Error(r.URL.RawQuery)
			}
			w.Header().Set("X-Openstack-Request-Id", "req-reset")
			w.WriteHeader(http.StatusOK)
		default:
			t.Error(r.Method)
		}
	})
	cloud.Mux.HandleFunc("/cinder/os-quota-sets/project-fixed/defaults", func(w http.ResponseWriter, r *http.Request) {
		defaults.Add(1)
		if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.70" {
			t.Errorf("defaults request=%s %s", r.Method, r.URL)
		}
		w.Header().Set("X-Openstack-Request-Id", "req-defaults")
		testcloud.JSON(w, 200, `{"quota_set":{"id":"default-wire-project","volumes":10,"groups":11,"volumes_SSD":-1,"counter":9007199254740993,"optional":null}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("escaped fixed target: %s", r.URL)
		testcloud.JSON(w, 404, "{}")
	})
	scope := newCinderProjectQuotaScope(t, cloud)
	if scope.ProjectID() != "project-fixed" || gets.Load()+usage.Load()+defaults.Load() != 0 {
		t.Fatal("explicit ID performed a lookup")
	}
	limits, err := scope.Get(context.Background())
	if err != nil || limits == nil || limits.ProjectID != "project-fixed" || limits.ID != "wire-project" || limits.Volumes != -1 || limits.Snapshots != 0 || limits.Extra["volumes_SSD"] != float64(8) || string(limits.Body["counter"]) != "9007199254740993" || string(limits.Body["optional"]) != "null" || limits.Header.Get("X-Openstack-Request-Id") != "req-get" {
		t.Fatalf("limits=%+v err=%v", limits, err)
	}
	baseline, err := scope.Defaults(context.Background())
	if err != nil || baseline == nil || baseline.ProjectID != "project-fixed" || baseline.ID != "default-wire-project" || baseline.Volumes != 10 || baseline.Groups != 11 || string(baseline.Body["volumes_SSD"]) != "-1" || string(baseline.Body["counter"]) != "9007199254740993" || baseline.Header.Get("X-Openstack-Request-Id") != "req-defaults" {
		t.Fatalf("defaults=%+v err=%v", baseline, err)
	}
	detail, err := scope.Usage(context.Background())
	if err != nil || detail == nil || detail.ProjectID != "project-fixed" || detail.ID != "wire-project" || detail.Volumes.Limit != -1 || detail.Volumes.InUse != 2 || detail.Volumes.Reserved != 3 || detail.Volumes.Allocated != 4 || detail.Snapshots.Limit != 0 || string(detail.Body["volumes_SSD"]) != `{"allocated":5,"counter":9007199254740993,"in_use":1,"limit":8,"reserved":2}` || string(detail.Body["optional"]) != "null" || detail.Header.Get("X-Openstack-Request-Id") != "req-usage" {
		t.Fatalf("usage=%+v err=%v", detail, err)
	}
	zero, unlimited := 0, -1
	extension := map[string]string{"tier": "requested"}
	option := quotasets.WithUpdateField("vendor", extension)
	extension["tier"] = "changed"
	updated, err := scope.Update(context.Background(), quotasets.UpdateOpts{Volumes: &unlimited, Snapshots: &zero, Force: true}, option, quotasets.WithUpdateForce(false), quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeVolumes, "SSD", 12))
	if err != nil || updated == nil || updated.ProjectID != "project-fixed" || updated.ID != "wire-project" || updated.Header.Get("X-Openstack-Request-Id") != "req-update" || string(updated.Body["counter"]) != "9007199254740993" {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	reset, err := scope.Reset(context.Background())
	if err != nil || reset == nil || reset.ProjectID != "project-fixed" || reset.Header.Get("X-Openstack-Request-Id") != "req-reset" || gets.Load() != 1 || defaults.Load() != 1 || usage.Load() != 1 || updates.Load() != 1 || resets.Load() != 1 {
		t.Fatalf("reset=%+v counts=%d/%d/%d/%d/%d err=%v", reset, gets.Load(), defaults.Load(), usage.Load(), updates.Load(), resets.Load(), err)
	}
	limits.Header.Set("X-Openstack-Request-Id", "changed")
	limits.Body["counter"][0] = '0'
	limits.Extra["volumes_SSD"] = 99
	if baseline.Header.Get("X-Openstack-Request-Id") != "req-defaults" || detail.Header.Get("X-Openstack-Request-Id") != "req-usage" || updated.Header.Get("X-Openstack-Request-Id") != "req-update" || reset.Header.Get("X-Openstack-Request-Id") != "req-reset" || string(updated.Body["counter"]) != "9007199254740993" || updated.Extra["volumes_SSD"] != float64(8) {
		t.Fatal("responses share mutable state")
	}
}

func TestCinderProjectQuotaDeepOptionSnapshotAndCoreCollisions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/cinder/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Quota map[string]json.RawMessage `json:"quota_set"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Quota) != 4 || string(body.Quota["volumes"]) != "7" || string(body.Quota["vendor"]) != `{"array":[{"counter":9007199254740993,"tier":"original"}],"null":null}` || string(body.Quota["gigabytes_SSD"]) != "0" || string(body.Quota["force"]) != "true" {
			t.Errorf("snapshot=%s", body.Quota)
		}
		testcloud.JSON(w, 200, cinderProjectQuotaLimits)
	})
	scope := newCinderProjectQuotaScope(t, cloud)
	limit := 7
	nested := map[string]any{"tier": "original", "counter": json.Number("9007199254740993")}
	extra := map[string]any{"vendor": map[string]any{"array": []any{nested}, "null": nil}, "gigabytes_SSD": 0}
	option := quotasets.WithQuotaOptions(quotasets.UpdateOpts{Volumes: &limit, Extra: extra, Force: true})
	limit = 99
	nested["tier"] = "changed"
	extra["gigabytes_SSD"] = 99
	extra["new"] = true
	for range 2 {
		if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{}, option); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{"volumes", "snapshots", "gigabytes", "per_volume_gigabytes", "backups", "backup_gigabytes", "groups", "force"} {
		for _, opts := range []quotasets.UpdateOpts{{Extra: map[string]any{key: 0}}, {Volumes: &limit, Extra: map[string]any{key: nil}}} {
			if result, err := scope.Update(context.Background(), opts); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("collision %s: result=%v err=%v", key, result, err)
			}
			if result, err := scope.Update(context.Background(), quotasets.UpdateOpts{}, quotasets.WithQuotaOptions(opts)); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("snapshot collision %s: result=%v err=%v", key, result, err)
			}
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestCinderProjectQuotaDeleteAndUpdateAcceptOnlyNative200(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		for _, status := range []int{201, 202, 204} {
			t.Run(method+http.StatusText(status), func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					if r.Method != method {
						t.Error(r.Method)
					}
					w.Header().Set("X-Openstack-Request-Id", "unexpected-code")
					w.WriteHeader(status)
				})
				scope := newCinderProjectQuotaScope(t, cloud)
				var err error
				if method == http.MethodPut {
					_, err = scope.Update(context.Background(), quotasets.UpdateOpts{})
				} else {
					_, err = scope.Reset(context.Background())
				}
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != status || len(response.Expected) != 1 || response.Expected[0] != 200 || response.ResponseHeader.Get("X-Openstack-Request-Id") != "unexpected-code" {
					t.Fatalf("response=%+v err=%v", response, err)
				}
			})
		}
	}
}

func TestCinderProjectQuotaTypedLimitsRetainIntegerPrecision(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("this native int limit requires a 64-bit platform")
	}
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/cinder/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Quota map[string]json.RawMessage `json:"quota_set"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		for _, key := range []string{"volumes", "snapshots", "gigabytes", "per_volume_gigabytes", "backups", "backup_gigabytes", "groups", "gigabytes_SSD"} {
			if string(body.Quota[key]) != "9007199254740993" {
				t.Errorf("%s lost integer precision: %s", key, body.Quota[key])
			}
		}
		testcloud.JSON(w, 200, `{"quota_set":{"volumes":9007199254740993}}`)
	})
	scope := newCinderProjectQuotaScope(t, cloud)
	var integer int64 = 9007199254740993
	large := int(integer)
	opts := quotasets.UpdateOpts{Volumes: &large, Snapshots: &large, Gigabytes: &large, PerVolumeGigabytes: &large, Backups: &large, BackupGigabytes: &large, Groups: &large}
	for _, options := range [][]quotasets.UpdateOption{
		{quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeGigabytes, "SSD", large)},
		{quotasets.WithQuotaOptions(opts), quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeGigabytes, "SSD", large)},
	} {
		value, err := scope.Update(context.Background(), opts, options...)
		if err != nil || value == nil || value.Volumes != large || string(value.Body["volumes"]) != "9007199254740993" {
			t.Fatalf("quota=%+v err=%v", value, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}
