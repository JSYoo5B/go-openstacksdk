package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestR2IronicACreateBodyHeaderAndExactStatus(t *testing.T) {
	for _, k := range r2ironicAKinds() {
		t.Run(k.name, func(t *testing.T) {
			for _, tc := range []struct {
				status  int
				success bool
			}{{201, true}, {200, false}, {202, false}, {204, false}} {
				cloud := testcloud.New(t)
				rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) {
					if tc.status == 204 {
						w.WriteHeader(204)
						return
					}
					testcloud.JSON(w, tc.status, k.item("new-1", ""))
				})
				id, err := k.create(context.Background(), ironicClient(cloud, "baremetal", "1.32"))
				got := rec.all()
				if len(got) != 1 || got[0].Method != "POST" || got[0].Path != "/v1/"+k.path || got[0].Header.Get(ironicAPIVersionHeader) != "1.32" {
					t.Fatalf("status %d requests=%+v", tc.status, got)
				}
				r2ironicAJSONEqual(t, got[0].Body, k.createBody)
				if tc.success != (err == nil) || tc.success && id != "new-1" {
					t.Fatalf("status %d id=%q err=%v", tc.status, id, err)
				}
				if !tc.success {
					ironicRequireStatus(t, err, tc.status)
				}
			}
		})
	}
}

func TestR2IronicACreateOmitsUnsetAndRejectsReservedFieldsBeforeHTTP(t *testing.T) {
	for _, k := range r2ironicAKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			ironicForbidHTTP(t, cloud)
			c := ironicClient(cloud, "baremetal", "")
			for _, key := range k.reserved {
				if err := k.createCollision(context.Background(), c, key); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("reserved %q: %v", key, err)
				}
			}
		})
	}
	// Unset typed values are omitted entirely and no microversion header is invented.
	cloud := testcloud.New(t)
	k := r2ironicAKinds()[2]
	rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) { testcloud.JSON(w, 201, k.item("x", "")) })
	if err := k.createCollision(context.Background(), ironicClient(cloud, "baremetal", ""), "unlisted"); err != nil {
		t.Fatal(err)
	}
	got := rec.all()
	if len(got) != 1 || got[0].Header.Get(ironicAPIVersionHeader) != "" {
		t.Fatalf("requests=%+v", got)
	}
	r2ironicAJSONEqual(t, got[0].Body, `{"unlisted":1}`)
}

func TestR2IronicAGetFieldsStatusAndEvidence(t *testing.T) {
	for _, k := range r2ironicAKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) {
				switch strings.TrimPrefix(req.Path, "/v1/"+k.path+"/") {
				case "ok":
					testcloud.JSON(w, 200, k.item("ok", "pg-name"))
				case "accepted":
					testcloud.JSON(w, 202, k.item("accepted", ""))
				default:
					testcloud.JSON(w, 404, `{"error_message":"gone"}`)
				}
			})
			c := ironicClient(cloud, "baremetal", "1.8")
			ctx := context.Background()
			if id, err := k.get(ctx, c, "ok", "uuid", "extra"); err != nil || id != "ok" {
				t.Fatalf("id=%q err=%v", id, err)
			}
			got := rec.all()
			if got[0].Method != "GET" || got[0].Path != "/v1/"+k.path+"/ok" || got[0].Query.Get("fields") != "uuid,extra" || got[0].Header.Get(ironicAPIVersionHeader) != "1.8" {
				t.Fatalf("request=%+v", got[0])
			}
			if ok, err := k.rawKeep(ctx, c, "ok"); err != nil || !ok {
				t.Fatalf("typed values, raw body, header and status must survive: ok=%v err=%v", ok, err)
			}
			if _, err := k.get(ctx, c, "accepted"); err == nil {
				t.Fatal("202 must not be accepted")
			}
			if _, err := k.get(ctx, c, "missing"); !r2ironicAIs404(err) {
				t.Fatalf("404 must stay native: %v", err)
			}
			before := len(rec.all())
			for _, bad := range [][]string{{}, {""}, {"a,b"}} {
				if _, err := k.get(ctx, c, "ok", bad...); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("fields %q: %v", bad, err)
				}
			}
			for _, id := range []string{"", "a/b", "a b", ".."} {
				if _, err := k.get(ctx, c, id); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("id %q: %v", id, err)
				}
			}
			if len(rec.all()) != before {
				t.Fatal("invalid input reached HTTP")
			}
		})
	}
}

