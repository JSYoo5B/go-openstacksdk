package image_test

import (
	"bytes"
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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image"
	nativeMembers "gophercloudsdk/image/v2/members"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const memberContractPrefix = "/reverse/members/glance/v2/"
const memberContractBase = "https://glance.invalid" + memberContractPrefix
const memberContractObject = `{"image_id":"response-parent","member_id":"response-member","status":"future","created_at":"literal-date","schema":"https://passive.invalid/schema"}`
const memberContractList = `{"members":[` + memberContractObject + `],"next":"https://passive.invalid/next","schema":42}`

type memberContractTransport func(*http.Request) (*http.Response, error)

func (f memberContractTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	if response != nil && response.Request == nil {
		response.Request = r
	}
	return response, err
}

type memberContractBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *memberContractBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type memberContractReader func([]byte) (int, error)

func (f memberContractReader) Read(p []byte) (int, error) { return f(p) }
func memberContractWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-member"}}, Body: body}
}
func memberContractClient(f memberContractTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "image", Endpoint: memberContractBase}
}

type memberContractOptions struct {
	common []image.ImageMemberOption
	remove []image.RemoveImageMemberOption
	find   []image.FindImageMemberOption
	list   []image.ListImageMembersOption
}
type memberContractResult struct {
	member *image.ImageMember
	ack    *image.ImageMemberAcknowledgement
	rows   []*image.ImageMember
}

func memberContractCall(s *image.Service, ctx context.Context, op string, parent resource.Ref, id string, o memberContractOptions) (*memberContractResult, error) {
	var v *image.ImageMember
	var e error
	switch op {
	case "Add":
		v, e = s.AddImageMember(ctx, parent, id, o.common...)
	case "Get":
		v, e = s.GetImageMember(ctx, parent, id, o.common...)
	case "Update":
		v, e = s.UpdateImageMember(ctx, parent, id, "accepted", o.common...)
	case "Find":
		v, e = s.FindImageMember(ctx, parent, id, o.find...)
	case "Remove":
		a, e := s.RemoveImageMember(ctx, parent, id, o.remove...)
		if a == nil {
			return nil, e
		}
		return &memberContractResult{ack: a}, e
	case "All":
		rows, e := s.AllImageMembers(ctx, parent, o.list...)
		if rows == nil {
			return nil, e
		}
		return &memberContractResult{rows: rows}, e
	case "List":
		rows := make([]*image.ImageMember, 0)
		for row, e := range s.ListImageMembers(ctx, parent, o.list...) {
			if e != nil {
				return nil, e
			}
			rows = append(rows, row)
		}
		return &memberContractResult{rows: rows}, nil
	default:
		panic("unknown member operation")
	}
	if v == nil {
		return nil, e
	}
	return &memberContractResult{member: v}, e
}

var memberContractOperations = []struct {
	name, method, suffix, body, raw string
	status                          int
}{
	{"Add", "POST", "", "{\"member\":\"member\"}", memberContractObject, 200},
	{"Get", "GET", "/member", "", memberContractObject, 200},
	{"Update", "PUT", "/member", "{\"status\":\"accepted\"}", memberContractObject, 200},
	{"Remove", "DELETE", "/member", "", "", 204},
	{"Find", "GET", "/member", "", memberContractObject, 200},
	{"List", "GET", "", "", memberContractList, 200},
	{"All", "GET", "", "", memberContractList, 200},
}

func memberContractProof(t *testing.T, e error, code int, raw []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(e, &proof) || proof.StatusCode != code || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-member" {
		t.Fatalf("response proof: %v %+v", e, proof)
	}
	return proof
}
func memberContractNative404() error {
	return gophercloud.ErrUnexpectedResponseCode{Method: "GET", URL: "not-wire://member", Expected: []int{200}, Actual: 404, Body: []byte("nested404")}
}
func memberContractHeaders() memberContractOptions {
	return memberContractOptions{
		common: []image.ImageMemberOption{image.WithImageMemberHeaders(map[string]string{"X-Option": "owned"}), image.WithImageMemberHeader("X-Final", "yes")},
		remove: []image.RemoveImageMemberOption{image.WithRemoveImageMemberHeaders(map[string]string{"X-Option": "owned"}), image.WithRemoveImageMemberHeader("X-Final", "yes")},
		find:   []image.FindImageMemberOption{image.WithFindImageMemberHeaders(map[string]string{"X-Option": "owned"}), image.WithFindImageMemberHeader("X-Final", "yes")},
		list:   []image.ListImageMembersOption{image.WithListImageMembersHeaders(map[string]string{"X-Option": "owned"}), image.WithListImageMembersHeader("X-Final", "yes")},
	}
}

