package transfers_test

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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type recordTransferCall struct {
	method, path, query, body string
	header                    http.Header
}

func recordTransferFixture(t *testing.T, selected, maximum, minimum string, reply func(*http.Request) (*http.Response, error)) (*transfers.API, *gophercloud.ServiceClient, *[]recordTransferCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]recordTransferCall{}
	cloud.Provider.HTTPClient.Transport = pythonTransferTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			raw = string(data)
		}
		*calls = append(*calls, recordTransferCall{method: req.Method, path: req.URL.Path, query: req.URL.RawQuery, body: raw, header: req.Header.Clone()})
		if req.URL.Path == "/cinder/v3/" {
			body, _ := json.Marshal(map[string]any{"version": map[string]any{"id": "v3.0", "status": "CURRENT", "max_version": maximum, "min_version": minimum}})
			return pythonTransferWire(200, string(body)), nil
		}
		return reply(req)
	})
	client := cloud.Client("volumev3", "/cinder/v3/project/")
	client.Microversion = selected
	return transfers.New(client), client, calls
}
func recordTransferField(t *testing.T, v *transfers.TransferRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s got %#v want %s", key, v, want)
	}
}
func recordTransferIDs(t *testing.T, api *transfers.API, options ...transfers.RecordListOption) ([]string, error) {
	t.Helper()
	var ids []string
	for v, err := range api.ListRecords(context.Background(), options...) {
		if err != nil {
			return ids, err
		}
		var id string
		if err := json.Unmarshal(v.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func TestPythonTransferDiscoveryAndVersionRouting(t *testing.T) {
	for _, item := range []struct{ selected, maximum, minimum, path, header string }{
		{"", "3.71", "3.0", "volume-transfers", "3.55"},
		{"3.54", "3.71", "3.0", "os-volume-transfer", "3.54"},
		{"3.55", "3.71", "3.0", "volume-transfers", "3.55"},
		{"3.71", "3.71", "3.0", "volume-transfers", "3.71"},
		{"", "3.54", "3.0", "os-volume-transfer", "3.54"},
		{"3.55", "", "3.0", "os-volume-transfer", "3.55"},
		{"", "", "3.0", "os-volume-transfer", ""},
		{"", "3.71", "", "os-volume-transfer", "3.55"},
	} {
		t.Run(item.selected+"_"+item.maximum+"_"+item.minimum, func(t *testing.T) {
			api, client, calls := recordTransferFixture(t, item.selected, item.maximum, item.minimum, func(*http.Request) (*http.Response, error) {
				return pythonTransferWire(202, `{"transfer":{"id":"t1"}}`), nil
			})
			v, err := api.CreateRecord(context.Background())
			if err != nil || v == nil || len(*calls) != 2 || (*calls)[0].path != "/cinder/v3/" || (*calls)[1].path != "/cinder/v3/project/"+item.path || (*calls)[1].body != `{"transfer":{}}` {
				t.Fatal(v, err, *calls)
			}
			if (*calls)[0].header.Get("OpenStack-API-Version") != "" || (*calls)[0].header.Get("X-OpenStack-Volume-API-Version") != "" || (*calls)[0].body != "" {
				t.Fatal("discovery must be versionless and bodyless", *calls)
			}
			if got := (*calls)[1].header.Get("X-OpenStack-Volume-API-Version"); got != item.header {
				t.Fatal("version", got, item.header)
			}
			if client.Microversion != item.selected || len(v.Discovery) != 1 || v.Discovery[0].StatusCode != 200 || v.StatusCode != 202 {
				t.Fatal("selection mutated shared policy or merged discovery/operation", v, client)
			}
		})
	}
}

func TestPythonTransferCreateRawSnapshotsExtensionsAndResponseViews(t *testing.T) {
	api, _, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(*http.Request) (*http.Response, error) {
		return pythonTransferWire(202, `{"transfer":{"id":"t1","created_at":"2026-10-11T01:02:03Z","auth_key":9007199254740993,"extension":[null,false]}}`), nil
	})
	fields := map[string]json.RawMessage{"volume_id": json.RawMessage(`"v1"`), "name": json.RawMessage(`""`), "no_snapshots": json.RawMessage(`false`), "unknown": json.RawMessage(`{bad`)}
	option := transfers.WithRecordCreateOpts(transfers.RecordCreateOpts{Fields: fields})
	fields["volume_id"][1] = 'x'
	extension := map[string]any{"list": []any{nil, false, 0}}
	extensionOption := transfers.WithRecordCreateExtension("future", extension)
	extension["list"] = "changed"
	v, err := api.CreateRecord(context.Background(), option, extensionOption, transfers.WithRecordCreateField("unused", make(chan int)))
	if err != nil || len(*calls) != 2 || (*calls)[1].body != `{"transfer":{"future":{"list":[null,false,0]},"name":"","no_snapshots":false,"volume_id":"v1"}}` {
		t.Fatal(v, err, *calls)
	}
	recordTransferField(t, v, "volume_id", `"v1"`)
	recordTransferField(t, v, "name", `""`)
	recordTransferField(t, v, "no_snapshots", `false`)
	recordTransferField(t, v, "auth_key", `9007199254740993`)
	recordTransferField(t, v, "created_at", `"2026-10-11T01:02:03Z"`)
	recordTransferField(t, v, "location", `null`)
	if len(v.Resource.Body) != 8 || string(v.Wire.Body["extension"]) != `[null,false]` {
		t.Fatal(v)
	}
	v.Resource.Body["auth_key"][0] = '1'
	if string(v.Wire.Body["auth_key"]) != `9007199254740993` || !strings.Contains(string(v.Envelope), "9007199254740993") {
		t.Fatal("record aliases wire")
	}
	v.Header.Set("X-One", "yes")
	if v.Wire.Header.Get("X-One") != "" || v.Resource.Header.Get("X-One") != "" {
		t.Fatal("header aliases wire")
	}
	*calls = nil
	if _, err := api.CreateRecord(context.Background(), transfers.WithRecordCreateField("volume_id", nil), transfers.WithRecordCreateField("name", 0), transfers.WithRecordCreateField("no_snapshots", []any{})); err != nil {
		t.Fatal(err)
	}
	if (*calls)[1].body != `{"transfer":{"name":0,"no_snapshots":[],"volume_id":null}}` {
		t.Fatal(*calls)
	}
	legacy, _, legacyCalls := recordTransferFixture(t, "3.0", "3.71", "3.0", func(*http.Request) (*http.Response, error) { return pythonTransferWire(202, `{}`), nil })
	if _, err := legacy.CreateRecord(context.Background(), transfers.WithRecordCreateNoSnapshots(true)); err != nil {
		t.Fatal(err)
	}
	if (*legacyCalls)[1].body != `{"transfer":{"no_snapshots":true}}` {
		t.Fatal("proxy installs attrs before cleanup params", *legacyCalls)
	}
	*legacyCalls = nil
	v, err = legacy.CreateRecord(context.Background(), transfers.WithRecordCreateNoSnapshots(true), transfers.WithRecordCreateSuppressLegacySnapshots(true))
	if err != nil || (*legacyCalls)[1].body != `{"transfer":{}}` {
		t.Fatal(v, err, *legacyCalls)
	}
	recordTransferField(t, v, "no_snapshots", `true`)
}

func TestPythonTransferListDefaultsSummaryQueriesAndRawFilters(t *testing.T) {
	for _, details := range []bool{true, false} {
		for _, selected := range []string{"3.0", "3.55"} {
			t.Run(fmt.Sprint(details)+selected, func(t *testing.T) {
				api, _, calls := recordTransferFixture(t, selected, "3.71", "3.0", func(*http.Request) (*http.Response, error) {
					return pythonTransferWire(200, `{"transfers":[{"id":"a","name":"ci","links":{"owner":"p1","count":9007199254740993}},{"id":"b","name":"other"}]}`), nil
				})
				ids, err := recordTransferIDs(t, api, transfers.WithRecordListDetails(details), transfers.WithRecordListAllProjects(true), transfers.WithRecordListQuery("name", "ci"), transfers.WithRecordListQuery("links", map[string]any{"owner": "p1"}), transfers.WithRecordListQuery("unknown", make(chan int)), transfers.WithRecordListQuery("all_tenants", true))
				path := "/cinder/v3/project/volume-transfers"
				if selected == "3.0" {
					path = "/cinder/v3/project/os-volume-transfer"
				}
				if details {
					path += "/detail"
				}
				if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(*calls) != 2 || (*calls)[1].path != path || (*calls)[1].query != "" || (*calls)[1].header.Get("Accept") != "application/json" {
					t.Fatal(ids, err, *calls)
				}
			})
		}
	}
	api, _, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(*http.Request) (*http.Response, error) { return pythonTransferWire(200, `{"transfers":[]}`), nil })
	ids, err := recordTransferIDs(t, api, transfers.WithRecordListServerQuery("all_tenants", "True"), transfers.WithRecordListLimit(0), transfers.WithRecordListMarker("before"))
	if err != nil || len(ids) != 0 || (*calls)[1].path != "/cinder/v3/project/volume-transfers/detail" || (*calls)[1].query != "all_tenants=True&limit=0&marker=before" {
		t.Fatal(ids, err, *calls)
	}
}

func TestPythonTransferListLazinessCapsAndContinuation(t *testing.T) {
	t.Run("raw cap before filter", func(t *testing.T) {
		api, _, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(*http.Request) (*http.Response, error) {
			return pythonTransferWire(200, `{"transfers":[{"id":"a","name":"other"},{"id":"b","name":"ci"}],"transfers_links":[{"rel":"next","href":"?marker=b"}]}`), nil
		})
		seq := api.ListRecords(context.Background(), transfers.WithRecordListQuery("name", "ci"), transfers.WithRecordListMaxItems(1))
		if len(*calls) != 0 {
			t.Fatal("constructed sequence performs HTTP", *calls)
		}
		var rows int
		for _, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			rows++
		}
		if rows != 0 || len(*calls) != 2 || (*calls)[1].query != "limit=1" {
			t.Fatal(rows, *calls)
		}
	})
	t.Run("short page marker", func(t *testing.T) {
		api, _, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(req *http.Request) (*http.Response, error) {
			if req.URL.Query().Get("marker") == "a" {
				return pythonTransferWire(200, `{"transfers":[]}`), nil
			}
			return pythonTransferWire(200, `{"transfers":[{"id":"a"}]}`), nil
		})
		ids, err := recordTransferIDs(t, api, transfers.WithRecordListLimit(2), transfers.WithRecordListDetails(false))
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(*calls) != 3 || (*calls)[2].query != "limit=2&marker=a" {
			t.Fatal(ids, err, *calls)
		}
	})
	t.Run("break avoids continuation", func(t *testing.T) {
		api, _, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(*http.Request) (*http.Response, error) {
			return pythonTransferWire(200, `{"transfers":[{"id":"a"}],"transfers_links":[{"rel":"next","href":"?marker=a"}]}`), nil
		})
		for _, err := range api.ListRecords(context.Background()) {
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if len(*calls) != 2 {
			t.Fatal(*calls)
		}
	})
	t.Run("HTTP links and late page proof", func(t *testing.T) {
		api, _, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(req *http.Request) (*http.Response, error) {
			if req.URL.Query().Get("marker") == "a" {
				return pythonTransferWire(200, `{"transfers":null}`), nil
			}
			r := pythonTransferWire(200, `{"transfers":[{"id":"a"}],"next":false}`)
			r.Header.Set("Link", `<?marker=a>; rel="next"`)
			return r, nil
		})
		ids, err := recordTransferIDs(t, api)
		var proof *resource.ResponseError
		if !reflect.DeepEqual(ids, []string{"a"}) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != `{"transfers":null}` || len(*calls) != 3 {
			t.Fatal(ids, err, proof, *calls)
		}
	})
}