func TestR2IronicAUpdateSendsExactPatchAndRejectsBadInputBeforeHTTP(t *testing.T) {
	for _, k := range r2ironicAKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			status := 200
			rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) { testcloud.JSON(w, status, k.item("u-1", "")) })
			c := ironicClient(cloud, "baremetal", "1.32")
			ctx := context.Background()
			ops := []r2IronicAOp{{Op: "add", Path: "/extra/flag", Value: false}, {Op: "replace", Path: "/extra", Value: map[string]any{}},
				{Op: "replace", Path: "/description", Value: nil}, {Path: "/extra/old", NoValue: true}}
			if id, err := k.update(ctx, c, "u-1", ops...); err != nil || id != "u-1" {
				t.Fatalf("id=%q err=%v", id, err)
			}
			got := rec.all()
			if len(got) != 1 || got[0].Method != "PATCH" || got[0].Path != "/v1/"+k.path+"/u-1" || got[0].Header.Get(ironicAPIVersionHeader) != "1.32" {
				t.Fatalf("requests=%+v", got)
			}
			r2ironicAJSONEqual(t, got[0].Body, `[{"op":"add","path":"/extra/flag","value":false},{"op":"replace","path":"/extra","value":{}},{"op":"replace","path":"/description","value":null},{"op":"remove","path":"/extra/old"}]`)
			for _, bad := range []int{201, 202, 204} {
				status = bad
				if _, err := k.update(ctx, c, "u-1", ops[:1]...); err == nil {
					t.Fatalf("status %d must be rejected", bad)
				}
			}
			before := len(rec.all())
			if _, err := k.update(ctx, c, "u-1"); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("empty patch: %v", err)
			}
			for _, name := range k.protect {
				if _, err := k.update(ctx, c, "u-1", r2IronicAOp{Op: "replace", Path: "/" + name, Value: "x"}); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("protected %q: %v", name, err)
				}
			}
			for _, op := range []r2IronicAOp{{Op: "add", Path: "extra"}, {Op: "add", Path: "/"}, {Op: "add", Path: ""}} {
				if _, err := k.update(ctx, c, "u-1", op); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("path %q: %v", op.Path, err)
				}
			}
			if _, err := k.update(ctx, c, "u-1", r2IronicAOp{Op: "move", Path: "/extra"}); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("op: %v", err)
			}
			if _, err := k.update(ctx, c, "a/b", ops[:1]...); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("id: %v", err)
			}
			if len(rec.all()) != before {
				t.Fatal("invalid input reached HTTP")
			}
		})
	}
}

