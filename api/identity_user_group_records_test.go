package api_test

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

const userGroupRecordsPath = "/reverse/identity/v3/users/user-id/groups"

func collectUserGroupRecords(t *testing.T, sequence iter.Seq2[*users.UserGroupRecord, error]) ([]*users.UserGroupRecord, error) {
	t.Helper()
	values := make([]*users.UserGroupRecord, 0)
	for value, err := range sequence {
		if err != nil {
			return values, err
		}
		if value == nil || value.Resource == nil || value.Wire == nil {
			t.Fatal("successful group lacks owned Resource or Wire", value)
		}
		values = append(values, value)
	}
	return values, nil
}

func TestIdentityUserGroupRecordsOwnedViewAndCachedConnection(t *testing.T) {
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/catalog/identity/v3/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Identity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cached, err := conn.IdentityV3(context.Background())
	if err != nil || cached != service || service.Users.RawClient() != service.RawClient() || service.RawClient().ProviderClient != cloud.Provider {
		t.Fatal(service, cached, err)
	}
	service.RawClient().ResourceBase = cloud.Server.URL + "/reverse/identity/v3/"
	service.RawClient().MoreHeaders = map[string]string{"X-Source": "cached"}
	cloud.Provider.SetToken("group-live")
	var calls atomic.Int32
	cloud.Mux.HandleFunc(userGroupRecordsPath, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		th.TestMethod(t, r, http.MethodGet)
		th.TestHeader(t, r, "X-Auth-Token", "group-live")
		th.TestHeader(t, r, "X-Source", "cached")
		if r.URL.RawQuery != "" || r.ContentLength != 0 {
			t.Error("public user_groups has no query or body", r.URL, r.ContentLength)
		}
		w.Header().Set("X-Group-Record", "actual")
		testcloud.JSON(w, 200, `{"groups":[{"id":null,"user_id":"server-user","domain_id":null,"name":false,"description":null,"links":null,"options":false,"vendor":9007199254740993123},{"id":"second","name":"owned","links":{"self":"https://passive.invalid/group"},"vendor":{"flags":[false,null]}}]}`)
	})
	sequence := service.Users.ListGroupRecords(context.Background(), "user-id")
	if calls.Load() != 0 {
		t.Fatal("owned group iterator was eager")
	}
	for repeat := 0; repeat < 2; repeat++ {
		values, err := collectUserGroupRecords(t, sequence)
		if err != nil || len(values) != 2 || calls.Load() != int32(repeat+1) {
			t.Fatal(values, err, calls.Load())
		}
		for index, value := range values {
			if string(value.Resource.Body["user_id"]) != `"user-id"` || value.Resource.StatusCode != 200 || value.Wire.StatusCode != 200 || value.Resource.Header.Get("X-Group-Record") != "actual" || value.Wire.Header.Get("X-Group-Record") != "actual" {
				t.Fatal("URI/receipt projection", index, value)
			}
		}
		for _, key := range []string{"id", "domain_id", "description", "links"} {
			if string(values[0].Resource.Body[key]) != "null" || string(values[0].Wire.Body[key]) != "null" {
				t.Fatal("nullable known field changed", key, values[0])
			}
		}
		if string(values[0].Resource.Body["name"]) != "false" || string(values[0].Resource.Body["options"]) != "false" || string(values[0].Wire.Body["options"]) != "false" || string(values[0].Resource.Body["vendor"]) != "9007199254740993123" || string(values[0].Wire.Body["user_id"]) != `"server-user"` {
			t.Fatal("group raw types coerced or wire URI changed", values[0])
		}
		if _, exists := values[1].Wire.Body["user_id"]; exists {
			t.Fatal("parent user leaked into actual Wire")
		}
		values[0].Resource.Body["name"][0] = 'X'
		values[0].Resource.Header.Set("X-Group-Record", "view changed")
		values[1].Wire.Body["name"][1] = 'X'
		values[1].Wire.Header.Set("X-Group-Record", "wire changed")
		if string(values[0].Wire.Body["name"]) != "false" || values[0].Wire.Header.Get("X-Group-Record") != "actual" || string(values[1].Resource.Body["name"]) != `"owned"` || values[1].Resource.Header.Get("X-Group-Record") != "actual" || values[0].Wire.Header.Get("X-Group-Record") != "actual" {
			t.Fatal("Resource/Wire, fields or row metadata alias")
		}
	}
}

