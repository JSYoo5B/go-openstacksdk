package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const getVolumesContractBase = "/reverse/cinder/v3/project/"
const getVolumesContractPath = getVolumesContractBase + "volumes/detail"

func getVolumesContractClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("volumev3", "/unused/cinder/")
	client.ResourceBase = cloud.Server.URL + getVolumesContractBase
	client.Microversion = "3.60"
	client.MoreHeaders = map[string]string{"x-source": "entry"}
	return client
}
func getVolumesContractWire(t *testing.T, r *http.Request, token string) {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != getVolumesContractPath || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "volume 3.60" {
		t.Error(r.Method, r.URL, r.Header)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Error("list request body", body, err)
		}
	}
}
func getVolumesContractPage(rows, href string) string {
	tail := ""
	if href != "" {
		encoded, _ := json.Marshal(href)
		tail = `,"volumes_links":[{"rel":"next","href":` + string(encoded) + `}]`
	}
	return `{"volumes":` + rows + tail + `}`
}
func getVolumesContractOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "GetVolumes" || operation.Resource != "volume" || operation.Cause == nil {
		t.Fatal("GetVolumes operation context lost", err)
	}
}

type getVolumesContractTransport func(*http.Request) (*http.Response, error)

func (f getVolumesContractTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func getVolumesContractResponse(r *http.Request, code int, body io.ReadCloser, proof string) *http.Response {
	return &http.Response{Request: r, StatusCode: code, Header: http.Header{"X-Proof": {proof}}, Body: body}
}

type getVolumesContractReader struct {
	data                  *strings.Reader
	readError, closeError error
	onClose               func()
	once                  sync.Once
	closes                atomic.Int32
}

func (b *getVolumesContractReader) Read(p []byte) (int, error) {
	if b.readError != nil {
		n, _ := b.data.Read(p)
		cause := b.readError
		b.readError = nil
		return n, cause
	}
	return b.data.Read(p)
}
func (b *getVolumesContractReader) Close() error {
	b.closes.Add(1)
	b.once.Do(func() {
		if b.onClose != nil {
			b.onClose()
		}
	})
	return b.closeError
}

func TestGetVolumesCompletePagingPreservesAttachmentDuplicatesAndDistinctRawRows(t *testing.T) {
	cloud := testcloud.New(t)
	client := getVolumesContractClient(cloud)
	var calls atomic.Int32
	first := getVolumesContractPage(`[{"id":false,"row":"first","status":23,"created_at":{},"size":false,"attachments":[{"server_id":true,"device":false},{"server_id":1,"device":{"opaque":true}},{"server_id":null}]},{"id":false,"row":"second","attachments":[{"server_id":1}]}]`, `?limit=2&marker=a%2Fb`)
	second := getVolumesContractPage(`[{"id":false,"row":"third","attachments":{"server_id":1,"device":null}}]`, "")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		getVolumesContractWire(t, r, "test-token")
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error("initial server/all-project/name filters leaked to list", r.URL)
			}
			w.Header().Set("X-Proof", "first")
			testcloud.JSON(w, 200, first)
		case 2:
			if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("marker") != "a/b" {
				t.Error("advertised native query changed", r.URL)
			}
			w.Header().Set("X-Proof", "second")
			testcloud.JSON(w, 200, second)
		default:
			t.Error("per-volume GET, duplicate refresh or list restart", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{ServerID: "unused"}, blockstorage.WithGetVolumesServerFields(map[string]json.RawMessage{"id": json.RawMessage(`1`)}))
	if err != nil || result == nil || len(result.Pages) != 2 || len(result.Volumes) != 4 || calls.Load() != 2 || result.Volumes[0] != result.Volumes[1] || result.Volumes[0] == result.Volumes[2] || result.Volumes[2] == result.Volumes[3] || string(result.Volumes[0].Body["row"]) != `"first"` || string(result.Volumes[2].Body["row"]) != `"second"` || string(result.Volumes[3].Body["row"]) != `"third"` {
		t.Fatal(result, err, calls.Load())
	}
	for _, row := range result.Volumes {
		if row.StatusCode != 200 || string(row.Body["id"]) != "false" {
			t.Fatal("known volume fields were coerced/decoded", row)
		}
	}
	if result.Volumes[0].Header.Get("X-Proof") != "first" || result.Volumes[3].Header.Get("X-Proof") != "second" {
		t.Fatal("raw row lost originating page metadata")
	}
	result.Pages[0].Body[0] = '!'
	result.Pages[0].Header.Set("X-Proof", "caller")
	result.Volumes[0].Header.Set("X-Proof", "duplicate")
	result.Volumes[0].Body["row"] = json.RawMessage(`"duplicate mutation"`)
	if string(result.Volumes[1].Body["row"]) != `"duplicate mutation"` || result.Volumes[1].Header.Get("X-Proof") != "duplicate" || result.Volumes[2].Header.Get("X-Proof") != "first" || string(result.Volumes[2].Body["row"]) != `"second"` || string(result.Pages[1].Body) != second || result.Pages[1].Header.Get("X-Proof") != "second" {
		t.Fatal("duplicates or independent rows/pages have wrong ownership", result)
	}
	clone := result.Volumes[2].Clone()
	clone.Body["row"][0] = '!'
	clone.Header.Set("X-Proof", "clone")
	if string(result.Volumes[2].Body["row"]) != `"second"` || result.Volumes[2].Header.Get("X-Proof") != "first" {
		t.Fatal("explicit row Clone did not separate ownership")
	}
}

