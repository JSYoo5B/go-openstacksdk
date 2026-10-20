package objects_test

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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestSwiftPythonObjectMetadataHeadOwnsQueryFreeResponseAndSourceStatuses(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("object-store", "/swift/v1/AUTH_selected/")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
			t.Error("nonempty request body", string(body), err)
		}
		if r.Method != "HEAD" || r.URL.EscapedPath() != "/swift/v1/AUTH_selected/"+url.PathEscape(objectMetadataContainer)+"/"+url.PathEscape(objectMetadataKey) || r.URL.RawQuery != "" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.Method, r.URL, r.Header)
		}
		w.Header().Set("X-Object-Meta-OWNER", "observed")
		w.Header().Set("X-Proof", "head")
		w.WriteHeader(203)
	})
	result, err := objects.New(client).HeadMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, objects.WithGetMetadataNewest(false))
	if err != nil || result.StatusCode != 203 || result.Metadata.Values["owner"] != "observed" || result.Header.Get("X-Proof") != "head" {
		t.Fatal(result, err)
	}
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399} {
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			objectMetadataBodyless(t, r, false)
			return objectMetadataResponse(code, "physical HEAD"), nil
		})
		result, err := objects.New(client).HeadMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
		if err != nil || result.StatusCode != code || string(result.Body) != "physical HEAD" {
			t.Fatal(code, result, err)
		}
	}
	for _, header := range []http.Header{{"Content-Length": {"invalid"}}, {"X-Delete-After": {"²"}}, {"X-Object-Meta-Owner": {"one", "two"}}} {
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			response := objectMetadataResponse(200, "physical")
			for key, value := range header {
				response.Header[key] = value
			}
			return response, nil
		})
		result, err := objects.New(client).HeadMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
		if err == nil || result.Metadata != nil {
			t.Fatal(result, err)
		}
		objectMetadataProof(t, err, 200, "physical")
	}
}

func TestSwiftPythonObjectMetadataSetterFreshMergeFalseyAliasesAndNoFinalHEAD(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("object-store", "/swift/v1/AUTH_selected/")
	client.MoreHeaders = map[string]string{"X-Source": "selected"}
	var methods []string
	oldSystems := map[string]string{"Accept-Ranges": "bytes", "Content-Disposition": "inline", "Content-Encoding": "gzip", "Content-Type": "application/observed", "X-Delete-After": "0007", "X-Delete-At": "1700000000", "X-Detect-Content-Type": "False", "X-Object-Manifest": "parts/old", "Access-Control-Allow-Origin": "https://observed.invalid"}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
			t.Error("nonempty request body", string(body), err)
		}
		methods = append(methods, r.Method)
		if r.URL.RawQuery != "" || r.Header.Get("X-Source") != "selected" || r.Header.Get("X-Probe") != "ordinary" {
			t.Error(r.Method, r.URL, r.Header)
		}
		if r.Method == "HEAD" {
			for key, value := range oldSystems {
				w.Header().Set(key, value)
			}
			for _, key := range []string{"owner", "false", "nil", "zero", "empty", "list", "dict"} {
				w.Header().Set("X-Object-Meta-"+key, "observed "+key)
			}
			w.Header().Set("X-Object-Meta-Blank", "")
			w.Header().Set("X-Symlink-Target", "target/object")
			w.WriteHeader(200)
			return
		}
		for key, value := range oldSystems {
			want := value
			if key == "X-Delete-After" {
				want = "7"
			}
			if key == "X-Detect-Content-Type" {
				want = "True"
			}
			if r.Header.Get(key) != want {
				t.Error(key, r.Header)
			}
		}
		expected := map[string]string{"owner": "requested", "false": "observed false", "nil": "observed nil", "zero": "observed zero", "empty": "observed empty", "list": "observed list", "dict": "observed dict", "large": "900719925474099312345", "enabled": "True", "nested": "{'active': True}", "external": "prefixed"}
		for key, want := range expected {
			if r.Header.Get("X-Object-Meta-"+key) != want {
				t.Error(key, r.Header)
			}
		}
		if values, ok := r.Header["X-Object-Meta-Blank"]; !ok || !reflect.DeepEqual(values, []string{""}) {
			t.Error("empty observed custom value lost", r.Header)
		}
		if r.Header.Get("X-Symlink-Target") != "" {
			t.Error("unrelated symlink response header reposted", r.Header)
		}
		w.Header().Set("X-Proof", "write ack")
		w.WriteHeader(202)
	})
	values := map[string]any{"Owner": "requested", "false": false, "nil": nil, "zero": json.Number("0e900"), "empty": "", "list": []any{}, "dict": map[string]any{}, "large": json.Number("900719925474099312345"), "enabled": true, "nested": map[string]any{"active": true}, "X-Object-Meta-External": "prefixed"}
	result, err := objects.New(client).SetMetadataFromFresh(context.Background(), "assets", "file", values, objects.WithMetadataHeader("X-Probe", "ordinary"))
	if err != nil || result.Before.Metadata.Values["owner"] != "observed owner" || result.Acknowledgement.StatusCode != 202 || result.Acknowledgement.Header.Get("X-Proof") != "write ack" || result.Refreshed != nil || result.NoOp || !reflect.DeepEqual(methods, []string{"HEAD", "POST"}) {
		t.Fatal(result, err, methods)
	}
	if !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "selected"}) {
		t.Fatal(client.MoreHeaders)
	}
}

