package gophercloudsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

func connectionCloudLimitsOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "GetVolumeLimits" || operation.Cause == nil {
		t.Fatal("wrong outer cloud limits operation", err, operation)
	}
}

func connectionCloudLimitsProject(t *testing.T, result *blockstorage.GetVolumeLimitsResult, rawID string, status int) {
	t.Helper()
	if result == nil || result.Project == nil || result.Project.Project == nil || result.Project.Observed == nil || result.Project.Observed.StatusCode != status || len(result.Project.Pages) != 0 || string(result.Project.ID) != rawID || string(result.RequestedProjectID) != rawID || result.Project.SeededID {
		t.Fatal("completed project proof was not retained independently", result)
	}
}

func TestConnectionGetVolumeLimitsOriginalsPrecedeIdentityResolutionAndCachedCinder(t *testing.T) {
	cases := []struct {
		name, rawID string
		values      []string
	}{
		{name: "string", rawID: `"returned-project"`, values: []string{"returned-project"}},
		{name: "boolean", rawID: `false`, values: []string{"False"}},
		{name: "nested Requests doseq", rawID: `[[1,2],[],null]`, values: []string{"1", "2"}},
		{name: "explicit null", rawID: `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, projectReads, limitsReads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				index := locates.Add(1)
				if callbacks.Load() != 1 || opts.Version != 3 {
					t.Error("originals must precede necessary v3 service selection", callbacks.Load(), opts)
				}
				if index == 1 && opts.Type == "identity" {
					return cloud.Server.URL + "/cloud-limits/identity/v3/", nil
				}
				if index == 2 && opts.Type == "block-storage" && projectReads.Load() == 1 {
					return cloud.Server.URL + "/cloud-limits/cinder/v3/scope/", nil
				}
				return "", errors.New("Cinder selected before completed Identity lookup or unrelated service selected")
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			var cachedIdentity, cachedCinder *gophercloud.ServiceClient
			cloud.Mux.HandleFunc("GET /cloud-limits/identity/v3/projects/project-input", func(w http.ResponseWriter, r *http.Request) {
				index := projectReads.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-limits-token" || index == 2 && r.Header.Get("X-Original") != "after-original" {
					t.Error(r.URL, r.Header)
				}
				w.Header().Set("X-Project-Proof", "actual")
				testcloud.JSON(w, 203, `{"project":{"id":`+tc.rawID+`,"name":"returned-name","unknown":9007199254740993}}`)
			})
			cloud.Mux.HandleFunc("GET /cloud-limits/cinder/v3/scope/limits", func(w http.ResponseWriter, r *http.Request) {
				index := limitsReads.Add(1)
				q := r.URL.Query()
				if len(q) != 0 && (len(q) != 1 || len(q["project_id"]) != len(tc.values) || strings.Join(q["project_id"], "|") != strings.Join(tc.values, "|")) || len(tc.values) != 0 && len(q) != 1 {
					t.Error("actual raw project ID was not encoded with Requests query semantics", r.URL, tc.values)
				}
				if len(tc.values) == 0 && r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-limits-token" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.60" || index == 2 && r.Header.Get("X-Original") != "after-original" {
					t.Error(r.URL, r.Header)
				}
				body, readErr := io.ReadAll(r.Body)
				if readErr != nil || len(body) != 0 {
					t.Error("limits GET must be bodyless", string(body), readErr)
				}
				w.Header().Set("X-Limits-Proof", "actual")
				testcloud.JSON(w, 203, `{"limits":{"absolute":{"maxTotalVolumes":"12"},"rate":[],"unknown":9007199254740993}}`)
			})
			for range 2 {
				identity := "project-input"
				result, err := conn.GetVolumeLimits(context.Background(), blockstorage.GetVolumeLimitsRequest{NameOrID: identity}, func(*blockstorage.GetVolumeLimitsOpts) error {
					callbacks.Add(1)
					cloud.Provider.SetToken("live-limits-token")
					identity = "caller-mutated"
					if cachedIdentity != nil {
						cachedIdentity.MoreHeaders = map[string]string{"X-Original": "after-original"}
					}
					if cachedCinder != nil {
						cachedCinder.MoreHeaders = map[string]string{"X-Original": "after-original"}
					}
					return nil
				})
				if err != nil {
					t.Fatal(result, err)
				}
				connectionCloudLimitsProject(t, result, tc.rawID, 203)
				if result.Project.Observed.Header.Get("X-Project-Proof") != "actual" || result.Observed == nil || result.Observed.StatusCode != 203 || result.Observed.Header.Get("X-Limits-Proof") != "actual" || result.Limits == nil || result.Limits.StatusCode != 203 || string(result.Limits.Body["unknown"]) != "9007199254740993" {
					t.Fatal(result)
				}
				absolute := connectionReadFields(t, connectionReadFields(t, result.Value)["absolute"])
				if string(absolute["max_total_volumes"]) != "12" || len(absolute) != 13 {
					t.Fatal(string(result.Value))
				}
				projectProof, limitsProof := bytes.Clone(result.Project.Observed.Body), bytes.Clone(result.Observed.Body)
				result.Project.ID[0] = '!'
				result.Project.Project.Body["id"][0] = '!'
				result.Value[0] = '!'
				result.Limits.Body["unknown"][0] = '!'
				result.Limits.Header.Set("X-Limits-Proof", "caller-mutated")
				if string(result.RequestedProjectID) != tc.rawID || !bytes.Equal(projectProof, result.Project.Observed.Body) || !bytes.Equal(limitsProof, result.Observed.Body) || result.Observed.Header.Get("X-Limits-Proof") != "actual" {
					t.Fatal("logical outputs alias physical or requested-ID evidence", result)
				}
				identityService, err := conn.IdentityV3(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				cinderService, err := conn.BlockStorageV3(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if cachedIdentity != nil && cachedIdentity != identityService.RawClient() || cachedCinder != nil && cachedCinder != cinderService.RawClient() {
					t.Fatal("cached service identity changed")
				}
				cachedIdentity, cachedCinder = identityService.RawClient(), cinderService.RawClient()
			}
			if callbacks.Load() != 2 || locates.Load() != 2 || projectReads.Load() != 2 || limitsReads.Load() != 2 {
				t.Fatal(callbacks.Load(), locates.Load(), projectReads.Load(), limitsReads.Load())
			}
		})
	}
}

func TestConnectionGetVolumeLimitsEmptyInputSkipsIdentityAndImplicitProjectFilter(t *testing.T) {
	for _, version := range []string{"", "3.0", "3.38", "3.60", "latest"} {
		t.Run("selected="+version, func(t *testing.T) {
			cloud := testcloud.New(t)
			if err := connectionReadRecordScope(cloud.Provider, "recorded-scope"); err != nil {
				t.Fatal(err)
			}
			var locates, reads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("empty input selected Identity/unrelated service", opts)
					return "", errors.New("unexpected service")
				}
				return cloud.Server.URL + "/cloud-limits/cinder/v3/scope/", nil
			}
			options := []sdk.ConnectionOption{sdk.WithRegion("owned-region")}
			if version != "" && version != "latest" {
				options = append(options, sdk.WithMicroversion(sdk.BlockStorage, version))
			}
			conn, err := sdk.FromProvider(cloud.Provider, options...)
			if err != nil {
				t.Fatal(err)
			}
			if version == "latest" {
				// Connection exact-version options remain numeric. This workflow
				// also preserves a symbolic version already selected on its cache.
				service, err := conn.BlockStorageV3(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				service.RawClient().Microversion = "latest"
			}
			cloud.Mux.HandleFunc("GET /cloud-limits/cinder/v3/scope/limits", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Volume-API-Version") != version {
					t.Error("implicit scope must not become a project filter or upgraded version", r.URL, r.Header)
				}
				testcloud.JSON(w, 203, `{"location":"wire","project_id":"wire-project","availability_zone":"wire-zone"}`)
			})
			result, err := conn.GetVolumeLimits(context.Background(), blockstorage.GetVolumeLimitsRequest{})
			if err != nil || result == nil || result.Project != nil || result.RequestedProjectID != nil || result.Observed == nil || result.Limits == nil || locates.Load() != 1 || reads.Load() != 1 {
				t.Fatal(result, err, locates.Load(), reads.Load())
			}
			view := connectionReadFields(t, result.Value)
			location := connectionReadFields(t, view["location"])
			project := connectionReadFields(t, location["project"])
			if len(view) != 5 || string(view["absolute"]) != "null" || string(view["rate"]) != "null" || string(project["id"]) != `"recorded-scope"` || string(location["region_name"]) != `"owned-region"` || string(location["zone"]) != "null" || string(result.Limits.Body["project_id"]) != `"wire-project"` {
				t.Fatal(string(result.Value), result.Limits)
			}
		})
	}
}

func TestConnectionGetVolumeLimitsProjectCompletesBeforeCinderGetterFailure(t *testing.T) {
	for _, kind := range []string{"getter error", "getter cancellation"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, getterError := errors.New("Cinder selection canceled"), errors.New("Cinder locator failed")
			var callbacks, locates, projects atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 1 || opts.Version != 3 {
					t.Error(callbacks.Load(), opts)
				}
				if opts.Type == "identity" {
					return cloud.Server.URL + "/cloud-limits/identity/v3/", nil
				}
				if opts.Type != "block-storage" || projects.Load() != 1 {
					t.Error("Cinder selection overtook project resolution", opts, projects.Load())
				}
				if kind == "getter cancellation" {
					cancel(cause)
				}
				return "", getterError
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("GET /cloud-limits/identity/v3/projects/input", func(w http.ResponseWriter, r *http.Request) {
				projects.Add(1)
				w.Header().Set("X-Project-Proof", "complete")
				testcloud.JSON(w, 203, `{"project":{"id":false,"name":"actual"}}`)
			})
			result, err := conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{NameOrID: "input"}, func(*blockstorage.GetVolumeLimitsOpts) error { callbacks.Add(1); return nil })
			if !errors.Is(err, getterError) || callbacks.Load() != 1 || locates.Load() != 2 || projects.Load() != 1 {
				t.Fatal(result, err, callbacks.Load(), locates.Load(), projects.Load())
			}
			connectionCloudLimitsOperation(t, err)
			connectionCloudLimitsProject(t, result, "false", 203)
			var physical *resource.ResponseError
			if errors.As(err, &physical) || result.Observed != nil || result.Limits != nil || result.Value != nil || result.Project.Observed.Header.Get("X-Project-Proof") != "complete" {
				t.Fatal("local getter failure borrowed prior accepted response", result, err)
			}
			if kind == "getter cancellation" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal("lost cancellation context/cause", err)
			}
		})
	}
}

func TestConnectionGetVolumeLimitsStrictProjectErrorsNeverSelectCinder(t *testing.T) {
	for _, kind := range []string{"missing", "duplicate", "terminal member"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates, members, pages, unused atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if opts.Type != "identity" || opts.Version != 3 {
					t.Error("Cinder selected after failed project find", opts)
					return "", errors.New("unexpected Cinder")
				}
				return cloud.Server.URL + "/cloud-limits/identity/v3/", nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("GET /cloud-limits/identity/v3/projects/input", func(w http.ResponseWriter, r *http.Request) {
				members.Add(1)
				status := 404
				if kind == "terminal member" {
					status = 503
				}
				testcloud.JSON(w, status, `{"error":"actual member rejection"}`)
			})
			cloud.Mux.HandleFunc("GET /cloud-limits/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
				pages.Add(1)
				if r.URL.RawQuery != "name=input" {
					t.Error(r.URL)
				}
				if kind == "duplicate" {
					testcloud.JSON(w, 203, `{"projects":[{"id":"first","name":"input"},{"id":"second","name":"input"}],"next":"?marker=unused"}`)
				} else {
					testcloud.JSON(w, 203, `{"projects":[],"next":"?marker=unused"}`)
				}
			})
			cloud.Mux.HandleFunc("GET /cloud-limits/cinder/v3/scope/limits", func(w http.ResponseWriter, r *http.Request) {
				unused.Add(1)
				t.Error("unexpected limits GET")
				w.WriteHeader(500)
			})
			result, err := conn.GetVolumeLimits(context.Background(), blockstorage.GetVolumeLimitsRequest{NameOrID: "input"})
			if err == nil || locates.Load() != 1 || members.Load() != 1 || unused.Load() != 0 {
				t.Fatal(result, err, locates.Load(), members.Load(), unused.Load())
			}
			connectionCloudLimitsOperation(t, err)
			if kind == "terminal member" {
				if !gophercloud.ResponseCodeIs(err, 503) || pages.Load() != 0 {
					t.Fatal(result, err, pages.Load())
				}
			} else {
				want := resource.ErrNotFound
				if kind == "duplicate" {
					want = resource.ErrAmbiguous
				}
				if !errors.Is(err, want) || pages.Load() != 1 || result == nil || result.Project == nil || len(result.Project.Pages) != 1 || result.Project.Observed != nil || result.Project.Pages[0].StatusCode != 203 {
					t.Fatal(result, err, pages.Load())
				}
			}
			if result != nil && (result.Observed != nil || result.Limits != nil || result.Value != nil || result.RequestedProjectID != nil) {
				t.Fatal("failed lookup published final output", result)
			}
		})
	}
}

func TestConnectionGetVolumeLimitsOwnsFactoriesOriginalSliceAndRetainedLocation(t *testing.T) {
	for _, factory := range []string{"bulk", "location", "callbacks"} {
		t.Run(factory, func(t *testing.T) {
			cloud := testcloud.New(t)
			defaultName, cloudName, name := "connection-default", "owned-cloud", "owned-name"
			defaults := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"default"`), Name: &defaultName}}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(defaults), sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/cloud-limits/identity/v3/"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cloud-limits/cinder/v3/scope/"))
			if err != nil {
				t.Fatal(err)
			}
			location := resource.CloudLocation{Cloud: &cloudName, Zone: json.RawMessage(`{"owned":true}`), Project: resource.CloudProject{ID: json.RawMessage(`"owned-scope"`), Name: &name}}
			var options []blockstorage.GetVolumeLimitsOption
			var retained *blockstorage.GetVolumeLimitsOpts
			first, second, injected := 0, 0, 0
			mutateCaller := func() {
				cloudName, name = "caller-cloud", "caller-name"
				location.Zone[0] = '!'
				location.Project.ID[0] = '!'
			}
			switch factory {
			case "bulk":
				options = []blockstorage.GetVolumeLimitsOption{blockstorage.WithGetVolumeLimitsOptions(blockstorage.GetVolumeLimitsOpts{Location: &location})}
			case "location":
				options = []blockstorage.GetVolumeLimitsOption{blockstorage.WithGetVolumeLimitsLocation(location)}
			default:
				options = make([]blockstorage.GetVolumeLimitsOption, 2)
				options[0] = func(o *blockstorage.GetVolumeLimitsOpts) error {
					first++
					o.Location = &location
					retained = o
					options[1] = func(*blockstorage.GetVolumeLimitsOpts) error {
						injected++
						return errors.New("replacement option must not run")
					}
					return nil
				}
				options[1] = func(o *blockstorage.GetVolumeLimitsOpts) error {
					second++
					mutateCaller()
					if o.Location == nil || o.Location.Cloud == nil || *o.Location.Cloud != "owned-cloud" || string(o.Location.Project.ID) != `"owned-scope"` || string(o.Location.Zone) != `{"owned":true}` {
						t.Error("first callback state was not owned", o)
					}
					return nil
				}
			}
			if factory != "callbacks" {
				mutateCaller()
			}
			cloud.Mux.HandleFunc("GET /cloud-limits/identity/v3/projects/input", func(w http.ResponseWriter, r *http.Request) {
				if retained != nil {
					retained.Location.Zone[0] = '!'
					retained.Location.Project.ID[0] = '!'
					*retained.Location.Cloud = "retained-pointer-change"
				}
				testcloud.JSON(w, 200, `{"project":{"id":"foreign-query"}}`)
			})
			cloud.Mux.HandleFunc("GET /cloud-limits/cinder/v3/scope/limits", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("project_id") != "foreign-query" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"limits":{"location":"wire-location","project_id":"wire-project","availability_zone":"wire-zone"}}`)
			})
			result, err := conn.GetVolumeLimits(context.Background(), blockstorage.GetVolumeLimitsRequest{NameOrID: "input"}, options...)
			if err != nil {
				t.Fatal(result, err)
			}
			locationFields := connectionReadFields(t, connectionReadFields(t, result.Value)["location"])
			project := connectionReadFields(t, locationFields["project"])
			if string(locationFields["cloud"]) != `"owned-cloud"` || string(locationFields["zone"]) != `{"owned":true}` || string(project["id"]) != `"owned-scope"` || string(project["name"]) != `"owned-name"` {
				t.Fatal("query or caller state replaced owned location", string(result.Value))
			}
			if factory == "callbacks" && (first != 1 || second != 1 || injected != 0) {
				t.Fatal(first, second, injected)
			}
			current, err := conn.CurrentLocation()
			if err != nil || current.Project.Name == nil || *current.Project.Name != "connection-default" || string(current.Project.ID) != `"default"` || current.Cloud != nil {
				t.Fatal("operation changed Connection defaults", current, err)
			}
		})
	}
}

func TestConnectionGetVolumeLimitsCapturesRecordedScopeAfterOptionsBeforeProjectResolution(t *testing.T) {
	cloud := testcloud.New(t)
	if err := connectionReadRecordScope(cloud.Provider, "before-original"); err != nil {
		t.Fatal(err)
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("owned-region"), sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/cloud-limits/identity/v3/"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cloud-limits/cinder/v3/scope/"))
	if err != nil {
		t.Fatal(err)
	}
	var callbacks, projectReads, limitsReads atomic.Int32
	cloud.Mux.HandleFunc("GET /cloud-limits/identity/v3/projects/input", func(w http.ResponseWriter, r *http.Request) {
		projectReads.Add(1)
		if err := connectionReadRecordScope(cloud.Provider, "between-stages"); err != nil {
			t.Error(err)
		}
		// Native SetToken deliberately clears the recorded authentication.
		// Keep the recorded next-call scope while installing its live token.
		recorded := cloud.Provider.GetAuthResult().(v3.CreateResult)
		recorded.Header.Set("X-Subject-Token", "between-stages-token")
		if err := cloud.Provider.SetTokenAndAuthResult(recorded); err != nil {
			t.Error(err)
		}
		testcloud.JSON(w, 200, `{"project":{"id":"foreign-query"}}`)
	})
	cloud.Mux.HandleFunc("GET /cloud-limits/cinder/v3/scope/limits", func(w http.ResponseWriter, r *http.Request) {
		limitsReads.Add(1)
		if r.URL.Query().Get("project_id") != "foreign-query" || r.Header.Get("X-Auth-Token") != "between-stages-token" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"limits":{"project_id":"wire-owner","location":"wire","availability_zone":"wire-zone"}}`)
	})
	for index := range 2 {
		result, err := conn.GetVolumeLimits(context.Background(), blockstorage.GetVolumeLimitsRequest{NameOrID: "input"}, func(*blockstorage.GetVolumeLimitsOpts) error {
			callbacks.Add(1)
			if index == 0 {
				return connectionReadRecordScope(cloud.Provider, "after-original")
			}
			return nil
		})
		if err != nil {
			t.Fatal(result, err)
		}
		want := `"after-original"`
		if index == 1 {
			want = `"between-stages"`
		}
		location := connectionReadFields(t, connectionReadFields(t, result.Value)["location"])
		project := connectionReadFields(t, location["project"])
		if string(project["id"]) != want || string(project["name"]) != "null" || string(location["region_name"]) != `"owned-region"` || string(location["zone"]) != "null" || string(result.RequestedProjectID) != `"foreign-query"` {
			t.Fatal("location scope reread or replaced by query", string(result.Value), result)
		}
	}
	if callbacks.Load() != 2 || projectReads.Load() != 2 || limitsReads.Load() != 2 {
		t.Fatal(callbacks.Load(), projectReads.Load(), limitsReads.Load())
	}
}

