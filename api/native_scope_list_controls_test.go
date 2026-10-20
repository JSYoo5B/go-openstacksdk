package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const nativeControlContainer = "container ?#% 한글"

type nativeControlRow struct {
	identity string
	value    any
}

type nativeControlAccess struct {
	list func(context.Context, ...resource.ListOption) iter.Seq2[nativeControlRow, error]
	all  func(context.Context, ...resource.ListOption) ([]nativeControlRow, error)
}

func nativeControlAdapter[T any](list func(context.Context, ...resource.ListOption) iter.Seq2[*T, error], all func(context.Context, ...resource.ListOption) ([]*T, error), identity func(*T) string) nativeControlAccess {
	return nativeControlAccess{
		list: func(ctx context.Context, options ...resource.ListOption) iter.Seq2[nativeControlRow, error] {
			return func(yield func(nativeControlRow, error) bool) {
				for value, err := range list(ctx, options...) {
					if err != nil {
						yield(nativeControlRow{}, err)
						return
					}
					if !yield(nativeControlRow{identity(value), value}, nil) {
						return
					}
				}
			}
		},
		all: func(ctx context.Context, options ...resource.ListOption) ([]nativeControlRow, error) {
			values, err := all(ctx, options...)
			if err != nil {
				return nil, err
			}
			result := make([]nativeControlRow, 0, len(values))
			for _, value := range values {
				result = append(result, nativeControlRow{identity(value), value})
			}
			return result, nil
		},
	}
}

type nativeControlCase struct {
	name, service, prefix, path, plural, links string
	parentID, parentPath, parentPlural         string
	swift, hasName, host                       bool
	new                                        func(*testing.T, *gophercloud.ServiceClient, resource.Ref) nativeControlAccess
}

func nativeControlCases() []nativeControlCase {
	return []nativeControlCase{
		{"actions", "compute", "/compute", "/compute/servers/parent/os-instance-actions", "instanceActions", "links", "parent", "/compute/servers/detail", "servers", false, false, false,
			func(t *testing.T, client *gophercloud.ServiceClient, parent resource.Ref) nativeControlAccess {
				client.Microversion = "2.84"
				scope, err := instanceactions.New(client).InServer(context.Background(), parent)
				if err != nil {
					t.Fatal(err)
				}
				if scope.ServerID() != "parent" {
					t.Fatal(scope.ServerID())
				}
				return nativeControlAdapter(scope.List, scope.All, func(value *instanceactions.ActionResource) string { return value.RequestID })
			}},
		{"containers", "object-store", "/swift", "/swift/", "", "", "", "", "", true, true, false,
			func(_ *testing.T, client *gophercloud.ServiceClient, _ resource.Ref) nativeControlAccess {
				collection := containers.New(client).Resources
				return nativeControlAdapter(collection.List, collection.All, func(value *containers.ContainerResource) string { return value.Name })
			}},
		{"objects", "object-store", "/swift", "/swift/" + url.PathEscape(nativeControlContainer), "", "", nativeControlContainer, "/swift/", "", true, true, false,
			func(t *testing.T, client *gophercloud.ServiceClient, parent resource.Ref) nativeControlAccess {
				scope, err := objects.New(client).InContainer(context.Background(), parent)
				if err != nil {
					t.Fatal(err)
				}
				return nativeControlAdapter(scope.List, scope.All, func(value *objects.ObjectResource) string { return value.Name })
			}},
		{"databases", "database", "/trove", "/trove/instances/parent/databases", "databases", "databases_links", "parent", "/trove/instances", "instances", false, true, false,
			func(t *testing.T, client *gophercloud.ServiceClient, parent resource.Ref) nativeControlAccess {
				scope, err := databases.New(client).InInstance(context.Background(), parent)
				if err != nil {
					t.Fatal(err)
				}
				return nativeControlAdapter(scope.List, scope.All, func(value *databases.Database) string { return value.Name })
			}},
		{"users", "database", "/trove", "/trove/instances/parent/users", "users", "users_links", "parent", "/trove/instances", "instances", false, true, true,
			func(t *testing.T, client *gophercloud.ServiceClient, parent resource.Ref) nativeControlAccess {
				scope, err := users.New(client).InInstance(context.Background(), parent, users.WithHost("db.example"))
				if err != nil {
					t.Fatal(err)
				}
				return nativeControlAdapter(scope.List, scope.All, func(value *users.UserResource) string { return value.Name })
			}},
	}
}