func TestSwiftPythonObjectMetadataSetterEverySystemAliasAndEmptyInput(t *testing.T) {
	aliases := map[string]string{"accept_ranges": "accept-ranges", "content_disposition": "content-disposition", "content_encoding": "content-encoding", "content_type": "content-type", "delete_after": "x-delete-after", "delete_at": "x-delete-at", "is_content_type_detected": "x-detect-content-type", "manifest": "x-object-manifest", "access_control_allow_origin": "access-control-allow-origin"}
	for _, wireNames := range []bool{false, true} {
		values := make(map[string]any)
		for alias, wire := range aliases {
			key := alias
			if wireNames {
				key = wire
			}
			values[key] = "input " + alias
		}
		var methods []string
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			objectMetadataBodyless(t, r, false)
			methods = append(methods, r.Method)
			if r.Method == "POST" {
				for alias, wire := range aliases {
					want := "input " + alias
					if wireNames && alias == "content_type" {
						want = "application/json"
					}
					if r.Header.Get(wire) != want {
						t.Error(alias, r.Header)
					}
				}
			}
			return objectMetadataResponse(204, ""), nil
		})
		if _, err := objects.New(client).SetMetadataFromFresh(context.Background(), objectMetadataContainer, objectMetadataKey, values); err != nil || !reflect.DeepEqual(methods, []string{"HEAD", "POST"}) {
			t.Fatal(err, methods)
		}
	}
	var methods []string
	client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		methods = append(methods, r.Method)
		response := objectMetadataResponse(204, "")
		if r.Method == "HEAD" {
			response.Header.Set("X-Object-Meta-Owner", "observed")
			response.Header.Set("X-Delete-After", "0")
			response.Header.Set("X-Detect-Content-Type", "")
		}
		if r.Method == "POST" && (r.Header.Get("X-Object-Meta-Owner") != "observed" || r.Header.Get("X-Delete-After") != "" || r.Header.Get("X-Detect-Content-Type") != "") {
			t.Error(r.Header)
		}
		return response, nil
	})
	if _, err := objects.New(client).SetMetadataFromFresh(context.Background(), objectMetadataContainer, objectMetadataKey, nil); err != nil || !reflect.DeepEqual(methods, []string{"HEAD", "POST"}) {
		t.Fatal(err, methods)
	}
	for _, tc := range []struct{ input, want string }{{"0007", "7"}, {"٠٧", "7"}, {"900719925474099312345", "900719925474099312345"}, {"-7", ""}, {"+7", ""}, {" 7 ", ""}, {"invalid", ""}, {"", ""}} {
		t.Run("descriptor "+tc.input, func(t *testing.T) {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				response := objectMetadataResponse(200, "")
				if r.Method == "HEAD" {
					response.Header.Set("X-Delete-After", tc.input)
				}
				if r.Method == "POST" {
					value, present := r.Header["X-Delete-After"]
					if tc.want == "" && present {
						t.Error("non-digit descriptor should become zero and be omitted", r.Header)
					}
					if tc.want != "" && (!present || !reflect.DeepEqual(value, []string{tc.want})) {
						t.Error(r.Header)
					}
				}
				return response, nil
			})
			if _, err := objects.New(client).SetMetadataFromFresh(context.Background(), objectMetadataContainer, objectMetadataKey, nil); err != nil || calls != 2 {
				t.Fatal(tc.input, err, calls)
			}
		})
	}

}