func TestPythonTransferFindGetFallbackSummaryAndMissingDefaults(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			api, _, calls := recordTransferFixture(t, "3.0", "3.71", "3.0", func(req *http.Request) (*http.Response, error) {
				if strings.HasSuffix(req.URL.Path, "/named") {
					return pythonTransferWire(status, `{"error":"missing"}`), nil
				}
				return pythonTransferWire(200, `{"transfers":[{"id":"other","name":"no"},{"id":"target","name":"named"}]}`), nil
			})
			v, err := api.FindRecord(context.Background(), "named")
			if err != nil || len(*calls) != 3 || (*calls)[1].path != "/cinder/v3/project/os-volume-transfer/named" || (*calls)[2].path != "/cinder/v3/project/volume-transfers" || (*calls)[2].query != "" {
				t.Fatal(v, err, *calls)
			}
			recordTransferField(t, v, "id", `"target"`)
		})
	}
	t.Run("ID from list and ambiguity", func(t *testing.T) {
		ambiguous := false
		api, _, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(req *http.Request) (*http.Response, error) {
			if strings.HasSuffix(req.URL.Path, "/target") {
				return pythonTransferWire(404, `{}`), nil
			}
			if ambiguous {
				return pythonTransferWire(200, `{"transfers":[{"id":"target","name":"first"},{"id":"other","name":"target"}]}`), nil
			}
			return pythonTransferWire(200, `{"transfers":[{"id":"target","name":"different"}]}`), nil
		})
		v, err := api.FindRecord(context.Background(), "target")
		if err != nil {
			t.Fatal(err)
		}
		recordTransferField(t, v, "id", `"target"`)
		ambiguous = true
		*calls = nil
		v, err = api.FindRecord(context.Background(), "target")
		if v != nil || !errors.Is(err, resource.ErrAmbiguous) || len(*calls) != 3 {
			t.Fatal(v, err, *calls)
		}
	})
	t.Run("missing default and strict", func(t *testing.T) {
		api, _, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(req *http.Request) (*http.Response, error) {
			if strings.HasSuffix(req.URL.Path, "/absent") {
				return pythonTransferWire(404, `{}`), nil
			}
			return pythonTransferWire(200, `{"transfers":[]}`), nil
		})
		v, err := api.FindRecord(context.Background(), "absent")
		if v != nil || err != nil || len(*calls) != 3 {
			t.Fatal(v, err, *calls)
		}
		v, err = api.FindRecord(context.Background(), "absent", transfers.WithRecordFindIgnoreMissing(false))
		if v != nil || !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(v, err)
		}
	})
}