func nativeControlBody(t *testing.T, facade nativeControlCase, rows []map[string]any, next string) string {
	t.Helper()
	if rows == nil {
		rows = []map[string]any{}
	}
	var body any = rows
	if !facade.swift {
		envelope := map[string]any{facade.plural: rows}
		if next != "" {
			envelope[facade.links] = []map[string]string{{"rel": "next", "href": next}}
		}
		body = envelope
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func nativeControlWireRow(facade nativeControlCase, identity, host string) map[string]any {
	switch facade.name {
	case "actions":
		return map[string]any{"action": "reboot", "request_id": identity, "instance_uuid": "wire-parent", "message": nil, "start_time": "2026-10-01T00:00:00.123456", "updated_at": "2026-10-01T00:01:00.000000", "vendor": json.RawMessage(`9007199254740993`), "events": []map[string]any{{"event": "compute_reboot_instance", "result": "Success", "details": "extension", "vendor": false}}}
	case "containers":
		return map[string]any{"name": identity, "count": 2, "bytes": 9007199254740993}
	case "objects":
		return map[string]any{"name": identity, "bytes": 5, "content_type": "text/plain", "hash": "checksum", "last_modified": "2030-01-02T03:04:05.123456", "is_latest": true, "version_id": "version"}
	case "databases":
		return map[string]any{"name": identity, "character_set": "utf8mb4", "collate": "utf8mb4_unicode_ci"}
	default:
		return map[string]any{"name": identity, "host": host, "password": "returned-password", "databases": []map[string]string{{"name": "logs", "character_set": "utf8mb4", "collate": "utf8mb4_unicode_ci"}}}
	}
}

func nativeControlAssertFields(t *testing.T, row nativeControlRow) {
	t.Helper()
	switch value := row.value.(type) {
	case *instanceactions.ActionResource:
		if value.ServerID != "parent" || value.InstanceUUID != "wire-parent" || value.Details != nil || value.UpdatedAt == nil || value.Events == nil || len(*value.Events) != 1 || (*value.Events)[0].Details == nil || string((*value.Events)[0].Body["vendor"]) != "false" || string(value.Body["vendor"]) != "9007199254740993" || value.Header.Get("X-Trace") != "native-page" {
			t.Fatal("action summary/detail/raw evidence changed", value)
		}
	case *containers.ContainerResource:
		if value.Count != 2 || value.Bytes != 9007199254740993 || value.Details != nil || value.Metadata != nil || value.Header != nil {
			t.Fatal("container list was confused with HEAD metadata", value)
		}
	case *objects.ObjectResource:
		if value.Container != nativeControlContainer || value.Bytes != 5 || value.ContentType != "text/plain" || value.Hash != "checksum" || value.VersionID != "version" || !value.IsLatest || value.LastModified.IsZero() || value.Details != nil || value.Metadata != nil || value.Header != nil {
			t.Fatal("object scope/list fields changed", value)
		}
	case *databases.Database:
		if value.CharSet != "utf8mb4" || value.Collate != "utf8mb4_unicode_ci" {
			t.Fatal("Trove database charset lost", value)
		}
	case *users.UserResource:
		if value.Host != "db.example" || value.Password != "returned-password" || len(value.Databases) != 1 || value.Databases[0].CharSet != "utf8mb4" || value.Databases[0].Collate != "utf8mb4_unicode_ci" {
			t.Fatal("Trove user native fields/host lost", value)
		}
	}
}

func nativeControlCollect(sequence iter.Seq2[nativeControlRow, error]) ([]nativeControlRow, error) {
	values := make([]nativeControlRow, 0)
	for value, err := range sequence {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestNativeScopeListControlsFiveBindingsRawCapBeforeNameAndHostFilters(t *testing.T) {
	for _, facade := range nativeControlCases() {
		for _, maximum := range []int{2, 3} {
			t.Run(facade.name+"/maximum="+strconv.Itoa(maximum), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodGet || r.URL.EscapedPath() != facade.path || r.URL.Query().Has("limit") || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") || r.URL.Query().Has("host") {
						t.Error("controls changed native route/query", r.Method, r.URL)
					}
					first, second := "other", "wanted"
					if facade.host {
						first, second = "wanted", "other"
					}
					rows := []map[string]any{nativeControlWireRow(facade, first, "other.example"), nativeControlWireRow(facade, second, "db.example"), nativeControlWireRow(facade, "wanted", "db.example")}
					w.Header().Set("X-Trace", "native-page")
					testcloud.JSON(w, 200, nativeControlBody(t, facade, rows, "https://foreign.invalid/next"))
				})
				access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.ID(facade.parentID))
				options := []resource.ListOption{resource.WithMaxItems(maximum)}
				if facade.hasName {
					options = append(options, resource.WithName("wanted"))
				}
				values, err := access.all(context.Background(), options...)
				want := maximum
				if facade.hasName {
					want--
				}
				if facade.host {
					want--
				}
				if err != nil || len(values) != want || requests.Load() != 1 {
					t.Fatal("cap counted matched name/host rows or processed continuation", err, len(values), requests.Load())
				}
				for _, row := range values {
					nativeControlAssertFields(t, row)
				}
			})
		}
	}
}

