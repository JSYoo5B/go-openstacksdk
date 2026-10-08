package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func glLocation() resource.CloudLocation {
	cloud, region, name, domainID, domainName := "cloud-original", "region-original", "project-original", "domain-ID", "domain-original"
	return resource.CloudLocation{Cloud: &cloud, RegionName: &region, Zone: json.RawMessage(`{"zone":"original"}`), Project: resource.CloudProject{ID: json.RawMessage(`"scope-original"`), Name: &name, DomainID: &domainID, DomainName: &domainName}}
}

func glMutateSource(client *gophercloud.ServiceClient, fact string) func() {
	switch fact {
	case "provider":
		prior := client.ProviderClient
		client.ProviderClient = &gophercloud.ProviderClient{}
		return func() { client.ProviderClient = prior }
	case "endpoint":
		prior := client.Endpoint
		client.Endpoint += "changed/"
		return func() { client.Endpoint = prior }
	case "resource base":
		prior := client.ResourceBase
		client.ResourceBase += "changed/"
		return func() { client.ResourceBase = prior }
	case "type":
		prior := client.Type
		client.Type = "wrong-role"
		return func() { client.Type = prior }
	case "microversion":
		prior := client.Microversion
		client.Microversion = "different"
		return func() { client.Microversion = prior }
	default:
		panic("unknown source fact")
	}
}

func TestGetVolumeLimitsFactoriesAndPreparedOptionsOwnCompleteLocation(t *testing.T) {
	for _, factory := range []string{"options", "location"} {
		t.Run(factory, func(t *testing.T) {
			location := glLocation()
			original, _ := json.Marshal(location)
			var option blockstorage.GetVolumeLimitsOption
			if factory == "options" {
				option = blockstorage.WithGetVolumeLimitsOptions(blockstorage.GetVolumeLimitsOpts{Location: &location})
			} else {
				option = blockstorage.WithGetVolumeLimitsLocation(location)
			}
			*location.Cloud = "caller changed"
			*location.RegionName = "caller changed"
			location.Zone[0] = '!'
			location.Project.ID[0] = '!'
			*location.Project.Name = "caller changed"
			*location.Project.DomainID = "caller changed"
			*location.Project.DomainName = "caller changed"
			first, err := blockstorage.PrepareGetVolumeLimitsOptions(context.Background(), option)
			if err != nil || first.Location == nil {
				t.Fatal(first, err)
			}
			raw, _ := json.Marshal(first.Location)
			if !glJSONSame(raw, original) {
				t.Fatal("factory retained caller state", string(raw), string(original))
			}
			*first.Location.Cloud = "prepared changed"
			first.Location.Zone[0] = '!'
			first.Location.Project.ID[0] = '!'
			second, err := blockstorage.PrepareGetVolumeLimitsOptions(context.Background(), option)
			if err != nil || second.Location == nil || second.Location == first.Location {
				t.Fatal(second, err)
			}
			raw, _ = json.Marshal(second.Location)
			if !glJSONSame(raw, original) {
				t.Fatal("factory reused prepared state", string(raw))
			}
			defaults, err := blockstorage.PrepareGetVolumeLimitsOptions(context.Background())
			if err != nil || defaults.Location != nil {
				t.Fatal(defaults, err)
			}
		})
	}
}

func TestGetVolumeLimitsOriginalsRunOnceInCapturedOrderAndRetainedConfigIsIsolated(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
	var calls atomic.Int32
	location := glLocation()
	var retained *blockstorage.GetVolumeLimitsOpts
	var order []int
	options := make([]blockstorage.GetVolumeLimitsOption, 3)
	options[0] = func(value *blockstorage.GetVolumeLimitsOpts) error {
		order = append(order, 1)
		value.Location = &location
		retained = value
		options[1] = func(*blockstorage.GetVolumeLimitsOpts) error { return errors.New("replacement option executed") }
		return nil
	}
	options[1] = func(*blockstorage.GetVolumeLimitsOpts) error { order = append(order, 2); return nil }
	options[2] = func(*blockstorage.GetVolumeLimitsOpts) error { order = append(order, 3); return nil }
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
			*retained.Location.Cloud = "retained changed"
			retained.Location.Project.ID[0] = '!'
			location.Zone[0] = '!'
			testcloud.JSON(w, 200, `{"project":{"id":"requested-project"}}`)
			return
		}
		volumeReadContractWire(t, r, glLimitsPath, "test-token")
		testcloud.JSON(w, 200, `{"limits":{}}`)
	})
	result, err := glCall(context.Background(), cinder, identity, "ref", options...)
	if err != nil || result == nil || calls.Load() != 2 || len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatal(result, err, calls.Load(), order)
	}
	view := glObject(t, result.Value)
	owned := glObject(t, view["location"])
	if string(owned["cloud"]) != `"cloud-original"` || string(owned["zone"]) != `{"zone":"original"}` || string(glObject(t, owned["project"])["id"]) != `"scope-original"` {
		t.Fatal("retained callback changed completed options", string(result.Value))
	}
}