type recordTransferFaultBody struct {
	reader            io.Reader
	readErr, closeErr error
}

func (b *recordTransferFaultBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if err == io.EOF && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *recordTransferFaultBody) Close() error { return b.closeErr }
func TestPythonTransferFindDeleteRejectedReadCloseAndCancellation(t *testing.T) {
	for _, failure := range []string{"read", "close", "cancel", "source"} {
		for _, operation := range []string{"find", "delete"} {
			t.Run(failure+operation, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("physical " + failure)
				var client *gophercloud.ServiceClient
				api, selected, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(*http.Request) (*http.Response, error) {
					r := pythonTransferWire(404, `{"error":"missing"}`)
					b := &recordTransferFaultBody{reader: strings.NewReader(`{"error":"missing"}`)}
					if failure == "read" {
						b.readErr = cause
					}
					if failure == "close" {
						b.closeErr = cause
					}
					if failure == "cancel" {
						cancel(cause)
					}
					if failure == "source" {
						client.Microversion = "3.54"
					}
					r.Body = b
					return r, nil
				})
				client = selected
				var err error
				if operation == "find" {
					_, err = api.FindRecord(ctx, "named")
				} else {
					err = api.DeleteRecord(ctx, "id")
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if err == nil || !errors.As(err, &native) || native.Actual != 404 || len(*calls) != 2 {
					t.Fatal(err, *calls)
				}
				if failure == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, cause) {
					t.Fatal("physical cause discarded", err)
				}
			})
		}
	}
}

