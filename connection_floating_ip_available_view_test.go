package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Keep the original fixture's physical request/deadline/version assertions.
// Only passive reply rows are replaced; these cases install no Close hook.
func availableViewReply(t *testing.T, cloud *testcloud.Cloud, reply func(*http.Request) (string, bool)) {
	t.Helper()
	availableReply(t, cloud, func(r *http.Request) (int, string, bool) {
		body, replace := reply(r)
		return 0, body, replace
	})
}

// Reuse request/deadline/source assertions while varying body and actual code.
func availableReply(t *testing.T, cloud *testcloud.Cloud, reply func(*http.Request) (int, string, bool)) {
	t.Helper()
	next := cloud.Provider.HTTPClient.Transport
	cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
		response, err := next.RoundTrip(r)
		if err != nil || response == nil {
			return response, err
		}
		if code, body, replace := reply(r); replace {
			if err := response.Body.Close(); err != nil {
				return response, err
			}
			response.Body = io.NopCloser(strings.NewReader(body))
			if code != 0 {
				response.StatusCode = code
			}
		}
		return response, nil
	})
}

func availableViewLocation(t *testing.T, record *compute.FloatingIPRecord) resource.CloudLocation {
	t.Helper()
	if record == nil || record.Resource == nil {
		t.Fatal("missing returned cloud view", record)
	}
	var value resource.CloudLocation
	if err := json.Unmarshal(record.Resource.Body["location"], &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAvailableIPNovaViewPreservesWireAndStrictAliases(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone} {
		for _, strict := range []bool{false, true} {
			for _, allocated := range []bool{false, true} {
				t.Run(fmt.Sprint(source, "/strict=", strict, "/allocated=", allocated), func(t *testing.T) {
					cloud, conn, state := connectionAvailableFixture(t, source)
					state.novaFree = !allocated
					id, address := "9007199254740993", "198.51.100.10"
					if allocated {
						id, address = "29", "198.51.100.11"
					}
					row := fmt.Sprintf(`{"id":%s,"ip":%q,"pool":"public","instance_id":null,"status":"DOWN","fixed_ip_address":null,"fixed_ip":"10.0.0.9","project_id":null,"tenant_id":"foreign","owner":"other","vendor":{"n":9007199254740993}}`, id, address)
					availableViewReply(t, cloud, func(r *http.Request) (string, bool) {
						switch r.Method + " " + r.URL.Path {
						case "GET /compute/os-floating-ips":
							if !allocated {
								return `{"floating_ips":[` + row + `]}`, true
							}
						case "POST /compute/os-floating-ips", "GET /compute/os-floating-ips/29":
							return `{"floating_ip":` + row + `}`, true
						}
						return "", false
					})
					name := "scope-name"
					location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
					result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest(), compute.WithAvailableIPStrict(strict), compute.WithAvailableIPLocation(location))
					if err != nil || result == nil || result.Nova == nil || result.Nova.FloatingIP == nil || result.FloatingIP == nil || result.FloatingIP.Resource == nil || result.FloatingIP.Wire == nil {
						t.Fatal(result, err)
					}
					record := result.FloatingIP
					if result.Backend != compute.FloatingIPNova || result.Allocated != allocated || result.Reused == allocated || result.ID != id || result.Address != address || record.Backend != compute.FloatingIPNova || record.NormalizationSource != compute.FloatingIPNova || !record.Normalized {
						t.Fatal(result)
					}
					for key, want := range map[string]string{"id": id, "status": `"ACTIVE"`, "attached": "false", "fixed_ip_address": "null", "floating_ip_address": fmt.Sprintf("%q", address), "network": `"public"`} {
						queryRaw(t, record.Resource, key, want)
					}
					queryRaw(t, record.Wire, "status", `"DOWN"`)
					queryRaw(t, record.Wire, "fixed_ip", `"10.0.0.9"`)
					queryRaw(t, record.Resource, "properties", `{"owner":"other","status":"DOWN","vendor":{"n":9007199254740993}}`)
					_, alias := record.Resource.Body["project_id"]
					_, vendor := record.Resource.Body["vendor"]
					if alias == strict || vendor == strict {
						t.Fatal(strict, record.Resource.Body)
					}
					if !strict {
						queryRaw(t, record.Resource, "project_id", "null")
					}
					gotLocation := availableViewLocation(t, record)
					if string(gotLocation.Project.ID) != `"scope"` || gotLocation.Project.Name == nil || *gotLocation.Project.Name != "scope-name" {
						t.Fatal(gotLocation)
					}
					wantEvents := []string{"GET /compute/os-floating-ips"}
					if allocated {
						wantEvents = append(wantEvents, "POST /compute/os-floating-ips", "GET /compute/os-floating-ips/29")
					}
					if !reflect.DeepEqual(state.events, wantEvents) || !reflect.DeepEqual(state.locators, []string{"compute"}) || string(result.Nova.FloatingIP.Body["status"]) != `"DOWN"` {
						t.Fatal(state, result)
					}
					if allocated && (result.Nova.AllocationResponse == nil || result.Nova.AllocationResponse.StatusCode != 200) {
						t.Fatal(result)
					}
					record.Resource.Body["properties"][0] = '['
					record.Resource.Header.Set("X-Available-Proof", "view-changed")
					record.Wire.Body["id"][0] = '8'
					if string(result.Nova.FloatingIP.Body["id"]) != id || string(result.Nova.FloatingIP.Body["vendor"]) != `{"n":9007199254740993}` || result.Nova.FloatingIP.Header.Get("X-Available-Proof") == "view-changed" || record.Wire.Header.Get("X-Available-Proof") == "view-changed" {
						t.Fatal("view shares native/raw storage", result)
					}
				})
			}
		}
	}
}