func TestSwiftPythonObjectMetadataDeleteNoOpsAndObservedFinalHEAD(t *testing.T) {
	for _, keys := range [][]string{nil, {}, {"missing"}, {"Owner"}, {"X-Object-Meta-Owner"}} {
		t.Run(fmt.Sprint(keys), func(t *testing.T) {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				objectMetadataBodyless(t, r, false)
				if r.Method != "HEAD" {
					t.Error(r.Method)
				}
				response := objectMetadataResponse(200, "")
				response.Header.Set("X-Object-Meta-Owner", "observed")
				return response, nil
			})
			result, err := objects.New(client).DeleteMetadataAndRefresh(context.Background(), objectMetadataContainer, objectMetadataKey, keys)
			want := 1
			if len(keys) == 0 {
				want = 0
			}
			if err != nil || !result.NoOp || result.Acknowledgement != nil || result.Refreshed != nil || calls != want {
				t.Fatal(result, err, calls)
			}
		})
	}
	cloud := testcloud.New(t)
	api := objects.New(cloud.Client("object-store", "/swift/v1/AUTH_selected/"))
	var methods []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
			t.Error("nonempty request body", string(body), err)
		}
		methods = append(methods, r.Method)
		if r.URL.RawQuery != "" {
			t.Error(r.Method, r.URL)
		}
		if len(methods) == 1 {
			w.Header().Set("X-Object-Meta-Owner", "old")
			w.Header().Set("X-Object-Meta-Keep", "old keep")
			w.Header().Set("Content-Type", "application/old")
			w.WriteHeader(200)
			return
		}
		if r.Method == "POST" {
			if _, exists := r.Header["X-Object-Meta-Owner"]; exists {
				t.Error("deleted suffix still sent", r.Header)
			}
			if r.Header.Get("X-Object-Meta-Keep") != "old keep" || r.Header.Get("Content-Type") != "application/old" {
				t.Error(r.Header)
			}
			w.Header().Set("X-Proof", "ack")
			w.WriteHeader(202)
			return
		}
		if r.Header.Get("X-Object-Meta-Keep") != "" {
			t.Error("write metadata leaked to final HEAD", r.Header)
		}
		w.Header().Set("X-Object-Meta-Keep", "fresh observed keep")
		w.Header().Set("X-Proof", "final HEAD")
		w.WriteHeader(203)
	})
	result, err := api.DeleteMetadataAndRefresh(context.Background(), "assets", "file", []string{"owner", "owner", "absent"})
	if err != nil || result.NoOp || result.Before.Metadata.Values["owner"] != "old" || result.Acknowledgement.Header.Get("X-Proof") != "ack" || result.Refreshed.Header.Get("X-Proof") != "final HEAD" || result.Refreshed.Metadata.Values["keep"] != "fresh observed keep" || !reflect.DeepEqual(methods, []string{"HEAD", "POST", "HEAD"}) {
		t.Fatal(result, err, methods)
	}
}

func TestSwiftPythonObjectMetadataDeleteAfterPreservesSourceBranch(t *testing.T) {
	for _, tc := range []struct {
		name        string
		hasDeleteAt bool
		keys        []string
		wantCalls   int
		wantError   bool
		noOp        bool
	}{
		{"missing delete_at", false, []string{"delete_after"}, 1, true, false},
		{"only delete_after has no POST", true, []string{"delete_after"}, 1, false, true},
		{"duplicate delete_after raises", true, []string{"delete_after", "delete_after", "owner"}, 1, true, false},
		{"combined actual deletion", true, []string{"delete_after", "owner"}, 3, false, false},
		{"wire system name is absent", true, []string{"x-delete-at"}, 1, false, true},
		{"canonical system alias deletes", true, []string{"delete_at"}, 3, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				objectMetadataBodyless(t, r, false)
				response := objectMetadataResponse(200, "physical")
				if calls == 1 {
					response.Header.Set("X-Object-Meta-Owner", "old")
					response.Header.Set("X-Delete-After", "7")
					if tc.hasDeleteAt {
						response.Header.Set("X-Delete-At", "1700000000")
					}
				}
				if r.Method == "POST" {
					if _, exists := r.Header["X-Delete-At"]; exists {
						t.Error("delete_at retained", r.Header)
					}
					if r.Header.Get("X-Delete-After") != "7" {
						t.Error("Source removes delete_at, not delete_after", r.Header)
					}
				}
				return response, nil
			})
			result, err := objects.New(client).DeleteMetadataAndRefresh(context.Background(), objectMetadataContainer, objectMetadataKey, tc.keys)
			if (err != nil) != tc.wantError || calls != tc.wantCalls || result.NoOp != tc.noOp {
				t.Fatal(result, err, calls)
			}
			if tc.wantError {
				var missing *objects.MetadataKeyError
				if !errors.As(err, &missing) || missing.Key != "delete_at" {
					t.Fatal(err)
				}
				objectMetadataProof(t, err, 200, "physical")
			}
		})
	}
}

