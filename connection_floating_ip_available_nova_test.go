package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"gophercloudsdk/compute"
	"gophercloudsdk/resource"
)

func TestAvailableIPNovaRawFilterBeforeNormalization(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone} {
		for _, test := range []struct {
			name, rows, id string
			failure        bool
		}{
			{"excluded rows are not normalized", `[{"instance_id":"attached"},{"instance_id":"","pool":"public"},{"instance_id":null,"pool":"other"},{"id":null,"ip":null,"pool":"public","instance_id":null}]`, "null", false},
			{"passive address is not an execution IPv4", `[{"id":7,"ip":"not an IPv4","pool":"public","instance_id":null}]`, "7", false},
			{"missing instance is read before mismatching pool", `[{"pool":"other"},{"id":7,"pool":"public","instance_id":null}]`, "", true},
			{"matching instance reads missing pool", `[{"id":7,"instance_id":null}]`, "", true},
		} {
			t.Run(fmt.Sprint(source, "/", test.name), func(t *testing.T) {
				cloud, conn, state := connectionAvailableFixture(t, source)
				body := `{"floating_ips":` + test.rows + `}`
				availableViewReply(t, cloud, func(r *http.Request) (string, bool) {
					return body, r.URL.Path == "/compute/os-floating-ips"
				})
				result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest())
				if result == nil || (err != nil) != test.failure || result.Allocated || result.Nova == nil || result.Nova.Inventory == nil || !reflect.DeepEqual(state.events, []string{"GET /compute/os-floating-ips"}) {
					t.Fatal(result, err, state)
				}
				if test.failure {
					var proof *resource.ResponseError
					if result.Reused || result.FloatingIP != nil || !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 {
						t.Fatal(result, err, proof)
					}
					return
				}
				if !result.Reused || len(result.Nova.Inventory.FloatingIPs) != 1 {
					t.Fatal(result)
				}
				queryRaw(t, result.FloatingIP.Resource, "id", test.id)
				queryRaw(t, result.FloatingIP.Resource, "status", `"ACTIVE"`)
				if test.id == "null" && (result.ID != "" || result.Nova.FloatingIP != nil) {
					t.Fatal("typed projection narrowed a passive null ID", result)
				}
			})
		}
	}
}

func TestAvailableIPNovaNormalizesEveryMatchBeforeFirst(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		t.Run(fmt.Sprint("late malformed=", malformed), func(t *testing.T) {
			cloud, _, state := connectionAvailableFixture(t, compute.FloatingIPNova)
			second := `{"id":8,"pool":"public","instance_id":null}`
			if malformed {
				second = `{"pool":"public","instance_id":null}`
			}
			body := `{"floating_ips":[{"id":7,"pool":"public","instance_id":null},` + second + `,{"instance_id":null,"pool":"other"}]}`
			availableViewReply(t, cloud, func(r *http.Request) (string, bool) { return body, r.URL.Path == "/compute/os-floating-ips" })
			client := cloud.Client("compute", "/compute/")
			client.Microversion = "2.35"
			policy, err := compute.PrepareServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNova))
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			service := compute.New(client, compute.Dependencies{ServerAddresses: policy, CloudLocation: func() (resource.CloudLocation, error) {
				calls++
				return resource.CloudLocation{}, nil
			}})
			result, err := service.AvailableFloatingIP(context.Background(), connectionAvailableRequest())
			if calls != 2 || result == nil || (err != nil) != malformed || result.Allocated || !reflect.DeepEqual(state.events, []string{"GET /compute/os-floating-ips"}) {
				t.Fatal(result, err, calls, state)
			}
			if malformed {
				var proof *resource.ResponseError
				if result.Reused || result.FloatingIP != nil || !errors.As(err, &proof) || string(proof.Body) != body {
					t.Fatal(result, err, proof)
				}
			} else {
				if !result.Reused || result.ID != "7" || len(result.Nova.Inventory.FloatingIPs) != 2 {
					t.Fatal(result)
				}
				result.FloatingIP.Wire.Body["id"][0] = '9'
				queryRaw(t, result.Nova.Inventory.FloatingIPs[0].Wire, "id", "7")
			}
		})
	}
	t.Run("outer fallback keeps configured Neutron view", func(t *testing.T) {
		cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
		state.pool = "legacy"
		availableViewReply(t, cloud, func(r *http.Request) (string, bool) {
			return `{"floating_ips":[{"id":7,"pool":"legacy","instance_id":null,"port_id":"attached-port","status":"DOWN"},{"id":8,"pool":"legacy","instance_id":null}]}`, r.URL.Path == "/compute/os-floating-ips"
		})
		input := connectionAvailableRequest()
		input.Networks = []resource.Ref{resource.Name("legacy")}
		result, err := conn.AvailableFloatingIP(context.Background(), input)
		if err != nil || result == nil || !result.Reused || result.FallbackError == nil || result.FloatingIP.NormalizationSource != compute.FloatingIPNeutron || len(result.Nova.Inventory.FloatingIPs) != 2 {
			t.Fatal(result, err)
		}
		queryRaw(t, result.FloatingIP.Resource, "attached", "true")
		queryRaw(t, result.FloatingIP.Resource, "status", `"DOWN"`)
		queryRaw(t, result.Nova.Inventory.FloatingIPs[1].Resource, "status", `"UNKNOWN"`)
	})
}