func TestAvailableIPViewUsesConfiguredSourceAfterFallback(t *testing.T) {
	for _, scenario := range []string{"semantic", "list404", "catalog absent"} {
		t.Run(scenario, func(t *testing.T) {
			cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
			input := connectionAvailableRequest()
			switch scenario {
			case "semantic":
				input.Networks = []resource.Ref{resource.Name("legacy")}
				state.pool = "legacy"
			case "list404":
				state.neutronCode = 404
			case "catalog absent":
				state.catalogError = fmt.Errorf("catalog: %w", gophercloud.ErrEndpointNotFound{})
			}
			availableViewReply(t, cloud, func(r *http.Request) (string, bool) {
				if r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips" {
					return fmt.Sprintf(`{"floating_ips":[{"id":12,"ip":"198.51.100.12","pool":%q,"instance_id":null,"port_id":"port","project_id":"foreign"}]}`, state.pool), true
				}
				return "", false
			})
			result, err := conn.AvailableFloatingIP(context.Background(), input, compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
			if scenario == "list404" {
				want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips", "POST /network/v2.0/floatingips"}
				if err != nil || result == nil || result.FloatingIP == nil || result.Backend != compute.FloatingIPNeutron || !result.Allocated || result.Reused || result.FloatingIP.Normalized || result.FloatingIP.NormalizationSource != compute.FloatingIPNeutron || result.Inventory == nil || result.Inventory.FallbackError == nil || result.FallbackError != nil || result.Creation == nil || result.Creation.Selection.NetworkID != "external" || !reflect.DeepEqual(state.events, want) {
					t.Fatal("internal fallback inventory must not become outer Nova reuse", result, err, state)
				}
				queryRaw(t, result.FloatingIP.Resource, "status", `"DOWN"`)
				queryRaw(t, result.FloatingIP.Wire, "status", `"DOWN"`)
				return
			}
			if err != nil || result == nil || result.FloatingIP == nil || result.Backend != compute.FloatingIPNova || !result.Reused || result.Allocated || !result.FloatingIP.Normalized || result.FloatingIP.Backend != compute.FloatingIPNova {
				t.Fatal(result, err)
			}
			wantMode, wantStatus, wantAttached := compute.FloatingIPNeutron, `"UNKNOWN"`, "true"
			wantEvents := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /compute/os-floating-ips"}
			if scenario == "list404" {
				wantEvents = []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips"}
			}
			if scenario == "catalog absent" {
				wantMode, wantStatus, wantAttached = compute.FloatingIPNova, `"ACTIVE"`, "false"
				wantEvents = []string{"GET /compute/os-floating-ips"}
			}
			if result.FloatingIP.NormalizationSource != wantMode || !reflect.DeepEqual(state.events, wantEvents) || !reflect.DeepEqual(state.locators, []string{"network", "compute"}) {
				t.Fatal(result, state)
			}
			queryRaw(t, result.FloatingIP.Resource, "status", wantStatus)
			queryRaw(t, result.FloatingIP.Resource, "attached", wantAttached)
			if _, present := result.FloatingIP.Wire.Body["status"]; present {
				t.Fatal("synthetic status leaked into wire")
			}
			if scenario != "catalog absent" && result.FallbackError == nil {
				t.Fatal("missing existing fallback proof")
			}
			if result.Nova.FloatingIP.InstanceID != nil || string(result.Nova.FloatingIP.Body["port_id"]) != `"port"` {
				t.Fatal(result)
			}
		})
	}
}