func TestIdentityUserGroupRecordsContinuationFormsAndConsumerStop(t *testing.T) {
	for _, mode := range []string{"dictionary", "links", "groups_links", "next", "HTTP Link", "server limit", "empty stops", "no content", "break"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(userGroupRecordsPath, func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.ContentLength != 0 || r.URL.Path != userGroupRecordsPath {
					t.Error(r.URL, r.ContentLength)
				}
				if call == 1 {
					th.TestHeader(t, r, "X-Auth-Token", "test-token")
					if r.URL.RawQuery != "" {
						t.Error("initial group query", r.URL)
					}
				} else {
					th.TestHeader(t, r, "X-Auth-Token", "group-next")
					wantQuery := "marker=next"
					if mode == "server limit" {
						wantQuery = "limit=25&marker=next"
					}
					if call != 2 || r.URL.RawQuery != wantQuery {
						t.Error("group continuation", r.URL, call)
					}
					testcloud.JSON(w, 200, `{"groups":[{"id":"second"}]}`)
					return
				}
				cloud.Provider.SetToken("group-next")
				if mode == "no content" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next := cloud.Server.URL + userGroupRecordsPath + "?marker=next"
				if mode == "server limit" {
					next += "&limit=25"
				}
				continuation := ""
				switch mode {
				case "dictionary", "break":
					continuation = `,"links":{"next":"` + next + `"}`
				case "links", "server limit", "empty stops":
					continuation = `,"links":[{"rel":"next","href":"` + next + `"}]`
				case "groups_links":
					continuation = `,"groups_links":[{"rel":"next","href":"` + next + `"}]`
				case "next":
					continuation = `,"next":"` + next + `"`
				case "HTTP Link":
					w.Header().Set("Link", "<"+next+">; rel=\"next\"")
				}
				rows := `{"id":"first"}`
				if mode == "empty stops" {
					rows = ""
				} else if mode == "break" {
					rows += `,{"id":"unconsumed"}`
				}
				testcloud.JSON(w, 200, `{"groups":[`+rows+`]`+continuation+`}`)
			})
			api := users.New(cloud.Client("identity", "/reverse/identity/v3"))
			ids := make([]string, 0)
			for value, err := range api.ListGroupRecords(context.Background(), "user-id") {
				if err != nil || value == nil || value.Resource == nil || value.Wire == nil || string(value.Resource.Body["user_id"]) != `"user-id"` {
					t.Fatal(value, err)
				}
				ids = append(ids, string(value.Wire.Body["id"]))
				if mode == "break" {
					break
				}
			}
			wantIDs, wantCalls := []string{`"first"`, `"second"`}, int32(2)
			if mode == "empty stops" || mode == "no content" {
				wantIDs, wantCalls = []string{}, 1
			} else if mode == "break" {
				wantIDs, wantCalls = []string{`"first"`}, 1
			}
			if !reflect.DeepEqual(ids, wantIDs) || calls.Load() != wantCalls {
				t.Fatal(ids, calls.Load(), wantIDs, wantCalls)
			}
		})
	}
}