func TestGetVolumesConsumesAssociationAndServerEvidenceOnlyAfterCompleteList(t *testing.T) {
	for _, tc := range []struct {
		name, rows    string
		fields        map[string]json.RawMessage
		responseError bool
		key           string
		success       bool
	}{{name: "response server key before bad provided ID", rows: `[{"attachments":[{}]}]`, fields: map[string]json.RawMessage{"id": json.RawMessage(`broken`)}, responseError: true, key: "server_id"}, {name: "invalid consumed provided ID", rows: `[{"attachments":[{"server_id":"server"}]}]`, fields: map[string]json.RawMessage{"id": json.RawMessage(`broken`)}, key: "id"}, {name: "invalid consumed UTF-8 server ID", rows: `[{"attachments":[{"server_id":"server"}]}]`, fields: map[string]json.RawMessage{"id": json.RawMessage([]byte{'"', 0xff, '"'})}, key: "id"}, {name: "missing provided canonical ID", rows: `[{"attachments":[{"server_id":"server"}]}]`, fields: map[string]json.RawMessage{"ID": json.RawMessage(`"server"`)}, key: "id"}, {name: "nil map is missing server evidence", rows: `[{"attachments":[{"server_id":"server"}]}]`, key: "id"}, {name: "no volumes never consume invalid server ID", rows: `[]`, fields: map[string]json.RawMessage{"id": json.RawMessage(`broken`)}, success: true}, {name: "empty attachments never consume missing server ID", rows: `[{"attachments":[]}]`, success: true}, {name: "empty attachments never consume invalid UTF-8 server ID", rows: `[{"attachments":[]}]`, fields: map[string]json.RawMessage{"id": json.RawMessage([]byte{'"', 0xff, '"'})}, success: true}, {name: "later malformed attachment fails entire result", rows: `[{"attachments":[{"server_id":"server"},{}]}]`, fields: map[string]json.RawMessage{"id": json.RawMessage(`"server"`)}, responseError: true, key: "server_id"}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			var calls atomic.Int32
			first, second := getVolumesContractPage(tc.rows, "?page=2"), getVolumesContractPage(`[]`, "")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				getVolumesContractWire(t, r, "test-token")
				if calls.Add(1) == 1 {
					w.Header().Set("X-Proof", "origin")
					testcloud.JSON(w, 200, first)
				} else {
					w.Header().Set("X-Proof", "later")
					testcloud.JSON(w, 200, second)
				}
			})
			result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{ServerID: "ignored"}, blockstorage.WithGetVolumesServerFields(tc.fields))
			expectedPages := 2
			if tc.rows == `[]` {
				expectedPages = 1
			}
			if result == nil || len(result.Pages) != expectedPages || calls.Load() != int32(expectedPages) {
				t.Fatal("association/server validation happened before list completion or ignored empty-page EOF", result, err, calls.Load())
			}
			if tc.success {
				if err != nil || result.Volumes == nil || len(result.Volumes) != 0 {
					t.Fatal(result, err)
				}
				return
			}
			var accepted *resource.ResponseError
			if err == nil || result.Volumes != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &accepted) != tc.responseError || !strings.Contains(err.Error(), tc.key) {
				t.Fatal(result, err)
			}
			if tc.responseError && (accepted.StatusCode != 200 || string(accepted.Body) != first || accepted.Header.Get("X-Proof") != "origin") {
				t.Fatal("association error borrowed latest page instead of originating row", err)
			}
			getVolumesContractOperation(t, err)
		})
	}
	for _, tc := range []struct {
		name, rows string
		fields     map[string]json.RawMessage
	}{{"later native rejection beats earlier malformed attachment", `[{"attachments":[{}]}]`, map[string]json.RawMessage{"id": json.RawMessage(`"server"`)}}, {"later native rejection beats earlier invalid server fields", `[{"attachments":[{"server_id":"server"}]}]`, map[string]json.RawMessage{"id": json.RawMessage(`broken`)}}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			var calls atomic.Int32
			first := getVolumesContractPage(tc.rows, "?page=2")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				getVolumesContractWire(t, r, "test-token")
				if calls.Add(1) == 1 {
					w.Header().Set("X-Proof", "first")
					testcloud.JSON(w, 200, first)
				} else {
					w.Header().Set("X-Proof", "rejected")
					testcloud.JSON(w, 403, `{"error":"denied later page"}`)
				}
			})
			result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{}, blockstorage.WithGetVolumesServerFields(tc.fields))
			var accepted *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.Volumes != nil || len(result.Pages) != 1 || string(result.Pages[0].Body) != first || calls.Load() != 2 || errors.As(err, &accepted) || errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "rejected" {
				t.Fatal("later page error was displaced by unconsumed association/server evidence", result, err, calls.Load())
			}
			getVolumesContractOperation(t, err)
		})
	}

}

