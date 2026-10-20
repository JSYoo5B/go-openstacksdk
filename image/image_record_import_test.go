package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func imageRecordImportFetched(t *testing.T, service *Service, handler *taskCoreTransport) *ImageRecord {
	t.Helper()
	return imageRecordUpdateFetched(t, service, `{"id":"fixed","status":"queued","container_format":"bare","disk_format":"qcow2","name":"before","size":"04","protected":"false","vendor":{"precise":900719925474099312345}}`, handler)
}
func imageRecordImportPayload(t *testing.T, request *http.Request, want string) map[string]json.RawMessage {
	t.Helper()
	got := taskCorePayload(t, request)
	decode := func(raw []byte) any {
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	actual, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	th.CheckDeepEquals(t, decode([]byte(want)), decode(actual))
	return got
}
func imageRecordImportStoreRecords(t *testing.T, client *gophercloud.ServiceClient, handler *taskCoreTransport, rows string, count int) []*serviceinfo.StoreRecord {
	t.Helper()
	*handler = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/info/stores" {
			t.Fatal(req.Method, req.URL)
		}
		return taskCoreJSON(req, 203, `{"stores":`+rows+`}`), nil
	}
	result := make([]*serviceinfo.StoreRecord, 0, count)
	// Stop after the requested raw rows, before malformed/null IDs can be
	// consumed as continuation markers. These are real SDK-owned resources.
	for value, err := range serviceinfo.New(client).ListStoreRecords(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, value)
		if len(result) == count {
			break
		}
	}
	if len(result) != count {
		t.Fatal("missing Store records", result)
	}
	return result
}

func TestImageRecordImportOpaqueAcceptedResponsesKeepPreparedRecordAndIndependentReceipt(t *testing.T) {
	for _, test := range []struct {
		code int
		raw  string
	}{{200, "opaque\xff"}, {201, "not JSON"}, {202, `{"id":"ACK decoy","status":"active","size":null}`}, {204, ""}, {299, `null`}, {300, `false`}, {304, `[]`}, {399, `"opaque"`}} {
		t.Run(fmt.Sprint(test.code), func(t *testing.T) {
			calls, locations := 0, 0
			cloud := "initial"
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
			seed := imageRecordImportFetched(t, service, &handler)
			id := "name-like /한:%?\\b"
			rawID, _ := json.Marshal(id)
			seed.bodyState.current["id"], seed.bodyState.original["id"] = rawID, append(json.RawMessage(nil), rawID...)
			seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
			seed.Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
			seed.Resource.Body["size"] = json.RawMessage(`"public invalid descriptor"`)
			reader := &imageUploadCoreReader{reader: strings.NewReader("private borrowed data")}
			seed.data = reader
			created, updated := "passive created", "passive updated"
			seed.Resource.CreatedAt, seed.Resource.UpdatedAt = &created, &updated
			seed.Resource.Links = []resource.Link{{Href: "https://foreign.test/passive", Rel: "self"}}
			before := cloneImageRecord(seed)
			cloud = "import location"
			body := &taskCoreBody{reader: strings.NewReader(test.raw)}
			var actualHeader http.Header
			handler = func(req *http.Request) (*http.Response, error) {
				if calls != 2 || req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/import" || req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") != "first-token" {
					t.Fatal("import performed lookup or moved identity", calls, req.Method, req.URL, req.Header)
				}
				imageRecordImportPayload(t, req, `{"method":{"name":"glance-direct"}}`)
				response := taskCoreHTTP(req, test.code, body)
				response.Header.Set("Location", "https://foreign.test/images/other")
				response.Header.Set("OpenStack-image-import-methods", "must remain passive")
				actualHeader = response.Header
				return response, nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed})
			if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 2 || locations != 2 || body.closes != 1 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 || reader.closes.Load() != 0 {
				t.Fatal(got, err, calls, locations, body.closes, reader)
			}
			ack, record := got.Acknowledgement, got.Record
			if ack.ImageID != id || ack.StatusCode != test.code || string(ack.Body) != test.raw || ack.Header.Get("X-Task-Proof") != "actual" || len(record.Resource.Body) != 65 || len(record.ImportMethods) != 0 || record.StatusCode != 203 || record.Resource.StatusCode != 203 || !reflect.DeepEqual(record.Header, seed.Header) || !reflect.DeepEqual(record.Wire, seed.Wire) || string(record.Envelope) != string(seed.Envelope) || !reflect.DeepEqual(record.bodyState, seed.bodyState) || record.data != reader || !reflect.DeepEqual(before, seed) {
				t.Fatal("opaque import changed body/status/fetch/data or input", got, seed)
			}
			for key, want := range map[string]string{"id": string(rawID), "status": `"queued"`, "size": "4", "is_protected": "true", "container_format": `"bare"`, "disk_format": `"qcow2"`} {
				th.AssertEquals(t, want, string(record.Resource.Body[key]))
			}
			if record.Resource.CreatedAt == seed.Resource.CreatedAt || record.Resource.UpdatedAt == seed.Resource.UpdatedAt || *record.Resource.CreatedAt != created || *record.Resource.UpdatedAt != updated || !reflect.DeepEqual(record.Resource.Links, seed.Resource.Links) {
				t.Fatal("passive metadata not cloned", record.Resource)
			}
			var location resource.CloudLocation
			if err := json.Unmarshal(record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "import location" {
				t.Fatal(location, err)
			}
			ack.Header.Set("X-Task-Proof", "mutated")
			if len(ack.Body) != 0 {
				ack.Body[0] = '!'
			}
			record.Header.Set("X-Task-Proof", "record")
			record.Resource.Header.Set("X-Task-Proof", "view")
			record.Wire.Body["id"][1] = '!'
			record.Envelope[0] = '!'
			record.bodyState.current["name"][1] = '!'
			record.bodyState.original["name"][1] = '!'
			*record.Resource.CreatedAt = "changed"
			record.Resource.Links[0].Href = "changed"
			if !reflect.DeepEqual(before, seed) || actualHeader.Get("X-Task-Proof") != "actual" {
				t.Fatal("result aliases source/receipt", seed)
			}
		})
	}
}