func TestAvailableIPNovaCleanListNotFoundAllocates(t *testing.T) {
	for _, test := range []struct {
		name                 string
		code                 int
		body                 string
		defaultPool, failure bool
	}{
		{"clean404 explicit pool", 404, `{"error":"gone"}`, false, false},
		{"clean404 default pool", 404, `{"error":"gone"}`, true, false},
		{"forbidden stays terminal", 403, `{"error":"denied"}`, false, true},
		{"accepted malformed stays terminal", 200, `{"floating_ips":`, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNone)
			input := connectionAvailableRequest()
			if test.defaultPool {
				input.Networks = nil
			}
			availableReply(t, cloud, func(r *http.Request) (int, string, bool) {
				return test.code, test.body, r.URL.Path == "/compute/os-floating-ips" && r.Method == http.MethodGet
			})
			result, err := conn.AvailableFloatingIP(context.Background(), input)
			want := []string{"GET /compute/os-floating-ips"}
			if !test.failure {
				want = append(want, "POST /compute/os-floating-ips", "GET /compute/os-floating-ips/29")
			}
			if test.defaultPool {
				want = append([]string{"GET /compute/os-floating-ip-pools"}, want...)
			}
			if result == nil || (err != nil) != test.failure || !reflect.DeepEqual(state.events, want) {
				t.Fatal(result, err, state)
			}
			if !test.failure && (!result.Allocated || result.Reused || result.Nova.Creation == nil || result.Nova.Inventory.SuppressedNotFound == nil || result.Nova.Inventory.Failure.StatusCode != 404 || string(result.Nova.Inventory.Failure.Envelope) != test.body) {
				t.Fatal(result)
			}
		})
	}
}

func TestAvailableIPNovaFreshKeepsCreateProofAndBudget(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprint("compatibility failure=", failure), func(t *testing.T) {
			cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNova)
			state.novaFree = false
			compatBody := `{"floating_ip":{"id":null,"ip":"passive","pool":null,"instance_id":null}}`
			code := 200
			if failure {
				code, compatBody = 403, `{"error":"compat denied"}`
			}
			availableReply(t, cloud, func(r *http.Request) (int, string, bool) {
				return code, compatBody, r.URL.Path == "/compute/os-floating-ips/29"
			})
			result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest(), compute.WithAvailableIPTimeout(time.Second))
			if result == nil || (err != nil) != failure || !result.Allocated || result.Reused || result.Nova.Creation == nil || result.Nova.Creation.AllocationResponse.StatusCode != 200 || result.Nova.Creation.Compatibility == nil || !reflect.DeepEqual(state.events, []string{"GET /compute/os-floating-ips", "POST /compute/os-floating-ips", "GET /compute/os-floating-ips/29"}) {
				t.Fatal(result, err, state)
			}
			for _, deadline := range state.deadlines {
				if deadline.IsZero() || !deadline.Equal(state.deadlines[0]) {
					t.Fatal("budget restarted", state.deadlines)
				}
			}
			queryRaw(t, result.Nova.Creation.Allocation.Wire, "id", "29")
			if failure {
				if result.Nova.Creation.Failure.StatusCode != 403 || string(result.Nova.Creation.Failure.Envelope) != compatBody {
					t.Fatal(result.Nova.Creation)
				}
				queryRaw(t, result.FloatingIP.Wire, "id", "29")
			} else {
				queryRaw(t, result.FloatingIP.Resource, "id", "null")
				queryRaw(t, result.FloatingIP.Resource, "floating_ip_address", `"passive"`)
				if result.Nova.FloatingIP != nil || string(result.Nova.Creation.Compatibility.Observed.Envelope) != compatBody {
					t.Fatal(result)
				}
			}
		})
	}
}