func TestGetVolumesV3ListifiedAttachmentsAndFullJSONServerEqualityIgnoreDeviceSchema(t *testing.T) {
	for _, tc := range []struct {
		name, attachments, serverID string
		override                    bool
		id                          json.RawMessage
		matches                     int
		invalid                     bool
	}{{name: "missing attachments", invalid: true}, {name: "string default ID does not coerce numeric attachment", serverID: "1", attachments: `[{"server_id":1}]`}, {name: "object listifies as one row", attachments: `{"server_id":1,"device":{"opaque":true}}`, override: true, id: json.RawMessage(`true`), matches: 1}, {name: "recursive bool numeric", attachments: `[{"server_id":[1,{"a":0}],"device":false}]`, override: true, id: json.RawMessage(`[true,{"a":false}]`), matches: 1}, {name: "full objects are not subset filters", attachments: `[{"server_id":{"a":1,"extra":true}}]`, override: true, id: json.RawMessage(`{"a":true}`)}, {name: "explicit nil raw ID is JSON null", attachments: `[{"server_id":null}]`, override: true, matches: 1}, {name: "exact numeric precision", attachments: `[{"server_id":9007199254740993}]`, override: true, id: json.RawMessage(`9007199254740992`)}, {name: "literal empty default server", attachments: `[{"server_id":"","device":""}]`, matches: 1}, {name: "literal punctuation default server", attachments: `[{"server_id":" /server?x# "}]`, serverID: " /server?x# ", matches: 1}, {name: "case sensitive default server", attachments: `[{"server_id":"SERVER"}]`, serverID: "server"}, {name: "null attachments", attachments: `null`, invalid: true}, {name: "empty object still listifies one incompatible row", attachments: `{}`, invalid: true}, {name: "empty string listifies incompatible row", attachments: `""`, invalid: true}, {name: "bool row", attachments: `false`, invalid: true}, {name: "numeric row", attachments: `23`, invalid: true}, {name: "null row", attachments: `[null]`, invalid: true}, {name: "wrong-case consumed key", attachments: `[{"Server_ID":"server"}]`, serverID: "server", invalid: true}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			var calls atomic.Int32
			row := `{"id":false,"status":{},"created_at":false}`
			if tc.attachments != "" {
				row = row[:len(row)-1] + `,"attachments":` + tc.attachments + `}`
			}
			body := getVolumesContractPage("["+row+"]", "")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				getVolumesContractWire(t, r, "test-token")
				testcloud.JSON(w, 200, body)
			})
			options := []blockstorage.GetVolumesOption{}
			if tc.override {
				options = append(options, blockstorage.WithGetVolumesServerFields(map[string]json.RawMessage{"id": tc.id}))
			}
			result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{ServerID: tc.serverID}, options...)
			if result == nil || len(result.Pages) != 1 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if tc.invalid {
				var accepted *resource.ResponseError
				if err == nil || result.Volumes != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body {
					t.Fatal(result, err)
				}
			} else if err != nil || result.Volumes == nil || len(result.Volumes) != tc.matches {
				t.Fatal(result, err)
			}
		})
	}
}