func TestAvailableIPNeutronViewAndLocationAreOwned(t *testing.T) {
	_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
	state.neutronBody = `{"floatingips":[{"id":"neutron","floating_network_id":"external","tenant_id":"owner","port_id":null,"floating_ip_address":"2001:db8::1","status":"ERROR","revision_number":12,"tags":["one"],"self":"link","extension":{"n":9007199254740993}}]}`
	cloudName, region, name := "dev", "region", "owner-name"
	location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`"owner"`), Name: &name}}
	input := connectionAvailableRequest()
	input.Server = resource.Name("must-stay-lazy")
	result, err := conn.AvailableFloatingIP(context.Background(), input, compute.WithAvailableIPLocation(location), compute.WithAvailableIPStrict(true), compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
	if err != nil || result == nil || result.Neutron == nil || result.FloatingIP == nil || result.FloatingIP.Resource == nil || result.FloatingIP.Wire == nil || result.FloatingIP.Normalized || result.FloatingIP.NormalizationSource != compute.FloatingIPNeutron || result.FloatingIP.Backend != compute.FloatingIPNeutron || !result.Reused || result.Allocated {
		t.Fatal(result, err)
	}
	for key, want := range map[string]string{"id": `"neutron"`, "name": `"2001:db8::1"`, "project_id": `"owner"`, "revision_number": "12", "port_id": "null", "if_match": "null", "status": `"ERROR"`, "tags": `["one"]`} {
		queryRaw(t, result.FloatingIP.Resource, key, want)
	}
	if _, present := result.FloatingIP.Resource.Body["self"]; present {
		t.Fatal("self was not removed")
	}
	if _, present := result.FloatingIP.Wire.Body["project_id"]; present {
		t.Fatal("alias inserted into wire")
	}
	queryRaw(t, result.FloatingIP.Wire, "self", `"link"`)
	gotLocation := availableViewLocation(t, result.FloatingIP)
	if gotLocation.Cloud == nil || *gotLocation.Cloud != "dev" || gotLocation.RegionName == nil || *gotLocation.RegionName != "region" || gotLocation.Project.Name == nil || *gotLocation.Project.Name != "owner-name" {
		t.Fatal(gotLocation)
	}
	if result.ID != "neutron" || result.Address != "2001:db8::1" || result.Neutron.FloatingIP.Status != "ERROR" || len(state.events) != 3 || !reflect.DeepEqual(state.locators, []string{"network"}) {
		t.Fatal(result, state)
	}
	result.FloatingIP.Resource.Body["extension"][0] = '['
	result.FloatingIP.Resource.Header.Set("X-Available-Proof", "view-changed")
	result.FloatingIP.Wire.Body["id"][1] = 'x'
	if string(result.Neutron.Metadata.Body["extension"]) != `{"n":9007199254740993}` || string(result.Neutron.Metadata.Body["id"]) != `"neutron"` || result.Neutron.Metadata.Header.Get("X-Available-Proof") == "view-changed" || result.FloatingIP.Wire.Header.Get("X-Available-Proof") == "view-changed" {
		t.Fatal("view shares lower metadata", result)
	}
}