func TestImageRecordImportFormatsUseRawPythonTruthinessWithoutStatusOrAllowList(t *testing.T) {
	for _, field := range []string{"container_format", "disk_format"} {
		for _, raw := range []string{"", `null`, `false`, `0`, `""`, `[]`, `{}`} {
			t.Run(field+"="+raw, func(t *testing.T) {
				calls := 0
				var handler taskCoreTransport
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
				service := New(client)
				seed := imageRecordImportFetched(t, service, &handler)
				if raw == "" {
					delete(seed.bodyState.current, field)
				} else {
					seed.bodyState.current[field] = json.RawMessage(raw)
				}
				seed.Resource.Body[field] = json.RawMessage(`"public truthy decoy"`)
				got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed})
				if got != nil || err == nil || calls != 1 {
					t.Fatal(got, err, calls)
				}
			})
		}
	}
	for _, raw := range []string{`true`, `1`, `" "`, `[null]`, `{"future":true}`} {
		t.Run("truthy formats="+raw, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			seed.bodyState.current["container_format"], seed.bodyState.current["disk_format"] = json.RawMessage(raw), json.RawMessage(raw)
			seed.bodyState.current["status"] = json.RawMessage(`{"untyped":"status"}`)
			handler = func(req *http.Request) (*http.Response, error) {
				imageRecordImportPayload(t, req, `{"method":{"name":"glance-direct"}}`)
				return taskCoreJSON(req, 202, "accepted"), nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed})
			if got == nil || got.Record == nil || err != nil || calls != 2 || string(got.Record.Resource.Body["container_format"]) != raw || string(got.Record.Resource.Body["disk_format"]) != raw {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordImportMethodPreservesOmittedNullUnknownAndAdminBranchValues(t *testing.T) {
	for _, raw := range []string{"", `null`, `""`, `"future-method"`, `false`, `17`, `[null]`, `{"future":true}`, `"web-download"`, `"glance-download"`, `"copy-image"`} {
		t.Run("method="+raw, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			want := raw
			var options []ImageRecordImportOption
			if raw == "" {
				want = `"glance-direct"`
			} else {
				options = []ImageRecordImportOption{WithImageRecordImportMethod(json.RawMessage(raw))}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				imageRecordImportPayload(t, req, `{"method":{"name":`+want+`}}`)
				return taskCoreJSON(req, 202, "async"), nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, options...)
			if got == nil || got.Record == nil || err != nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("helper nil explicitly sends null", func(t *testing.T) {
		calls := 0
		var handler taskCoreTransport
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
		service := New(client)
		seed := imageRecordImportFetched(t, service, &handler)
		handler = func(req *http.Request) (*http.Response, error) {
			imageRecordImportPayload(t, req, `{"method":{"name":null}}`)
			return taskCoreJSON(req, 202, ""), nil
		}
		got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, WithImageRecordImportMethod(nil))
		if got == nil || err != nil || calls != 2 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestImageRecordImportURIUsesTruthyDomainAndExactWebDownloadGuard(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		t.Run("falsey URI="+raw, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				imageRecordImportPayload(t, req, `{"method":{"name":"glance-direct"}}`)
				return taskCoreJSON(req, 202, ""), nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, WithImageRecordImportURI(json.RawMessage(raw)))
			if got == nil || err != nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, raw := range []string{`"not-a-URL"`, `true`, `1`, `[null]`, `{"future":true}`} {
		t.Run("truthy web URI="+raw, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				imageRecordImportPayload(t, req, `{"method":{"name":"web-download","uri":`+raw+`}}`)
				return taskCoreJSON(req, 202, ""), nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, WithImageRecordImportMethod("web-download"), WithImageRecordImportURI(json.RawMessage(raw)))
			if got == nil || err != nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, method := range []any{"Web-download", "", nil, map[string]any{"name": "web-download"}} {
		t.Run(fmt.Sprintf("wrong method=%v", method), func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, WithImageRecordImportMethod(method), WithImageRecordImportURI("truthy"))
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordImportRemoteTupleRequiresAllThreeTruthyValuesUnderAnyMethod(t *testing.T) {
	for _, test := range []struct{ name, method, region, id, iface, want string }{
		{"complete other method", `"future"`, `"RegionOne"`, `"remote image"`, `"internal"`, `{"method":{"name":"future","glance_region":"RegionOne","glance_image_id":"remote image","glance_service_interface":"internal"}}`},
		{"complete untyped tuple", `null`, `true`, `17`, `{"interface":[null]}`, `{"method":{"name":null,"glance_region":true,"glance_image_id":17,"glance_service_interface":{"interface":[null]}}}`},
		{"complete copy admin", `"copy-image"`, `[null]`, `{"id":true}`, `" "`, `{"method":{"name":"copy-image","glance_region":[null],"glance_image_id":{"id":true},"glance_service_interface":" "}}`},
		{"interface missing", `"glance-download"`, `"region"`, `"id"`, "", `{"method":{"name":"glance-download"}}`},
		{"region falsey", `"glance-download"`, `[]`, `"id"`, `"public"`, `{"method":{"name":"glance-download"}}`},
		{"image falsey", `"glance-direct"`, `"region"`, `null`, `"public"`, `{"method":{"name":"glance-direct"}}`},
		{"interface falsey", `"web-download"`, `"region"`, `"id"`, `false`, `{"method":{"name":"web-download"}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			options := []ImageRecordImportOption{WithImageRecordImportMethod(json.RawMessage(test.method)), WithImageRecordImportRemoteRegion(json.RawMessage(test.region)), WithImageRecordImportRemoteImageID(json.RawMessage(test.id))}
			if test.iface != "" {
				options = append(options, WithImageRecordImportRemoteServiceInterface(json.RawMessage(test.iface)))
			}
			handler = func(req *http.Request) (*http.Response, error) {
				imageRecordImportPayload(t, req, test.want)
				return taskCoreJSON(req, 202, ""), nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, options...)
			if got == nil || err != nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordImportSingularStoreEmitsBodyAndCompatibilityHeaderAndRemovesConfiguredDecoys(t *testing.T) {
	for _, mode := range []string{"literal", "empty literal", "SDK private Record", "no typed selection", "plural only"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			var selector ImageRecordImportStore
			if mode == "SDK private Record" {
				records := imageRecordImportStoreRecords(t, client, &handler, `[{"id":"canonical store","name":"name decoy"}]`, 1)
				records[0].Resource.Body["id"] = json.RawMessage(`"view decoy"`)
				records[0].Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
				records[0].Envelope[0] = '!'
				selector.Record = records[0]
			} else if mode == "literal" {
				selector.ID = "literal store"
			}
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			initial := calls
			client.MoreHeaders = map[string]string{"x-image-meta-store": "configured decoy"}
			wantBody := `{"method":{"name":"glance-direct"}}`
			wantHeader := ""
			var options []ImageRecordImportOption
			switch mode {
			case "literal":
				wantHeader = "literal store"
				wantBody = `{"method":{"name":"glance-direct"},"stores":["literal store"]}`
				options = []ImageRecordImportOption{WithImageRecordImportStore(selector)}
			case "empty literal":
				wantBody = `{"method":{"name":"glance-direct"},"stores":[""]}`
				options = []ImageRecordImportOption{WithImageRecordImportStore(selector)}
			case "SDK private Record":
				wantHeader = "canonical store"
				wantBody = `{"method":{"name":"glance-direct"},"stores":["canonical store"]}`
				options = []ImageRecordImportOption{WithImageRecordImportStore(selector)}
			case "plural only":
				wantBody = `{"method":{"name":"glance-direct"},"stores":["plural"]}`
				options = []ImageRecordImportOption{WithImageRecordImportStores(ImageRecordImportStore{ID: "plural"})}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				imageRecordImportPayload(t, req, wantBody)
				if req.Header.Get("X-Image-Meta-Store") != wantHeader {
					t.Fatal("configured header retargeted import", req.Header)
				}
				if mode == "empty literal" {
					if _, present := req.Header[http.CanonicalHeaderKey("X-Image-Meta-Store")]; !present {
						t.Fatal("explicit empty store header omitted", req.Header)
					}
				}
				if mode == "no typed selection" || mode == "plural only" {
					if _, present := req.Header[http.CanonicalHeaderKey("X-Image-Meta-Store")]; present {
						t.Fatal("plural/no selection acquired compatibility header", req.Header)
					}
				}
				return taskCoreJSON(req, 202, ""), nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, options...)
			if got == nil || err != nil || calls != initial+1 {
				t.Fatal(got, err, calls, initial)
			}
		})
	}
}

func TestImageRecordImportPluralStoreConstructorsPreserveRawIDsOrderDuplicatesAndMissingNull(t *testing.T) {
	calls := 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	records := imageRecordImportStoreRecords(t, client, &handler, `[{"id":"SDK"},{"id":null},{"name":"not alternate ID"},{"id":900719925474099312345}]`, 4)
	records[0].Resource.Body["id"] = json.RawMessage(`"public decoy"`)
	records[1].Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
	service := New(client)
	seed := imageRecordImportFetched(t, service, &handler)
	selectors := []ImageRecordImportStore{
		{ID: "one"}, {Record: records[0]}, {ID: "one"}, {}, {Record: records[1]}, {Record: records[2]}, {Record: records[3]},
		{RawID: json.RawMessage(`false`)}, {RawID: json.RawMessage(`[null]`)}, {RawID: json.RawMessage(`{"id":"literal object"}`)},
		{Attributes: map[string]any{}}, {Attributes: map[string]any{"name": "not ID"}}, {Attributes: map[string]any{"id": map[string]any{"dict": true}, "properties": "unrelated"}},
	}
	option := WithImageRecordImportStores(selectors...)
	selectors[0].ID = "later"
	selectors[7].RawID[0] = 't'
	selectors[12].Attributes["id"] = "later attributes"
	handler = func(req *http.Request) (*http.Response, error) {
		imageRecordImportPayload(t, req, `{"method":{"name":"glance-direct"},"stores":["one","SDK","one","",null,null,900719925474099312345,false,[null],{"id":"literal object"},null,null,{"dict":true}]}`)
		if _, present := req.Header[http.CanonicalHeaderKey("X-Image-Meta-Store")]; present {
			t.Fatal("plural emitted compatibility header", req.Header)
		}
		return taskCoreJSON(req, 202, ""), nil
	}
	got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, option)
	if got == nil || err != nil || calls != 3 {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordImportStoreSelectorsRejectConflictsUnsafeHeadersAndConstructorControls(t *testing.T) {
	calls := 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	records := imageRecordImportStoreRecords(t, client, &handler, `[{"id":null},{"id":17},{"id":"safe"}]`, 3)
	service := New(client)
	seed := imageRecordImportFetched(t, service, &handler)
	tests := []struct {
		name     string
		selector ImageRecordImportStore
		plural   bool
	}{
		{"singular SDK null", ImageRecordImportStore{Record: records[0]}, false},
		{"singular SDK number", ImageRecordImportStore{Record: records[1]}, false},
		{"singular raw bool", ImageRecordImportStore{RawID: json.RawMessage(`true`)}, false},
		{"singular attributes missing id", ImageRecordImportStore{Attributes: map[string]any{"name": "name"}}, false},
		{"singular header control", ImageRecordImportStore{ID: "bad\nheader"}, false},
		{"record and literal", ImageRecordImportStore{ID: "literal", Record: records[2]}, true},
		{"raw and attributes", ImageRecordImportStore{RawID: json.RawMessage(`null`), Attributes: map[string]any{}}, true},
		{"handcrafted StoreRecord", ImageRecordImportStore{Record: &serviceinfo.StoreRecord{Resource: records[2].Resource.Clone()}}, true},
		{"invalid JSON", ImageRecordImportStore{RawID: json.RawMessage(`{`)}, true},
		{"invalid UTF8", ImageRecordImportStore{RawID: json.RawMessage{'"', 0xff, '"'}}, true},
	}
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		tests = append(tests, struct {
			name     string
			selector ImageRecordImportStore
			plural   bool
		}{"constructor control=" + key, ImageRecordImportStore{Attributes: map[string]any{"id": "safe", key: nil}}, true})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			option := WithImageRecordImportStore(test.selector)
			if test.plural {
				option = WithImageRecordImportStores(test.selector)
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, option)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordImportFlagsPreserveFalseyPresenceAndRejectTruthyStoreConflicts(t *testing.T) {
	for _, field := range []string{"all_stores", "all_stores_must_succeed"} {
		for _, raw := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`, `"truthy"`} {
			t.Run(field+"="+raw, func(t *testing.T) {
				calls := 0
				var handler taskCoreTransport
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
				service := New(client)
				seed := imageRecordImportFetched(t, service, &handler)
				option := WithImageRecordImportAllStores(json.RawMessage(raw))
				if field == "all_stores_must_succeed" {
					option = WithImageRecordImportAllStoresMustSucceed(json.RawMessage(raw))
				}
				want := `{"method":{"name":"glance-direct"}}`
				if raw != `null` {
					want = `{"method":{"name":"glance-direct"},"` + field + `":` + raw + `}`
				}
				handler = func(req *http.Request) (*http.Response, error) {
					imageRecordImportPayload(t, req, want)
					return taskCoreJSON(req, 202, ""), nil
				}
				got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, option)
				if got == nil || err != nil || calls != 2 {
					t.Fatal(got, err, calls)
				}
			})
		}
	}
	for _, mode := range []string{"truthy singular", "truthy plural", "singular plus plural", "false singular allowed", "empty plural omitted"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			var options []ImageRecordImportOption
			switch mode {
			case "truthy singular":
				options = []ImageRecordImportOption{WithImageRecordImportAllStores([]any{nil}), WithImageRecordImportStore(ImageRecordImportStore{})}
			case "truthy plural":
				options = []ImageRecordImportOption{WithImageRecordImportAllStores(map[string]any{"truthy": true}), WithImageRecordImportStores(ImageRecordImportStore{ID: "store"})}
			case "singular plus plural":
				options = []ImageRecordImportOption{WithImageRecordImportStore(ImageRecordImportStore{}), WithImageRecordImportStores(ImageRecordImportStore{ID: "store"})}
			case "false singular allowed":
				options = []ImageRecordImportOption{WithImageRecordImportAllStores(false), WithImageRecordImportAllStoresMustSucceed(false), WithImageRecordImportStore(ImageRecordImportStore{ID: "store"})}
			case "empty plural omitted":
				options = []ImageRecordImportOption{WithImageRecordImportStores(), WithImageRecordImportAllStores(true)}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				want := `{"method":{"name":"glance-direct"},"all_stores":true}`
				if mode == "false singular allowed" {
					want = `{"method":{"name":"glance-direct"},"all_stores":false,"all_stores_must_succeed":false,"stores":["store"]}`
				}
				imageRecordImportPayload(t, req, want)
				return taskCoreJSON(req, 202, ""), nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, options...)
			if mode == "false singular allowed" || mode == "empty plural omitted" {
				if got == nil || err != nil || calls != 2 {
					t.Fatal(got, err, calls)
				}
			} else if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordImportExtensionAndFullOptionsFactoriesAreSnapshotsAndReusable(t *testing.T) {
	precise := json.RawMessage(`900719925474099312345`)
	nested := map[string]any{"exact": precise, "nil": nil}
	methodNested := map[string]any{"future": []any{1, true, nil}}
	rootFields := map[string]any{"vendor": nested}
	methodFields := map[string]any{"vendor_method": methodNested}
	headers := map[string]string{"x-note": "captured"}
	options := []ImageRecordImportOption{WithImageRecordImportFields(rootFields), WithImageRecordImportMethodFields(methodFields), WithImageRecordImportHeaders(headers), WithImageRecordImportField("root_last", "first"), WithImageRecordImportField("root_last", "last"), WithImageRecordImportMethodField("method_last", false), WithImageRecordImportFields(nil), WithImageRecordImportMethodFields(map[string]any{})}
	precise[0] = '8'
	nested["nil"] = "changed"
	methodNested["future"] = "changed"
	headers["x-note"] = "changed"
	rootFields["later"] = true
	for repeat := 0; repeat < 2; repeat++ {
		t.Run(fmt.Sprintf("reused=%d", repeat), func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				imageRecordImportPayload(t, req, `{"method":{"name":"glance-direct","vendor_method":{"future":[1,true,null]},"method_last":false},"vendor":{"exact":900719925474099312345,"nil":null},"root_last":"last"}`)
				if req.Header.Get("X-Note") != "captured" {
					t.Fatal(req.Header)
				}
				return taskCoreJSON(req, 202, ""), nil
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, options...)
			if got == nil || err != nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("full opts replaces every namespace and selection", func(t *testing.T) {
		calls := 0
		var handler taskCoreTransport
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
		service := New(client)
		seed := imageRecordImportFetched(t, service, &handler)
		method := json.RawMessage(`"copy-image"`)
		opts := ImageRecordImportOpts{Method: method, Headers: map[string]string{"X-Final": "new"}, Fields: map[string]any{"retained": nil}, MethodFields: map[string]any{"retained_method": true}}
		replacement := WithImageRecordImportOpts(opts)
		method[1] = 'X'
		opts.Headers["X-Final"] = "changed"
		opts.Fields["retained"] = "changed"
		opts.MethodFields["retained_method"] = false
		handler = func(req *http.Request) (*http.Response, error) {
			imageRecordImportPayload(t, req, `{"method":{"name":"copy-image","retained_method":true},"retained":null}`)
			if req.Header.Get("X-Discarded") != "" || req.Header.Get("X-Final") != "new" || req.Header.Get("X-Added") != "yes" || req.Header.Get("X-Image-Meta-Store") != "" {
				t.Fatal(req.Header)
			}
			return taskCoreJSON(req, 202, ""), nil
		}
		got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, WithImageRecordImportStore(ImageRecordImportStore{ID: "discarded store"}), WithImageRecordImportHeader("X-Discarded", "old"), WithImageRecordImportField("discarded", true), WithImageRecordImportMethodField("discarded_method", true), replacement, WithImageRecordImportHeaders(map[string]string{"X-Added": "yes"}))
		if got == nil || err != nil || calls != 2 {
			t.Fatal(got, err, calls)
		}
	})
	t.Run("plural helper replacement and ordinary canonical last value", func(t *testing.T) {
		calls := 0
		var handler taskCoreTransport
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
		service := New(client)
		seed := imageRecordImportFetched(t, service, &handler)
		handler = func(req *http.Request) (*http.Response, error) {
			imageRecordImportPayload(t, req, `{"method":{"name":"glance-direct"},"stores":["final"]}`)
			if req.Header.Get("X-Note") != "last" || req.Header.Get("X-Merged") != "yes" {
				t.Fatal(req.Header)
			}
			return taskCoreJSON(req, 202, ""), nil
		}
		got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, WithImageRecordImportStores(ImageRecordImportStore{ID: "discarded"}), WithImageRecordImportStores(), WithImageRecordImportStores(ImageRecordImportStore{ID: "final"}), WithImageRecordImportHeader("x-note", "first"), WithImageRecordImportHeaders(map[string]string{"X-Merged": "yes"}), WithImageRecordImportHeader("X-NOTE", "last"))
		if got == nil || err != nil || calls != 2 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestImageRecordImportCapturesPrivateImageStoreOptionsAndLocationBeforeCallbacks(t *testing.T) {
	calls, callbacks, locations := 0, 0, 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	stores := imageRecordImportStoreRecords(t, client, &handler, `[{"id":"captured store"}]`, 1)
	service := New(client)
	seed := imageRecordImportFetched(t, service, &handler)
	selector := &ImageRecordImportStore{Record: stores[0]}
	headers := map[string]string{"x-note": "captured"}
	method := json.RawMessage(`"copy-image"`)
	factory := WithImageRecordImportOpts(ImageRecordImportOpts{Method: method, Store: selector, Headers: headers, AllStores: json.RawMessage(`false`)})
	method[1] = 'X'
	selector.ID = "later selector"
	headers["x-note"] = "later header"
	cloud := "captured location"
	var retained *ImageRecordImportOpts
	options := []ImageRecordImportOption{factory, func(config *ImageRecordImportOpts) error {
		callbacks++
		retained = config
		return WithImageRecordImportHeader("X-Final", "yes")(config)
	}}
	client.MoreHeaders = map[string]string{"X-Source": "captured source"}
	service.dependencies.CloudLocation = func() (resource.CloudLocation, error) {
		locations++
		seed.bodyState.current["id"] = json.RawMessage(`"later private target"`)
		seed.bodyState.current["name"] = json.RawMessage(`"later private name"`)
		*stores[0] = serviceinfo.StoreRecord{}
		options[1] = func(*ImageRecordImportOpts) error { t.Fatal("options slice was not captured"); return nil }
		client.MoreHeaders["X-Source"] = "later source"
		client.SetToken("live import token")
		return resource.CloudLocation{Cloud: &cloud}, nil
	}
	handler = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/import" || req.Header.Get("X-Auth-Token") != "live import token" || req.Header.Get("X-Source") != "captured source" || req.Header.Get("X-Note") != "captured" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Image-Meta-Store") != "captured store" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		retained.Method[1] = 'Y'
		retained.Headers["X-Note"] = "retained mutation"
		retained.Store.ID = "retained store"
		cloud = "later location"
		imageRecordImportPayload(t, req, `{"method":{"name":"copy-image"},"all_stores":false,"stores":["captured store"]}`)
		return taskCoreJSON(req, 202, ""), nil
	}
	got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, options...)
	if got == nil || got.Record == nil || err != nil || calls != 3 || callbacks != 1 || locations != 1 || string(got.Record.Resource.Body["name"]) != `"before"` {
		t.Fatal(got, err, calls, callbacks, locations)
	}
	var location resource.CloudLocation
	if err := json.Unmarshal(got.Record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured location" {
		t.Fatal(location, err)
	}
}

func TestImageRecordImportPreflightRejectsUnsafeSelectorsDescriptorsJSONAndProtectedNamespaces(t *testing.T) {
	calls := 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	seed := imageRecordImportFetched(t, service, &handler)
	for _, test := range []struct {
		name    string
		request ImageRecordImportRequest
	}{
		{"no selector", ImageRecordImportRequest{}}, {"literal needs caller metadata", ImageRecordImportRequest{ID: "literal"}},
		{"ID and Record", ImageRecordImportRequest{ID: "literal", Record: seed}},
		{"handcrafted Record", ImageRecordImportRequest{Record: &ImageRecord{Resource: seed.Resource.Clone()}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.ImportImageRecord(context.Background(), test.request)
			if got != nil || err == nil || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, test := range []struct{ name, field, raw string }{
		{"private lone surrogate", "id", `"\ud800"`}, {"private size overflow", "size", `1e9999`}, {"private float error", "instance_type_rxtx_factor", `"invalid"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := cloneImageRecord(seed)
			bad.bodyState.current[test.field] = json.RawMessage(test.raw)
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: bad})
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, test := range []struct {
		name   string
		option ImageRecordImportOption
	}{
		{"nil option", nil}, {"raw syntax", WithImageRecordImportOpts(ImageRecordImportOpts{Method: json.RawMessage(`{`)})},
		{"raw UTF8", WithImageRecordImportMethod(json.RawMessage{'"', 0xff, '"'})},
		{"raw nested surrogate", WithImageRecordImportMethod(map[string]any{"nested": json.RawMessage(`"\ud800"`)})},
		{"invalid Go string", WithImageRecordImportURI(string([]byte{0xff}))},
		{"unsupported JSON value", WithImageRecordImportField("future", make(chan int))},
		{"root core alias", WithImageRecordImportField("ALL-STORES", false)},
		{"root core method", WithImageRecordImportFields(map[string]any{"Method": map[string]any{"name": "rogue"}})},
		{"root core stores", WithImageRecordImportField("store_id", "rogue")},
		{"method core alias", WithImageRecordImportMethodField("REMOTE-IMAGE-ID", "rogue")},
		{"method core name", WithImageRecordImportMethodFields(map[string]any{"NAME": "rogue"})},
		{"method core URI", WithImageRecordImportMethodField("URI", "rogue")},
		{"blank field", WithImageRecordImportField(" ", 1)},
		{"nonUTF8 method field", WithImageRecordImportMethodField(string([]byte{0xff}), true)},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, test.option)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, header := range []string{"Authorization", "X-Auth-Token", "Content-Type", "Accept", "Content-Length", "OpenStack-API-Version", "x-image-meta-store"} {
		t.Run("owned header="+header, func(t *testing.T) {
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, WithImageRecordImportHeader(header, "rogue"))
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("pre-canceled context and nil context avoid callbacks", func(t *testing.T) {
		callbacks := 0
		option := func(*ImageRecordImportOpts) error { callbacks++; return nil }
		got, err := service.ImportImageRecord(nil, ImageRecordImportRequest{Record: seed}, option)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal(got, err, callbacks)
		}
		marker := errors.New("pre-canceled import")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(marker)
		got, err = service.ImportImageRecord(ctx, ImageRecordImportRequest{Record: seed}, option)
		if got != nil || !errors.Is(err, marker) || !errors.Is(err, context.Canceled) || callbacks != 0 || calls != 1 {
			t.Fatal(got, err, callbacks, calls)
		}
	})
	t.Run("invalid source before option", func(t *testing.T) {
		client.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
		callbacks := 0
		got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed}, func(*ImageRecordImportOpts) error { callbacks++; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 || calls != 1 {
			t.Fatal(got, err, callbacks, calls)
		}
	})
}

func TestImageRecordImportOptionAndNestedMarshalerGuardsStopLaterWorkWithCauses(t *testing.T) {
	for _, mode := range []string{"cancel", "source", "binding", "outer", "callback cause", "nested marshaler source restored later"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("import option guard")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, callbacks, later := 0, 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			got, err := service.ImportImageRecord(ctx, ImageRecordImportRequest{Record: seed}, func(config *ImageRecordImportOpts) error {
				callbacks++
				switch mode {
				case "cancel":
					cancel(marker)
				case "source":
					client.Endpoint = "https://foreign.test/"
				case "binding":
					service.API = nil
				case "outer":
					outerBad = true
				case "callback cause":
					return marker
				case "nested marshaler source restored later":
					config.Fields = map[string]any{"a_first": imageRecordMarshalCallback(func() ([]byte, error) { client.Endpoint = "https://foreign.test/"; return []byte(`true`), nil }), "z_restore": imageRecordMarshalCallback(func() ([]byte, error) {
						later++
						client.Endpoint = "https://glance.example/reverse/glance/v2/"
						return []byte(`true`), nil
					})}
				}
				return nil
			}, func(*ImageRecordImportOpts) error { later++; return nil })
			if got != nil || err == nil || calls != 1 || callbacks != 1 || later != 0 {
				t.Fatal(got, err, calls, callbacks, later)
			}
			if mode == "source" || mode == "binding" || strings.HasPrefix(mode, "nested") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal("callback cause lost", err)
			}
		})
	}
}

func TestImageRecordImportAcknowledgementDoesNotCleanPendingBodyOrDiscardBorrowedData(t *testing.T) {
	calls := 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	fetched := imageRecordImportFetched(t, service, &handler)
	fetched.data = &imageUploadCoreReader{reader: strings.NewReader("borrowed")}
	handler = func(req *http.Request) (*http.Response, error) {
		imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/name","value":"pending"}]`)
		return taskCoreJSON(req, 203, "invalid update JSON"), nil
	}
	pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: fetched, Attributes: map[string]any{"name": "pending"}})
	if pending == nil || err != nil {
		t.Fatal(pending, err)
	}
	before := cloneImageRecord(pending)
	handler = func(req *http.Request) (*http.Response, error) {
		if calls != 3 || req.Method != http.MethodPost {
			t.Fatal(calls, req.Method)
		}
		imageRecordImportPayload(t, req, `{"method":{"name":"glance-direct"}}`)
		response := taskCoreJSON(req, 202, `{"name":"ACK replacement","status":"active"}`)
		response.Header.Set("OpenStack-image-import-methods", "ACK must not restore")
		return response, nil
	}
	got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: pending})
	if got == nil || got.Record == nil || err != nil || !reflect.DeepEqual(got.Record.bodyState, before.bodyState) || got.Record.data != before.data || got.Record.StatusCode != before.StatusCode || string(got.Record.Envelope) != string(before.Envelope) || len(got.Record.ImportMethods) != 0 || !reflect.DeepEqual(pending, before) {
		t.Fatal(got, err)
	}
	handler = func(req *http.Request) (*http.Response, error) {
		if calls != 4 || req.Method != http.MethodPatch {
			t.Fatal(calls, req.Method)
		}
		imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/name","value":"pending"}]`)
		return taskCoreJSON(req, 200, `{}`), nil
	}
	committed, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
	if committed == nil || err != nil || len(committed.bodyState.dirty) != 0 || committed.data != fetched.data || calls != 4 {
		t.Fatal(committed, err, calls)
	}
}

func TestImageRecordImportAcceptedPhysicalFailuresRetainOnlyActualPartialAcknowledgement(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel", "source drift restored on Close", "outer drift restored on Close"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("accepted import handling")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			before := cloneImageRecord(seed)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			body := &taskCoreBody{reader: strings.NewReader("actual accepted import")}
			action := func() {}
			switch mode {
			case "read":
				body.reader = &taskCoreReader{body: "actual accepted import", err: marker}
			case "close":
				body.closeErr = marker
			case "cancel":
				action = func() { cancel(marker) }
			case "source drift restored on Close":
				action = func() { client.Endpoint = "https://foreign.test/" }
			case "outer drift restored on Close":
				action = func() { outerBad = true }
			}
			if mode != "read" {
				body.reader = &taskCoreReader{body: "actual accepted import", err: io.EOF, action: action}
			}
			var selected io.ReadCloser = body
			if strings.Contains(mode, "restored") {
				selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; outerBad = false }}
			}
			handler = func(req *http.Request) (*http.Response, error) { return taskCoreHTTP(req, 201, selected), nil }
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := service.ImportImageRecord(ctx, ImageRecordImportRequest{Record: seed})
			if got == nil || got.Record != nil || got.Acknowledgement == nil || got.Acknowledgement.ImageID != "fixed" || got.Acknowledgement.StatusCode != 201 || string(got.Acknowledgement.Body) != "actual accepted import" || err == nil || calls != 2 || retries != 0 || body.closes != 1 || !reflect.DeepEqual(seed, before) {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			if strings.HasPrefix(mode, "source") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal("accepted cause lost", err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			taskCoreProof(t, err, 201, "actual accepted import")
		})
	}
}

func TestImageRecordImportNativeRejectionsPreserveErrorEvidenceAndNeverAcknowledge(t *testing.T) {
	for _, code := range []int{400, 403, 404, 500, 599} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, retries := 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			body := &taskCoreBody{reader: strings.NewReader("actual rejection")}
			handler = func(req *http.Request) (*http.Response, error) { return taskCoreHTTP(req, code, body), nil }
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed})
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if got != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "actual rejection" || native.ResponseHeader.Get("X-Task-Proof") != "actual" || errors.As(err, &accepted) || calls != 2 || retries != 1 || body.closes != 1 || errors.Is(err, resource.ErrNotFound) {
				t.Fatal(got, err, native, calls, retries, body.closes)
			}
		})
	}
}

func TestImageRecordImportNativeRetryFreezesBodyTargetStoreHeadersAndKeepsLiveAuthentication(t *testing.T) {
	for _, mode := range []string{"successful retry", "expanded OkCodes rejection", "body ownership", "store header ownership", "store header removal", "absent store header ownership", "source change", "canceled retry", "hook cause", "final missing"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("import retry cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordImportFetched(t, service, &handler)
			submitted := `{"method":{"name":"copy-image"},"stores":["selected"]}`
			wantStore := "selected"
			options := []ImageRecordImportOption{WithImageRecordImportMethod("copy-image"), WithImageRecordImportStore(ImageRecordImportStore{ID: "selected"})}
			if mode == "absent store header ownership" {
				submitted = `{"method":{"name":"copy-image"}}`
				wantStore = ""
				options = []ImageRecordImportOption{WithImageRecordImportMethod("copy-image")}
				client.MoreHeaders = map[string]string{"X-Image-Meta-Store": "configured decoy"}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/import" || req.URL.RawQuery != "" || req.Header.Get("X-Image-Meta-Store") != wantStore {
					t.Fatal("retry moved import scope/selection", req.Method, req.URL, req.Header)
				}
				if mode == "absent store header ownership" {
					if _, present := req.Header[http.CanonicalHeaderKey("X-Image-Meta-Store")]; present {
						t.Fatal("retry added removed unselected store", req.Header)
					}
				}
				imageRecordImportPayload(t, req, submitted)
				if calls == 2 {
					return taskCoreJSON(req, 503, "initial rejection"), nil
				}
				if calls != 3 || req.Header.Get("X-Auth-Token") != "retry token" || req.Header.Get("X-Retry") != "ordinary" {
					t.Fatal(calls, req.Header)
				}
				if mode == "expanded OkCodes rejection" {
					return taskCoreJSON(req, 418, "actual rejected status"), nil
				}
				if mode == "final missing" {
					return taskCoreJSON(req, 404, "final missing"), nil
				}
				return taskCoreJSON(req, 203, "actual accepted"), nil
			}
			client.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, count uint) error {
				retries++
				if method != http.MethodPost || endpoint != "https://glance.example/reverse/glance/v2/images/fixed/import" {
					t.Fatal(method, endpoint)
				}
				if count > 1 {
					return original
				}
				client.SetToken("retry token")
				if opts.MoreHeaders == nil {
					opts.MoreHeaders = make(map[string]string)
				}
				opts.MoreHeaders["X-Retry"] = "ordinary"
				switch mode {
				case "expanded OkCodes rejection":
					opts.OkCodes = append(opts.OkCodes, 418)
				case "body ownership":
					opts.JSONBody = map[string]any{"method": map[string]any{"name": "rogue"}}
				case "store header removal":
					delete(opts.MoreHeaders, "X-Image-Meta-Store")
				case "store header ownership", "absent store header ownership":
					opts.MoreHeaders["X-Image-Meta-Store"] = "rogue"
				case "source change":
					client.Endpoint = "https://foreign.test/"
				case "canceled retry":
					cancel(marker)
				case "hook cause":
					return errors.Join(original, marker)
				}
				return nil
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			got, err := service.ImportImageRecord(ctx, ImageRecordImportRequest{Record: seed}, options...)
			wantCalls := 2
			if mode == "successful retry" || mode == "expanded OkCodes rejection" || mode == "final missing" {
				wantCalls = 3
			}
			if calls != wantCalls || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatal(calls, retries)
			}
			if mode == "successful retry" {
				if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || got.Acknowledgement.StatusCode != 203 || retries != 1 {
					t.Fatal(got, err, retries)
				}
				return
			}
			code, body := 503, "initial rejection"
			if mode == "expanded OkCodes rejection" {
				code, body = 418, "actual rejected status"
			}
			if mode == "final missing" {
				code, body = 404, "final missing"
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Task-Proof") != "actual" {
				t.Fatal(got, err, native)
			}
			if mode == "body ownership" || mode == "store header ownership" || mode == "store header removal" || mode == "absent store header ownership" || mode == "source change" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			if mode == "canceled retry" || mode == "hook cause" {
				if !errors.Is(err, marker) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestImageRecordImportSafeReauthenticationRetriesOnlyCapturedJSONSubmission(t *testing.T) {
	calls, reauth := 0, 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	seed := imageRecordImportFetched(t, service, &handler)
	handler = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/import" || req.URL.RawQuery != "" {
			t.Fatal(req.Method, req.URL)
		}
		imageRecordImportPayload(t, req, `{"method":{"name":"glance-direct"}}`)
		if calls == 2 {
			return taskCoreJSON(req, 401, "expired"), nil
		}
		if calls != 3 || req.Header.Get("X-Auth-Token") != "reauth token" {
			t.Fatal(calls, req.Header)
		}
		return taskCoreJSON(req, 202, "accepted once"), nil
	}
	client.ProviderClient.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("reauth token"); return nil }
	got, err := service.ImportImageRecord(context.Background(), ImageRecordImportRequest{Record: seed})
	if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 3 || reauth != 1 || string(got.Acknowledgement.Body) != "accepted once" {
		t.Fatal(got, err, calls, reauth)
	}
}