func TestSwiftPythonObjectMetadataObservedAliasesRejectOnlyBeforePOST(t *testing.T) {
	pairs := [][2]string{
		{"accept_ranges", "accept-ranges"},
		{"content_disposition", "content-disposition"},
		{"content_encoding", "content-encoding"},
		{"content_type", "content-type"},
		{"delete_after", "x-delete-after"},
		{"delete_at", "x-delete-at"},
		{"is_content_type_detected", "x-detect-content-type"},
		{"manifest", "x-object-manifest"},
		{"access_control_allow_origin", "access-control-allow-origin"},
	}
	for _, pair := range pairs {
		for _, operation := range []string{"empty setter", "setter overwrite", "actual deletion"} {
			t.Run(pair[0]+"/"+operation, func(t *testing.T) {
				calls := 0
				body := &objectMetadataBody{Reader: strings.NewReader("observed alias HEAD")}
				client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					objectMetadataBodyless(t, r, false)
					if r.Method != "HEAD" || calls != 1 {
						t.Error("ambiguous observation reached write", r.Method, calls)
					}
					response := objectMetadataWire(200, body)
					response.Header.Set("X-Object-Meta-"+pair[0], "canonical custom")
					response.Header.Set("X-Object-Meta-"+pair[1], "wire custom")
					response.Header.Set("X-Object-Meta-Owner", "observed owner")
					response.Header.Set("X-Proof", "ambiguous observation")
					return response, nil
				})
				api := objects.New(client)
				var result *objects.FreshMetadataChangeResult
				var err error
				switch operation {
				case "empty setter":
					result, err = api.SetMetadataFromFresh(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
				case "setter overwrite":
					result, err = api.SetMetadataFromFresh(context.Background(), objectMetadataContainer, objectMetadataKey, map[string]any{pair[0]: "updated canonical"})
				case "actual deletion":
					result, err = api.DeleteMetadataAndRefresh(context.Background(), objectMetadataContainer, objectMetadataKey, []string{"owner"})
				}
				if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "ambiguous observed Python object metadata aliases") || calls != 1 || body.closes.Load() != 1 {
					t.Fatal(result, err, calls, body.closes.Load())
				}
				if result == nil || result.Before == nil || result.Before.Metadata == nil || result.Acknowledgement != nil || result.Refreshed != nil || result.NoOp {
					t.Fatal("lost successful HEAD or manufactured write", result, err)
				}
				if result.Before.Metadata.Values[pair[0]] != "canonical custom" || result.Before.Metadata.Values[pair[1]] != "wire custom" || result.Before.Metadata.Values["owner"] != "observed owner" {
					t.Fatal("ambiguous source observation was changed", result.Before.Metadata)
				}
				proof := objectMetadataProof(t, err, 200, "observed alias HEAD")
				if proof.Header.Get("X-Proof") != "ambiguous observation" || proof.Header.Get("X-Object-Meta-"+pair[0]) != "canonical custom" || proof.Header.Get("X-Object-Meta-"+pair[1]) != "wire custom" {
					t.Fatal("lost ambiguous HEAD proof", proof)
				}
			})
		}
	}
}

