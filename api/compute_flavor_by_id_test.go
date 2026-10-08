package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Cloud get_flavor_by_id only forwards its default-false enrichment flag to
// strict proxy get_flavor. Existing identity/native specs tests own its matrix.
func TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment(t *testing.T) {
	for _, tc := range []struct {
		name, inline        string
		extra               *bool
		status, extraStatus int
		wantExtra, failure  bool
	}{
		{"omitted default", "", nil, 200, 200, false, false},
		{"explicit false", "", flavorIdentityBool(false), 200, 200, false, false},
		{"true missing", "", flavorIdentityBool(true), 200, 200, true, false},
		{"true null", `,"extra_specs":null`, flavorIdentityBool(true), 200, 200, true, false},
		{"true empty", `,"extra_specs":{}`, flavorIdentityBool(true), 200, 200, true, false},
		{"true inline", `,"extra_specs":{"inline":"kept"}`, flavorIdentityBool(true), 200, 200, false, false},
		{"strict400 never lists", "", nil, 400, 200, false, true},
		{"strict403 never lists", "", nil, 403, 200, false, true},
		{"strict404 never lists", "", nil, 404, 200, false, true},
		{"extra404 stays terminal", "", flavorIdentityBool(true), 200, 404, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			cloud.Provider.SetToken("flavor-cloud-live")
			var gets, extras, other, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "flavor-cloud-live")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.URL.RawQuery != "" {
					t.Error(r.URL, string(body), err)
				}
				w.Header().Set("X-Flavor-Cloud-Proof", tc.name)
				switch r.URL.Path {
				case flavorIdentityPath + "/lookup":
					gets.Add(1)
					if tc.status != 200 {
						testcloud.JSON(w, tc.status, `{"error":"strict member"}`)
						return
					}
					testcloud.JSON(w, 200, `{"flavor":{"id":"7","name":"canonical","ram":2048,"disk":12,"vcpus":2,"swap":"3"`+tc.inline+`}}`)
				case flavorIdentityPath + "/7/os-extra_specs":
					extras.Add(1)
					testcloud.JSON(w, tc.extraStatus, `{"extra_specs":{"cpu":"dedicated"}}`)
				default:
					other.Add(1)
					w.WriteHeader(500)
				}
			})
			var options []compute.FlavorByIDOption
			if tc.extra != nil {
				options = append(options, compute.WithFlavorByIDExtraSpecs(*tc.extra))
			}
			options = append(options, func(*request.Config[compute.FlavorByIDOpts]) error { callbacks.Add(1); return nil })
			value, err := compute.New(client, compute.Dependencies{}).GetFlavorByID(context.Background(), "lookup", options...)
			wantExtras := int32(0)
			if tc.wantExtra {
				wantExtras = 1
			}
			if (err != nil) != tc.failure || gets.Load() != 1 || extras.Load() != wantExtras || other.Load() != 0 || callbacks.Load() != 1 || client.Microversion != "2.55" {
				t.Fatal(value, err, gets.Load(), extras.Load(), other.Load(), callbacks.Load())
			}
			if tc.failure {
				var native gophercloud.ErrUnexpectedResponseCode
				status := tc.status
				target := cloud.Server.URL + flavorIdentityPath + "/lookup"
				if tc.status == 200 {
					status = tc.extraStatus
					target = cloud.Server.URL + flavorIdentityPath + "/7/os-extra_specs"
				}
				if value != nil || !errors.As(err, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != target || native.ResponseHeader.Get("X-Flavor-Cloud-Proof") != tc.name {
					t.Fatal(value, err, native)
				}
				return
			}
			if value == nil || value.ID != "7" || value.Name != "canonical" || value.RAM != 2048 || value.Swap != 3 || value.Disk != 12 {
				t.Fatal(value, err)
			}
			if tc.wantExtra && value.ExtraSpecs["cpu"] != "dedicated" || tc.inline != "" && tc.extra != nil && *tc.extra && !tc.wantExtra && value.ExtraSpecs["inline"] != "kept" {
				t.Fatal(value)
			}
		})
	}
}

func TestComputeCloudGetFlavorByIDTypedOptionsAndPreflight(t *testing.T) {
	for _, mode := range []string{"nil service", "nil context", "nil option", "unsafe ID", "unsupported header", "unsupported query", "unsupported field", "unsupported argument", "source callback", "cancel callback", "callback error"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			service := compute.New(client, compute.Dependencies{})
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var operationCtx context.Context = ctx
			id := "7"
			cause := errors.New("flavor option cause")
			var calls, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			var options []compute.FlavorByIDOption
			switch mode {
			case "nil service":
				service = nil
			case "nil context":
				operationCtx = nil
			case "nil option":
				options = append(options, nil)
			case "unsafe ID":
				id = "unsafe/name"
			case "unsupported header":
				options = append(options, request.WithHeader[compute.FlavorByIDOpts]("X-Custom", "value"))
			case "unsupported query":
				options = append(options, request.WithQuery[compute.FlavorByIDOpts]("vendor", "value"))
			case "unsupported field":
				options = append(options, request.WithField[compute.FlavorByIDOpts]("vendor", "value"))
			case "unsupported argument":
				options = append(options, request.WithArgument[compute.FlavorByIDOpts]("vendor", "value"))
			case "source callback":
				options = append(options, func(*request.Config[compute.FlavorByIDOpts]) error { client.Microversion = "2.55"; return nil })
			case "cancel callback":
				options = append(options, func(*request.Config[compute.FlavorByIDOpts]) error { cancel(cause); return nil })
			case "callback error":
				options = append(options, func(*request.Config[compute.FlavorByIDOpts]) error { return cause })
			}
			options = append(options, func(*request.Config[compute.FlavorByIDOpts]) error { later.Add(1); return nil })
			value, err := service.GetFlavorByID(operationCtx, id, options...)
			if value != nil || err == nil || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
			if (mode == "source callback" || mode == "unsafe ID" || mode == "nil service" || mode == "nil context" || mode == "nil option") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("preflight lost invalid-option classification", err)
			}
			if mode == "cancel callback" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) || mode == "callback error" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if (mode == "source callback" || mode == "cancel callback" || mode == "callback error" || mode == "nil service" || mode == "nil context" || mode == "nil option") && later.Load() != 0 {
				t.Fatal("preflight ran later option", later.Load())
			}
		})
	}
	t.Run("typed options are a reusable value", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		var extras atomic.Int32
		cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"flavor":{"id":"7"}}`) })
		cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
			extras.Add(1)
			testcloud.JSON(w, 200, `{"extra_specs":{"fetched":"kept"}}`)
		})
		original := compute.FlavorByIDOpts{GetExtraSpecs: true}
		option := compute.WithFlavorByIDOptions(original)
		original.GetExtraSpecs = false
		for range 2 {
			value, err := compute.New(client, compute.Dependencies{}).GetFlavorByID(context.Background(), "7", option)
			if err != nil || value == nil || value.ExtraSpecs["fetched"] != "kept" {
				t.Fatal(value, err)
			}
		}
		if extras.Load() != 2 {
			t.Fatal("typed options did not capture enrichment value", extras.Load())
		}
	})
}