func TestGetVolumesContinuationScopeCyclesAndCanonicalEnvelopeUseActualPageProof(t *testing.T) {
	for _, href := range []string{"https://foreign.invalid/volumes/detail", "/other/volumes/detail", "/reverse/cinder/v3/project/volumes/detail/child", "http://user:password@localhost/path", "?bad=%zz", "?bad=1;other=2", "?page=2#fragment"} {
		t.Run(href, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			var calls atomic.Int32
			body := getVolumesContractPage(`[{"attachments":[]}]`, href)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "current")
				testcloud.JSON(w, 200, body)
			})
			result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{})
			var accepted *resource.ResponseError
			if result == nil || result.Volumes != nil || len(result.Pages) != 1 || err == nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Proof") != "current" || calls.Load() != 1 {
				t.Fatal("unsafe continuation was followed or lacked originating page evidence", result, err, calls.Load())
			}
		})
	}
	t.Run("query-order canonical cycle", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := getVolumesContractClient(cloud)
		var calls atomic.Int32
		first, second := getVolumesContractPage(`[{"attachments":[]}]`, "?b=2&a=1"), getVolumesContractPage(`[{"attachments":[]}]`, "?a=1&b=2")
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			step := calls.Add(1)
			if step == 1 {
				testcloud.JSON(w, 200, first)
			} else if step == 2 {
				if r.URL.Query().Get("a") != "1" || r.URL.Query().Get("b") != "2" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Proof", "cycle")
				testcloud.JSON(w, 200, second)
			} else {
				t.Error("equivalent query cycle was physically followed", r.URL)
				w.WriteHeader(500)
			}
		})
		result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{})
		var accepted *resource.ResponseError
		if result == nil || result.Volumes != nil || len(result.Pages) != 2 || !errors.Is(err, resource.ErrPaginationCycle) || !errors.As(err, &accepted) || string(accepted.Body) != second || calls.Load() != 2 {
			t.Fatal(result, err, calls.Load())
		}
	})
	for _, tc := range []struct {
		name, body string
		success    bool
	}{{"HTTP Link and wrong-case links are ignored", `{"volumes":[{"attachments":[]}],"Volumes_Links":[{"rel":"next","href":"https://foreign.invalid/"}]}`, true}, {"wrong-case envelope", `{"Volumes":[]}`, false}, {"null volumes", `{"volumes":null}`, false}, {"nonobject raw volume row", `{"volumes":[null]}`, false}, {"empty EOF ignores malformed next data", `{"volumes":[],"volumes_links":false}`, true}, {"empty EOF ignores unsafe advertised next", `{"volumes":[],"volumes_links":[{"rel":"next","href":"https://foreign.invalid/"}]}`, true}, {"malformed native link data on nonempty page", `{"volumes":[{"attachments":[]}],"volumes_links":false}`, false}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Link", "<https://foreign.invalid/>; rel=next")
				testcloud.JSON(w, 200, tc.body)
			})
			result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{})
			if result == nil || len(result.Pages) != 1 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if tc.success {
				if err != nil || result.Volumes == nil || len(result.Volumes) != 0 {
					t.Fatal(result, err)
				}
			} else {
				var accepted *resource.ResponseError
				if err == nil || result.Volumes != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 {
					t.Fatal(result, err)
				}
			}
		})
	}
}