func TestPythonTransferModernGetAcceptDeleteAndCleanRetries(t *testing.T) {
	status := 200
	api, client, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			return pythonTransferWire(status, `{"error":"missing"}`), nil
		}
		if strings.HasSuffix(req.URL.Path, "/accept") {
			return pythonTransferWire(202, `{"transfer":{"name":"accepted","links":[],"future":true}}`), nil
		}
		return pythonTransferWire(200, `{"transfer":{"created_at":"2026-10-11T01:02:03+09:00","future":true}}`), nil
	})
	v, err := api.GetRecord(context.Background(), "id")
	if err != nil || (*calls)[1].path != "/cinder/v3/project/volume-transfers/id" {
		t.Fatal(v, err, *calls)
	}
	recordTransferField(t, v, "id", `"id"`)
	recordTransferField(t, v, "created_at", `"2026-10-11T01:02:03+09:00"`)
	*calls = nil
	v, err = api.AcceptRecord(context.Background(), "id", transfers.WithRecordAcceptAuthKey("secret"))
	if err != nil || (*calls)[1].path != "/cinder/v3/project/volume-transfers/id/accept" || (*calls)[1].body != `{"accept":{"auth_key":"secret"}}` {
		t.Fatal(v, err, *calls)
	}
	recordTransferField(t, v, "id", `null`)
	status = 404
	client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		return original
	}
	*calls = nil
	if err := api.DeleteRecord(context.Background(), "id"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 2 || (*calls)[1].path != "/cinder/v3/project/volume-transfers/id" {
		t.Fatal(*calls)
	}
	if err := api.DeleteRecord(context.Background(), "id", transfers.WithRecordDeleteIgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
}

func TestPythonTransferInvalidOptionsAndStickySelection(t *testing.T) {
	api, client, calls := recordTransferFixture(t, "3.55", "3.71", "3.0", func(*http.Request) (*http.Response, error) { return pythonTransferWire(202, `{}`), nil })
	for _, invoke := range []func() error{
		func() error { _, err := api.CreateRecord(nil); return err },
		func() error { _, err := api.CreateRecord(context.Background(), nil); return err },
		func() error {
			_, err := api.CreateRecord(context.Background(), transfers.WithRecordCreateOpts(transfers.RecordCreateOpts{Fields: map[string]json.RawMessage{"name": json.RawMessage(`{bad`)}}))
			return err
		},
		func() error {
			_, err := api.CreateRecord(context.Background(), transfers.WithRecordCreateExtension("name", true))
			return err
		},
		func() error { _, err := api.GetRecord(context.Background(), ".."); return err },
		func() error { _, err := api.AcceptRecord(context.Background(), "id", nil); return err },
		func() error { _, err := recordTransferIDs(t, api, transfers.WithRecordListMaxItems(-1)); return err },
	} {
		if err := invoke(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(*calls) != 0 {
		t.Fatal("invalid preflight performs HTTP", *calls)
	}
	_, err := api.CreateRecord(context.Background(), func(*transfers.RecordCreateOpts) error { client.Microversion = "3.54"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
}