func TestSwiftPythonObjectMetadataObservedAliasesPreserveReadsAndDeleteNoOps(t *testing.T) {
	for _, tc := range []struct {
		name      string
		getter    bool
		keys      []string
		wantCalls int
	}{
		{"getter always HEAD", true, nil, 1},
		{"nil deletion", false, nil, 0},
		{"empty deletion", false, []string{}, 0},
		{"missing deletion", false, []string{"missing"}, 1},
		{"case mismatched deletion", false, []string{"Owner"}, 1},
		{"full prefix deletion", false, []string{"X-Object-Meta-Owner"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				objectMetadataBodyless(t, r, false)
				if r.Method != "HEAD" {
					t.Error("no-op reached write", r.Method)
				}
				response := objectMetadataResponse(200, "observed alias HEAD")
				response.Header.Set("X-Object-Meta-Content_type", "canonical custom")
				response.Header.Set("X-Object-Meta-Content-Type", "wire custom")
				response.Header.Set("X-Object-Meta-Owner", "observed owner")
				return response, nil
			})
			api := objects.New(client)
			var before *objects.GetMetadataResult
			if tc.getter {
				var err error
				before, err = api.HeadMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
				if err != nil {
					t.Fatal(before, err)
				}
			} else {
				result, err := api.DeleteMetadataAndRefresh(context.Background(), objectMetadataContainer, objectMetadataKey, tc.keys)
				if err != nil || result == nil || !result.NoOp || result.Acknowledgement != nil || result.Refreshed != nil {
					t.Fatal(result, err)
				}
				before = result.Before
			}
			if calls != tc.wantCalls || (before != nil) != (tc.wantCalls == 1) {
				t.Fatal(before, calls)
			}
			if before != nil && (before.Metadata == nil || before.Metadata.Values["content_type"] != "canonical custom" || before.Metadata.Values["content-type"] != "wire custom") {
				t.Fatal(before)
			}
		})
	}
	t.Run("deleting a colliding suffix makes the actual POST unambiguous", func(t *testing.T) {
		var methods []string
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			objectMetadataBodyless(t, r, false)
			methods = append(methods, r.Method)
			response := objectMetadataResponse(200, "phase proof")
			if len(methods) == 1 {
				response.Header.Set("X-Object-Meta-Content_type", "canonical custom")
				response.Header.Set("X-Object-Meta-Content-Type", "wire custom")
				response.Header.Set("X-Object-Meta-Owner", "observed owner")
			} else if r.Method == "POST" {
				if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Object-Meta-Owner") != "observed owner" {
					t.Error(r.Header)
				}
				if _, present := r.Header["X-Object-Meta-Content-Type"]; present {
					t.Error("removed suffix retained", r.Header)
				}
			} else {
				response.Header.Set("X-Object-Meta-Owner", "fresh owner")
			}
			return response, nil
		})
		result, err := objects.New(client).DeleteMetadataAndRefresh(context.Background(), objectMetadataContainer, objectMetadataKey, []string{"content-type"})
		if err != nil || result == nil || result.NoOp || result.Before == nil || result.Acknowledgement == nil || result.Refreshed == nil || result.Refreshed.Metadata.Values["owner"] != "fresh owner" || !reflect.DeepEqual(methods, []string{"HEAD", "POST", "HEAD"}) {
			t.Fatal(result, err, methods)
		}
	})
}