func TestR2IronicADeleteAndRemoveMissingPolicy(t *testing.T) {
	for _, k := range r2ironicAKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) {
				switch strings.TrimPrefix(req.Path, "/v1/"+k.path+"/") {
				case "d204":
					w.WriteHeader(204)
				case "d202":
					w.WriteHeader(202)
				case "d200":
					w.WriteHeader(200)
				case "forbidden":
					testcloud.JSON(w, 403, `{}`)
				default:
					testcloud.JSON(w, 404, `{"error_message":"gone"}`)
				}
			})
			c := ironicClient(cloud, "baremetal", "")
			ctx := context.Background()
			if err := k.del(ctx, c, "d204"); err != nil {
				t.Fatal(err)
			}
			ironicRequireStatus(t, k.del(ctx, c, "d202"), 202)
			ironicRequireStatus(t, k.del(ctx, c, "d200"), 200)
			if err := k.del(ctx, c, "missing"); !r2ironicAIs404(err) {
				t.Fatalf("strict delete must keep 404: %v", err)
			}
			rec.mu.Lock()
			rec.requests = nil
			rec.mu.Unlock()
			if err := k.remove(ctx, c, "missing"); err != nil {
				t.Fatalf("Remove ignores a missing ID by default: %v", err)
			}
			if err := k.remove(ctx, c, "missing", resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("WithMissingError: %v", err)
			}
			if err := k.remove(ctx, c, "forbidden"); err == nil || errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("403 must never be ignored: %v", err)
			}
			for _, r := range rec.all() {
				if r.Method != "DELETE" || !strings.HasPrefix(r.Path, "/v1/"+k.path+"/") {
					t.Fatalf("Remove by ID must send DELETE directly: %+v", r)
				}
			}
			if len(rec.all()) != 3 {
				t.Fatalf("each removal is one request and is never replayed: %+v", rec.all())
			}
		})
	}
}

func TestR2IronicAListFiltersPagingAndDetailRoutes(t *testing.T) {
	for _, k := range r2ironicAKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) {
				row := func(id string) string { return k.item(id, "") }
				switch req.Query.Get("marker") {
				case "":
					testcloud.JSON(w, 200, `{"`+k.plural+`":[`+row("a")+`,`+row("b")+`],"next":"`+cloud.Server.URL+req.Path+`?`+req.Query.Encode()+`&marker=b"}`)
				case "b":
					testcloud.JSON(w, 200, `{"`+k.plural+`":[`+row("c")+`],"next":null}`)
				}
			})
			c := ironicClient(cloud, "baremetal", "1.8")
			ctx := context.Background()
			ids, err := k.list(ctx, c, false, 0, "node-9", nil, [][2]string{{"vendor_filter", "x y"}})
			if err != nil || strings.Join(ids, ",") != "a,b,c" {
				t.Fatalf("ids=%v err=%v", ids, err)
			}
			got := rec.all()
			if len(got) != 2 || got[0].Method != "GET" || got[0].Path != "/v1/"+k.path || got[0].Query.Has("detail") || got[0].Query.Get("vendor_filter") != "x y" ||
				got[0].Query.Get("marker") != "" || got[1].Query.Get("marker") != "b" || got[1].Query.Get("vendor_filter") != "x y" || got[0].Header.Get(ironicAPIVersionHeader) != "1.8" {
				t.Fatalf("requests=%+v", got)
			}
			if k.filter != "" && (got[0].Query.Get(k.filter) != "node-9" || got[1].Query.Get(k.filter) != "node-9") {
				t.Fatalf("typed filter must be kept on every page: %+v", got)
			}
			// Fields are a summary-list feature only.
			rec.mu.Lock()
			rec.requests = nil
			rec.mu.Unlock()
			if _, err := k.list(ctx, c, false, 0, "", []string{"uuid"}, nil); err != nil {
				t.Fatal(err)
			}
			if rec.all()[0].Query.Get("fields") != "uuid" {
				t.Fatalf("fields: %+v", rec.all()[0])
			}
			rec.mu.Lock()
			rec.requests = nil
			rec.mu.Unlock()
			if _, err := k.list(ctx, c, true, 0, "", nil, nil); err != nil {
				t.Fatal(err)
			}
			d := rec.all()[0]
			if k.detailPath && (d.Path != "/v1/"+k.path+"/detail" || d.Query.Has("detail")) || !k.detailPath && (d.Path != "/v1/"+k.path || d.Query.Get("detail") != "True") {
				t.Fatalf("detail request=%+v", d)
			}
			before := len(rec.all())
			if _, err := k.list(ctx, c, true, 0, "", []string{"uuid"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("fields with detail: %v", err)
			}
			for _, key := range []string{"limit", "marker", "fields", "sort_key", "sort_dir", "node", "address", "detail"} {
				if _, err := k.list(ctx, c, false, 0, "", nil, [][2]string{{key, "x"}}); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("extension %q: %v", key, err)
				}
			}
			if _, err := k.list(ctx, c, false, -1, "", nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("negative limit: %v", err)
			}
			if len(rec.all()) != before {
				t.Fatal("invalid input reached HTTP")
			}
		})
	}
}