func TestNativeScopeListControlsSinglePageSkipsForeignAndCyclicContinuation(t *testing.T) {
	for _, facade := range nativeControlCases() {
		modes := []string{"cycle", "foreign"}
		if facade.swift {
			modes = []string{"cycle"} // Swift's native page owns a marker protocol.
		}
		for _, mode := range modes {
			t.Run(facade.name+"/"+mode, func(t *testing.T) {
				cloud, foreign := testcloud.New(t), testcloud.New(t)
				var requests, foreignRequests atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { foreignRequests.Add(1); w.WriteHeader(500) })
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					next := cloud.Server.URL + r.URL.String()
					if mode == "foreign" {
						next = foreign.Server.URL + facade.path
					}
					w.Header().Set("X-Trace", "native-page")
					rows := []map[string]any{nativeControlWireRow(facade, "one", "db.example"), nativeControlWireRow(facade, "two", "db.example")}
					testcloud.JSON(w, 200, nativeControlBody(t, facade, rows, next))
				})
				access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.ID(facade.parentID))
				options := []resource.ListOption{resource.WithPaginated(false)}
				if facade.swift {
					options = append(options, resource.WithQuery("marker", "two"))
				}
				values, err := nativeControlCollect(access.list(context.Background(), options...))
				if err != nil || len(values) != 2 || requests.Load() != 1 || foreignRequests.Load() != 0 {
					t.Fatal("single page evaluated continuation", err, len(values), requests.Load(), foreignRequests.Load())
				}
			})
		}
	}
}

func TestNativeScopeListControlsCrossPageQueryFixedParentAndIteratorReuse(t *testing.T) {
	for _, facade := range nativeControlCases() {
		t.Run(facade.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.EscapedPath() != facade.path || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") || r.URL.Query().Has("host") {
					t.Error(r.URL)
				}
				if facade.service != "database" && (r.URL.Query().Get("limit") != "1" || r.URL.Query().Get("vendor") != "a&b") {
					t.Error("typed native builder query changed", r.URL)
				}
				marker := r.URL.Query().Get("marker")
				rows := []map[string]any{nativeControlWireRow(facade, "one", "other.example"), nativeControlWireRow(facade, "two", "db.example")}
				next := ""
				if marker == "" {
					query := r.URL.Query()
					query.Set("marker", "two")
					next = cloud.Server.URL + facade.path + "?" + query.Encode()
				} else if marker == "two" {
					rows = []map[string]any{nativeControlWireRow(facade, "three", "db.example")}
				} else {
					t.Error("cap should stop before third page", marker)
				}
				w.Header().Set("X-Trace", "native-page")
				testcloud.JSON(w, 200, nativeControlBody(t, facade, rows, next))
			})
			access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.ID(facade.parentID))
			options := []resource.ListOption{resource.WithMaxItems(3)}
			if facade.service != "database" {
				options = append(options, resource.WithPageSize(1), resource.WithQuery("vendor", "a&b"))
			}
			sequence := access.list(context.Background(), options...)
			for iteration := 0; iteration < 2; iteration++ {
				values, err := nativeControlCollect(sequence)
				want := 3
				if facade.host {
					want = 2
				}
				if err != nil || len(values) != want || requests.Load() != int32(2*(iteration+1)) {
					t.Fatal("controlled paging changed fixed route/count or was not reusable", err, len(values), requests.Load())
				}
				for _, row := range values {
					nativeControlAssertFields(t, row)
				}
			}
		})
	}
}

