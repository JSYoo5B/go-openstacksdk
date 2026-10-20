package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

type floatingDeleteState struct {
	readError        error
	locators, events []string
	deadlines        []time.Time
	reply            func(*http.Request) (int, string)
	afterClose       func(*http.Request) error
	catalogError     error
}

func floatingDeleteFixture(t *testing.T, source compute.FloatingIPSource) (*testcloud.Cloud, *sdk.Connection, *floatingDeleteState) {
	t.Helper()
	cloud := testcloud.New(t)
	state := &floatingDeleteState{}
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		state.locators = append(state.locators, o.Type)
		if o.Type == "network" && state.catalogError != nil {
			return "", state.catalogError
		}
		if o.Type != "network" && o.Type != "compute" {
			t.Error("unexpected locator", o.Type)
		}
		return cloud.Server.URL + "/delete-" + o.Type + "/", nil
	}
	cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "DELETE" && r.Method != "GET" {
			t.Error("unexpected mutation", r.Method, r.URL)
		}
		q := r.URL.Query()
		if (len(q) > 0 && (len(q) != 1 || q.Get("marker") == "")) || r.ContentLength != 0 {
			t.Error("unexpected request fields", r.URL, r.ContentLength)
		}
		state.events = append(state.events, r.Method+" "+r.URL.Path)
		deadline, _ := r.Context().Deadline()
		state.deadlines = append(state.deadlines, deadline)
		if strings.Contains(r.URL.Path, "compute") && r.Header.Get("X-OpenStack-Nova-API-Version") != "2.35" {
			t.Error("Nova version", r.Header)
		}
		if state.reply == nil {
			return nil, errors.New("missing delete fixture")
		}
		code, body := state.reply(r)
		var reader io.ReadCloser = io.NopCloser(strings.NewReader(body))
		if state.readError != nil {
			reader = floatingDeleteFailingBody{Reader: strings.NewReader(body), cause: state.readError}
		}
		if state.afterClose != nil {
			reader = readyConnectionCloseBody{ReadCloser: reader, close: func() error { return state.afterClose(r) }}
		}
		return &http.Response{StatusCode: code, Body: reader, Header: http.Header{"X-Delete-Proof": {r.Method + " " + r.URL.Path}}, Request: r}, nil
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.35"), sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(source)), sdk.WithCloudLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}))
	if err != nil {
		t.Fatal(err)
	}
	return cloud, conn, state
}
func TestFloatingIPDeleteDefaultRetryAndPublicLookup(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
	deletes, gets := 0, 0
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "DELETE" {
			deletes++
			return 204, ""
		}
		gets++
		if gets == 1 {
			return 200, `{"floatingips":[{"id":"ip","status":"ACTIVE"}]}`
		}
		return 200, `{"floatingips":[]}`
	}
	result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"})
	want := []string{"DELETE /delete-network/v2.0/floatingips/ip", "GET /delete-network/v2.0/floatingips", "DELETE /delete-network/v2.0/floatingips/ip", "GET /delete-network/v2.0/floatingips"}
	if err != nil || !result.Deleted || !result.Absent || result.Down || deletes != 2 || gets != 2 || len(result.Attempts) != 2 || !reflect.DeepEqual(state.events, want) || !reflect.DeepEqual(state.locators, []string{"network"}) {
		t.Fatal(result, err, state)
	}
	for _, attempt := range result.Attempts {
		if !attempt.Accepted || !attempt.Verified || attempt.Response.StatusCode != 204 || len(attempt.Verification.Pages) != 1 {
			t.Fatal(attempt)
		}
	}
}
func TestFloatingIPDeleteRetryZeroNegativeAndBulkOptions(t *testing.T) {
	for _, mode := range []string{"zero", "negative", "bulk zero"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 202, "not-json"
				}
				return 200, `{"floatingips":[]}`
			}
			option := compute.WithFloatingIPDeleteRetries(0)
			if mode == "negative" {
				option = compute.WithFloatingIPDeleteRetries(-5)
			}
			if mode == "bulk zero" {
				option = compute.WithFloatingIPDeleteOptions(compute.FloatingIPDeleteOpts{})
			}
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"}, option)
			if err != nil || !result.Deleted || len(result.Attempts) != 1 || string(result.Attempts[0].Response.Envelope) != "not-json" {
				t.Fatal(result, err)
			}
			if mode == "negative" {
				if len(state.events) != 2 || !result.Absent || !result.Attempts[0].Verified {
					t.Fatal(result, state)
				}
			} else if len(state.events) != 1 || result.Absent || result.Down || result.Attempts[0].Verified || result.LastVerification != nil {
				t.Fatal(result, state)
			}
		})
	}
}
func TestFloatingIPDeleteDownIsPresentAndJSONSemantic(t *testing.T) {
	for _, status := range []string{`"DOWN"`, `"\u0044OWN"`} {
		t.Run(status, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 204, ""
				}
				return 200, `{"floatingips":[{"id":"ip","status":` + status + `}]}`
			}
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"})
			if err != nil || !result.Deleted || result.Absent || !result.Down || len(state.events) != 2 || result.LastVerification.FloatingIP == nil || len(result.Attempts) != 1 {
				t.Fatal(result, err, state)
			}
			queryRaw(t, result.LastVerification.FloatingIP.Wire, "status", status)
		})
	}
}
func TestFloatingIPDeleteVerificationExhaustionKeepsLastRow(t *testing.T) {
	for _, status := range []string{`"ACTIVE"`, `"ERROR"`, `null`, `false`, `"down"`} {
		t.Run(status, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 204, ""
				}
				return 200, `{"floatingips":[{"id":"ip","status":` + status + `}]}`
			}
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"})
			var failed *compute.FloatingIPDeleteVerificationError
			if !errors.As(err, &failed) || failed.ID != "ip" || failed.Attempts != 2 || failed.FloatingIP == nil || result.Deleted || result.Absent || result.Down || len(result.Attempts) != 2 || len(state.events) != 4 {
				t.Fatal(result, err, state)
			}
		})
	}
}
func TestFloatingIPDeleteNotFoundIsFalseWithoutFallback(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNeutron, compute.FloatingIPNova, compute.FloatingIPNone} {
		for _, retry := range []int{0, 1, 3, -1} {
			t.Run(fmt.Sprint(source, retry), func(t *testing.T) {
				_, conn, state := floatingDeleteFixture(t, source)
				state.reply = func(*http.Request) (int, string) { return 404, `{"missing":true}` }
				result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"}, compute.WithFloatingIPDeleteRetries(retry))
				if err != nil || result.Deleted || len(result.Attempts) != 1 || result.Attempts[0].Accepted || !gophercloud.ResponseCodeIs(result.Attempts[0].NotFound, 404) || result.Failure.StatusCode != 404 || len(state.events) != 1 || len(state.locators) != 1 {
					t.Fatal(result, err, state)
				}
			})
		}
	}
}
func TestFloatingIPDeleteSecondMissingRetainsAcceptedHistory(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
	deletes := 0
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "GET" {
			return 200, `{"floatingips":[{"id":"ip","status":"ACTIVE"}]}`
		}
		deletes++
		if deletes == 1 {
			return 204, ""
		}
		return 404, "second missing"
	}
	result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"})
	if err != nil || result.Deleted || len(result.Attempts) != 2 || !result.Attempts[0].Accepted || result.Attempts[1].Accepted || result.Attempts[1].NotFound == nil || len(state.events) != 3 || result.LastVerification.FloatingIP == nil {
		t.Fatal(result, err, state)
	}
	if result.Attempts[0].Response.Header.Get("X-Delete-Proof") != "DELETE /delete-network/v2.0/floatingips/ip" || string(result.Failure.Envelope) != "second missing" {
		t.Fatal(result)
	}
}
func TestFloatingIPDeleteNovaAndNoneVerificationUsesNormalizedCloudView(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone} {
		t.Run(string(source), func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, source)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 202, ""
				}
				return 200, `{"floating_ips":[{"id":7,"status":"DOWN","ip":"actual"}]}`
			}
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "7"})
			var failed *compute.FloatingIPDeleteVerificationError
			if !errors.As(err, &failed) || result.Deleted || result.Backend != compute.FloatingIPNova || len(state.events) != 4 || len(state.locators) != 1 {
				t.Fatal(result, err, state)
			}
			queryRaw(t, result.LastVerification.FloatingIP.Resource, "status", `"ACTIVE"`)
			queryRaw(t, result.LastVerification.FloatingIP.Wire, "status", `"DOWN"`)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 204, ""
				}
				return 404, "legacy list missing"
			}
			result, err = conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "7"})
			if err != nil || !result.Deleted || !result.Absent || result.LastVerification.SuppressedNotFound == nil {
				t.Fatal(result, err)
			}
		})
	}
}
func TestFloatingIPDeleteVerificationCanFallbackAfterNeutronAcceptance(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "DELETE" {
			return 204, ""
		}
		if strings.Contains(r.URL.Path, "network") {
			return 404, "network list missing"
		}
		return 200, `{"floating_ips":[{"id":"ip","status":"\u0044OWN"}]}`
	}
	result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"})
	if err != nil || !result.Deleted || !result.Down || result.Absent || result.Backend != compute.FloatingIPNeutron || result.LastVerification.Backend != compute.FloatingIPNova || result.LastVerification.FallbackError == nil || len(state.events) != 3 {
		t.Fatal(result, err, state)
	}
	queryRaw(t, result.LastVerification.FloatingIP.Resource, "status", `"\u0044OWN"`)
}
func TestFloatingIPDeletePassiveBodyAndSuccessfulStatusPolicy(t *testing.T) {
	for _, code := range []int{200, 201, 202, 203, 204, 205, 206, 299, 300, 301, 304, 307, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			state.reply = func(*http.Request) (int, string) { return code, "malformed body {" }
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"}, compute.WithFloatingIPDeleteRetries(0))
			if err != nil || !result.Deleted || !result.Attempts[0].Accepted || result.Attempts[0].Response.StatusCode != code || string(result.Attempts[0].Response.Envelope) != "malformed body {" || len(state.events) != 1 {
				t.Fatal(result, err, state)
			}
		})
	}
}

type floatingDeleteFailingBody struct {
	Reader *strings.Reader
	cause  error
}

func (b floatingDeleteFailingBody) Read(p []byte) (int, error) {
	n, _ := b.Reader.Read(p)
	return n, b.cause
}
func (b floatingDeleteFailingBody) Close() error { return nil }
