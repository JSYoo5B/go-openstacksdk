package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Pinned Cloud _compute.py:138–163, 243–253, 482–522, 769–802.
// These fixtures exercise composition; leaf paging/projection/version matrices
// remain in the existing keypair record/find/create/delete tests.
func TestComputeCloudKeypairQueryCompositionAndPresence(t *testing.T) {
	for _, tc := range []struct {
		name, operation, identity, filters, rows, nextRows, queryOwner, rawValue string
		wantNames                                                                []string
		wantInventory                                                            int
		member, missing, failure, lateFailure                                    bool
	}{
		{"eager list preserves page order and duplicates", "list", "", "", `{"keypair":{"name":"blue-2"}},{"keypair":{"name":"red"}}`, `{"keypair":{"name":"blue-1"}},{"keypair":{"name":"blue-2"}}`, "", "", []string{`"blue-2"`, `"red"`, `"blue-1"`, `"blue-2"`}, 4, false, false, false, false},
		{"glob preserves matching order and duplicates", "search", "blue-*", "", `{"keypair":{"name":"blue-2"}},{"keypair":{"name":"red"}}`, `{"keypair":{"name":"blue-1"}},{"keypair":{"name":"blue-2"}}`, "", "", []string{`"blue-2"`, `"blue-1"`, `"blue-2"`}, 4, false, false, false, false},
		{"logical numeric identity is stringified", "search", "17", "", `{"keypair":{"name":17}},{"keypair":{"name":null}}`, "", "", "", []string{`17`}, 2, false, false, false, false},
		{"server owner query is filtered again", "search", "", `{"user_id":"requested"}`, `{"keypair":{"name":"wrong","user_id":"foreign"}},{"keypair":{"name":"right","user_id":"requested"}}`, "", "requested", "", []string{`"right"`}, 2, false, false, false, false},
		{"earlier owner mismatch leaves later unknown unconsumed", "search", "", `{"user_id":"requested","unknown":true}`, `{"keypair":{"name":"key","user_id":"foreign"}}`, "", "requested", "", []string{}, 1, false, false, false, false},
		{"earlier unknown raises before owner mismatch", "search", "", `{"unknown":true,"user_id":"requested"}`, `{"keypair":{"name":"key","user_id":"foreign"}}`, "", "requested", "", nil, 1, false, false, true, false},
		{"first body filter removes row before cloud unknown predicate", "search", "", `{"type":"x509","unknown":true}`, `{"keypair":{"name":"key","type":"ssh"}}`, "", "", "", []string{}, 0, false, false, false, false},
		{"surviving row exposes missing cloud attribute", "search", "", `{"type":"x509","unknown":true}`, `{"keypair":{"name":"key","type":"x509"}}`, "", "", "", nil, 1, false, false, true, false},
		{"expression returns scalar without inventing record", "search", "blue-*", `"length(@)"`, `{"keypair":{"name":"blue-1"}},{"keypair":{"name":"red"}}`, "", "", `1`, nil, 2, false, false, false, false},
		{"absent get uses owner member lookup", "get", "key", "", "", "", "owner+team", "", []string{`"key"`}, -1, true, false, false, false},
		{"null get also uses owner member lookup", "get", "key", `null`, "", "", "owner+team", "", []string{`"key"`}, -1, true, false, false, false},
		{"empty object get ignores supplied owner and lists", "get", "key", `{}`, `{"keypair":{"name":"key","user_id":"foreign"}}`, "", "", "", []string{`"key"`}, 1, false, false, false, false},
		{"empty string get ignores supplied owner and lists", "get", "key", `""`, `{"keypair":{"name":"key","user_id":"foreign"}}`, "", "", "", []string{`"key"`}, 1, false, false, false, false},
		{"filtered get missing returns no value", "get", "missing", `{}`, `{"keypair":{"name":"other"}}`, "", "", "", nil, 1, false, true, false, false},
		{"filtered get duplicate waits for inventory", "get", "key", `{}`, `{"keypair":{"name":"key"}}`, `{"keypair":{"name":"key"}}`, "", "", nil, 2, false, false, true, false},
		{"expression first preserves false and has no record", "get", "", "\"`[false]`\"", `{"keypair":{"name":"key"}}`, "", "", `false`, nil, 1, false, false, false, false},
		{"late inventory failure cannot return early match", "search", "blue-*", `{}`, `{"keypair":{"name":"blue-1"}}`, `{"keypair":{"name":"unused"}}`, "", "", nil, 1, false, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			service := compute.New(client, compute.Dependencies{})
			var calls atomic.Int32
			query := make(url.Values)
			if tc.queryOwner != "" {
				query.Set("user_id", tc.queryOwner)
			}
			nextQuery := make(url.Values, len(query)+1)
			for key, values := range query {
				nextQuery[key] = append([]string(nil), values...)
			}
			nextQuery.Set("marker", "page-2")
			next := cloud.Server.URL + computeKeypairCreatePath + "?" + nextQuery.Encode()
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				index := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 {
					t.Error("query sent body", string(body), err)
				}
				wantPath, wantQuery := computeKeypairCreatePath, query
				if tc.member {
					wantPath += "/key"
				} else if index == 2 {
					wantQuery = nextQuery
				}
				if r.URL.Path != wantPath || r.URL.RawQuery != wantQuery.Encode() {
					t.Error("query changed member/owner or performed extra lookup", r.URL, wantPath, wantQuery)
				}
				w.Header().Set("X-Cloud-Keypair-Proof", tc.name)
				if tc.member {
					testcloud.JSON(w, 200, `{"keypair":{}}`)
					return
				}
				if index == 2 && tc.lateFailure {
					testcloud.JSON(w, 403, `{"error":"late inventory denied"}`)
					return
				}
				rows, links := tc.rows, ""
				if index == 2 {
					rows = tc.nextRows
				} else if tc.nextRows != "" {
					encoded, _ := json.Marshal(next)
					links = `,"keypairs_links":[{"rel":"next","href":` + string(encoded) + `}]`
				}
				testcloud.JSON(w, 200, `{"keypairs":[`+rows+`]`+links+`}`)
			})
			var options []compute.KeypairQueryOption
			if tc.filters != "" {
				options = append(options, compute.WithKeypairQueryFilters(json.RawMessage(tc.filters)))
			}
			if tc.operation == "get" {
				options = append(options, compute.WithKeypairQueryUserID("owner+team"))
			}
			var inventory, selected []*keypairs.KeypairRecord
			var value json.RawMessage
			var failure error
			switch tc.operation {
			case "list":
				result, err := service.ListKeypairs(context.Background(), options...)
				failure = err
				if result != nil {
					inventory, selected, value = result.Inventory, result.Keypairs, result.Value
				}
			case "search":
				result, err := service.SearchKeypairs(context.Background(), tc.identity, options...)
				failure = err
				if result != nil {
					inventory, selected, value = result.Inventory, result.Keypairs, result.Value
				}
			default:
				result, err := service.GetKeypair(context.Background(), tc.identity, options...)
				failure = err
				if result != nil {
					inventory, value = result.Inventory, result.Value
					if result.Keypair != nil {
						selected = []*keypairs.KeypairRecord{result.Keypair}
					}
				}
			}
			wantCalls := int32(1)
			if tc.nextRows != "" {
				wantCalls = 2
			}
			if (failure != nil) != tc.failure || calls.Load() != wantCalls || tc.wantInventory >= 0 && len(inventory) != tc.wantInventory {
				t.Fatal(failure, calls.Load(), len(inventory), tc.wantInventory, string(value))
			}
			if tc.failure {
				if value != nil || selected != nil {
					t.Fatal("failed composition returned completed selection", string(value), selected)
				}
				if tc.lateFailure {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(failure, &native) || native.Actual != 403 || native.Method != http.MethodGet || native.URL != next || string(native.Body) != `{"error":"late inventory denied"}` || native.ResponseHeader.Get("X-Cloud-Keypair-Proof") != tc.name {
						t.Fatal("late physical error lost", failure, native)
					}
				}
				return
			}
			if tc.missing {
				if value != nil || selected != nil {
					t.Fatal("missing filtered get synthesized a result", string(value), selected)
				}
				return
			}
			if tc.rawValue != "" {
				if string(value) != tc.rawValue || selected != nil {
					t.Fatal("expression invented record association or changed raw output", string(value), selected)
				}
				return
			}
			names := make([]string, len(selected))
			for i, record := range selected {
				if record == nil || record.Resource == nil {
					t.Fatal("selected row is not source-shaped", record)
				}
				names[i] = string(record.Resource.Body["name"])
			}
			if !reflect.DeepEqual(names, tc.wantNames) || value == nil || !json.Valid(value) {
				t.Fatal("order/duplicate/logical name or normalized result lost", names, tc.wantNames, string(value))
			}
		})
	}
}