func TestR2IronicAListMarkerFallbackWithLimitAndStatus(t *testing.T) {
	k := r2ironicAKinds()[2]
	cloud := testcloud.New(t)
	status := 200
	rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) {
		if status != 200 {
			testcloud.JSON(w, status, `{"`+k.plural+`":[]}`)
			return
		}
		switch req.Query.Get("marker") {
		case "":
			testcloud.JSON(w, 200, `{"`+k.plural+`":[`+k.item("a", "")+`]}`)
		case "a":
			testcloud.JSON(w, 200, `{"`+k.plural+`":[`+k.item("b", "")+`]}`)
		default:
			testcloud.JSON(w, 200, `{"`+k.plural+`":[]}`)
		}
	})
	c := ironicClient(cloud, "baremetal", "")
	ids, err := k.list(context.Background(), c, false, 1, "", nil, nil)
	if err != nil || strings.Join(ids, ",") != "a,b" {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	got := rec.all()
	if len(got) != 3 || got[1].Query.Get("marker") != "a" || got[2].Query.Get("marker") != "b" || got[1].Query.Get("limit") != "1" {
		t.Fatalf("with an explicit limit and no next link Python asks again with the last uuid as marker: %+v", got)
	}
	status = 202
	if _, err = k.list(context.Background(), c, false, 0, "", nil, nil); err == nil {
		t.Fatal("202 must not be accepted for a list")
	}
}

func TestR2IronicAListRejectsForeignNextLinks(t *testing.T) {
	k := r2ironicAKinds()[0]
	cloud := testcloud.New(t)
	r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) {
		testcloud.JSON(w, 200, `{"`+k.plural+`":[`+k.item("a", "")+`],"next":"http://other.invalid/v1/chassis?marker=a"}`)
	})
	_, err := k.list(context.Background(), ironicClient(cloud, "baremetal", ""), false, 0, "", nil, nil)
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("a continuation must stay on the collection origin: %v", err)
	}
}

func TestR2IronicAFindIdentityMatchesPythonFallback(t *testing.T) {
	for _, k := range r2ironicAKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) {
				id := strings.TrimPrefix(req.Path, "/v1/"+k.path+"/")
				switch {
				case req.Method == "GET" && id == "direct":
					testcloud.JSON(w, 200, k.item("direct", ""))
				case req.Method == "GET" && id == "bad-request":
					testcloud.JSON(w, 400, `{}`)
				case req.Method == "GET" && id == "denied":
					testcloud.JSON(w, 403, `{}`)
				case req.Method == "GET" && id == "broken":
					testcloud.JSON(w, 500, `{}`)
				case req.Method == "GET" && req.Path != "/v1/"+k.path && !strings.HasSuffix(req.Path, "detail"):
					testcloud.JSON(w, 404, `{}`)
				default:
					testcloud.JSON(w, 200, `{"`+k.plural+`":[`+k.item("bad-request", "")+`,`+k.item("denied", "")+`,`+k.item("listed", "listed-name")+`]}`)
				}
			})
			c := ironicClient(cloud, "baremetal", "")
			ctx := context.Background()
			if id, err := k.find(ctx, c, "direct"); err != nil || id != "direct" || len(rec.all()) != 1 {
				t.Fatalf("direct id=%q err=%v requests=%+v", id, err, rec.all())
			}
			for _, identity := range []string{"bad-request", "denied"} {
				before := len(rec.all())
				if id, err := k.find(ctx, c, identity); err != nil || id != identity {
					t.Fatalf("%s fallback id=%q err=%v", identity, id, err)
				}
				if got := rec.all()[before:]; len(got) != 2 || got[0].Method != "GET" || got[1].Method != "GET" || !strings.HasPrefix(got[1].Path, "/v1/"+k.path) || strings.HasSuffix(got[1].Path, identity) {
					t.Fatalf("%s sequence=%+v", identity, got)
				}
			}
			if k.hasNames {
				if id, err := k.find(ctx, c, "listed-name"); err != nil || id != "listed" {
					t.Fatalf("name match id=%q err=%v", id, err)
				}
			}
			listRequests := 0
			for _, r := range rec.all() {
				if r.Path == "/v1/"+k.path+"/detail" || r.Path == "/v1/"+k.path {
					listRequests++
					if k.detailPath && r.Path != "/v1/"+k.path+"/detail" || !k.detailPath && r.Query.Get("detail") != "True" {
						t.Fatalf("the fallback scan uses the detailed list by default: %+v", r)
					}
				}
			}
			if listRequests == 0 {
				t.Fatal("no fallback list request observed")
			}
			if id, err := k.find(ctx, c, "nobody"); err != nil || id != "" {
				t.Fatalf("missing is ignored by default: id=%q err=%v", id, err)
			}
			if _, err := k.find(ctx, c, "nobody", resource.WithIdentityFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("strict missing: %v", err)
			}
			if _, err := k.find(ctx, c, "broken"); err == nil || errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("a 500 is never converted to a lookup miss: %v", err)
			}
		})
	}
}