func TestGetVolumesAcceptedPageFailuresKeepCurrentProofAndLaterRejectionsNeverBorrowOldPage(t *testing.T) {
	for _, mode := range []string{"read", "close", "context", "source", "malformed JSON", "invalid UTF-8", "native rejection", "transport cause", "blocked cancellation"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("list page failed " + mode)
			var calls, retries atomic.Int32
			first, second := getVolumesContractPage(`[{"attachments":[{"server_id":"server"}]}]`, "?page=2"), getVolumesContractPage(`[]`, "")
			if mode == "malformed JSON" {
				second = `{"volumes":[`
			}
			if mode == "invalid UTF-8" {
				second = string([]byte{'{', '"', 'v', 'o', 'l', 'u', 'm', 'e', 's', '"', ':', '[', '{', '"', 'x', '"', ':', '"', 0xff, '"', '}', ']', '}'})
			}
			broken := &getVolumesContractReader{data: strings.NewReader(second)}
			if mode == "read" {
				broken.readError = cause
			}
			if mode == "close" {
				broken.closeError = cause
			}
			if mode == "context" {
				broken.onClose = func() { cancel(cause) }
			}
			if mode == "source" {
				broken.onClose = func() { client.Microversion = "3.99" }
			}
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(first)), "first"), nil
				}
				if calls.Load() > 2 {
					t.Error("page failure caused list replay or extra lookup", r.URL)
					return nil, errors.New("unexpected HTTP")
				}
				if mode == "native rejection" {
					return getVolumesContractResponse(r, 403, io.NopCloser(strings.NewReader(`{"error":"current rejection"}`)), "rejected"), nil
				}
				if mode == "transport cause" {
					return nil, cause
				}
				if mode == "blocked cancellation" {
					go cancel(cause)
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return getVolumesContractResponse(r, 200, broken, "current"), nil
			})
			result, err := blockstorage.GetVolumes(ctx, client, blockstorage.GetVolumesRequest{ServerID: "server"})
			var accepted *resource.ResponseError
			if result == nil || result.Volumes != nil || err == nil || calls.Load() != 2 || len(result.Pages) == 0 || string(result.Pages[0].Body) != first || result.Pages[0].Header.Get("X-Proof") != "first" {
				t.Fatal(result, err, calls.Load())
			}
			if mode == "native rejection" {
				var native gophercloud.ErrUnexpectedResponseCode
				if len(result.Pages) != 1 || errors.As(err, &accepted) || !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "rejected" {
					t.Fatal(result, err)
				}
			} else if mode == "transport cause" {
				if len(result.Pages) != 1 || errors.As(err, &accepted) || !errors.Is(err, cause) {
					t.Fatal(result, err)
				}
			} else if mode == "blocked cancellation" {
				if len(result.Pages) != 1 || errors.As(err, &accepted) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(result, err)
				}
			} else {
				if len(result.Pages) != 2 || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != second || accepted.Header.Get("X-Proof") != "current" || string(result.Pages[1].Body) != second || broken.closes.Load() != 1 || retries.Load() != 0 {
					t.Fatal(result, err, retries.Load())
				}
				if mode == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if mode == "malformed JSON" {
					var syntax *json.SyntaxError
					if !errors.As(err, &syntax) {
						t.Fatal("actual JSON parse cause lost", err)
					}
				} else if mode == "invalid UTF-8" {
					if !strings.Contains(err.Error(), "UTF-8") {
						t.Fatal("encoding failure cause lost", err)
					}
				} else if !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if mode == "context" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				accepted.Body[0] = '!'
				accepted.Header.Set("X-Proof", "caller")
				if string(result.Pages[1].Body) != second || result.Pages[1].Header.Get("X-Proof") != "current" {
					t.Fatal("page proof aliases accepted error")
				}
			}
			getVolumesContractOperation(t, err)
		})
	}
}