func TestConnectionGetVolumeLimitsPreflightAndGetterErrorsKeepOuterOperationAndCause(t *testing.T) {
	kinds := []string{"nil context", "nil connection", "already canceled", "nil option", "option error", "option cancellation", "Identity getter error", "Identity getter cancellation", "empty Cinder getter error"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, optionError, getterError := errors.New("cloud limits canceled"), errors.New("cloud limits original failed"), errors.New("cloud limits locator failed")
			callbacks, locates := 0, 0
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates++
				expected := "identity"
				if kind == "empty Cinder getter error" {
					expected = "block-storage"
				}
				if callbacks != 1 || opts.Type != expected || opts.Version != 3 {
					t.Error(callbacks, opts)
				}
				if kind == "Identity getter cancellation" {
					cancel(cause)
				}
				return "", getterError
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			options := []blockstorage.GetVolumeLimitsOption{func(*blockstorage.GetVolumeLimitsOpts) error {
				callbacks++
				if kind == "option cancellation" {
					cancel(cause)
				}
				if kind == "option error" || kind == "option cancellation" {
					return optionError
				}
				return nil
			}}
			identity := "input"
			want := resource.ErrInvalidOption
			wantCallbacks, wantLocates := 0, 0
			switch kind {
			case "nil context":
				ctx = nil
			case "nil connection":
				conn = nil
			case "already canceled":
				cancel(cause)
				want = context.Canceled
			case "nil option":
				options = append(options, nil)
				wantCallbacks = 1
			case "option error":
				want = optionError
				wantCallbacks = 1
			case "option cancellation":
				want = context.Canceled
				wantCallbacks = 1
			default:
				want = getterError
				wantCallbacks, wantLocates = 1, 1
				if kind == "empty Cinder getter error" {
					identity = ""
				}
			}
			result, err := conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{NameOrID: identity}, options...)
			if !errors.Is(err, want) || callbacks != wantCallbacks || locates != wantLocates {
				t.Fatal(result, err, callbacks, locates)
			}
			if result != nil && (result.Project != nil || result.RequestedProjectID != nil || result.Observed != nil || result.Limits != nil || result.Value != nil) {
				t.Fatal("preflight invented phase proof", result)
			}
			connectionCloudLimitsOperation(t, err)
			var physical *resource.ResponseError
			if errors.As(err, &physical) {
				t.Fatal("preflight/getter error borrowed response evidence", err)
			}
			if strings.Contains(kind, "cancellation") || kind == "already canceled" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal("lost cancellation causes", err)
				}
			}
			if kind == "option cancellation" && !errors.Is(err, optionError) {
				t.Fatal("lost original error", err)
			}
		})
	}
}