func TestR2IronicAFindIdentityReportsDuplicatesAndFollowsPages(t *testing.T) {
	k := r2ironicAKinds()[1]
	cloud := testcloud.New(t)
	r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) {
		if req.Method == "GET" && req.Path != "/v1/"+k.path && !strings.HasSuffix(req.Path, "detail") {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		if req.Query.Get("marker") == "" {
			testcloud.JSON(w, 200, `{"portgroups":[`+k.item("same", "")+`],"next":"`+cloud.Server.URL+`/v1/portgroups/detail?marker=same"}`)
			return
		}
		testcloud.JSON(w, 200, `{"portgroups":[`+k.item("other", "same")+`]}`)
	})
	_, err := k.find(context.Background(), ironicClient(cloud, "baremetal", ""), "same")
	var ambiguous *resource.AmbiguousError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("an id match plus a name match on a later page must be ambiguous: %v", err)
	}
}

func TestR2IronicACancellationMakesNoRequestAndKeepsCause(t *testing.T) {
	for _, k := range r2ironicAKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			ironicForbidHTTP(t, cloud)
			c := ironicClient(cloud, "baremetal", "")
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := k.create(canceled, c); !errors.Is(err, context.Canceled) {
				t.Fatalf("create: %v", err)
			}
			if _, err := k.get(canceled, c, "x"); !errors.Is(err, context.Canceled) {
				t.Fatalf("get: %v", err)
			}
			if _, err := k.update(canceled, c, "x", r2IronicAOp{Op: "add", Path: "/extra", Value: 1}); !errors.Is(err, context.Canceled) {
				t.Fatalf("update: %v", err)
			}
			if err := k.del(canceled, c, "x"); !errors.Is(err, context.Canceled) {
				t.Fatalf("delete: %v", err)
			}
			if _, err := k.list(canceled, c, false, 0, "", nil, nil); !errors.Is(err, context.Canceled) {
				t.Fatalf("list: %v", err)
			}
			if _, err := k.find(canceled, c, "x"); !errors.Is(err, context.Canceled) {
				t.Fatalf("find: %v", err)
			}
		})
	}
}

func TestR2IronicADeadlineStaysObservableAndMutationIsNotReplayed(t *testing.T) {
	t.Parallel()
	for _, k := range r2ironicAKinds() {
		cloud, calls := ironicBlockingCloud(t)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_, err := k.create(ctx, ironicClient(cloud, "baremetal", ""))
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
			t.Fatalf("%s: err=%v calls=%d", k.name, err, calls.Load())
		}
	}
}