func TestImageMembersFixedParentRoutesAndBodies(t *testing.T) {
	for _, op := range memberContractOperations {
		t.Run(op.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + memberContractPrefix
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			client.Microversion = "2.10"
			cloud.Provider.SetToken("live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc(memberContractPrefix, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != op.method || r.URL.Path != memberContractPrefix+"images/parent/members"+op.suffix || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "yes" || r.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Error(r.Method, r.URL, r.Header)
				}
				raw, e := io.ReadAll(r.Body)
				if e != nil || string(raw) != op.body {
					t.Error(string(raw), e)
				}
				w.Header().Set("X-Request-Id", "actual-member")
				testcloud.JSON(w, op.status, op.raw)
			})
			v, e := memberContractCall(image.New(client), context.Background(), op.name, resource.ID("parent"), "member", memberContractHeaders())
			if e != nil || v == nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			if v.member != nil && (*v.member.ImageID != "response-parent" || *v.member.MemberID != "response-member" || v.member.StatusCode != 200 || v.member.Header.Get("X-Request-Id") != "actual-member") {
				t.Fatal(v.member)
			}
			if v.ack != nil && (v.ack.ImageID != "parent" || v.ack.MemberID != "member" || v.ack.StatusCode != 204) {
				t.Fatal(v.ack)
			}
			if v.rows != nil && (len(v.rows) != 1 || *v.rows[0].MemberID != "response-member") {
				t.Fatal(v.rows)
			}
		})
	}
	for _, op := range memberContractOperations {
		t.Run(op.name+" unicode identity", func(t *testing.T) {
			var calls atomic.Int32
			client := memberContractClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				suffix := ""
				if op.suffix != "" {
					suffix = "/" + url.PathEscape("회원-7")
				}
				if r.URL.EscapedPath() != memberContractPrefix+"images/"+url.PathEscape("이미지-7")+"/members"+suffix || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				if op.name == "Add" {
					raw, _ := io.ReadAll(r.Body)
					if string(raw) != `{"member":"회원-7"}` {
						t.Error(string(raw))
					}
				}
				return memberContractWire(op.status, io.NopCloser(strings.NewReader(op.raw))), nil
			})
			v, e := memberContractCall(image.New(client), context.Background(), op.name, resource.ID("이미지-7"), "회원-7", memberContractOptions{})
			if e != nil || v == nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	for _, op := range memberContractOperations {
		for _, mode := range []string{"last page", "missing", "ambiguous", "late404", "late read", "unsafe resolved", "cancel", "provider replaced"} {
			t.Run(op.name+" parentName "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("parent lookup cause")
				var requests []string
				var bodies []*memberContractBody
				client := memberContractClient(nil)
				client.MoreHeaders = map[string]string{"X-Source": "captured"}
				client.HTTPClient.Transport = memberContractTransport(func(r *http.Request) (*http.Response, error) {
					requests = append(requests, r.Method+" "+r.URL.String())
					if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" {
						t.Error(r.Header)
					}
					if r.URL.Path != memberContractPrefix+"images" {
						if r.URL.String() != memberContractBase+"images/parent/members"+op.suffix || r.Method != op.method {
							t.Error(r.Method, r.URL)
						}
						return memberContractWire(op.status, io.NopCloser(strings.NewReader(op.raw))), nil
					}
					first := r.URL.Query().Get("marker") == ""
					raw := `{"images":[{"id":"near","name":"needle-suffix"}],"next":"/v2/images?marker=second"}`
					code := 200
					if !first {
						raw = `{"images":[{"id":"parent","name":"needle","status":"killed","protected":true}]}`
					}
					switch mode {
					case "missing":
						raw = `{"images":[]}`
					case "ambiguous":
						raw = `{"images":[{"id":"a","name":"needle"},{"id":"b","name":"needle"}]}`
					case "unsafe resolved":
						raw = `{"images":[{"id":"../escape","name":"needle"}]}`
					case "late404", "late read":
						if first {
							raw = `{"images":[{"id":"parent","name":"needle"}],"next":"/v2/images?marker=second"}`
						} else {
							code = 404
							raw = "actual parent404"
						}
					}
					b := &memberContractBody{Reader: strings.NewReader(raw)}
					if mode == "late read" && !first {
						code = 200
						b.Reader = memberContractReader(func(p []byte) (int, error) { return copy(p, `{"images":`), cause })
					}
					if mode == "cancel" {
						b.onClose = func() { cancel(cause) }
					}
					if mode == "provider replaced" {
						b.onClose = func() { client.ProviderClient = &gophercloud.ProviderClient{} }
					}
					bodies = append(bodies, b)
					return memberContractWire(code, b), nil
				})
				v, e := memberContractCall(image.New(client), ctx, op.name, resource.Name("needle"), "member", memberContractHeaders())
				if mode == "last page" {
					if e != nil || v == nil || len(requests) != 3 {
						t.Fatal(v, e, requests)
					}
				} else {
					if v != nil || e == nil {
						t.Fatal(v, e, requests)
					}
					for _, r := range requests {
						if strings.Contains(r, "/images/") {
							t.Fatal("member request after failed parent lookup", requests)
						}
					}
				}
				if mode == "missing" && !errors.Is(e, resource.ErrNotFound) || mode == "ambiguous" && !errors.Is(e, resource.ErrAmbiguous) || (mode == "unsafe resolved" || mode == "provider replaced") && !errors.Is(e, resource.ErrInvalidOption) || mode == "late404" && !gophercloud.ResponseCodeIs(e, 404) || (mode == "late read" || mode == "cancel") && !errors.Is(e, cause) {
					t.Fatal(e, requests)
				}
				for _, b := range bodies {
					if b.closes.Load() != 1 {
						t.Fatal("parent body close", b.closes.Load())
					}
				}
			})
		}
	}
}