func TestGetVolumeLimitsSelectedSourcesAreCapturedBeforeOriginalsAndViolationIsTerminal(t *testing.T) {
	for _, stage := range []string{"cinder", "identity"} {
		for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(stage+"/"+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
				var calls, late atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				selected := cinder
				if stage == "identity" {
					selected = identity
				}
				var restore func()
				result, err := glCall(context.Background(), cinder, identity, "ref", func(*blockstorage.GetVolumeLimitsOpts) error { restore = glMutateSource(selected, fact); return nil }, func(*blockstorage.GetVolumeLimitsOpts) error { late.Add(1); restore(); return nil })
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || late.Load() != 0 || calls.Load() != 0 {
					t.Fatal("source restored after observed violation", result, err, late.Load(), calls.Load())
				}
				glOperation(t, err)
			})
		}
	}
}

func TestGetVolumeLimitsPreflightCancellationAndInvalidSelectionsNeverSendHTTP(t *testing.T) {
	for _, which := range []string{"nil context", "canceled context", "nil cinder", "nil identity", "wrong cinder role", "wrong identity role", "reserved auth header", "reserved cookie header", "conflicting microversion", "bad input UTF8", "control input", "original error", "original cancellation", "nil original"} {
		t.Run(which, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls, originals atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("specific preflight cause")
			name := "ref"
			var inputContext context.Context = ctx
			option := blockstorage.GetVolumeLimitsOption(func(*blockstorage.GetVolumeLimitsOpts) error { originals.Add(1); return nil })
			switch which {
			case "nil context":
				inputContext = nil
			case "canceled context":
				cancel(cause)
			case "nil cinder":
				cinder = nil
			case "nil identity":
				identity = nil
			case "wrong cinder role":
				cinder.Type = "compute"
			case "wrong identity role":
				identity.Type = "compute"
			case "reserved auth header":
				cinder.MoreHeaders["X-Auth-Token"] = "caller"
			case "reserved cookie header":
				identity.MoreHeaders["Cookie"] = "caller"
			case "conflicting microversion":
				cinder.MoreHeaders["OpenStack-API-Version"] = "volume 3.39"
			case "bad input UTF8":
				name = string([]byte{0xff})
			case "control input":
				name = "name\n"
			case "original error":
				option = func(*blockstorage.GetVolumeLimitsOpts) error { originals.Add(1); return cause }
			case "original cancellation":
				option = func(*blockstorage.GetVolumeLimitsOpts) error { originals.Add(1); cancel(cause); return nil }
			case "nil original":
				option = nil
			}
			result, err := glCall(inputContext, cinder, identity, name, option)
			if err == nil || result != nil || calls.Load() != 0 {
				t.Fatal(result, err, calls.Load())
			}
			glOperation(t, err)
			if which == "canceled context" || which == "original cancellation" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal("cancel cause lost", err)
				}
			}
			if which == "original error" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if which == "nil context" || which == "canceled context" || which == "nil cinder" || which == "nil identity" {
				if originals.Load() != 0 {
					t.Fatal("original ran before invalid required source", originals.Load())
				}
			}
		})
	}
}

func TestGetVolumeLimitsHeaderSnapshotsStayOwnedWhileProviderTokensRemainLive(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
	var calls, options atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			glIdentityWire(t, r, glProjectsPath+"/ref", "option-token")
			cinder.MoreHeaders["x-source"] = "lookup changed"
			identity.MoreHeaders["x-identity"] = "lookup changed"
			cloud.Provider.SetToken("after-lookup-token")
			testcloud.JSON(w, 200, `{"project":{"id":"actual-project"}}`)
			return
		}
		volumeReadContractWire(t, r, glLimitsPath, "after-lookup-token")
		if r.URL.Query().Get("project_id") != "actual-project" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"limits":{}}`)
	})
	result, err := glCall(context.Background(), cinder, identity, "ref", func(*blockstorage.GetVolumeLimitsOpts) error {
		options.Add(1)
		cinder.MoreHeaders["x-source"] = "option changed"
		identity.MoreHeaders["x-identity"] = "option changed"
		cloud.Provider.SetToken("option-token")
		return nil
	})
	if err != nil || result == nil || calls.Load() != 2 || options.Load() != 1 {
		t.Fatal(result, err, calls.Load(), options.Load())
	}
}

func TestGetVolumeLimitsInvalidOwnedLocationFailsLocallyAfterProjectWithoutFinalProof(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
	var calls atomic.Int32
	location := glLocation()
	location.Project.ID = json.RawMessage(`not-json`)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
		w.Header().Set("X-Proof", "project")
		testcloud.JSON(w, 200, `{"project":{"id":"requested"}}`)
	})
	result, err := glCall(context.Background(), cinder, identity, "ref", blockstorage.WithGetVolumeLimitsLocation(location))
	var physical *resource.ResponseError
	if result == nil || result.Project == nil || result.Project.Project == nil || result.Project.Observed == nil || result.Project.Observed.Header.Get("X-Proof") != "project" || string(result.RequestedProjectID) != `"requested"` || result.Observed != nil || result.Limits != nil || result.Value != nil || calls.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) {
		t.Fatal(result, err, calls.Load())
	}
	glOperation(t, err)
}