func TestGetVolumesNativeAuthBackoffRetryKeepsBodylessExactListAndLiveHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	client := getVolumesContractClient(cloud)
	var calls, reauths, backoffs, retries atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed"); return nil }
	cloud.Provider.RetryBackoffFunc = func(_ context.Context, code *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
		backoffs.Add(1)
		if code.Actual != 429 || count != 1 {
			t.Error(code, count)
		}
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, count uint) error {
		retries.Add(1)
		if !gophercloud.ResponseCodeIs(err, 503) {
			return err
		}
		if method != http.MethodGet || target != cloud.Server.URL+getVolumesContractPath || count != 2 || options.JSONBody != nil || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody || !reflect.DeepEqual(options.OkCodes, []int{200}) {
			t.Error(method, target, count, options)
		}
		if options.MoreHeaders == nil {
			options.MoreHeaders = map[string]string{}
		}
		options.MoreHeaders["X-Retry"] = "native"
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		step := calls.Add(1)
		token := "refreshed"
		if step == 1 {
			token = "live-entry"
		}
		getVolumesContractWire(t, r, token)
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		switch step {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 429, `{"error":"throttled"}`)
		case 3:
			testcloud.JSON(w, 503, `{"error":"retry"}`)
		default:
			if r.Header.Get("X-Retry") != "native" {
				t.Error("retry header lost")
			}
			testcloud.JSON(w, 200, getVolumesContractPage(`[{"attachments":[{"server_id":"server"}]}]`, ""))
		}
	})
	result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{ServerID: "server"}, func(*blockstorage.GetVolumesOpts) error {
		client.MoreHeaders["x-source"] = "late"
		cloud.Provider.SetToken("live-entry")
		return nil
	})
	if err != nil || result == nil || len(result.Pages) != 1 || len(result.Volumes) != 1 || calls.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || client.MoreHeaders["x-source"] != "late" {
		t.Fatal(result, err, calls.Load(), reauths.Load(), backoffs.Load(), retries.Load())
	}
	for _, mode := range []string{"body", "response target", "status expansion"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			var calls atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				switch mode {
				case "body":
					options.JSONBody = map[string]any{"bad": true}
				case "response target":
					options.JSONResponse = new(any)
				case "status expansion":
					options.OkCodes = append(options.OkCodes, 201)
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				} else {
					testcloud.JSON(w, 201, getVolumesContractPage(`[]`, ""))
				}
			})
			result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{})
			if result == nil || result.Volumes != nil || len(result.Pages) != 0 || err == nil {
				t.Fatal(result, err)
			}
			if mode == "status expansion" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 201 || !reflect.DeepEqual(native.Expected, []int{200}) || calls.Load() != 2 {
					t.Fatal(err, calls.Load())
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}