func TestImageMembersCanonicalRawModelsAndFiniteList(t *testing.T) {
	for _, raw := range []string{`{}`, `{"image_id":null,"member_id":null,"status":null,"schema":null,"created_at":null,"updated_at":null}`, `{"image_id":"","member_id":"","status":"","schema":"","created_at":"","updated_at":""}`, `{"image_id":"foreign-parent","member_id":"foreign-member","status":"future-status","schema":"https://passive.invalid/x","created_at":"unparsed 2026 +09:00","updated_at":"not-a-date","links":false,"Member_ID":42,"extension":9007199254740993}`} {
		t.Run("nullable passive "+raw, func(t *testing.T) {
			b := &memberContractBody{Reader: strings.NewReader(raw)}
			wire := memberContractWire(200, b)
			client := memberContractClient(func(*http.Request) (*http.Response, error) { return wire, nil })
			v, e := image.New(client).GetImageMember(context.Background(), resource.ID("parent"), "member")
			if e != nil || v == nil || v.Links != nil || v.StatusCode != 200 || b.closes.Load() != 1 {
				t.Fatal(v, e, b.closes.Load())
			}
			for _, f := range []struct {
				k string
				p *string
			}{{"image_id", v.ImageID}, {"member_id", v.MemberID}, {"status", v.Status}, {"schema", v.Schema}, {"created_at", v.CreatedAt}, {"updated_at", v.UpdatedAt}} {
				token, exists := v.Body[f.k]
				if !exists || string(token) == "null" {
					if f.p != nil {
						t.Fatal(f.k, f.p)
					}
				} else {
					var want string
					if json.Unmarshal(token, &want) != nil || f.p == nil || *f.p != want {
						t.Fatal(f.k, f.p, string(token))
					}
				}
			}
			if strings.Contains(raw, "extension") && string(v.Body["extension"]) != "9007199254740993" {
				t.Fatal(v.Body)
			}
			wire.Header.Set("X-Request-Id", "wire changed")
			if v.Header.Get("X-Request-Id") != "actual-member" {
				t.Fatal("wire header alias", v.Header)
			}
			if v.MemberID != nil {
				before := *v.MemberID
				v.Body["member_id"][1] = '!'
				if *v.MemberID != before {
					t.Fatal("raw and typed strings alias")
				}
			}
		})
	}
	for _, field := range []string{"image_id", "member_id", "status", "schema", "created_at", "updated_at"} {
		for _, token := range []string{"1", "true", "[]", "{}"} {
			t.Run(field+" rejects "+token, func(t *testing.T) {
				raw := []byte(fmt.Sprintf(`{%q:%s}`, field, token))
				b := &memberContractBody{Reader: bytes.NewReader(raw)}
				client := memberContractClient(func(*http.Request) (*http.Response, error) { return memberContractWire(200, b), nil })
				v, e := image.New(client).GetImageMember(context.Background(), resource.ID("parent"), "member")
				memberContractProof(t, e, 200, raw)
				if v != nil || b.closes.Load() != 1 {
					t.Fatal(v, e, b.closes.Load())
				}
			})
		}
	}
	for _, raw := range [][]byte{[]byte(`null`), []byte(`[]`), []byte(`1`), []byte(`{"member_id":`), []byte("{\"unknown\":\"\xff\"}")} {
		t.Run("strict single root "+string(raw), func(t *testing.T) {
			b := &memberContractBody{Reader: bytes.NewReader(raw)}
			client := memberContractClient(func(*http.Request) (*http.Response, error) { return memberContractWire(200, b), nil })
			v, e := image.New(client).GetImageMember(context.Background(), resource.ID("parent"), "member")
			memberContractProof(t, e, 200, raw)
			if v != nil || b.closes.Load() != 1 {
				t.Fatal(v, e, b.closes.Load())
			}
		})
	}
	t.Run("finite local cap break and ownership", func(t *testing.T) {
		raw := `{"members":[{"member_id":"first","extension":9007199254740993},{"member_id":"second"},null],"next":"https://other.invalid/escape","links":false,"schema":{},"Members":[null]}`
		var calls atomic.Int32
		var bodies []*memberContractBody
		client := memberContractClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.String() != memberContractBase+"images/parent/members" || r.URL.RawQuery != "" || r.Body != nil {
				t.Error(r.URL, r.Body)
			}
			b := &memberContractBody{Reader: strings.NewReader(raw)}
			bodies = append(bodies, b)
			wire := memberContractWire(200, b)
			wire.Header.Set("Link", `<https://other.invalid/next>; rel="next"`)
			return wire, nil
		})
		s := image.New(client)
		rows, e := s.AllImageMembers(context.Background(), resource.ID("parent"), image.WithListImageMembersMaxItems(2))
		if e != nil || len(rows) != 2 || calls.Load() != 1 {
			t.Fatal(rows, e, calls.Load())
		}
		rows[0].Header.Set("X-Request-Id", "caller")
		rows[0].Body["member_id"][1] = '!'
		if rows[1].Header.Get("X-Request-Id") != "actual-member" || *rows[0].MemberID != "first" || string(rows[0].Body["extension"]) != "9007199254740993" {
			t.Fatal(rows)
		}
		n := 0
		for row, e := range s.ListImageMembers(context.Background(), resource.ID("parent")) {
			if e != nil || row == nil {
				t.Fatal(row, e)
			}
			n++
			break
		}
		if n != 1 || calls.Load() != 2 {
			t.Fatal(n, calls.Load())
		}
		rows, e = s.AllImageMembers(context.Background(), resource.ID("parent"))
		memberContractProof(t, e, 200, []byte(raw))
		if rows != nil || calls.Load() != 3 {
			t.Fatal(rows, e, calls.Load())
		}
		for _, b := range bodies {
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		}
	})
	for _, raw := range []string{`{}`, `null`, `[]`, `{"members":null}`, `{"members":{}}`, `{"members":true}`, `{"Members":[]}`, `{"members":[null]}`, `{"members":[1]}`, `{"members":[[]]}`, `{"members":[{"status":false}]}`, "{\"members\":[{}],\"unknown\":\"\xff\"}"} {
		t.Run("strict list "+raw, func(t *testing.T) {
			b := &memberContractBody{Reader: strings.NewReader(raw)}
			client := memberContractClient(func(*http.Request) (*http.Response, error) { return memberContractWire(200, b), nil })
			rows, e := image.New(client).AllImageMembers(context.Background(), resource.ID("parent"))
			memberContractProof(t, e, 200, []byte(raw))
			if rows != nil || b.closes.Load() != 1 {
				t.Fatal(rows, e, b.closes.Load())
			}
		})
	}
	t.Run("empty nonnil and repeat lazy iterator", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		client := memberContractClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return memberContractWire(200, io.NopCloser(strings.NewReader(`{"members":[]}`))), nil
		})
		s := image.New(client)
		seq := s.ListImageMembers(context.Background(), resource.ID("parent"), func(*image.ListImageMembersOpts) error { callbacks.Add(1); return nil })
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("eager list")
		}
		for i := 0; i < 2; i++ {
			for row, e := range seq {
				t.Fatal("unexpected empty row", row, e)
			}
		}
		rows, e := s.AllImageMembers(context.Background(), resource.ID("parent"))
		if e != nil || rows == nil || len(rows) != 0 || calls.Load() != 3 || callbacks.Load() != 2 {
			t.Fatal(rows, e, calls.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"cancel", "provider replaced"} {
		t.Run("between consumed rows "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("row consumption cancelled")
			raw := `{"members":[{"member_id":"first"},{"member_id":"second"}]}`
			var calls atomic.Int32
			client := memberContractClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return memberContractWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			n, terminal := 0, 0
			for row, e := range image.New(client).ListImageMembers(ctx, resource.ID("parent")) {
				if e != nil {
					memberContractProof(t, e, 200, []byte(raw))
					terminal++
					if row != nil || mode == "cancel" && (!errors.Is(e, cause) || !errors.Is(e, context.Canceled)) || mode == "provider replaced" && !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal(row, e)
					}
					continue
				}
				n++
				if mode == "cancel" {
					cancel(cause)
				} else {
					client.ProviderClient = &gophercloud.ProviderClient{}
				}
			}
			if n != 1 || terminal != 1 || calls.Load() != 1 {
				t.Fatal(n, terminal, calls.Load())
			}
		})
	}
}

