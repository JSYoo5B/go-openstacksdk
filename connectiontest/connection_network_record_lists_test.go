package connectiontest_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionPortRecordsCachedServiceAndAutomaticPerRowLocation(t *testing.T) {
	fixture := testcloud.New(t)
	cloudName, projectName := "configured-cloud", "configured-project"
	configured := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	locates, requests, callbacks := 0, 0, 0
	fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return fixture.Server.URL + "/network/", nil
	}
	conn, err := sdk.FromProvider(fixture.Provider, sdk.WithCloudLocation(configured))
	if err != nil {
		t.Fatal(err)
	}
	cloudName = "caller changed"
	projectName = "caller changed"
	service, err := conn.NetworkV2(context.Background())
	if err != nil || service.Ports.RawClient() != service.RawClient() {
		t.Fatal(service, err)
	}
	fixture.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 203, `{"ports":[{"id":"current","name":"alpha","project_id":"scope","location":{"cloud":"forged"}},{"id":"foreign","name":"beta","project_id":"other"}]}`)
	})
	records, err := conn.ListPorts(context.Background(), func(*ports.RecordListOpts) error { callbacks++; return nil })
	if err != nil || len(records) != 2 || requests != 1 || locates != 1 || callbacks != 1 {
		t.Fatal(records, err, requests, locates, callbacks)
	}
	for i, record := range records {
		loc := connectionSearchFields(t, record.Resource.Body["location"])
		project := connectionSearchFields(t, loc["project"])
		expectedID, expectedName := `"scope"`, `"configured-project"`
		if i == 1 {
			expectedID, expectedName = `"other"`, `null`
		}
		if string(loc["cloud"]) != `"configured-cloud"` || string(project["id"]) != expectedID || string(project["name"]) != expectedName || record.StatusCode != 203 {
			t.Fatal(record.Resource.Body)
		}
	}
	if string(records[0].Wire.Body["location"]) != `{"cloud":"forged"}` {
		t.Fatal("computed location altered wire", records[0].Wire.Body)
	}
	result, err := conn.SearchPorts(context.Background(), "a*", ports.WithRecordSearchFilters(json.RawMessage(`{"location":{"cloud":"configured-cloud","project":{"id":"scope"}}}`)), func(*ports.RecordSearchOpts) error { callbacks++; return nil })
	if err != nil || result == nil || len(result.Ports) != 1 || len(result.Inventory) != 2 || string(result.Ports[0].Resource.Body["id"]) != `"current"` || requests != 2 || locates != 1 || callbacks != 2 {
		t.Fatal(result, err, requests, locates, callbacks)
	}
	cached, err := conn.NetworkV2(context.Background())
	if err != nil || cached != service {
		t.Fatal("Connection replaced cached service", cached, err)
	}
}

func TestConnectionPortRecordsCatalogAbsencePrecedesListOptionsButSearchStillEvaluatesEmpty(t *testing.T) {
	for _, catalogErr := range []error{gophercloud.ErrEndpointNotFound{}, fmt.Errorf("catalog wrapper: %w", &gophercloud.ErrEndpointNotFound{})} {
		fixture := testcloud.New(t)
		locates, requests, listCallbacks := 0, 0, 0
		fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { locates++; return "", catalogErr }
		fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests++; t.Errorf("absence sent request %s", r.URL) })
		conn, err := sdk.FromProvider(fixture.Provider)
		if err != nil {
			t.Fatal(err)
		}
		records, err := conn.ListPorts(context.Background(), func(*ports.RecordListOpts) error {
			listCallbacks++
			return errors.New("must not evaluate missing-service filters")
		}, nil)
		if err != nil || records == nil || len(records) != 0 || listCallbacks != 0 || requests != 0 || locates != 1 {
			t.Fatal(records, err, listCallbacks, requests, locates)
		}
		for _, tc := range []struct {
			filter, want string
			bad          bool
		}{
			{`true`, `[]`, false}, {`[1]`, `[]`, false}, {`"length(@)"`, `0`, false}, {`"[0].id"`, `null`, false}, {`"["`, ``, true},
		} {
			result, err := conn.SearchPorts(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(tc.filter)))
			if result == nil || result.Inventory == nil || len(result.Inventory) != 0 || requests != 0 {
				t.Fatal(tc, result, err, requests)
			}
			if tc.bad {
				if !errors.Is(err, resource.ErrInvalidOption) || result.Value != nil {
					t.Fatal(tc, result, err)
				}
			} else if err != nil || string(result.Value) != tc.want {
				t.Fatal(tc, result, err)
			}
		}
	}
}

func TestConnectionPortRecordsMixedCatalogFailureAndCancellationDoNotClaimEmpty(t *testing.T) {
	for _, mode := range []string{"mixed", "cancel"} {
		fixture := testcloud.New(t)
		cause := errors.New("catalog handling failed")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
			if mode == "cancel" {
				cancel(cause)
				return "", gophercloud.ErrEndpointNotFound{}
			}
			return "", errors.Join(gophercloud.ErrEndpointNotFound{}, cause)
		}
		conn, err := sdk.FromProvider(fixture.Provider)
		if err != nil {
			t.Fatal(err)
		}
		var proof *resource.ResponseError
		records, err := conn.ListPorts(ctx)
		if records != nil || !errors.Is(err, cause) || errors.As(err, &proof) {
			t.Fatal(mode, records, err, proof)
		}
		if mode == "cancel" && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
}

func TestConnectionPortRecordsPreflightAndLatePageFailureRetainActualEvidence(t *testing.T) {
	fixture := testcloud.New(t)
	requests, locates := 0, 0
	fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return fixture.Server.URL + "/network/", nil
	}
	conn, err := sdk.FromProvider(fixture.Provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"nil context", "nil connection", "canceled"} {
		ctx := context.Background()
		current := conn
		cause := errors.New("caller canceled")
		if mode == "nil context" {
			ctx = nil
		}
		if mode == "nil connection" {
			current = nil
		}
		if mode == "canceled" {
			next, cancel := context.WithCancelCause(ctx)
			cancel(cause)
			ctx = next
		}
		result, err := current.SearchPorts(ctx, "")
		want := resource.ErrInvalidOption
		if mode == "canceled" {
			want = context.Canceled
		}
		if result != nil || !errors.Is(err, want) || locates != 0 || requests != 0 {
			t.Fatal(mode, result, err, locates, requests)
		}
		if mode == "canceled" && !errors.Is(err, cause) {
			t.Fatal(err)
		}
	}
	fixture.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("marker") == "late" {
			w.Header().Set("X-Proof", "late-forbidden")
			testcloud.JSON(w, 403, `{"error":"late denied"}`)
			return
		}
		testcloud.JSON(w, 200, `{"ports":[{"id":"first","name":"alpha"}],"ports_links":[{"rel":"next","href":"?marker=late"}]}`)
	})
	records, err := conn.ListPorts(context.Background())
	var native gophercloud.ErrUnexpectedResponseCode
	if records != nil || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"late denied"}` || native.ResponseHeader.Get("X-Proof") != "late-forbidden" || requests != 2 {
		t.Fatal(records, err, native, requests)
	}
}