func TestConnectionGetVolumeLimitsSeparatesLocalRejectedAndAcceptedFinalProof(t *testing.T) {
	for _, kind := range []string{"invalid owned location", "native final503", "accepted invalid UTF8", "accepted nested self"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/cloud-limits/identity/v3/"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cloud-limits/cinder/v3/scope/"))
			if err != nil {
				t.Fatal(err)
			}
			var projects, limits atomic.Int32
			cloud.Mux.HandleFunc("GET /cloud-limits/identity/v3/projects/input", func(w http.ResponseWriter, r *http.Request) {
				projects.Add(1)
				w.Header().Set("X-Project-Proof", "actual")
				testcloud.JSON(w, 200, `{"project":{"id":"foreign-query"}}`)
			})
			cloud.Mux.HandleFunc("GET /cloud-limits/cinder/v3/scope/limits", func(w http.ResponseWriter, r *http.Request) {
				limits.Add(1)
				w.Header().Set("X-Limits-Proof", "actual")
				switch kind {
				case "native final503":
					testcloud.JSON(w, 503, `{"error":"actual final rejection"}`)
				case "accepted invalid UTF8":
					w.WriteHeader(203)
					_, _ = w.Write([]byte{0xff, 0x00})
				default:
					testcloud.JSON(w, 202, `{"limits":{"absolute":{"self":null}}}`)
				}
			})
			var options []blockstorage.GetVolumeLimitsOption
			if kind == "invalid owned location" {
				options = []blockstorage.GetVolumeLimitsOption{blockstorage.WithGetVolumeLimitsLocation(resource.CloudLocation{Zone: json.RawMessage(`not-json`)})}
			}
			result, err := conn.GetVolumeLimits(context.Background(), blockstorage.GetVolumeLimitsRequest{NameOrID: "input"}, options...)
			if err == nil || projects.Load() != 1 {
				t.Fatal(result, err, projects.Load())
			}
			connectionCloudLimitsOperation(t, err)
			connectionCloudLimitsProject(t, result, `"foreign-query"`, 200)
			if result.Limits != nil || result.Value != nil || result.Project.Observed.Header.Get("X-Project-Proof") != "actual" {
				t.Fatal("failed final stage published logical output", result)
			}
			var physical *resource.ResponseError
			switch kind {
			case "invalid owned location":
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || result.Observed != nil || limits.Load() != 0 {
					t.Fatal("local location error borrowed completed project response", result, err, limits.Load())
				}
			case "native final503":
				if !gophercloud.ResponseCodeIs(err, 503) || errors.As(err, &physical) || result.Observed != nil || limits.Load() != 1 {
					t.Fatal(result, err, limits.Load())
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Limits-Proof") != "actual" || !strings.Contains(string(native.Body), "actual final rejection") {
					t.Fatal("lost rejected native evidence", err)
				}
			default:
				want := 203
				if kind == "accepted nested self" {
					want = 202
				}
				if !errors.As(err, &physical) || physical.StatusCode != want || physical.Header.Get("X-Limits-Proof") != "actual" || result.Observed == nil || result.Observed.StatusCode != want || !bytes.Equal(physical.Body, result.Observed.Body) || limits.Load() != 1 {
					t.Fatal("accepted final error borrowed old project status", result, err, physical, limits.Load())
				}
			}
		})
	}
}