func TestNativeScopeListControlsKeepDefaultContinuationCycleGuard(t *testing.T) {
	for _, facade := range nativeControlCases() {
		t.Run(facade.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				testcloud.JSON(w, 200, nativeControlBody(t, facade, []map[string]any{nativeControlWireRow(facade, "one", "db.example")}, cloud.Server.URL+r.URL.String()))
			})
			access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.ID(facade.parentID))
			var options []resource.ListOption
			if facade.swift {
				options = append(options, resource.WithQuery("marker", "one"))
			}
			values, err := access.all(context.Background(), options...)
			if !errors.Is(err, resource.ErrPaginationCycle) || values != nil || requests.Load() != 1 {
				t.Fatal("default native paging no longer rejects cycles", err, requests.Load())
			}
		})
	}
}

func TestNativeScopeListControlsZeroDefaultsFollowNativeContinuation(t *testing.T) {
	for _, facade := range nativeControlCases() {
		t.Run(facade.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.EscapedPath() != facade.path || r.URL.Query().Has("limit") || r.URL.Query().Has("host") {
					t.Error("zero controls changed native route/query", r.URL)
				}
				rows := []map[string]any{nativeControlWireRow(facade, "one", "db.example")}
				next := cloud.Server.URL + facade.path + "?marker=one"
				switch r.URL.Query().Get("marker") {
				case "":
				case "one":
					rows = []map[string]any{nativeControlWireRow(facade, "two", "db.example")}
					next = ""
				case "two":
					rows, next = nil, ""
				default:
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, nativeControlBody(t, facade, rows, next))
			})
			access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.ID(facade.parentID))
			values, err := access.all(context.Background())
			wantRequests := int32(2)
			if facade.swift {
				wantRequests = 3 // Marker paging terminates on the empty third response.
			}
			if err != nil || len(values) != 2 || requests.Load() != wantRequests {
				t.Fatal("default collection became bounded or single page", err, len(values), requests.Load())
			}
		})
	}
}

func TestNativeScopeListControlsKeepWholePageDecodeErrors(t *testing.T) {
	for _, facade := range nativeControlCases() {
		t.Run(facade.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				bad := nativeControlWireRow(facade, "bad", "db.example")
				field := map[string]string{"actions": "request_id", "containers": "count", "objects": "bytes", "databases": "character_set", "users": "host"}[facade.name]
				bad[field] = []any{}
				testcloud.JSON(w, 200, nativeControlBody(t, facade, []map[string]any{nativeControlWireRow(facade, "good", "db.example"), bad}, ""))
			})
			access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.ID(facade.parentID))
			values, err := access.all(context.Background(), resource.WithMaxItems(1), resource.WithPaginated(false))
			var typed *json.UnmarshalTypeError
			if !errors.As(err, &typed) || values != nil || requests.Load() != 1 {
				t.Fatal("native full-page decoder error beyond cap was hidden", err, values, requests.Load())
			}
		})
	}
}

func TestNativeScopeListControlsParentCancellationAndConsumerBreak(t *testing.T) {
	for _, facade := range nativeControlCases() {
		for _, mode := range []string{"preflight", "cancel", "break"} {
			t.Run(facade.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					testcloud.JSON(w, 200, nativeControlBody(t, facade, []map[string]any{nativeControlWireRow(facade, "one", "db.example"), nativeControlWireRow(facade, "two", "db.example")}, cloud.Server.URL+facade.path+"?marker=two"))
				})
				access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.ID(facade.parentID))
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if mode == "preflight" {
					cancel()
				}
				var terminal error
				var yielded int
				for _, err := range access.list(ctx, resource.WithMaxItems(2), resource.WithPaginated(false)) {
					if err != nil {
						terminal = err
						break
					}
					yielded++
					if mode == "break" {
						break
					}
					cancel()
				}
				want := 1
				if mode == "preflight" {
					want = 0
				}
				if mode != "break" && !errors.Is(terminal, context.Canceled) || mode == "break" && terminal != nil || requests.Load() != int32(want) || yielded != want {
					t.Fatal("control suppressed cancellation or continued after break", terminal, requests.Load(), yielded)
				}
			})
		}
	}
}

