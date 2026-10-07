package nativefind_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/nativefind"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeListItem struct{ ID, Name, Status string }
type nativeListMode struct {
	kind string
	list func(context.Context, *gophercloud.ServiceClient, url.Values, bool) iter.Seq2[*nativeListItem, error]
}

func nativeListModes() []nativeListMode {
	return []nativeListMode{
		{"servers", func(ctx context.Context, client *gophercloud.ServiceClient, query url.Values, details bool) iter.Seq2[*nativeListItem, error] {
			sequence := nativefind.IterateServers(ctx, client, query, details)
			return func(yield func(*nativeListItem, error) bool) {
				for value, err := range sequence {
					if err != nil {
						yield(nil, err)
						return
					}
					if !yield(&nativeListItem{value.ID, value.Name, value.Status}, nil) {
						return
					}
				}
			}
		}},
		{"volumes", func(ctx context.Context, client *gophercloud.ServiceClient, query url.Values, details bool) iter.Seq2[*nativeListItem, error] {
			sequence := nativefind.IterateVolumes(ctx, client, query, details)
			return func(yield func(*nativeListItem, error) bool) {
				for value, err := range sequence {
					if err != nil {
						yield(nil, err)
						return
					}
					if !yield(&nativeListItem{value.ID, value.Name, value.Status}, nil) {
						return
					}
				}
			}
		}},
	}
}

func nativeListPath(mode nativeListMode, details bool) string {
	path := "/reverse/" + mode.kind
	if details {
		path += "/detail"
	}
	return path
}

func nativeListPayload(kind, id, status, next string) string {
	links := "[]"
	if next != "" {
		links = fmt.Sprintf(`[{"rel":"next","href":%q}]`, next)
	}
	return fmt.Sprintf(`{%q:[{"id":%q,"name":"literal-name","status":%q}],%q:%s}`, kind, id, status, kind+"_links", links)
}

func TestNativeIdentityListModesRoutesFrozenQueriesAndLiveSource(t *testing.T) {
	for _, mode := range nativeListModes() {
		for _, details := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/details=%t", mode.kind, details), func(t *testing.T) {
				cloud := testcloud.New(t)
				service := "compute"
				if mode.kind == "volumes" {
					service = "block-storage"
				}
				client := cloud.Client(service, "/unused")
				client.ResourceBase = cloud.Server.URL + "/reverse/"
				client.MoreHeaders = map[string]string{"X-Source": "original"}
				client.Microversion = "2.7"
				provider := client.ProviderClient
				originalEndpoint, originalBase := client.Endpoint, client.ResourceBase
				path := nativeListPath(mode, details)
				input := url.Values{"tag": {"first", "second"}, "name": {"a&b /한글"}, "empty": nil, "all_tenants": {"true"}}
				expected := input.Encode()
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, request *http.Request) {
					call := calls.Add(1)
					requestQuery := request.URL.Query()
					requestQuery.Del("marker")
					if requestQuery.Encode() != expected || request.Header.Get("X-Source") != "original" {
						t.Error(request.URL, request.Header)
					}
					if call == 1 && request.Header.Get("X-Auth-Token") != "test-token" {
						t.Error(request.Header)
					}
					if call > 1 && request.Header.Get("X-Auth-Token") != "renewed" {
						t.Error(request.Header)
					}
					next := ""
					id := "second"
					if request.URL.Query().Get("marker") == "" {
						next = cloud.Server.URL + path + "?" + expected + "&marker=next"
						id = "first"
					}
					// First-page and server-owned continuation queries retain the
					// same caller values; marker is introduced only by the link.
					if request.URL.Query().Get("marker") != "" && request.URL.Query().Get("marker") != "next" {
						t.Error(request.URL)
					}
					testcloud.JSON(w, 200, nativeListPayload(mode.kind, id, "ACTIVE", next))
				})
				// Handle the extra server continuation marker separately from
				// the literal first-page encoded query assertion above.
				baseTransport := provider.HTTPClient.Transport
				if baseTransport == nil {
					baseTransport = http.DefaultTransport
				}
				var middleware atomic.Int32
				provider.HTTPClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
					middleware.Add(1)
					return baseTransport.RoundTrip(request)
				})
				sequence := mode.list(context.Background(), client, input, details)
				input["tag"][0] = "caller-mutation"
				input["name"] = []string{"changed"}
				delete(input, "all_tenants")
				var ids []string
				for value, err := range sequence {
					if err != nil {
						t.Fatal(err)
					}
					ids = append(ids, value.ID)
					if value.ID == "first" {
						provider.SetToken("renewed")
					}
				}
				if !reflect.DeepEqual(ids, []string{"first", "second"}) || calls.Load() != 2 || middleware.Load() != 2 {
					t.Fatal(ids, calls.Load(), middleware.Load())
				}
				if client.ProviderClient != provider || client.Endpoint != originalEndpoint || client.ResourceBase != originalBase || client.Microversion != "2.7" || client.MoreHeaders["X-Source"] != "original" {
					t.Fatal("source client changed", client)
				}
			})
		}
	}
}