func TestGetVolumesOptionsSnapshotsReplacementOrderedCallbacksAndDelayedFields(t *testing.T) {
	fields := map[string]json.RawMessage{"id": json.RawMessage(`{"nested":[true,9007199254740993]}`)}
	pointer := &fields
	option := blockstorage.WithGetVolumesOptions(blockstorage.GetVolumesOpts{ServerFields: pointer})
	fields["id"][0] = '!'
	replacement := map[string]json.RawMessage{"id": json.RawMessage(`false`)}
	pointer = &replacement
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			prepared, err := blockstorage.PrepareGetVolumesOptions(context.Background(), option)
			if err != nil || prepared.ServerFields == nil || string((*prepared.ServerFields)["id"]) != `{"nested":[true,9007199254740993]}` {
				t.Error(prepared, err)
				return
			}
			(*prepared.ServerFields)["id"][0] = '!'
		}()
	}
	wait.Wait()
	prepared, err := blockstorage.PrepareGetVolumesOptions(context.Background(), option, blockstorage.WithGetVolumesOptions(blockstorage.GetVolumesOpts{}))
	if err != nil || prepared.ServerFields != nil {
		t.Fatal("complete replacement did not restore request.ServerID", prepared, err)
	}
	var retained *blockstorage.GetVolumesOpts
	calls := 0
	options := []blockstorage.GetVolumesOption{nil, nil}
	options[0] = func(v *blockstorage.GetVolumesOpts) error {
		calls++
		local := map[string]json.RawMessage{"id": json.RawMessage(`"original"`)}
		v.ServerFields = &local
		retained = v
		options[1] = nil
		return nil
	}
	options[1] = func(v *blockstorage.GetVolumesOpts) error {
		calls++
		(*retained.ServerFields)["id"][0] = '!'
		if string((*v.ServerFields)["id"]) != `"original"` {
			t.Error("retained handle aliases next callback")
		}
		return nil
	}
	prepared, err = blockstorage.PrepareGetVolumesOptions(context.Background(), options...)
	if err != nil || calls != 2 || prepared.ServerFields == nil || string((*prepared.ServerFields)["id"]) != `"original"` {
		t.Fatal("original ordered callbacks were rewritten by caller slice/handles", prepared, err, calls)
	}
	cause := errors.New("option canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	calls = 0
	_, err = blockstorage.PrepareGetVolumesOptions(ctx, func(*blockstorage.GetVolumesOpts) error { calls++; cancel(cause); return nil }, func(*blockstorage.GetVolumesOpts) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 1 {
		t.Fatal(err, calls)
	}
	prepared, err = blockstorage.PrepareGetVolumesOptions(context.Background(), blockstorage.WithGetVolumesServerFields(map[string]json.RawMessage{"id": json.RawMessage(`broken`)}))
	if err != nil || prepared.ServerFields == nil {
		t.Fatal("raw server fields validated before consumption", prepared, err)
	}
}

func TestGetVolumesPreflightAndCapturedSourceFactsBeforeOptionsOrSubsequentPages(t *testing.T) {
	for _, mode := range []string{"nil context", "nil Cinder", "nil provider", "wrong type", "foreign base", "token header", "nil option", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			ctx := context.Background()
			cause := errors.New("canceled before list")
			switch mode {
			case "nil context":
				ctx = nil
			case "nil Cinder":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type = "compute"
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/"
			case "token header":
				client.MoreHeaders["X-Auth-Token"] = "other"
			case "canceled":
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
			}
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("preflight reached list", r.URL)
				w.WriteHeader(500)
			})
			options := []blockstorage.GetVolumesOption{func(*blockstorage.GetVolumesOpts) error { callbacks.Add(1); return nil }}
			expectedCallbacks := int32(0)
			if mode == "nil option" {
				options = append(options, nil)
				expectedCallbacks = 1
			}
			result, err := blockstorage.GetVolumes(ctx, client, blockstorage.GetVolumesRequest{}, options...)
			want := resource.ErrInvalidOption
			if mode == "wrong type" {
				want = resource.ErrUnsupported
			}
			if mode == "canceled" {
				want = context.Canceled
			}
			if result != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != expectedCallbacks || mode == "canceled" && !errors.Is(err, cause) {
				t.Fatal(result, err, calls.Load(), callbacks.Load())
			}
		})
	}
	changes := []struct {
		name  string
		apply func(*gophercloud.ServiceClient)
	}{{"provider", func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }}, {"endpoint", func(c *gophercloud.ServiceClient) { c.Endpoint += "changed/" }}, {"base", func(c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" }}, {"type", func(c *gophercloud.ServiceClient) { c.Type = "compute" }}, {"version", func(c *gophercloud.ServiceClient) { c.Microversion = "3.99" }}}
	for _, change := range changes {
		t.Run(change.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			var calls atomic.Int32
			first := getVolumesContractPage(`[]`, "?page=2")
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				change.apply(client)
				return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(first)), "actual"), nil
			})
			result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{})
			var accepted *resource.ResponseError
			if result == nil || result.Volumes != nil || len(result.Pages) != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != first || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
	for _, change := range changes {
		t.Run("original option boundary "+change.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := getVolumesContractClient(cloud)
			original := *client
			var calls atomic.Int32
			callbacks := 0
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("mutated source reached HTTP", r.URL)
				w.WriteHeader(500)
			})
			result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{}, func(*blockstorage.GetVolumesOpts) error { callbacks++; change.apply(client); return nil }, func(*blockstorage.GetVolumesOpts) error { callbacks++; *client = original; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 1 || calls.Load() != 0 {
				t.Fatal("original source was captured after options or a later callback restored an observed mutation", result, err, callbacks, calls.Load())
			}
			getVolumesContractOperation(t, err)
		})
	}
	t.Run("empty original Type uses copied native volume header", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := getVolumesContractClient(cloud)
		client.Type = ""
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			getVolumesContractWire(t, r, "test-token")
			testcloud.JSON(w, 200, getVolumesContractPage(`[]`, ""))
		})
		result, err := blockstorage.GetVolumes(context.Background(), client, blockstorage.GetVolumesRequest{})
		if err != nil || result == nil || result.Volumes == nil || client.Type != "" {
			t.Fatal(result, err, client.Type)
		}
	})
}