func TestImageMembersOwnedOptionsAndPreflight(t *testing.T) {
	for _, op := range memberContractOperations {
		for _, mode := range []string{"nil context", "cancelled context", "nil service", "nil provider", "bad source", "bad parent", "invalid parent UTF8"} {
			t.Run(op.name+" "+mode, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				s := image.New(client)
				ctx := context.Background()
				parent := resource.Name("needs-lookup")
				cause := errors.New("preflight cancellation")
				switch mode {
				case "nil context":
					ctx = nil
				case "cancelled context":
					c, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = c
				case "nil service":
					s = nil
				case "nil provider":
					client.ProviderClient = nil
				case "bad source":
					client.Endpoint = "https://user:secret@glance.invalid/v2/"
				case "bad parent":
					parent = resource.ID("../escape")
				case "invalid parent UTF8":
					parent = resource.Name(string([]byte{255}))
				}
				o := memberContractOptions{common: []image.ImageMemberOption{func(*image.ImageMemberOpts) error { callbacks.Add(1); return nil }}, remove: []image.RemoveImageMemberOption{func(*image.RemoveImageMemberOpts) error { callbacks.Add(1); return nil }}, find: []image.FindImageMemberOption{func(*image.FindImageMemberOpts) error { callbacks.Add(1); return nil }}, list: []image.ListImageMembersOption{func(*image.ListImageMembersOpts) error { callbacks.Add(1); return nil }}}
				v, e := memberContractCall(s, ctx, op.name, parent, "member", o)
				if v != nil || e == nil || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
				if mode == "cancelled context" {
					if !errors.Is(e, context.Canceled) || !errors.Is(e, cause) {
						t.Fatal(e)
					}
				} else if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			})
		}
	}
	for _, op := range memberContractOperations {
		if op.name == "List" || op.name == "All" {
			continue
		}
		for _, id := range []string{"", ".", "..", "slash/id", "back\\id", "%2F", "query?id", "fragment#id", "white space", "colon:id", "line\nfeed", string([]byte{255})} {
			t.Run(op.name+" invalid member "+id, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				o := memberContractOptions{common: []image.ImageMemberOption{func(*image.ImageMemberOpts) error { callbacks.Add(1); return nil }}, remove: []image.RemoveImageMemberOption{func(*image.RemoveImageMemberOpts) error { callbacks.Add(1); return nil }}, find: []image.FindImageMemberOption{func(*image.FindImageMemberOpts) error { callbacks.Add(1); return nil }}}
				v, e := memberContractCall(image.New(client), context.Background(), op.name, resource.Name("needle"), id, o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
			})
		}
	}
	for _, status := range []string{"", "Accepted", " accepted", "accepted ", "unknown"} {
		t.Run("invalid status "+status, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			v, e := image.New(client).UpdateImageMember(context.Background(), resource.Name("needle"), "member", status, func(*image.ImageMemberOpts) error { callbacks.Add(1); return nil })
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(v, e, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, status := range []string{"pending", "accepted", "rejected"} {
		t.Run("exact required status "+status, func(t *testing.T) {
			client := memberContractClient(func(r *http.Request) (*http.Response, error) {
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != fmt.Sprintf(`{"status":%q}`, status) {
					t.Error(string(raw))
				}
				return memberContractWire(200, io.NopCloser(strings.NewReader(memberContractObject))), nil
			})
			if v, e := image.New(client).UpdateImageMember(context.Background(), resource.ID("parent"), "member", status); e != nil || v == nil {
				t.Fatal(v, e)
			}
		})
	}
	for _, op := range memberContractOperations {
		for _, header := range []string{"x-auth-token", "Content-Type", "Accept", "Content-Length", "Transfer-Encoding", "Host", "OpenStack-API-Version", "Bad Header"} {
			t.Run(op.name+" protected option "+header, func(t *testing.T) {
				var calls atomic.Int32
				client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				o := memberContractOptions{common: []image.ImageMemberOption{image.WithImageMemberHeader(header, "value")}, remove: []image.RemoveImageMemberOption{image.WithRemoveImageMemberHeader(header, "value")}, find: []image.FindImageMemberOption{image.WithFindImageMemberHeader(header, "value")}, list: []image.ListImageMembersOption{image.WithListImageMembersHeader(header, "value")}}
				v, e := memberContractCall(image.New(client), context.Background(), op.name, resource.Name("needle"), "member", o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	for _, op := range memberContractOperations {
		for _, mode := range []string{"nil callback", "callback cause", "source protected", "provider replacement", "callback cancellation"} {
			t.Run(op.name+" "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("caller option cause")
				var calls, callbacks atomic.Int32
				client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				if mode == "source protected" {
					client.MoreHeaders = map[string]string{"x-auth-token": "forged"}
				}
				apply := func() error {
					callbacks.Add(1)
					switch mode {
					case "callback cause":
						return cause
					case "provider replacement":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "callback cancellation":
						cancel(cause)
					}
					return nil
				}
				o := memberContractOptions{common: []image.ImageMemberOption{func(*image.ImageMemberOpts) error { return apply() }}, remove: []image.RemoveImageMemberOption{func(*image.RemoveImageMemberOpts) error { return apply() }}, find: []image.FindImageMemberOption{func(*image.FindImageMemberOpts) error { return apply() }}, list: []image.ListImageMembersOption{func(*image.ListImageMembersOpts) error { return apply() }}}
				if mode == "nil callback" {
					o = memberContractOptions{common: []image.ImageMemberOption{nil}, remove: []image.RemoveImageMemberOption{nil}, find: []image.FindImageMemberOption{nil}, list: []image.ListImageMembersOption{nil}}
				}
				v, e := memberContractCall(image.New(client), ctx, op.name, resource.Name("needle"), "member", o)
				if v != nil || e == nil || calls.Load() != 0 || mode == "source protected" && callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
				if (mode == "callback cause" || mode == "callback cancellation") && !errors.Is(e, cause) || (mode == "nil callback" || mode == "source protected" || mode == "provider replacement") && !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			})
		}
	}
	t.Run("negative local cap before lookup", func(t *testing.T) {
		var calls atomic.Int32
		client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
		rows, e := image.New(client).AllImageMembers(context.Background(), resource.Name("needle"), image.WithListImageMembersMaxItems(-1))
		if rows != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(rows, e, calls.Load())
		}
	})
	for _, op := range memberContractOperations {
		t.Run(op.name+" copied full opts and callback maps", func(t *testing.T) {
			sourceHeaders := map[string]string{"X-Source": "captured"}
			input := map[string]string{"X-Owned": "before"}
			flag := true
			var escaped map[string]string
			var escapedFlag *bool
			var callbacks atomic.Int32
			mutate := func() {
				callbacks.Add(1)
				escaped["X-Owned"] = "escaped mutation"
				if escapedFlag != nil {
					*escapedFlag = false
				}
			}
			o := memberContractOptions{
				common: []image.ImageMemberOption{image.WithImageMemberHeader("X-Dropped", "gone"), image.WithImageMemberOpts(image.ImageMemberOpts{Headers: input}), func(c *image.ImageMemberOpts) error { escaped = c.Headers; return nil }, func(c *image.ImageMemberOpts) error { mutate(); return nil }},
				remove: []image.RemoveImageMemberOption{image.WithRemoveImageMemberHeader("X-Dropped", "gone"), image.WithRemoveImageMemberOpts(image.RemoveImageMemberOpts{Headers: input, IgnoreMissing: &flag}), func(c *image.RemoveImageMemberOpts) error {
					escaped = c.Headers
					escapedFlag = c.IgnoreMissing
					return nil
				}, func(c *image.RemoveImageMemberOpts) error { mutate(); return nil }},
				find: []image.FindImageMemberOption{image.WithFindImageMemberHeader("X-Dropped", "gone"), image.WithFindImageMemberOpts(image.FindImageMemberOpts{Headers: input, IgnoreMissing: &flag}), func(c *image.FindImageMemberOpts) error {
					escaped = c.Headers
					escapedFlag = c.IgnoreMissing
					return nil
				}, func(c *image.FindImageMemberOpts) error { mutate(); return nil }},
				list: []image.ListImageMembersOption{image.WithListImageMembersHeader("X-Dropped", "gone"), image.WithListImageMembersOpts(image.ListImageMembersOpts{Headers: input, MaxItems: 1}), func(c *image.ListImageMembersOpts) error { escaped = c.Headers; return nil }, func(c *image.ListImageMembersOpts) error { mutate(); return nil }},
			}
			input["X-Owned"] = "caller mutation"
			flag = false
			client := memberContractClient(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("X-Owned") != "before" || r.Header.Get("X-Dropped") != "" || r.Header.Get("X-Source") != "captured" {
					t.Error(r.Header)
				}
				code, raw := op.status, op.raw
				if op.name == "Remove" || op.name == "Find" {
					code = 404
					raw = "physical404"
				}
				return memberContractWire(code, io.NopCloser(strings.NewReader(raw))), nil
			})
			client.MoreHeaders = sourceHeaders
			v, e := memberContractCall(image.New(client), context.Background(), op.name, resource.ID("parent"), "member", o)
			missing := op.name == "Remove" || op.name == "Find"
			if e != nil || !missing && v == nil || missing && v != nil || callbacks.Load() != 1 || sourceHeaders["X-Source"] != "captured" {
				t.Fatal(v, e, callbacks.Load(), sourceHeaders)
			}
		})
	}
	t.Run("captured route headers and live token", func(t *testing.T) {
		client := memberContractClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "before"}
		var calls atomic.Int32
		client.HTTPClient.Transport = memberContractTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.String() != memberContractBase+"images/parent/members/member" || r.Header.Get("X-Source") != "before" || r.Header.Get("X-Auth-Token") != "live" {
				t.Error(r.URL, r.Header)
			}
			return memberContractWire(200, io.NopCloser(strings.NewReader(memberContractObject))), nil
		})
		v, e := image.New(client).GetImageMember(context.Background(), resource.ID("parent"), "member", func(*image.ImageMemberOpts) error {
			client.Endpoint = "https://glance.invalid/changed/v2/"
			client.ResourceBase = "https://glance.invalid/replacement/v2/"
			client.MoreHeaders["X-Source"] = "later"
			client.SetToken("live")
			return nil
		})
		if e != nil || v == nil || calls.Load() != 1 || client.MoreHeaders["X-Source"] != "later" {
			t.Fatal(v, e, calls.Load())
		}
	})
	t.Run("lazy option slice and parallel iterations", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		input := map[string]string{"X-Option": "snapshot"}
		client := memberContractClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Option") != "snapshot" || r.URL.RawQuery != "" {
				t.Error(r.Header, r.URL)
			}
			return memberContractWire(200, io.NopCloser(strings.NewReader(`{"members":[{},null]}`))), nil
		})
		opts := []image.ListImageMembersOption{image.WithListImageMembersOpts(image.ListImageMembersOpts{Headers: input, MaxItems: 1}), func(*image.ListImageMembersOpts) error { callbacks.Add(1); return nil }}
		seq := image.New(client).ListImageMembers(context.Background(), resource.ID("parent"), opts...)
		opts[0] = nil
		input["X-Option"] = "later"
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("eager work")
		}
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				n := 0
				for v, e := range seq {
					if v == nil || e != nil {
						t.Error(v, e)
					}
					n++
				}
				if n != 1 {
					t.Error(n)
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 4 || callbacks.Load() != 4 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
}