func TestSwiftPythonObjectMetadataPhysicalFailuresKeepEachCompletedPhase(t *testing.T) {
	for _, operation := range []string{"get", "set", "delete"} {
		maxPhase := 1
		if operation == "set" {
			maxPhase = 2
		}
		if operation == "delete" {
			maxPhase = 3
		}
		for failed := 1; failed <= maxPhase; failed++ {
			for _, kind := range []string{"status", "close", "read and close", "cancel"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation, failed, kind), func(t *testing.T) {
					calls := 0
					closeCause := errors.New("physical close")
					readCause := errors.New("physical read")
					cancelCause := errors.New("physical cancel")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					bodies := []*objectMetadataBody{}
					client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
						calls++
						body := &objectMetadataBody{Reader: strings.NewReader("physical")}
						bodies = append(bodies, body)
						status := 200
						if calls == failed {
							if kind == "status" || kind == "read and close" {
								status = 404
							}
							if kind == "close" || kind == "read and close" {
								body.closeErr = closeCause
							}
							if kind == "read and close" {
								body.Reader = objectMetadataReader(func([]byte) (int, error) { return 0, readCause })
							}
							if kind == "cancel" {
								body.onClose = func() { cancel(cancelCause) }
							}
						}
						response := objectMetadataWire(status, body)
						response.Header.Set("X-Phase", fmt.Sprint(calls))
						if r.Method == "HEAD" {
							response.Header.Set("X-Object-Meta-Owner", "old")
						}
						return response, nil
					})
					api := objects.New(client)
					var result *objects.FreshMetadataChangeResult
					var head *objects.GetMetadataResult
					var err error
					if operation == "get" {
						head, err = api.HeadMetadata(ctx, objectMetadataContainer, objectMetadataKey)
					}
					if operation == "set" {
						result, err = api.SetMetadataFromFresh(ctx, objectMetadataContainer, objectMetadataKey, map[string]any{"owner": "input"})
					}
					if operation == "delete" {
						result, err = api.DeleteMetadataAndRefresh(ctx, objectMetadataContainer, objectMetadataKey, []string{"owner"})
					}
					if err == nil || calls != failed {
						t.Fatal(result, err, calls)
					}
					for _, body := range bodies {
						if body.closes.Load() != 1 {
							t.Fatal("body Close count", body.closes.Load())
						}
					}
					rejected := kind == "status" || kind == "read and close"
					checkHead := func(value *objects.GetMetadataResult, phase int, want bool) {
						t.Helper()
						if (value != nil) != want {
							t.Fatal("HEAD receipt presence", phase, value, err)
						}
						if value == nil {
							return
						}
						if value.StatusCode != 200 || string(value.Body) != "physical" || value.Header.Get("X-Phase") != fmt.Sprint(phase) {
							t.Fatal("lost actual accepted HEAD", phase, value, err)
						}
						if phase < failed {
							if value.Metadata == nil || value.Metadata.Values["owner"] != "old" {
								t.Fatal("lost prior successful observation", phase, value, err)
							}
						} else if value.Metadata != nil {
							t.Fatal("failed accepted HEAD published metadata", phase, value, err)
						}
					}
					checkAck := func(value *objects.MetadataResponse, phase int, want bool) {
						t.Helper()
						if (value != nil) != want {
							t.Fatal("POST acknowledgement presence", phase, value, err)
						}
						if value != nil && (value.StatusCode != 200 || string(value.Body) != "physical" || value.Header.Get("X-Phase") != fmt.Sprint(phase)) {
							t.Fatal("lost actual accepted POST", phase, value, err)
						}
					}
					if operation == "get" {
						checkHead(head, 1, !rejected)
					} else {
						// A rejected initial setter HEAD returns no change result. Deletion may
						// retain an empty result, but neither may manufacture a phase receipt.
						if result == nil && !(operation == "set" && failed == 1 && rejected) {
							t.Fatal("lost change result", result, err)
						}
						var before, refreshed *objects.GetMetadataResult
						var ack *objects.MetadataResponse
						if result != nil {
							before, ack, refreshed = result.Before, result.Acknowledgement, result.Refreshed
							if result.NoOp {
								t.Fatal("failure cannot claim no-op", result, err)
							}
						}
						checkHead(before, 1, failed > 1 || !rejected)
						checkAck(ack, 2, failed > 2 || failed == 2 && !rejected)
						checkHead(refreshed, 3, failed == 3 && !rejected)
					}
					if rejected {
						expectedBody := "physical"
						if kind == "read and close" {
							expectedBody = ""
						}
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != expectedBody || native.ResponseHeader.Get("X-Phase") != fmt.Sprint(failed) || native.ResponseHeader.Get("X-Trans-Id") != "actual-object" {
							t.Fatal("lost rejected phase proof", err, native)
						}
					} else {
						// Accepted responses keep their actual receipt even when closing the
						// body or cancellation fails. Rejections keep proof in the native error.
						proof := objectMetadataProof(t, err, 200, "physical")
						if proof.Header.Get("X-Phase") != fmt.Sprint(failed) {
							t.Fatal("failed phase proof header", err, proof)
						}
					}
					if (kind == "close" || kind == "read and close") && !errors.Is(err, closeCause) {
						t.Fatal(err)
					}
					if kind == "read and close" && !errors.Is(err, readCause) {
						t.Fatal(err)
					}
					if kind == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestSwiftPythonObjectMetadataOwnedInputsCallbacksConcurrentReuseAndPreflight(t *testing.T) {
	values := map[string]any{"owner": "original", "nested": []any{"before"}}
	var kept *objects.MetadataOpts
	option := func(cfg *objects.MetadataOpts) error {
		values["owner"] = "caller"
		values["nested"].([]any)[0] = "changed"
		cfg.Headers["X-Probe"] = "snapshot"
		kept = cfg
		return nil
	}
	var calls int
	client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		kept.Headers = map[string]string{"X-Probe": "late"}
		if r.Header.Get("X-Probe") != "snapshot" {
			t.Error(r.Header)
		}
		if r.Method == "POST" && (r.Header.Get("X-Object-Meta-Owner") != "original" || r.Header.Get("X-Object-Meta-Nested") != "['before']") {
			t.Error(r.Header)
		}
		return objectMetadataResponse(204, ""), nil
	})
	if _, err := objects.New(client).SetMetadataFromFresh(context.Background(), objectMetadataContainer, objectMetadataKey, values, option); err != nil || calls != 2 {
		t.Fatal(err, calls)
	}
	keys := []string{"owner"}
	calls = 0
	client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		response := objectMetadataResponse(204, "")
		if r.Method == "HEAD" {
			response.Header.Set("X-Object-Meta-Owner", "old")
		}
		if r.Method == "POST" {
			if _, present := r.Header["X-Object-Meta-Owner"]; present {
				t.Error(r.Header)
			}
		}
		return response, nil
	})
	if _, err := objects.New(client).DeleteMetadataAndRefresh(context.Background(), objectMetadataContainer, objectMetadataKey, keys, func(*objects.MetadataOpts) error { keys[0] = "missing"; return nil }); err != nil || calls != 3 {
		t.Fatal(err, calls)
	}
	cloud := testcloud.New(t)
	api := objects.New(cloud.Client("object-store", "/swift/v1/AUTH_selected/"))
	var concurrent atomic.Int32
	headers := map[string]string{"X-Probe": "snapshot"}
	reusable := objects.WithMetadataHeaders(headers)
	headers["X-Probe"] = "caller"
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		concurrent.Add(1)
		if r.Header.Get("X-Probe") != "snapshot" {
			t.Error(r.Header)
		}
		w.WriteHeader(204)
	})
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := api.SetMetadataFromFresh(context.Background(), "assets", "file", map[string]any{"owner": "same"}, reusable); err != nil {
				t.Error(err)
			}
		}()
	}
	wait.Wait()
	if concurrent.Load() != 16 {
		t.Fatal(concurrent.Load())
	}
	calls = 0
	client = objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataResponse(204, ""), nil })
	api = objects.New(client)
	for _, values := range []map[string]any{{"owner": "x", "X-Object-Meta-Owner": "collision"}, {"bad\r\nkey": "value"}, {"custom": func() {}}} {
		if _, err := api.SetMetadataFromFresh(context.Background(), objectMetadataContainer, objectMetadataKey, values); err == nil {
			t.Fatal(values)
		}
	}
	cause := errors.New("precanceled caller")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if _, err := api.HeadMetadata(ctx, objectMetadataContainer, objectMetadataKey); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if _, err := api.DeleteMetadataAndRefresh(context.Background(), objectMetadataContainer, objectMetadataKey, nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal(calls)
	}
}

type swiftPythonObjectMetadataMarshaler func() ([]byte, error)

func (marshal swiftPythonObjectMetadataMarshaler) MarshalJSON() ([]byte, error) { return marshal() }
func TestSwiftPythonObjectMetadataCapturesSourceBeforeCustomJSONMarshaler(t *testing.T) {
	calls := 0
	client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataResponse(204, ""), nil })
	input := swiftPythonObjectMetadataMarshaler(func() ([]byte, error) {
		client.Endpoint = "https://swift.invalid/changed/AUTH_other/"
		return []byte(`"value"`), nil
	})
	_, err := objects.New(client).SetMetadataFromFresh(context.Background(), objectMetadataContainer, objectMetadataKey, map[string]any{"custom": input})
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal("marshal retargeted Source", err, calls)
	}
}
