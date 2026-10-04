package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gophercloud/gophercloud/v2"
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
	"net/http"
	"sync/atomic"
	"testing"
)

func connectionSearchRecord(t *testing.T, provider *gophercloud.ProviderClient, id string) {
	t.Helper()
	value := v3.CreateResult{}
	value.Header = http.Header{"X-Subject-Token": {"recorded-token"}}
	value.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": id, "name": "token-name-must-not-be-inferred"}}}
	if err := provider.SetTokenAndAuthResult(value); err != nil {
		t.Fatal(err)
	}
}
func connectionSearchFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(string(raw), err)
	}
	return fields
}

func TestConnectionVolumeSearchCachedCinderOriginalCallbacksOnceAndOwnedLocation(t *testing.T) {
	cloud := testcloud.New(t)
	connectionSearchRecord(t, cloud.Provider, "scope")
	var locates, lists, callbacks atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return cloud.Server.URL + "/connection-search/v3/p/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cached, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /connection-search/v3/p/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.Header.Get("X-Option") != "after" {
			t.Error("Connection package capture preceded original callback", r.Header)
		}
		testcloud.JSON(w, 200, `{"volumes":[{"id":"id","project_id":"scope","availability_zone":false,"location":{"cloud":"untrusted"}}]}`)
	})
	option := func(*blockstorage.VolumeSearchOpts) error {
		callbacks.Add(1)
		current, err := conn.BlockStorageV3(context.Background())
		if err != nil || current != cached {
			t.Error(current, err)
			return errors.New("cached service changed")
		}
		cached.RawClient().MoreHeaders = map[string]string{"X-Option": "after"}
		return nil
	}
	search, err := conn.SearchVolumes(context.Background(), blockstorage.SearchVolumesRequest{}, option)
	if err != nil || search == nil || len(search.Volumes) != 1 {
		t.Fatal(search, err)
	}
	result, err := conn.GetVolume(context.Background(), blockstorage.GetVolumeRequest{NameOrID: "id"}, blockstorage.WithVolumeSearchFilters(json.RawMessage(`false`)), option)
	if err != nil || result == nil || result.Volume == nil || locates.Load() != 1 || lists.Load() != 2 || callbacks.Load() != 2 {
		t.Fatal(result, err, locates.Load(), lists.Load(), callbacks.Load())
	}
	fields := connectionSearchFields(t, result.Value)
	location := connectionSearchFields(t, fields["location"])
	project := connectionSearchFields(t, location["project"])
	if string(location["cloud"]) != "null" || string(location["zone"]) != "false" || string(project["id"]) != `"scope"` || string(project["name"]) != "null" {
		t.Fatal(string(result.Value))
	}
}