func TestImageMembersAcceptedEvidenceAndMutationNoReplay(t *testing.T) {
	for _, op := range memberContractOperations {
		for _, mode := range []string{"read", "Close", "read Close cancel", "cancel only"} {
			t.Run(op.name+" accepted "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("member Read"), errors.New("member Close"), errors.New("member cancellation")
				raw := []byte(op.raw)
				if op.name == "Remove" {
					raw = []byte{0, 255, 'x'}
				}
				b := &memberContractBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "read") {
					b.Reader = memberContractReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					b.closeErr = closeCause
				}
				wire := memberContractWire(op.status, b)
				b.onClose = func() {
					wire.Header.Set("X-Request-Id", "mutated during Close")
					if strings.Contains(mode, "cancel") {
						cancel(cancelCause)
					}
				}
				var calls, retries atomic.Int32
				client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("accepted replay")
				}
				v, e := memberContractCall(image.New(client), ctx, op.name, resource.ID("parent"), "member", memberContractOptions{})
				proof := memberContractProof(t, e, op.status, raw)
				if calls.Load() != 1 || retries.Load() != 0 || b.closes.Load() != 1 || op.name == "Remove" && v == nil || op.name != "Remove" && v != nil {
					t.Fatal(v, e, calls.Load(), retries.Load(), b.closes.Load())
				}
				if strings.Contains(mode, "read") && !errors.Is(e, readCause) || strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) {
					t.Fatal("cause lost", e)
				}
				if v != nil {
					a := v.ack
					if a.ImageID != "parent" || a.MemberID != "member" || a.StatusCode != 204 || !bytes.Equal(a.Body, raw) || a.Header.Get("X-Request-Id") != "actual-member" {
						t.Fatal(a)
					}
					a.Body[0] = '!'
					a.Header.Set("X-Request-Id", "caller")
					if proof.Body[0] != raw[0] || proof.Header.Get("X-Request-Id") != "actual-member" {
						t.Fatal("ack and proof alias", a, proof)
					}
				}
			})
		}
	}
	for _, op := range memberContractOperations {
		for _, code := range []int{200, 201, 202, 204, 206, 400, 403, 404, 409, 500} {
			if code == op.status || (op.name == "Find" || op.name == "Remove") && code == 404 {
				continue
			}
			t.Run(fmt.Sprintf("%s unexpected%d", op.name, code), func(t *testing.T) {
				b := &memberContractBody{Reader: strings.NewReader("native status bytes")}
				var calls atomic.Int32
				client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return memberContractWire(code, b), nil })
				v, e := memberContractCall(image.New(client), context.Background(), op.name, resource.ID("parent"), "member", memberContractOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				var owned *resource.ResponseError
				expected := []int{op.status}
				if op.name == "Find" || op.name == "Remove" {
					expected = append(expected, 404)
				}
				if v != nil || !errors.As(e, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, expected) || native.Method != op.method || native.URL != memberContractBase+"images/parent/members"+op.suffix || string(native.Body) != "native status bytes" || native.ResponseHeader.Get("X-Request-Id") != "actual-member" || errors.As(e, &owned) || calls.Load() != 1 || b.closes.Load() != 1 {
					t.Fatal(v, e, native, calls.Load(), b.closes.Load())
				}
			})
		}
	}
	for _, op := range []string{"Find", "Remove"} {
		for _, mode := range []string{"default", "explicit true", "nil replacement", "explicit false", "false native retry"} {
			t.Run(op+" physical404 "+mode, func(t *testing.T) {
				var calls, retries atomic.Int32
				var bodies []*memberContractBody
				client := memberContractClient(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					code := 404
					raw := "physical404 bytes"
					if mode == "false native retry" && n == 2 {
						if op == "Find" {
							code = 200
							raw = memberContractObject
						} else {
							code = 204
						}
					}
					if r.URL.String() != memberContractBase+"images/parent/members/member" || r.Body != nil {
						t.Error(r.URL, r.Body)
					}
					b := &memberContractBody{Reader: strings.NewReader(raw)}
					bodies = append(bodies, b)
					return memberContractWire(code, b), nil
				})
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, e error, _ uint) error {
					retries.Add(1)
					if mode == "false native retry" && gophercloud.ResponseCodeIs(e, 404) {
						return nil
					}
					return e
				}
				o := memberContractOptions{}
				switch mode {
				case "explicit true":
					o.find = []image.FindImageMemberOption{image.WithFindImageMemberIgnoreMissing(true)}
					o.remove = []image.RemoveImageMemberOption{image.WithRemoveImageMemberIgnoreMissing(true)}
				case "nil replacement":
					o.find = []image.FindImageMemberOption{image.WithFindImageMemberIgnoreMissing(false), image.WithFindImageMemberOpts(image.FindImageMemberOpts{})}
					o.remove = []image.RemoveImageMemberOption{image.WithRemoveImageMemberIgnoreMissing(false), image.WithRemoveImageMemberOpts(image.RemoveImageMemberOpts{})}
				case "explicit false", "false native retry":
					o.find = []image.FindImageMemberOption{image.WithFindImageMemberIgnoreMissing(false)}
					o.remove = []image.RemoveImageMemberOption{image.WithRemoveImageMemberIgnoreMissing(false)}
				}
				v, e := memberContractCall(image.New(client), context.Background(), op, resource.ID("parent"), "member", o)
				switch mode {
				case "explicit false":
					var native gophercloud.ErrUnexpectedResponseCode
					var owned *resource.ResponseError
					if v != nil || !errors.As(e, &native) || native.Actual != 404 || !reflect.DeepEqual(native.Expected, []int{map[string]int{"Find": 200, "Remove": 204}[op]}) || errors.As(e, &owned) || calls.Load() != 1 || retries.Load() != 1 {
						t.Fatal(v, e, native, calls.Load(), retries.Load())
					}
				case "false native retry":
					if e != nil || v == nil || calls.Load() != 2 || retries.Load() != 1 {
						t.Fatal(v, e, calls.Load(), retries.Load())
					}
				default:
					if v != nil || e != nil || calls.Load() != 1 || retries.Load() != 0 {
						t.Fatal(v, e, calls.Load(), retries.Load())
					}
				}
				for _, b := range bodies {
					if b.closes.Load() != 1 {
						t.Fatal(b.closes.Load())
					}
				}
			})
		}
	}
	for _, op := range []string{"Find", "Remove"} {
		for _, mode := range []string{"read", "Close", "cancel", "read Close cancel"} {
			t.Run(op+" owned404 failure "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("404 Read"), errors.New("404 Close"), errors.New("404 cause")
				raw := []byte("physical404 partial bytes")
				b := &memberContractBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "read") {
					b.Reader = memberContractReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					b.closeErr = closeCause
				}
				if strings.Contains(mode, "cancel") {
					b.onClose = func() { cancel(cancelCause) }
				}
				var calls, retries atomic.Int32
				client := memberContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return memberContractWire(404, b), nil })
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("unexpected retry")
				}
				v, e := memberContractCall(image.New(client), ctx, op, resource.ID("parent"), "member", memberContractOptions{})
				memberContractProof(t, e, 404, raw)
				if v != nil || calls.Load() != 1 || retries.Load() != 0 || b.closes.Load() != 1 || strings.Contains(mode, "read") && !errors.Is(e, readCause) || strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled)) {
					t.Fatal(v, e, calls.Load(), retries.Load(), b.closes.Load())
				}
			})
		}
	}
	for _, op := range []string{"Find", "Remove"} {
		for _, mode := range []string{"transport nested404", "callback nested404", "reauth nested404"} {
			t.Run(op+" "+mode, func(t *testing.T) {
				cause := errors.New("nested policy cause")
				nested := memberContractNative404()
				var calls, hooks atomic.Int32
				b := &memberContractBody{Reader: strings.NewReader("native failure")}
				client := memberContractClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					if mode == "transport nested404" {
						return nil, errors.Join(cause, nested)
					}
					code := 503
					if mode == "reauth nested404" {
						code = 401
					}
					return memberContractWire(code, b), nil
				})
				if mode == "callback nested404" {
					client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						hooks.Add(1)
						return errors.Join(cause, nested)
					}
				}
				if mode == "reauth nested404" {
					client.ReauthFunc = func(context.Context) error { hooks.Add(1); return errors.Join(cause, nested) }
				}
				v, e := memberContractCall(image.New(client), context.Background(), op, resource.ID("parent"), "member", memberContractOptions{})
				if v != nil || e == nil || calls.Load() != 1 {
					t.Fatal(v, e, calls.Load())
				}
				if mode == "reauth nested404" {
					var native *gophercloud.ErrUnableToReauthenticate
					if !errors.As(e, &native) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) || !errors.Is(native.ErrReauth, cause) || !gophercloud.ResponseCodeIs(native.ErrReauth, 404) || hooks.Load() != 1 {
						t.Fatal(e, native, hooks.Load())
					}
				} else if !errors.Is(e, cause) || mode == "callback nested404" && (!gophercloud.ResponseCodeIs(e, 503) || hooks.Load() != 1) || mode == "transport nested404" && !gophercloud.ResponseCodeIs(e, 404) {
					t.Fatal(e, hooks.Load())
				}
				if mode != "transport nested404" && b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			})
		}
	}
}