func TestIdentityUserGroupRecordsPreflightFaultsAndPartialResults(t *testing.T) {
	for _, mode := range []string{"invalid ID", "nil API", "cancel before", "read", "close", "HTTP403", "late bad object", "cancel after row", "source transport", "late source transport"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			client := cloud.Client("identity", "/reverse/identity/v3")
			api := users.New(client)
			userID := "user-id"
			cause := errors.New("group record " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			first := `{"groups":[{"id":"first"}],"links":{"next":"` + cloud.Server.URL + userGroupRecordsPath + `?marker=next"}}`
			second := `{"groups":[{"id":"second"}]}`
			forbidden := `{"error":{"message":"group membership denied"}}`
			bad := `{"groups":[false]}`
			cloud.Mux.HandleFunc(userGroupRecordsPath, func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				if r.ContentLength != 0 || call == 1 && r.URL.RawQuery != "" || call == 2 && r.URL.RawQuery != "marker=next" || call > 2 {
					t.Error(r.URL, call, r.ContentLength)
				}
				w.Header().Set("X-Group-Page", strconv.Itoa(int(call)))
				if mode == "HTTP403" {
					testcloud.JSON(w, 403, forbidden)
				} else if call == 1 {
					testcloud.JSON(w, 200, first)
				} else if mode == "late bad object" {
					testcloud.JSON(w, 200, bad)
				} else {
					testcloud.JSON(w, 200, second)
				}
			})
			var track *payloadContractTracking
			switch mode {
			case "invalid ID":
				userID = "."
			case "nil API":
				api = nil
			case "cancel before":
				cancel(cause)
			case "read":
				track = payloadContractTrack(cloud, cause, nil)
			case "close":
				track = payloadContractTrack(cloud, nil, cause)
			case "source transport", "late source transport":
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil && (mode == "source transport" || calls.Load() == 2) {
						client.ResourceBase = cloud.Server.URL + "/changed/identity/v3/"
					}
					return response, err
				})
			}
			sequence := api.ListGroupRecords(ctx, userID)
			if calls.Load() != 0 {
				t.Fatal("group failure iterator was eager")
			}
			rows, failures := 0, 0
			for value, err := range sequence {
				if err == nil {
					rows++
					if value == nil || value.Resource == nil || value.Wire == nil || string(value.Wire.Body["id"]) != `"first"` || value.Wire.StatusCode != 200 || value.Wire.Header.Get("X-Group-Page") != "1" {
						t.Fatal("partial group lost", value)
					}
					if mode == "cancel after row" {
						cancel(cause)
					}
					continue
				}
				failures++
				if value != nil {
					t.Fatal("failed group returned a successful record", value, err)
				}
				switch mode {
				case "invalid ID", "nil API":
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				case "cancel before", "cancel after row":
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal("custom cancellation cause lost", err)
					}
				case "HTTP403":
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != forbidden || native.ResponseHeader.Get("X-Group-Page") != "1" {
						t.Fatal(err, native)
					}
				default:
					var proof *resource.ResponseError
					wantBody, wantPage := first, "1"
					if mode == "late bad object" {
						wantBody, wantPage = bad, "2"
					} else if mode == "late source transport" {
						wantBody, wantPage = second, "2"
					}
					if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != wantBody || proof.Header.Get("X-Group-Page") != wantPage {
						t.Fatal("accepted page proof lost", err, proof)
					}
					if mode == "read" || mode == "close" {
						if !errors.Is(err, cause) {
							t.Fatal("original body failure lost", err)
						}
					} else if mode == "source transport" || mode == "late source transport" {
						if !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal("source replacement admitted", err)
						}
					}
				}
			}
			wantRows, wantCalls := 0, int32(1)
			switch mode {
			case "invalid ID", "nil API", "cancel before":
				wantCalls = 0
			case "late bad object", "late source transport":
				wantRows, wantCalls = 1, 2
			case "cancel after row":
				wantRows = 1
			}
			if rows != wantRows || failures != 1 || calls.Load() != wantCalls {
				t.Fatal(rows, failures, calls.Load(), wantRows, wantCalls)
			}
			if track != nil && (track.calls.Load() != 1 || track.last(t).closes.Load() != 1) {
				t.Fatal("failed body replay/close ownership", track.calls.Load(), track.last(t).closes.Load())
			}
		})
	}
}