func TestConnectionVolumeSearchOptionsBeforeGetterAndNilCancellationPreventSelection(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, callbacks atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if callbacks.Load() != 2 {
			t.Error("getter selected before originals", callbacks.Load())
		}
		return cloud.Server.URL + "/connection-search/v3/p/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("option rejected")
	result, err := conn.SearchVolumes(context.Background(), blockstorage.SearchVolumesRequest{}, func(*blockstorage.VolumeSearchOpts) error { callbacks.Add(1); return sentinel })
	if result != nil || !errors.Is(err, sentinel) || locates.Load() != 0 {
		t.Fatal(result, err, locates.Load())
	}
	cloud.Mux.HandleFunc("GET /connection-search/v3/p/volumes/detail", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"volumes":[]}`) })
	result, err = conn.SearchVolumes(context.Background(), blockstorage.SearchVolumesRequest{}, func(*blockstorage.VolumeSearchOpts) error { callbacks.Add(1); return nil })
	if err != nil || result == nil || locates.Load() != 1 {
		t.Fatal(result, err, locates.Load())
	}
	for _, kind := range []string{"nil context", "nil connection", "nil option", "callback cancel"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			current := conn
			cause := errors.New("volume callback canceled")
			count := 0
			opts := []blockstorage.VolumeSearchOption{func(*blockstorage.VolumeSearchOpts) error { count++; return nil }}
			want := resource.ErrInvalidOption
			wantCount := 0
			switch kind {
			case "nil context":
				ctx = nil
			case "nil connection":
				current = nil
			case "nil option":
				opts = append(opts, nil)
				wantCount = 1
			case "callback cancel":
				next, cancel := context.WithCancelCause(ctx)
				ctx = next
				opts = []blockstorage.VolumeSearchOption{func(*blockstorage.VolumeSearchOpts) error { count++; cancel(cause); return nil }, func(*blockstorage.VolumeSearchOpts) error { count++; return nil }}
				want = context.Canceled
				wantCount = 1
			}
			result, err := current.GetVolume(ctx, blockstorage.GetVolumeRequest{NameOrID: "id"}, opts...)
			if result != nil || !errors.Is(err, want) || count != wantCount || locates.Load() != 1 {
				t.Fatal(result, err, count, locates.Load())
			}
			if want == context.Canceled && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}

func TestConnectionCurrentLocationTracksRecordedScopeWithoutServiceOrAuthIO(t *testing.T) {
	cloud := testcloud.New(t)
	var locates atomic.Int32
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return "", errors.New("unexpected locator")
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second"} {
		connectionSearchRecord(t, cloud.Provider, id)
		location, err := conn.CurrentLocation()
		if err != nil || string(location.Project.ID) != `"`+id+`"` || location.Cloud != nil || location.RegionName == nil || *location.RegionName != "" || location.Project.Name != nil || locates.Load() != 0 {
			t.Fatal(location, err)
		}
		location.Project.ID[0] = '!'
		again, err := conn.CurrentLocation()
		if err != nil || !json.Valid(again.Project.ID) {
			t.Fatal("location not owned", again, err)
		}
	}
	cloud.Provider.SetToken("manual")
	location, err := conn.CurrentLocation()
	if err != nil || location.Project.ID != nil || locates.Load() != 0 {
		t.Fatal(location, err)
	}
	var missing *sdk.Connection
	if _, err := missing.CurrentLocation(); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestConnectionCloudLocationOverrideAndOperationLocationFactoriesOwnInputs(t *testing.T) {
	cloud := testcloud.New(t)
	cloudName, name := "owned-cloud", "owned-project"
	input := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"owned-scope"`), Name: &name}}
	option := sdk.WithCloudLocation(input)
	cloudName = "caller"
	name = "caller"
	input.Project.ID[0] = '!'
	conn, err := sdk.FromProvider(cloud.Provider, option, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-search/v3/p/"))
	if err != nil {
		t.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil || location.Cloud == nil || *location.Cloud != "owned-cloud" || string(location.Project.ID) != `"owned-scope"` {
		t.Fatal(location, err)
	}
	*location.Cloud = "changed"
	location.Project.ID[0] = '!'
	overrideName := "per-call"
	perCall := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"owned-scope"`), Name: &overrideName}}
	override := blockstorage.WithVolumeSearchLocation(perCall)
	overrideName = "caller"
	perCall.Project.ID[0] = '!'
	cloud.Mux.HandleFunc("GET /connection-search/v3/p/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"volumes":[{"id":"id","project_id":"owned-scope"}]}`)
	})
	result, err := conn.GetVolume(context.Background(), blockstorage.GetVolumeRequest{NameOrID: "id"}, blockstorage.WithVolumeSearchFilters(json.RawMessage(`false`)), override)
	if err != nil || result == nil || result.Volume == nil {
		t.Fatal(result, err)
	}
	resultProject := connectionSearchFields(t, connectionSearchFields(t, connectionSearchFields(t, result.Value)["location"])["project"])
	if string(resultProject["name"]) != `"per-call"` {
		t.Fatal(string(result.Value))
	}
	search, err := conn.SearchVolumes(context.Background(), blockstorage.SearchVolumesRequest{}, override)
	if err != nil || search == nil {
		t.Fatal(search, err)
	}
	var rows []map[string]json.RawMessage
	_ = json.Unmarshal(search.Value, &rows)
	project := connectionSearchFields(t, connectionSearchFields(t, rows[0]["location"])["project"])
	if string(project["name"]) != `"per-call"` {
		t.Fatal(string(search.Value))
	}
	again, err := conn.CurrentLocation()
	if err != nil || again.Cloud == nil || *again.Cloud != "owned-cloud" || again.Project.Name == nil || *again.Project.Name != "owned-project" {
		t.Fatal(again, err)
	}
}