func TestImageMembersNativeHooksAndCompatibility(t *testing.T) {
	t.Run("configured prebody hooks live auth and original provider", func(t *testing.T) {
		var calls, reauth, backoff, retries atomic.Int32
		var bodies []*memberContractBody
		transportCause := errors.New("prebody transport")
		client := memberContractClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); client.SetToken("reauth"); return nil }
		client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff.Add(1)
			client.SetToken("backoff")
			return nil
		}
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
			retries.Add(1)
			if !o.KeepResponseBody || o.JSONResponse != nil || o.RawBody != nil {
				t.Error(o)
			}
			encoded, _ := json.Marshal(o.JSONBody)
			if string(encoded) != `{"member":"member"}` {
				t.Error(string(encoded))
			}
			if gophercloud.ResponseCodeIs(e, 503) {
				client.SetToken("retry503")
				return nil
			}
			if errors.Is(e, transportCause) {
				client.SetToken("retrytransport")
				return nil
			}
			return e
		}
		originalProvider, originalRetry := client.ProviderClient, reflect.ValueOf(client.RetryFunc).Pointer()
		client.HTTPClient.Transport = memberContractTransport(func(r *http.Request) (*http.Response, error) {
			i := int(calls.Add(1)) - 1
			codes := []int{401, 429, 503, 0, 200}
			tokens := []string{"initial", "reauth", "backoff", "retry503", "retrytransport"}
			if i >= len(codes) {
				return nil, errors.New("replay")
			}
			raw, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || r.URL.String() != memberContractBase+"images/parent/members" || string(raw) != `{"member":"member"}` || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[i] {
				t.Error(r.Method, r.URL, string(raw), r.Header)
			}
			if codes[i] == 0 {
				return nil, transportCause
			}
			b := &memberContractBody{Reader: strings.NewReader(memberContractObject)}
			bodies = append(bodies, b)
			return memberContractWire(codes[i], b), nil
		})
		v, e := image.New(client).AddImageMember(context.Background(), resource.ID("parent"), "member")
		if e != nil || v == nil || calls.Load() != 5 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 2 || client.ProviderClient != originalProvider || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
			t.Fatal(v, e, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
		}
		for _, b := range bodies {
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		}
	})
	for _, change := range []string{"in-place RawMessage", "changed JSON", "JSON nil", "JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSONBody"} {
		t.Run("Add retry guard "+change, func(t *testing.T) {
			callbackCause := errors.New("callback cause")
			var calls, retries, borrowedReads atomic.Int32
			b := &memberContractBody{Reader: strings.NewReader("original503")}
			borrowed := &memberContractBody{Reader: memberContractReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			client := memberContractClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if r.Method != "POST" || string(raw) != `{"member":"member"}` {
					t.Error(r.Method, string(raw))
				}
				return memberContractWire(503, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					t.Error(e)
				}
				switch change {
				case "in-place RawMessage":
					raw, ok := o.JSONBody.(json.RawMessage)
					if !ok {
						t.Error("missing raw request body", o.JSONBody)
					} else {
						raw[len(raw)-3] = 'z'
					}
				case "changed JSON":
					o.JSONBody = map[string]string{"member": "other"}
				case "JSON nil":
					o.JSONBody = nil
				case "JSON null":
					o.JSONBody = json.RawMessage("null")
				case "KeepResponseBody":
					o.KeepResponseBody = false
				case "JSONResponse":
					o.JSONResponse = new(any)
				case "RawBody":
					o.RawBody = borrowed
				case "unsupported JSONBody":
					o.JSONBody = make(chan int)
				}
				return callbackCause
			}
			v, e := image.New(client).AddImageMember(context.Background(), resource.ID("parent"), "member")
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, callbackCause) || !errors.As(e, &native) || native.Actual != 503 || string(native.Body) != "original503" || calls.Load() != 1 || retries.Load() != 1 || b.closes.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 {
				t.Fatal(v, e, native, calls.Load(), retries.Load(), b.closes.Load())
			}
			if change == "unsupported JSONBody" {
				var encoding *json.UnsupportedTypeError
				if !errors.As(e, &encoding) {
					t.Fatal("encoding cause lost", e)
				}
			}
		})
	}
	t.Run("bodyless null differs from absent", func(t *testing.T) {
		var calls atomic.Int32
		b := &memberContractBody{Reader: strings.NewReader("original503")}
		client := memberContractClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Body != nil {
				t.Error("request body", r.Body)
			}
			return memberContractWire(503, b), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = json.RawMessage("null")
			return nil
		}
		v, e := image.New(client).RemoveImageMember(context.Background(), resource.ID("parent"), "member")
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 1 || b.closes.Load() != 1 {
			t.Fatal(v, e, calls.Load())
		}
	})
	for _, op := range []string{"Add", "Update"} {
		t.Run(op+" same serialized replacement", func(t *testing.T) {
			var calls, retries atomic.Int32
			var bodies []*memberContractBody
			want := `{"member":"member"}`
			if op == "Update" {
				want = `{"status":"accepted"}`
			}
			client := memberContractClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != want {
					t.Error(string(raw), want)
				}
				code := 503
				reply := "original503"
				if n == 2 {
					code = 200
					reply = memberContractObject
				}
				b := &memberContractBody{Reader: strings.NewReader(reply)}
				bodies = append(bodies, b)
				return memberContractWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				retries.Add(1)
				if op == "Add" {
					o.JSONBody = map[string]string{"member": "member"}
				} else {
					o.JSONBody = map[string]string{"status": "accepted"}
				}
				return nil
			}
			v, e := memberContractCall(image.New(client), context.Background(), op, resource.ID("parent"), "member", memberContractOptions{})
			if v == nil || e != nil || calls.Load() != 2 || retries.Load() != 1 {
				t.Fatal(v, e, calls.Load(), retries.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, op := range []string{"Add", "Remove", "Find", "All"} {
		t.Run(op+" expanded codes cannot accept actual status", func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("unexpected Read"), errors.New("unexpected Close"), errors.New("unexpected cancel")
			var calls, retries atomic.Int32
			var bodies []*memberContractBody
			client := memberContractClient(func(*http.Request) (*http.Response, error) {
				n := calls.Add(1)
				code, raw := 503, "original503"
				if n == 2 {
					code = 202
					raw = "unexpected accepted bytes"
				}
				b := &memberContractBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.Reader = memberContractReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return memberContractWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					return e
				}
				o.OkCodes = []int{202}
				return nil
			}
			v, e := memberContractCall(image.New(client), ctx, op, resource.ID("parent"), "member", memberContractOptions{})
			var native gophercloud.ErrUnexpectedResponseCode
			var owned *resource.ResponseError
			expected := []int{200}
			if op == "Find" {
				expected = append(expected, 404)
			}
			if op == "Remove" {
				expected = []int{204, 404}
			}
			if v != nil || !errors.As(e, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, expected) || string(native.Body) != "unexpected accepted bytes" || native.ResponseHeader.Get("X-Request-Id") != "actual-member" || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled) || errors.As(e, &owned) || calls.Load() != 2 || retries.Load() != 1 {
				t.Fatal(v, e, native, calls.Load(), retries.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, redirect := range []string{"same target", "foreign origin", "changed path", "changed query", "changed method"} {
		t.Run("configured redirect "+redirect, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*memberContractBody
			client := memberContractClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				b := &memberContractBody{Reader: strings.NewReader(memberContractObject)}
				bodies = append(bodies, b)
				if n == 2 {
					raw, _ := io.ReadAll(r.Body)
					if r.Method != "POST" || string(raw) != `{"member":"member"}` {
						t.Error(r.Method, string(raw))
					}
					return memberContractWire(200, b), nil
				}
				code, target := 307, memberContractBase+"images/parent/members"
				switch redirect {
				case "foreign origin":
					target = "https://foreign.invalid/member"
				case "changed path":
					target = memberContractBase + "images/other/members"
				case "changed query":
					target += "?q=x"
				case "changed method":
					code = 303
				}
				wire := memberContractWire(code, b)
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			v, e := image.New(client).AddImageMember(context.Background(), resource.ID("parent"), "member")
			if redirect == "same target" {
				if e != nil || v == nil || calls.Load() != 2 || redirects.Load() != 1 {
					t.Fatal(v, e, calls.Load(), redirects.Load())
				}
			} else if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load(), redirects.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, op := range []string{"Create", "Get", "Update", "Delete", "List", "Scope"} {
		t.Run("existing native "+op, func(t *testing.T) {
			var calls atomic.Int32
			var bodies []*memberContractBody
			raw := `{"member_id":"member","image_id":"parent","status":"accepted","created_at":"2026-10-03T00:00:00Z","updated_at":"2026-10-03T00:00:00Z"}`
			client := memberContractClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				code, payload := 200, raw
				if r.Method == "DELETE" {
					code = 204
					payload = ""
				}
				if op == "List" {
					payload = `{"members":[` + raw + `]}`
				}
				if r.URL.Path != memberContractPrefix+"images/parent/members" && r.URL.Path != memberContractPrefix+"images/parent/members/member" {
					t.Error(r.URL)
				}
				b := &memberContractBody{Reader: strings.NewReader(payload)}
				bodies = append(bodies, b)
				return memberContractWire(code, b), nil
			})
			api := nativeMembers.New(client)
			ctx := context.Background()
			var e error
			switch op {
			case "Create":
				v, err := api.Create(ctx, "parent", "member")
				e = err
				if v == nil || v.MemberID != "member" {
					t.Error(v)
				}
			case "Get":
				v, err := image.New(client).API.Members.Get(ctx, "parent", "member")
				e = err
				if v == nil || v.CreatedAt.IsZero() {
					t.Error(v)
				}
			case "Update":
				v, err := api.Update(ctx, "parent", "member", nativeMembers.UpdateOpts{Status: "accepted"})
				e = err
				if v == nil || v.Status != "accepted" {
					t.Error(v)
				}
			case "Delete":
				e = api.Delete(ctx, "parent", "member")
			case "List":
				n := 0
				for v, err := range api.List(ctx, "parent") {
					if err != nil {
						e = err
					}
					if v == nil {
						t.Error(v)
					}
					n++
				}
				if n != 1 {
					t.Error(n)
				}
			case "Scope":
				scope, err := api.InImage(ctx, resource.ID("parent"))
				if err != nil {
					t.Fatal(err)
				}
				if calls.Load() != 0 {
					t.Fatal("scope eager HTTP")
				}
				v, err := scope.Get(ctx, "member")
				e = err
				if v == nil || v.MemberID != "member" {
					t.Error(v)
				}
				if _, err = scope.ResolveID(ctx, resource.Name("member")); !errors.Is(err, resource.ErrUnsupported) {
					t.Error("invented member Name support", err)
				}
			}
			if e != nil || calls.Load() != 1 {
				t.Fatal(e, calls.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	t.Run("literal timestamps differ from native time decoder", func(t *testing.T) {
		var calls atomic.Int32
		client := memberContractClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return memberContractWire(200, io.NopCloser(strings.NewReader(`{"created_at":"unparsed","member_id":"member"}`))), nil
		})
		v, e := image.New(client).GetImageMember(context.Background(), resource.ID("parent"), "member")
		if e != nil || v == nil || v.CreatedAt == nil || *v.CreatedAt != "unparsed" {
			t.Fatal(v, e)
		}
		if _, e = nativeMembers.New(client).Get(context.Background(), "parent", "member"); e == nil || calls.Load() != 2 {
			t.Fatal("native time decode boundary", e, calls.Load())
		}
	})
}