func TestAvailableIPViewOptionsSnapshotAndBudget(t *testing.T) {
	_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNova)
	input := connectionAvailableRequest()
	cloudName, projectName := "owned-cloud", "owned-project"
	projectID := json.RawMessage(`"scope"`)
	locationOption := compute.WithAvailableIPLocation(resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: projectID, Name: &projectName}})
	cloudName, projectName = "caller-changed", "caller-changed"
	projectID[1] = 'x'
	outer, inner, replaced := 0, 0, 0
	innerOptions := []network.AvailableFloatingIPOption{beforeAvailableOption(func() { inner++ }, network.WithAvailableProject("owner"))}
	networkOption := compute.WithAvailableIPNetworkOptions(innerOptions...)
	var options []compute.AvailableFloatingIPOption
	options = []compute.AvailableFloatingIPOption{
		beforeAvailableOption(func() {
			outer++
			input.Networks[0] = resource.Name("changed")
			options[1] = compute.WithAvailableIPStrict(false)
			innerOptions[0] = beforeAvailableOption(func() { replaced++ }, network.WithAvailableProject("changed"))
		}, locationOption),
		compute.WithAvailableIPStrict(true), networkOption, compute.WithAvailableIPTimeout(17 * time.Second),
	}
	started := time.Now()
	result, err := conn.AvailableFloatingIP(context.Background(), input, options...)
	if err != nil || result == nil || result.FloatingIP == nil || outer != 1 || inner != 1 || replaced != 0 || result.Nova.FloatingIP.Pool != "public" || len(state.deadlines) != 1 {
		t.Fatal(result, err, outer, inner, replaced, state)
	}
	if _, alias := result.FloatingIP.Resource.Body["project_id"]; alias {
		t.Fatal("caller replaced snapshotted option")
	}
	gotLocation := availableViewLocation(t, result.FloatingIP)
	if gotLocation.Cloud == nil || *gotLocation.Cloud != "owned-cloud" || gotLocation.Project.Name == nil || *gotLocation.Project.Name != "owned-project" || string(gotLocation.Project.ID) != `"scope"` {
		t.Fatal(gotLocation)
	}
	if remaining := state.deadlines[0].Sub(started); remaining < 16*time.Second || remaining > 18*time.Second {
		t.Fatal("deadline restarted or lost", state.deadlines)
	}
	if !reflect.DeepEqual(state.events, []string{"GET /compute/os-floating-ips"}) {
		t.Fatal(state)
	}
	for _, mode := range []string{"source", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNova)
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("option cancellation")
			later := 0
			first := beforeAvailableOption(func() {
				if mode == "source" {
					service.API = nil
				} else {
					cancel(cause)
				}
			}, compute.WithAvailableIPStrict(true))
			second := beforeAvailableOption(func() { later++ }, compute.WithAvailableIPStrict(false))
			result, err := service.AvailableFloatingIP(ctx, connectionAvailableRequest(), first, second)
			if result != nil || err == nil || later != 0 || len(state.events) != 0 {
				t.Fatal(result, err, later, state)
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
		})
	}
}

func TestAvailableIPViewLocationErrorKeepsKnownAllocation(t *testing.T) {
	for _, mode := range []string{"error", "source", "cancel", "override"} {
		t.Run(mode, func(t *testing.T) {
			cloud, _, state := connectionAvailableFixture(t, compute.FloatingIPNova)
			state.novaFree = false
			client := cloud.Client("compute", "/compute/")
			client.Microversion = "2.35"
			policy, err := compute.PrepareServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNova))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("view location failure")
			calls := 0
			var service *compute.Service
			service = compute.New(client, compute.Dependencies{ServerAddresses: policy, CloudLocation: func() (resource.CloudLocation, error) {
				calls++
				switch mode {
				case "error":
					return resource.CloudLocation{}, cause
				case "source":
					service.API = nil
				case "cancel":
					cancel(cause)
				}
				return resource.CloudLocation{}, nil
			}})
			var options []compute.AvailableFloatingIPOption
			if mode == "override" {
				options = append(options, compute.WithAvailableIPLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}))
			}
			result, err := service.AvailableFloatingIP(ctx, connectionAvailableRequest(), options...)
			if result == nil || result.Backend != compute.FloatingIPNova || !result.Allocated || result.Reused || result.ID != "29" || result.Nova == nil || result.Nova.FloatingIP == nil || result.Nova.AllocationResponse == nil || result.Nova.AllocationResponse.StatusCode != 200 || result.FloatingIP == nil || result.FloatingIP.Wire == nil || string(result.FloatingIP.Wire.Body["id"]) != "29" {
				t.Fatal(result, err)
			}
			if !reflect.DeepEqual(state.events, []string{"GET /compute/os-floating-ips", "POST /compute/os-floating-ips", "GET /compute/os-floating-ips/29"}) {
				t.Fatal(state)
			}
			if mode == "override" {
				if err != nil || calls != 0 || result.FloatingIP.Resource == nil {
					t.Fatal(result, err, calls)
				}
				queryRaw(t, result.FloatingIP.Resource, "status", `"ACTIVE"`)
				return
			}
			if err == nil || calls != 1 || result.FloatingIP.Resource != nil {
				t.Fatal(result, err, calls)
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode != "source" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || len(proof.Body) == 0 || proof.Header.Get("X-Available-Proof") != "/compute/os-floating-ips/29" {
				t.Fatal("missing actual view response proof", err, proof)
			}
			var envelope map[string]json.RawMessage
			if decodeErr := json.Unmarshal(proof.Body, &envelope); decodeErr != nil {
				t.Fatal(decodeErr)
			}
			if _, present := envelope["floating_ip"]; !present {
				t.Fatal("response proof was replaced by a serialized row", string(proof.Body))
			}
		})
	}
}