func TestNativeIdentityListModesNativeCodesAndWholePageErrors(t *testing.T) {
	for _, mode := range nativeListModes() {
		for _, details := range []bool{false, true} {
			for _, status := range []int{200, 204, 300, 203, 403, 404} {
				t.Run(fmt.Sprintf("%s/%t/code%d", mode.kind, details, status), func(t *testing.T) {
					cloud := testcloud.New(t)
					client := cloud.Client("compute", "/reverse")
					var calls atomic.Int32
					cloud.Mux.HandleFunc("GET "+nativeListPath(mode, details), func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if status == 204 {
							w.WriteHeader(status)
							return
						}
						w.Header().Set("X-Evidence", "native-page")
						testcloud.JSON(w, status, nativeListPayload(mode.kind, "row", "ACTIVE", ""))
					})
					var values int
					var result error
					for _, err := range mode.list(context.Background(), client, nil, details) {
						if err != nil {
							result = err
							break
						}
						values++
					}
					if status == 200 || status == 300 {
						if result != nil || values != 1 {
							t.Fatal(values, result)
						}
					} else if status == 204 {
						if result != nil || values != 0 {
							t.Fatal(values, result)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(result, &native) || native.Actual != status || native.ResponseHeader.Get("X-Evidence") != "native-page" || len(native.Body) == 0 {
							t.Fatal(values, result, native)
						}
					}
					if calls.Load() != 1 {
						t.Fatal("request replay", calls.Load())
					}
				})
			}
			t.Run(fmt.Sprintf("%s/%t/malformed-trailing-row", mode.kind, details), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("compute", "/reverse")
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+nativeListPath(mode, details), func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"valid","name":"n"},{"id":true}]}`, mode.kind))
				})
				values := 0
				var result error
				for _, err := range mode.list(context.Background(), client, nil, details) {
					if err != nil {
						result = err
						break
					}
					values++
				}
				var typed *json.UnmarshalTypeError
				if !errors.As(result, &typed) || values != 0 || calls.Load() != 1 {
					t.Fatal(values, result, calls.Load())
				}
			})
		}
	}
}

func TestNativeIdentityListModesBreakCancellationAndPreflight(t *testing.T) {
	for _, mode := range nativeListModes() {
		for _, details := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", mode.kind, details), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("compute", "/reverse")
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+nativeListPath(mode, details), func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, nativeListPayload(mode.kind, "first", "ACTIVE", cloud.Server.URL+"/must-not-follow"))
				})
				for value, err := range mode.list(context.Background(), client, nil, details) {
					if err != nil || value.ID != "first" {
						t.Fatal(value, err)
					}
					break
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var result error
				for value, err := range mode.list(ctx, client, nil, details) {
					if err != nil {
						result = err
						break
					}
					if value.ID != "first" {
						t.Fatal(value)
					}
					cancel()
				}
				if !errors.Is(result, context.Canceled) || calls.Load() != 2 {
					t.Fatal(result, calls.Load())
				}
				cancelled, stop := context.WithCancel(context.Background())
				stop()
				for _, err := range mode.list(cancelled, client, nil, details) {
					if !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				}
				for _, source := range []*gophercloud.ServiceClient{nil, {Endpoint: client.Endpoint}} {
					for _, err := range mode.list(context.Background(), source, nil, details) {
						if !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(source, err)
						}
					}
				}
				for _, err := range mode.list(nil, client, nil, details) {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				}
				for _, endpoint := range []string{"/relative/", cloud.Server.URL + "/reverse/?query=old", cloud.Server.URL + "/reverse/#fragment", "http://user:password@host/reverse/", "ftp://host/reverse/", "http://host/%zz/"} {
					copy := *client
					copy.ResourceBase = endpoint
					for _, err := range mode.list(context.Background(), &copy, nil, details) {
						if !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(endpoint, err)
						}
					}
				}
				if calls.Load() != 2 {
					t.Fatal("preflight or broken consumer performed HTTP", calls.Load())
				}
			})
		}
	}
}

func TestNativeIdentityListModesCyclesForeignLinksAndLaterFailure(t *testing.T) {
	for _, mode := range nativeListModes() {
		for _, details := range []bool{false, true} {
			for _, scenario := range []string{"cycle", "native-next", "later-403", "malformed-link"} {
				t.Run(fmt.Sprintf("%s/%t/%s", mode.kind, details, scenario), func(t *testing.T) {
					cloud := testcloud.New(t)
					nextCloud := testcloud.New(t)
					client := cloud.Client("compute", "/reverse")
					var firstCalls, nextCalls atomic.Int32
					nextCloud.Mux.HandleFunc("GET /next", func(w http.ResponseWriter, r *http.Request) {
						nextCalls.Add(1)
						if scenario == "later-403" {
							testcloud.JSON(w, 403, `{"error":"denied"}`)
							return
						}
						testcloud.JSON(w, 200, nativeListPayload(mode.kind, "second", "ACTIVE", ""))
					})
					cloud.Mux.HandleFunc("GET "+nativeListPath(mode, details), func(w http.ResponseWriter, r *http.Request) {
						firstCalls.Add(1)
						next := nextCloud.Server.URL + "/next"
						if scenario == "cycle" {
							next = cloud.Server.URL + nativeListPath(mode, details)
						}
						payload := nativeListPayload(mode.kind, "first", "ACTIVE", next)
						if scenario == "malformed-link" {
							payload = fmt.Sprintf(`{%q:[{"id":"first","name":"n"}],%q:[{"rel":"next","href":true}]}`, mode.kind, mode.kind+"_links")
						}
						testcloud.JSON(w, 200, payload)
					})
					var ids []string
					var result error
					for value, err := range mode.list(context.Background(), client, nil, details) {
						if err != nil {
							result = err
							break
						}
						ids = append(ids, value.ID)
					}
					switch scenario {
					case "cycle":
						var cycle *resource.PaginationCycleError
						if !errors.As(result, &cycle) || nextCalls.Load() != 0 {
							t.Fatal(ids, result, nextCalls.Load())
						}
					case "native-next":
						if result != nil || !reflect.DeepEqual(ids, []string{"first", "second"}) || nextCalls.Load() != 1 {
							t.Fatal(ids, result, nextCalls.Load())
						}
					case "later-403":
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(result, &native) || native.Actual != 403 || nextCalls.Load() != 1 {
							t.Fatal(ids, result, nextCalls.Load())
						}
					case "malformed-link":
						var typed *json.UnmarshalTypeError
						if !errors.As(result, &typed) || nextCalls.Load() != 0 {
							t.Fatal(ids, result, nextCalls.Load())
						}
					}
					if firstCalls.Load() != 1 {
						t.Fatal("initial page replay", firstCalls.Load())
					}
				})
			}
		}
	}
}

type nativeListReadError struct{ cause error }

func (body nativeListReadError) Read([]byte) (int, error) { return 0, body.cause }
func (nativeListReadError) Close() error                  { return nil }

func TestNativeIdentityListModesTransportAndAcceptedReadErrorsStayTerminal(t *testing.T) {
	for _, mode := range nativeListModes() {
		for _, details := range []bool{false, true} {
			for _, read := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%t/read=%t", mode.kind, details, read), func(t *testing.T) {
					cloud := testcloud.New(t)
					client := cloud.Client("compute", "/reverse")
					sentinel := errors.New("original-native-cause")
					var calls atomic.Int32
					cloud.Provider.HTTPClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
						calls.Add(1)
						if !read {
							return nil, sentinel
						}
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Request: request, Body: nativeListReadError{sentinel}}, nil
					})
					var result error
					values := 0
					for _, err := range mode.list(context.Background(), client, nil, details) {
						if err != nil {
							result = err
							break
						}
						values++
					}
					if !errors.Is(result, sentinel) || values != 0 || calls.Load() != 1 {
						t.Fatal(values, result, calls.Load())
					}
				})
			}
		}
	}
}

func TestNativeIdentityListModesReusableAndConcurrentIndependentSnapshots(t *testing.T) {
	for _, mode := range nativeListModes() {
		t.Run(mode.kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/reverse")
			var calls atomic.Int32
			for _, details := range []bool{false, true} {
				cloud.Mux.HandleFunc("GET "+nativeListPath(mode, details), func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					name := r.URL.Query().Get("name")
					if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"first", "second"}) {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, nativeListPayload(mode.kind, name, "ACTIVE", ""))
				})
			}
			q := url.Values{"name": {"summary"}, "tag": {"first", "second"}}
			summary := mode.list(context.Background(), client, q, false)
			q["name"][0] = "details"
			detailed := mode.list(context.Background(), client, q, true)
			q["name"][0] = "caller-changed"
			q["tag"][0] = "caller-changed"
			var wg sync.WaitGroup
			for _, entry := range []struct {
				name     string
				sequence iter.Seq2[*nativeListItem, error]
			}{{"summary", summary}, {"details", detailed}} {
				for range 3 {
					wg.Add(1)
					go func(name string, sequence iter.Seq2[*nativeListItem, error]) {
						defer wg.Done()
						count := 0
						for value, err := range sequence {
							if err != nil || value.ID != name {
								t.Error(value, err, name)
							}
							count++
						}
						if count != 1 {
							t.Error(count)
						}
					}(entry.name, entry.sequence)
				}
			}
			wg.Wait()
			if calls.Load() != 6 {
				t.Fatal(calls.Load())
			}
		})
	}
}