func TestComputeCloudKeypairCreateAndDeleteOutcomes(t *testing.T) {
	for _, tc := range []struct{ name, publicKey, body string }{
		{"", "", `{"keypair":{"name":""}}`},
		{"../body/name", "", `{"keypair":{"name":"../body/name"}}`},
		{"imported", "ssh-ed25519 public", `{"keypair":{"name":"imported","public_key":"ssh-ed25519 public"}}`},
	} {
		t.Run("create "+tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.1"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				th.TestHeader(t, r, "X-Cloud-Create", "owned")
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != tc.body || r.URL.Path != computeKeypairCreatePath || r.URL.RawQuery != "" {
					t.Error("Cloud name/public-key transform changed or added lookup", string(body), tc.body, err, r.URL)
				}
				testcloud.JSON(w, 201, `{"keypair":{}}`)
			})
			record, err := compute.New(client, compute.Dependencies{}).CreateKeypair(context.Background(), tc.name, compute.WithCloudKeypairPublicKey(tc.publicKey), compute.WithCloudKeypairCreateHeader("X-Cloud-Create", "owned"))
			name, _ := json.Marshal(tc.name)
			if err != nil || record == nil || record.Resource == nil || string(record.Resource.Body["name"]) != string(name) || string(record.Resource.Body["type"]) != `"ssh"` || string(record.Resource.Body["location"]) != `null` || calls.Load() != 1 {
				t.Fatal("Cloud create lost seed/default or rejected body name", record, err, calls.Load())
			}
		})
	}
	for _, tc := range []struct {
		name                             string
		status                           int
		close404, wantDeleted, wantError bool
	}{
		{"accepted", 202, false, true, false},
		{"expanded passive success", 203, false, true, false},
		{"ordinary missing", 404, false, false, false},
		{"retry callback404 is terminal", 404, false, false, true},
		{"forbidden", 403, false, false, true},
		{"accepted close404 is not missing", 202, true, false, true},
		{"missing with source failure is terminal", 404, false, false, true},
		{"missing with cancellation is terminal", 404, false, false, true},
	} {
		t.Run("delete "+tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("Cloud delete cancellation")
			const body = "actual passive delete body"
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodDelete)
				if r.URL.Path != computeKeypairCreatePath+"/name" || r.URL.RawQuery != "" {
					t.Error("Cloud delete looked up name or changed target", r.URL)
				}
				w.Header().Set("X-Cloud-Delete-Proof", tc.name)
				testcloud.JSON(w, tc.status, body)
			})
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodDelete, URL: "nested", Body: []byte("original nested404")}
			var tracking *payloadContractTracking
			if tc.close404 {
				tracking = payloadContractTrack(cloud, nil, nested)
			}
			if tc.name == "missing with source failure is terminal" || tc.name == "missing with cancellation is terminal" {
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil {
						if tc.name == "missing with source failure is terminal" {
							client.ResourceBase = cloud.Server.URL + "/changed/"
						} else {
							cancel(cause)
						}
					}
					return response, err
				})
			}
			if tc.name != "ordinary missing" {
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					return err
				}
			}
			deleted, err := compute.New(client, compute.Dependencies{}).DeleteKeypair(ctx, "name")
			if deleted != tc.wantDeleted || (err != nil) != tc.wantError || calls.Load() != 1 {
				t.Fatal(deleted, err, calls.Load())
			}
			if tc.close404 {
				var proof *resource.ResponseError
				var original gophercloud.ErrUnexpectedResponseCode
				if errors.Is(err, resource.ErrNotFound) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Cloud-Delete-Proof") != tc.name || !errors.As(err, &original) || original.URL != "nested" || retries.Load() != 0 {
					t.Fatal("terminal receipt became False,nil", err, proof, original, retries.Load())
				}
				physical := tracking.last(t)
				if physical.closes.Load() != 1 {
					t.Fatal("accepted body was not closed once", physical.closes.Load())
				}
			} else if tc.name == "missing with source failure is terminal" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("source error converted to ordinary missing", err)
				}
			} else if tc.name == "missing with cancellation is terminal" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal("context cause converted to ordinary missing", err)
				}
			} else if tc.name == "retry callback404 is terminal" {
				var native gophercloud.ErrUnexpectedResponseCode
				if errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) || native.Actual != 404 || retries.Load() != 1 {
					t.Fatal("retry failure converted to ordinary missing", err, native, retries.Load())
				}
			} else if tc.wantError {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 403 || native.Method != http.MethodDelete || native.URL != cloud.Server.URL+computeKeypairCreatePath+"/name" || string(native.Body) != body || native.ResponseHeader.Get("X-Cloud-Delete-Proof") != tc.name {
					t.Fatal("nonmissing HTTP cause lost", err, native)
				}
			}
		})
	}
}

func TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors(t *testing.T) {
	t.Run("location is captured before options and filtered after inventory", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		cloudName, region := "original-cloud", "original-region"
		project := json.RawMessage(`"scope"`)
		location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Project: resource.CloudProject{ID: project}}
		wantLocation, err := location.ForResource(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		var calls, locations, callbacks atomic.Int32
		service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations.Add(1); return location, nil }})
		const row = `{"keypair":{"name":"key-a","location":{"cloud":"wire-cloud"}}}`
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			if r.URL.Path != computeKeypairCreatePath || r.URL.RawQuery != "" {
				t.Error("computed location became a wire parameter", r.URL)
			}
			testcloud.JSON(w, 200, `{"keypairs":[`+row+`,{"keypair":{"name":"key-b","location":false}}]}`)
		})
		result, err := service.SearchKeypairs(context.Background(), "key-*", compute.WithKeypairQueryFilters(json.RawMessage(`{"location":{"cloud":"original-cloud","project":{"id":"scope"}}}`)),
			func(_ *request.Config[compute.KeypairQueryOpts]) error {
				callbacks.Add(1)
				if locations.Load() != 1 {
					t.Error("location was not captured before options", locations.Load())
				}
				cloudName, region = "caller-changed", "caller-changed"
				project[1] = 'x'
				return nil
			})
		if err != nil || result == nil || len(result.Inventory) != 2 || len(result.Keypairs) != 2 || calls.Load() != 1 || locations.Load() != 1 || callbacks.Load() != 1 {
			t.Fatal("location was discarded by cloud second filter or recaptured", result, err, calls.Load(), locations.Load(), callbacks.Load())
		}
		for _, record := range result.Keypairs {
			if string(record.Resource.Body["location"]) != string(wantLocation) {
				t.Fatal("caller location mutation changed captured view", record.Resource, string(wantLocation))
			}
		}
		if string(result.Inventory[0].Envelope) != row || string(result.Inventory[0].Wire.Body["keypair"]) != `{"name":"key-a","location":{"cloud":"wire-cloud"}}` || string(result.Inventory[1].Wire.Body["keypair"]) != `{"name":"key-b","location":false}` {
			t.Fatal("computed location rewrote actual Wire/Envelope", result.Inventory)
		}
		result.Keypairs[0].Resource.Body["location"][0] = 'x'
		if string(result.Keypairs[1].Resource.Body["location"]) != string(wantLocation) || string(result.Inventory[0].Envelope) != row || !json.Valid(result.Value) {
			t.Fatal("location bytes alias another row, physical evidence or Value", result)
		}
	})
	t.Run("member and create capture location while delete does not read it", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		cloudName := "create-cloud"
		var calls, locations atomic.Int32
		service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
			locations.Add(1)
			return resource.CloudLocation{Cloud: &cloudName}, nil
		}})
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestHeader(t, r, "X-Auth-Token", "test-token")
			switch r.Method {
			case http.MethodPost:
				th.TestMethod(t, r, http.MethodPost)
				if r.URL.Path != computeKeypairCreatePath || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != `{"keypair":{"name":"created"}}` {
					t.Error("computed location entered create request", string(body), err)
				}
				testcloud.JSON(w, 201, `{"keypair":{"name":"created","location":false}}`)
			case http.MethodGet:
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != computeKeypairCreatePath+"/fetched" || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"keypair":{"name":"actual","location":{"response":17}}}`)
			case http.MethodDelete:
				th.TestMethod(t, r, http.MethodDelete)
				if r.URL.Path != computeKeypairCreatePath+"/fetched" || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.WriteHeader(202)
			default:
				t.Error("unexpected operation", r.Method)
				w.WriteHeader(500)
			}
		})
		created, err := service.CreateKeypair(context.Background(), "created", func(_ *request.Config[compute.CloudKeypairCreateOpts]) error {
			if locations.Load() != 1 {
				t.Error("create location was not captured before option", locations.Load())
			}
			cloudName = "member-cloud"
			return nil
		})
		var createLocation resource.CloudLocation
		if err != nil || created == nil || created.Resource == nil || json.Unmarshal(created.Resource.Body["location"], &createLocation) != nil || createLocation.Cloud == nil || *createLocation.Cloud != "create-cloud" || created.Wire == nil || string(created.Wire.Body["location"]) != `false` || string(created.Envelope) != `{"keypair":{"name":"created","location":false}}` || locations.Load() != 1 {
			t.Fatal("create location/result evidence changed", created, err, locations.Load())
		}
		fetched, err := service.GetKeypair(context.Background(), "fetched")
		var memberLocation resource.CloudLocation
		if err != nil || fetched == nil || fetched.Keypair == nil || fetched.Keypair.Resource == nil || fetched.Keypair.Wire == nil || json.Unmarshal(fetched.Keypair.Resource.Body["location"], &memberLocation) != nil || memberLocation.Cloud == nil || *memberLocation.Cloud != "member-cloud" || string(fetched.Keypair.Wire.Body["location"]) != `{"response":17}` || locations.Load() != 2 {
			t.Fatal("member did not attach one current location snapshot", fetched, err, locations.Load())
		}
		deleted, err := service.DeleteKeypair(context.Background(), "fetched")
		if err != nil || !deleted || locations.Load() != 2 || calls.Load() != 3 {
			t.Fatal("deletion read location or extra resource", deleted, err, locations.Load(), calls.Load())
		}
	})
	t.Run("captured options and callback maps are reusable", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.1"
		service := compute.New(client, compute.Dependencies{})
		filters, version, paginated := json.RawMessage(`{"user_id":"owner"}`), "2.55", false
		input := compute.KeypairQueryOpts{Filters: &filters, UserID: "ignored", Microversion: &version, MaxItems: 1, Paginated: &paginated}
		captured := compute.WithKeypairQueryOptions(input)
		filters[2], version, paginated, input.UserID = 'x', "2.1", true, "changed-owner"
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-Auth-Token", "cloud-live")
			th.TestHeader(t, r, "X-Cloud-Read", "owned")
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
			if r.URL.Path != computeKeypairCreatePath || !reflect.DeepEqual(r.URL.Query(), url.Values{"user_id": {"owner"}, "limit": {"1"}}) {
				t.Error("captured pointer/query controls changed", r.URL)
			}
			testcloud.JSON(w, 200, `{"keypairs":[{"keypair":{"name":"owned","type":"x509","user_id":"owner"}}]}`)
		})
		for repeat := 0; repeat < 2; repeat++ {
			headers := map[string]string{"X-Cloud-Read": "owned"}
			result, err := service.SearchKeypairs(context.Background(), "owned", captured,
				func(config *request.Config[compute.KeypairQueryOpts]) error {
					callbacks.Add(1)
					config.Headers = headers
					return nil
				},
				func(_ *request.Config[compute.KeypairQueryOpts]) error {
					callbacks.Add(1)
					headers["X-Cloud-Read"] = "caller-changed"
					cloud.Provider.SetToken("cloud-live")
					return nil
				})
			if err != nil || result == nil || len(result.Keypairs) != 1 || len(result.Inventory) != 1 || string(result.Keypairs[0].Resource.Body["name"]) != `"owned"` || client.Microversion != "2.1" {
				t.Fatal("Cloud wrapper replayed options or lost owned state", result, err, client)
			}
		}
		if calls.Load() != 2 || callbacks.Load() != 4 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"nil option", "negative max", "invalid filters", "source", "cancel"} {
		t.Run("preflight "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			service := compute.New(client, compute.Dependencies{})
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, original := errors.New("Cloud keypair cancellation"), errors.New("Cloud option error")
			var calls, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			var option compute.KeypairQueryOption
			switch mode {
			case "negative max":
				option = compute.WithKeypairQueryMaxItems(-1)
			case "invalid filters":
				option = compute.WithKeypairQueryFilters(json.RawMessage(`{`))
			case "source":
				option = func(_ *request.Config[compute.KeypairQueryOpts]) error {
					client.ResourceBase = cloud.Server.URL + "/changed/"
					return nil
				}
			case "cancel":
				option = func(_ *request.Config[compute.KeypairQueryOpts]) error { cancel(cause); return original }
			}
			_, err := service.SearchKeypairs(ctx, "key", option, func(_ *request.Config[compute.KeypairQueryOpts]) error { later.Add(1); return nil })
			if err == nil || calls.Load() != 0 {
				t.Fatal("invalid Cloud preparation reached HTTP", err, calls.Load())
			}
			if mode == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, original)) {
				t.Fatal("Cloud context/option cause lost", err)
			}
			if (mode == "nil option" || mode == "source" || mode == "cancel") && later.Load() != 0 {
				t.Fatal("later callback ran after preflight failure", later.Load())
			}
		})
	}
	t.Run("eager read terminal404 never becomes absence", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		const body = `{"keypairs":[{"keypair":{"name":"key"}}]}`
		var calls, retries atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			w.Header().Set("X-Cloud-Read-Proof", "accepted")
			testcloud.JSON(w, 200, body)
		})
		nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested", Body: []byte("original nested404")}
		tracking := payloadContractTrack(cloud, nil, nested)
		cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retries.Add(1)
			return err
		}
		result, err := compute.New(client, compute.Dependencies{}).GetKeypair(context.Background(), "key", compute.WithKeypairQueryFilters(json.RawMessage(`{}`)))
		var proof *resource.ResponseError
		var original gophercloud.ErrUnexpectedResponseCode
		if err == nil || errors.Is(err, resource.ErrNotFound) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Cloud-Read-Proof") != "accepted" || !errors.As(err, &original) || original.URL != "nested" || result != nil && (result.Value != nil || result.Keypair != nil) || calls.Load() != 1 || retries.Load() != 0 {
			t.Fatal("eager accepted failure converted to missing/completed result", result, err, proof, original, calls.Load(), retries.Load())
		}
		physical := tracking.last(t)
		if physical.closes.Load() != 1 {
			t.Fatal("accepted inventory body was not closed once", physical.closes.Load())
		}
	})
}