func TestAvailableIPViewAcceptedFailuresPreserveRawReceipts(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNeutron} {
		for _, failure := range []string{"close", "source", "cancel"} {
			t.Run(string(source)+"/"+failure, func(t *testing.T) {
				_, conn, state := connectionAvailableFixture(t, source)
				state.novaFree, state.neutronFree = false, false
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("accepted processing failure")
				state.afterPost = func(string) error {
					switch failure {
					case "close":
						return cause
					case "source":
						if source == compute.FloatingIPNova {
							service, _ := conn.Compute(context.Background())
							service.API = nil
						} else {
							service, _ := conn.Network(context.Background())
							service.API = nil
						}
					case "cancel":
						cancel(cause)
					}
					return nil
				}
				result, err := conn.AvailableFloatingIP(ctx, connectionAvailableRequest(), compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
				if err == nil || result == nil || !result.Allocated || result.Reused || result.Backend != source || result.FloatingIP == nil || result.FloatingIP.Wire == nil || result.FloatingIP.Backend != source {
					t.Fatal(result, err)
				}
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || len(proof.Body) == 0 || proof.Header.Get("X-Available-Proof") == "" {
					t.Fatal(err, proof)
				}
				var envelope map[string]json.RawMessage
				if decodeErr := json.Unmarshal(proof.Body, &envelope); decodeErr != nil {
					t.Fatal(decodeErr)
				}
				key := "floating_ip"
				if source == compute.FloatingIPNeutron {
					key = "floatingip"
				}
				if _, present := envelope[key]; !present {
					t.Fatal("accepted proof lost original envelope", string(proof.Body))
				}
				if failure == "source" && !errors.Is(err, resource.ErrInvalidOption) || failure != "source" && !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if failure != "close" && result.FloatingIP.Resource != nil {
					t.Fatal("unguarded successful view", result)
				}
				if source == compute.FloatingIPNova {
					queryRaw(t, result.FloatingIP.Wire, "id", "29")
					if result.Nova == nil || result.Nova.FloatingIP.ID != "29" || result.Nova.AllocationResponse.StatusCode != 200 || !reflect.DeepEqual(state.events, []string{"GET /compute/os-floating-ips", "POST /compute/os-floating-ips"}) {
						t.Fatal(result, state)
					}
				} else {
					queryRaw(t, result.FloatingIP.Wire, "id", `"allocated-neutron"`)
					if result.Neutron == nil || result.Neutron.FloatingIP.ID != "allocated-neutron" || result.Neutron.AllocationResponse.StatusCode != 201 || len(state.events) != 4 || !reflect.DeepEqual(state.locators, []string{"network"}) {
						t.Fatal(result, state)
					}
				}
			})
		}
	}
}

func TestAvailableIPNeutronViewErrorsKeepPhysicalReceipts(t *testing.T) {
	for _, allocated := range []bool{false, true} {
		t.Run(fmt.Sprint("allocated=", allocated), func(t *testing.T) {
			_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
			state.neutronFree = !allocated
			result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest(),
				compute.WithAvailableIPLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage("invalid")}}),
				compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
			if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Backend != compute.FloatingIPNeutron || result.Allocated != allocated || result.Reused {
				t.Fatal(result, err)
			}
			var proof *resource.ResponseError
			if allocated {
				if result.FloatingIP == nil || result.FloatingIP.Wire == nil || result.FloatingIP.Resource != nil || result.Neutron == nil || result.Neutron.FloatingIP == nil {
					t.Fatal(result, err)
				}
				receipt := result.Neutron.AllocationResponse
				if !errors.As(err, &proof) || receipt == nil || proof.StatusCode != receipt.StatusCode || string(proof.Body) != string(receipt.Envelope) || !reflect.DeepEqual(proof.Header, receipt.Header) || result.ID != "allocated-neutron" || len(state.events) != 4 {
					t.Fatal(result, err, proof, state)
				}
			} else {
				inventory := result.Inventory
				if !errors.As(err, &proof) || proof.StatusCode != 200 || inventory == nil || inventory.Failure == nil || len(inventory.Pages) != 1 || string(proof.Body) != string(inventory.Pages[0].Envelope) || !reflect.DeepEqual(proof.Header, inventory.Pages[0].Header) || result.FloatingIP != nil || result.Neutron != nil || result.Creation != nil || len(state.events) != 3 {
					t.Fatal("list view error lost physical page or returned a free candidate", result, err, proof, state)
				}
			}
			if !reflect.DeepEqual(state.locators, []string{"network"}) {
				t.Fatal("view failure selected another backend", state)
			}
		})
	}
}