func TestNativeScopeListControlsPreflightNegativeAndTroveQueryPolicy(t *testing.T) {
	for _, facade := range nativeControlCases() {
		t.Run(facade.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); w.WriteHeader(500) })
			access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.ID(facade.parentID))
			if values, err := access.all(context.Background(), resource.WithMaxItems(-1)); !errors.Is(err, resource.ErrInvalidOption) || values != nil || requests.Load() != 0 {
				t.Fatal(err, values, requests.Load())
			}
			if facade.service == "database" {
				for _, option := range []resource.ListOption{resource.WithPageSize(1), resource.WithQuery("host", "db.example"), resource.WithQuery("vendor", "a&b")} {
					values, err := access.all(context.Background(), resource.WithPaginated(false), option)
					if !errors.Is(err, resource.ErrUnsupported) || values != nil || requests.Load() != 0 {
						t.Fatal("new control widened Trove initial query contract", err, values, requests.Load())
					}
				}
			}
		})
	}
}

func TestNativeScopeListControlsSwiftSubdirectoryConsumesRawCap(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, heads atomic.Int32
	cloud.Mux.HandleFunc("/swift/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			heads.Add(1)
			w.WriteHeader(500)
			return
		}
		lists.Add(1)
		if r.URL.EscapedPath() != "/swift/"+url.PathEscape(nativeControlContainer) || r.URL.Query().Get("prefix") != "folder/my file?#%.txt" || r.URL.Query().Get("delimiter") != "/" || r.URL.Query().Has("limit") {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `[{"subdir":"folder/"},{"name":"folder/my file?#%.txt","bytes":5,"content_type":"text/plain","hash":"checksum","last_modified":"2030-01-02T03:04:05.123456","is_latest":true,"version_id":"version"}]`)
	})
	scope, err := objects.New(cloud.Client("object-store", "/swift")).InContainer(context.Background(), resource.ID(nativeControlContainer))
	if err != nil {
		t.Fatal(err)
	}
	for _, maximum := range []int{1, 2} {
		values, err := scope.All(context.Background(), resource.WithName("folder/my file?#%.txt"), resource.WithQuery("delimiter", "/"), resource.WithMaxItems(maximum))
		if err != nil || len(values) != maximum-1 || lists.Load() != int32(maximum) || heads.Load() != 0 {
			t.Fatal("subdirectory row did not consume cap or list performed HEAD", err, values, lists.Load(), heads.Load())
		}
		if maximum == 2 {
			nativeControlAssertFields(t, nativeControlRow{values[0].Name, values[0]})
		}
	}
}

func TestNativeScopeListControlsResolveNamedParentOnce(t *testing.T) {
	for _, facade := range nativeControlCases() {
		if facade.parentPath == "" {
			continue
		}
		t.Run(facade.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var parents, children atomic.Int32
			parentName := "parent-name"
			if facade.name == "objects" {
				parentName = nativeControlContainer
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.EscapedPath() == facade.parentPath {
					parents.Add(1)
					if facade.swift {
						if r.URL.Query().Get("prefix") != parentName {
							t.Error(r.URL)
						}
						if r.URL.Query().Has("marker") {
							testcloud.JSON(w, 200, `[]`)
							return
						}
						testcloud.JSON(w, 200, fmt.Sprintf(`[{"name":%q,"count":2,"bytes":5}]`, facade.parentID))
					} else {
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":%q,"name":%q}]}`, facade.parentPlural, facade.parentID, parentName))
					}
					return
				}
				if r.URL.EscapedPath() != facade.path {
					t.Error("fixed child parent changed", r.URL)
				}
				children.Add(1)
				w.Header().Set("X-Trace", "native-page")
				testcloud.JSON(w, 200, nativeControlBody(t, facade, []map[string]any{nativeControlWireRow(facade, "one", "db.example")}, ""))
			})
			access := facade.new(t, cloud.Client(facade.service, facade.prefix), resource.Name(parentName))
			wantParents := int32(1)
			if facade.swift {
				wantParents = 2 // One Swift lookup includes its terminal empty page.
			}
			for iteration := 0; iteration < 2; iteration++ {
				values, err := access.all(context.Background(), resource.WithPaginated(false))
				if err != nil || len(values) != 1 || parents.Load() != wantParents || children.Load() != int32(iteration+1) {
					t.Fatal("scope performed another parent lookup", err, len(values), parents.Load(), children.Load())
				}
			}
		})
	}
}
