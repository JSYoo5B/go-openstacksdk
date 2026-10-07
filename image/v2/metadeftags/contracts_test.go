package metadeftags_test

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

	tags "github.com/JSYoo5B/gophercloudsdk/image/v2/metadeftags"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const tagPrefix = "/reverse/tag/glance/v2/"
const tagBase = "https://glance.invalid" + tagPrefix
const tagParent = "OS::Compute::Libvirt"
const tagName = "CPU Limits"
const tagCollection = "metadefs/namespaces/" + tagParent + "/tags"
const tagJSON = `{"name":"passive foreign name","created_at":"literal-date","updated_at":"another literal","links":42,"self":"https://passive.invalid","extension":9007199254740993}`
const tagListJSON = `{"tags":[` + tagJSON + `],"next":"https://passive.invalid","schema":false}`

func tagCallbacks(counter *atomic.Int32, cause error) tagOptions {
	return tagOptions{
		create:    []tags.CreateOption{func(*tags.CreateOpts) error { counter.Add(1); return cause }},
		get:       []tags.GetOption{func(*tags.GetOpts) error { counter.Add(1); return cause }},
		update:    []tags.UpdateOption{func(*tags.UpdateOpts) error { counter.Add(1); return cause }},
		deletion:  []tags.DeleteOption{func(*tags.DeleteOpts) error { counter.Add(1); return cause }},
		deleteAll: []tags.DeleteAllOption{func(*tags.DeleteAllOpts) error { counter.Add(1); return cause }},
		set:       []tags.SetOption{func(*tags.SetOpts) error { counter.Add(1); return cause }},
		list:      []tags.ListOption{func(*tags.ListOpts) error { counter.Add(1); return cause }},
	}
}
func tagHeaderOption(key, value string) tagOptions {
	return tagOptions{create: []tags.CreateOption{tags.WithCreateHeader(key, value)}, get: []tags.GetOption{tags.WithGetHeader(key, value)}, update: []tags.UpdateOption{tags.WithUpdateHeader(key, value)}, deletion: []tags.DeleteOption{tags.WithDeleteHeader(key, value)}, deleteAll: []tags.DeleteAllOption{tags.WithDeleteAllHeader(key, value)}, set: []tags.SetOption{tags.WithSetHeader(key, value)}, list: []tags.ListOption{tags.WithListHeader(key, value)}}
}

func TestMetadefTagsOwnedPreparationAndSource(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "a/b", "a\\b", "a%b", "a?b", "a#b", "a\x00b", "a\x7fb", "\xff", strings.Repeat("界", 81)} {
		t.Run("unsafe parent "+bad, func(t *testing.T) {
			var calls atomic.Int32
			client := tagClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			v, e := tags.New(client).InNamespace(context.Background(), bad)
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, e, calls.Load())
			}
		})
		for _, op := range []string{"Create", "Get", "Update", "Delete"} {
			t.Run(op+" unsafe child "+bad, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := tagClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				v, e := tagCall(tagScope(t, client, tagParent), context.Background(), op, bad, tagCallbacks(&callbacks, nil))
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
			})
		}
	}
	for _, op := range tagOperations {
		for _, mode := range []string{"nil context", "canceled", "nil scope", "provider", "type", "endpoint", "base", "microversion", "source header"} {
			t.Run(op.name+" preflight "+mode, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := tagClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				s := tagScope(t, client, tagParent)
				ctx := context.Background()
				want := resource.ErrInvalidOption
				switch mode {
				case "nil context":
					ctx = nil
				case "canceled":
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
					want = context.Canceled
				case "nil scope":
					s = nil
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "type":
					client.Type = "compute"
				case "endpoint":
					client.Endpoint = "https://other.invalid/v2/"
				case "base":
					client.ResourceBase = "https://glance.invalid/other/v2/"
				case "microversion":
					client.Microversion = "2.3"
				case "source header":
					client.MoreHeaders = map[string]string{"X-Auth-Token": "forbidden"}
				}
				v, e := tagCall(s, ctx, op.name, tagName, tagCallbacks(&callbacks, nil))
				if v != nil || !errors.Is(e, want) || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
			})
		}
	}
	for _, source := range []*gophercloud.ServiceClient{nil, {}, {Type: "image"}, {Type: "compute", ProviderClient: &gophercloud.ProviderClient{}, Endpoint: tagBase}, {Type: "image", ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "https://glance.invalid/v2/?x=1"}} {
		t.Run(fmt.Sprintf("initial source %p", source), func(t *testing.T) {
			v, e := tags.New(source).InNamespace(context.Background(), tagParent)
			if v != nil || e == nil {
				t.Fatal(v, e)
			}
		})
	}
	for _, op := range tagOperations {
		for _, header := range []string{"X-Auth-Token", "Authorization", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "Accept", "Content-Type", "OpenStack-API-Version", "X-OpenStack-Glance-Api-Version", "x-openstack-append", "Bad Header"} {
			t.Run(op.name+" protected "+header, func(t *testing.T) {
				var calls atomic.Int32
				client := tagClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				v, e := tagCall(tagScope(t, client, tagParent), context.Background(), op.name, tagName, tagHeaderOption(header, "caller"))
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	for _, header := range []map[string]string{{"X-OpenStack-Append": "False"}, {"X-Alias": "a", "x-alias": "b"}, {"X-Ordinary": "line\nbreak"}} {
		t.Run(fmt.Sprint(header), func(t *testing.T) {
			client := tagClient(func(*http.Request) (*http.Response, error) { t.Error("HTTP"); return nil, errors.New("unexpected") })
			client.MoreHeaders = header
			v, e := tags.New(client).InNamespace(context.Background(), tagParent)
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(v, e)
			}
		})
	}
	t.Run("options invalid after callbacks and replacement", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		client := tagClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return tagWire(200, &tagBody{Reader: strings.NewReader(tagJSON)}), nil
		})
		s := tagScope(t, client, tagParent)
		for _, o := range []tags.ListOption{nil, tags.WithListLimit(-1), tags.WithListMaxItems(-1), tags.WithListMarker("line\n"), tags.WithListSortKey("\xff"), tags.WithListSortDir(""), tags.WithListSortDir("ASC")} {
			v, e := s.All(context.Background(), func(*tags.ListOpts) error { callbacks.Add(1); return nil }, o)
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(v, e)
			}
		}
		for _, name := range []string{"", "../other", "\xff"} {
			v, e := s.Update(context.Background(), tagName, tags.WithUpdateName(name))
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(v, e)
			}
		}
		v, e := s.Set(context.Background(), []string{"\xff"})
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 7 {
			t.Fatal(v, e, calls.Load(), callbacks.Load())
		}
		v2, e := s.Update(context.Background(), tagName, tags.WithUpdateName("unsafe/name"), tags.WithUpdateOpts(tags.UpdateOpts{}))
		if e != nil || v2 == nil || calls.Load() != 1 {
			t.Fatal(v2, e, calls.Load())
		}
	})
	for _, op := range tagOperations {
		t.Run(op.name+" callback causes", func(t *testing.T) {
			var calls, callbacks atomic.Int32
			cause := errors.New("option callback")
			client := tagClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			v, e := tagCall(tagScope(t, client, tagParent), context.Background(), op.name, tagName, tagCallbacks(&callbacks, cause))
			if v != nil || !errors.Is(e, cause) || calls.Load() != 0 || callbacks.Load() != 1 {
				t.Fatal(v, e, calls.Load(), callbacks.Load())
			}
		})
	}
	t.Run("Set input header pointer and callback ownership", func(t *testing.T) {
		input := []string{"before"}
		header := map[string]string{"X-Own": "factory"}
		appendValue := true
		full := tags.WithSetOpts(tags.SetOpts{Headers: header, Append: &appendValue})
		header["X-Own"] = "late"
		appendValue = false
		var borrowedHeaders map[string]string
		var borrowedAppend *bool
		var calls, callbacks atomic.Int32
		client := tagClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, _ := io.ReadAll(r.Body)
			if string(raw) != `{"tags":[{"name":"before"}]}` || r.Header.Get("X-Own") != "factory" || r.Header.Get("X-Step") != "first" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-OpenStack-Append") != "True" {
				t.Error(string(raw), r.Header)
			}
			return tagWire(201, &tagBody{Reader: strings.NewReader(`{"tags":[]}`)}), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		s := tagScope(t, client, tagParent)
		v, e := s.Set(context.Background(), input, full, func(o *tags.SetOpts) error {
			callbacks.Add(1)
			o.Headers["X-Step"] = "first"
			borrowedHeaders = o.Headers
			borrowedAppend = o.Append
			input[0] = "caller late"
			client.MoreHeaders["X-Source"] = "future"
			return nil
		}, func(o *tags.SetOpts) error {
			callbacks.Add(1)
			borrowedHeaders["X-Step"] = "late"
			*borrowedAppend = false
			return nil
		})
		if e != nil || v == nil || calls.Load() != 1 || callbacks.Load() != 2 || client.MoreHeaders["X-Source"] != "future" {
			t.Fatal(v, e, calls.Load(), callbacks.Load())
		}
	})
	t.Run("full header options replace and preserve original provider", func(t *testing.T) {
		for _, op := range tagOperations {
			var calls atomic.Int32
			h := map[string]string{"X-Own": "snapshot"}
			client := tagClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Header.Get("X-Discard") != "" || r.Header.Get("X-Own") != "snapshot" {
					t.Error(r.Header)
				}
				return tagWire(op.status, &tagBody{Reader: strings.NewReader(op.reply)}), nil
			})
			p := client.ProviderClient
			opts := tagHeaderOption("X-Discard", "discard")
			switch op.name {
			case "Create":
				opts.create = append(opts.create, tags.WithCreateOpts(tags.CreateOpts{Headers: h}))
			case "Get":
				opts.get = append(opts.get, tags.WithGetOpts(tags.GetOpts{Headers: h}))
			case "Update":
				opts.update = append(opts.update, tags.WithUpdateOpts(tags.UpdateOpts{Headers: h}))
			case "Delete":
				opts.deletion = append(opts.deletion, tags.WithDeleteOpts(tags.DeleteOpts{Headers: h}))
			case "DeleteAll":
				opts.deleteAll = append(opts.deleteAll, tags.WithDeleteAllOpts(tags.DeleteAllOpts{Headers: h}))
			case "Set":
				opts.set = append(opts.set, tags.WithSetOpts(tags.SetOpts{Headers: h}))
			default:
				opts.list = append(opts.list, tags.WithListOpts(tags.ListOpts{Headers: h}))
			}
			h["X-Own"] = "late"
			v, e := tagCall(tagScope(t, client, tagParent), context.Background(), op.name, tagName, opts)
			if e != nil || v == nil || calls.Load() != 1 || client.ProviderClient != p || p.HTTPClient.CheckRedirect != nil {
				t.Fatal(op.name, v, e, calls.Load())
			}
		}
	})
}

type tagTransport func(*http.Request) (*http.Response, error)

func (f tagTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, e := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, e
}

type tagBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *tagBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type tagReader func([]byte) (int, error)

func (f tagReader) Read(p []byte) (int, error) { return f(p) }
func tagWire(status int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"X-Request-Id": {"actual-tag"}, "Content-Type": {"application/json"}}, Body: body}
}
func tagClient(f tagTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "image", Endpoint: tagBase}
}
func tagScope(t *testing.T, client *gophercloud.ServiceClient, parent string) *tags.NamespaceScope {
	t.Helper()
	api := tags.New(client)
	s, e := api.InNamespace(context.Background(), parent)
	if e != nil || s == nil || api.RawClient() != client || s.RawClient() != client || s.NamespaceName() != parent {
		t.Fatalf("scope %v %v", s, e)
	}
	return s
}
func tagString(v string) *string { return &v }
func tagInt(v int) *int          { return &v }
func tagBool(v bool) *bool       { return &v }

type tagOptions struct {
	create    []tags.CreateOption
	get       []tags.GetOption
	update    []tags.UpdateOption
	deletion  []tags.DeleteOption
	deleteAll []tags.DeleteAllOption
	set       []tags.SetOption
	list      []tags.ListOption
}
type tagResult struct {
	value *tags.Tag
	ack   *tags.Acknowledgement
	set   *tags.SetResult
	rows  []*tags.Tag
}

func tagCall(s *tags.NamespaceScope, ctx context.Context, op, name string, options tagOptions) (*tagResult, error) {
	var v *tags.Tag
	var e error
	switch op {
	case "Create":
		v, e = s.Create(ctx, name, options.create...)
	case "Get":
		v, e = s.Get(ctx, name, options.get...)
	case "Update":
		v, e = s.Update(ctx, name, options.update...)
	case "Delete":
		v, e := s.Delete(ctx, name, options.deletion...)
		if v == nil {
			return nil, e
		}
		return &tagResult{ack: v}, e
	case "DeleteAll":
		v, e := s.DeleteAll(ctx, options.deleteAll...)
		if v == nil {
			return nil, e
		}
		return &tagResult{ack: v}, e
	case "Set":
		v, e := s.Set(ctx, []string{tagName}, options.set...)
		if v == nil {
			return nil, e
		}
		return &tagResult{set: v}, e
	case "All":
		v, e := s.All(ctx, options.list...)
		if v == nil {
			return nil, e
		}
		return &tagResult{rows: v}, e
	case "List":
		rows := make([]*tags.Tag, 0)
		for v, e := range s.List(ctx, options.list...) {
			if e != nil {
				return nil, e
			}
			rows = append(rows, v)
		}
		return &tagResult{rows: rows}, nil
	default:
		panic("unknown operation")
	}
	if v == nil {
		return nil, e
	}
	return &tagResult{value: v}, e
}

var tagOperations = []struct {
	name, method, body, reply string
	status                    int
	named                     bool
}{
	{"Create", "POST", "", tagJSON, 201, true},
	{"Get", "GET", "", tagJSON, 200, true},
	{"Update", "PUT", `{"name":"CPU Limits"}`, tagJSON, 200, true},
	{"Delete", "DELETE", "", "", 204, true},
	{"DeleteAll", "DELETE", "", "", 204, false},
	{"Set", "POST", `{"tags":[{"name":"CPU Limits"}]}`, tagListJSON, 201, false},
	{"List", "GET", "", tagListJSON, 200, false},
	{"All", "GET", "", tagListJSON, 200, false},
}

func tagProof(t *testing.T, e error, status int, raw []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(e, &proof) || proof.StatusCode != status || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-tag" {
		t.Fatalf("owned proof %v %+v", e, proof)
	}
	return proof
}
func tagHeaders() tagOptions {
	m := map[string]string{"X-Option": "owned"}
	return tagOptions{
		create:    []tags.CreateOption{tags.WithCreateHeaders(m), tags.WithCreateHeader("X-Final", "yes")},
		get:       []tags.GetOption{tags.WithGetHeaders(m), tags.WithGetHeader("X-Final", "yes")},
		update:    []tags.UpdateOption{tags.WithUpdateHeaders(m), tags.WithUpdateHeader("X-Final", "yes")},
		deletion:  []tags.DeleteOption{tags.WithDeleteHeaders(m), tags.WithDeleteHeader("X-Final", "yes")},
		deleteAll: []tags.DeleteAllOption{tags.WithDeleteAllHeaders(m), tags.WithDeleteAllHeader("X-Final", "yes")},
		set:       []tags.SetOption{tags.WithSetHeaders(m), tags.WithSetHeader("X-Final", "yes")},
		list:      []tags.ListOption{tags.WithListHeaders(m), tags.WithListHeader("X-Final", "yes")},
	}
}

func TestMetadefTagsFixedScopedRoutesAndPayloads(t *testing.T) {
	for _, op := range tagOperations {
		t.Run(op.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/unused/")
			client.ResourceBase = cloud.Server.URL + tagPrefix
			client.Microversion = "2.2"
			var calls atomic.Int32
			s := tagScope(t, client, tagParent)
			if calls.Load() != 0 {
				t.Fatal("scope HTTP")
			}
			client.MoreHeaders = map[string]string{"X-Source": "latest"}
			cloud.Provider.SetToken("live")
			cloud.Mux.HandleFunc(tagPrefix, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				want := tagPrefix + tagCollection
				if op.named {
					want += "/" + url.PathEscape(tagName)
				}
				if r.Method != op.method || r.URL.EscapedPath() != want || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("X-Source") != "latest" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "yes" || r.Header.Get("OpenStack-API-Version") != "image 2.2" {
					t.Error(r.Method, r.URL, r.Header, want)
				}
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != op.body {
					t.Error(string(raw), op.body)
				}
				if op.name == "Set" && r.Header.Get("X-OpenStack-Append") != "False" {
					t.Error(r.Header)
				}
				w.Header().Set("X-Request-Id", "actual-tag")
				w.Header().Set("Location", "https://passive.invalid/new")
				testcloud.JSON(w, op.status, op.reply)
			})
			v, e := tagCall(s, context.Background(), op.name, tagName, tagHeaders())
			if e != nil || v == nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			if v.value != nil && (v.value.Name == nil || *v.value.Name != "passive foreign name" || v.value.StatusCode != op.status || v.value.Header.Get("Location") != "https://passive.invalid/new") {
				t.Fatal(v.value)
			}
			if v.ack != nil && (v.ack.Namespace != tagParent || v.ack.StatusCode != 204 || op.named && (v.ack.Name == nil || *v.ack.Name != tagName) || !op.named && v.ack.Name != nil) {
				t.Fatal(v.ack)
			}
			if v.set != nil && (len(v.set.Tags) != 1 || v.set.StatusCode != 201) {
				t.Fatal(v.set)
			}
			if v.rows != nil && len(v.rows) != 1 {
				t.Fatal(v.rows)
			}
		})
	}
	t.Run("literal route identities and rename", func(t *testing.T) {
		for _, parent := range []string{"OS::Vendor Name::한글", strings.Repeat("界", 80)} {
			for _, name := range []string{"tag::한글 name", strings.Repeat("名", 80)} {
				var calls atomic.Int32
				client := tagClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					want := tagPrefix + "metadefs/namespaces/" + url.PathEscape(parent) + "/tags/" + url.PathEscape(name)
					raw, _ := io.ReadAll(r.Body)
					var body map[string]string
					if e := json.Unmarshal(raw, &body); e != nil || body["name"] != "새 이름" || r.URL.EscapedPath() != want || r.Method != "PUT" {
						t.Error(r.Method, r.URL, string(raw), e)
					}
					return tagWire(200, &tagBody{Reader: strings.NewReader(tagJSON)}), nil
				})
				v, e := tagScope(t, client, parent).Update(context.Background(), name, tags.WithUpdateName("새 이름"))
				if e != nil || v == nil || calls.Load() != 1 {
					t.Fatal(v, e, calls.Load())
				}
			}
		}
	})
	for _, input := range [][]string{nil, {}, {"", "line\ncontrol\t", "a/b%?#", strings.Repeat("界", 100), "duplicate", "duplicate"}} {
		for _, mode := range []string{"nil", "false", "true"} {
			t.Run(fmt.Sprintf("Set %s %d", mode, len(input)), func(t *testing.T) {
				var calls atomic.Int32
				client := tagClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					raw, _ := io.ReadAll(r.Body)
					var body struct {
						Tags []struct {
							Name string `json:"name"`
						} `json:"tags"`
					}
					if e := json.Unmarshal(raw, &body); e != nil || body.Tags == nil || len(body.Tags) != len(input) {
						t.Error(string(raw), e)
					}
					for i, row := range body.Tags {
						if row.Name != input[i] {
							t.Error(row.Name, input[i])
						}
					}
					want := "False"
					if mode == "true" {
						want = "True"
					}
					if r.Method != "POST" || r.URL.String() != tagBase+tagCollection || r.Header.Get("X-OpenStack-Append") != want {
						t.Error(r.Method, r.URL, r.Header)
					}
					return tagWire(201, &tagBody{Reader: strings.NewReader(`{"tags":[]}`)}), nil
				})
				var options []tags.SetOption
				if mode != "nil" {
					options = []tags.SetOption{tags.WithSetAppend(mode == "true")}
				}
				v, e := tagScope(t, client, tagParent).Set(context.Background(), input, options...)
				if e != nil || v == nil || v.Tags == nil || len(v.Tags) != 0 || calls.Load() != 1 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
}

func TestMetadefTagsResponseOwnershipAndNativeHooks(t *testing.T) {
	for _, op := range tagOperations {
		for _, mode := range []string{"Read", "Close", "cancel", "Read Close cancel", "source after Close"} {
			t.Run(op.name+" accepted "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted Read"), errors.New("accepted Close"), errors.New("accepted cause")
				raw := []byte(op.reply)
				if op.status == 204 {
					raw = []byte{'a', 'c', 'k', 255}
				}
				b := &tagBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "Read") {
					b.Reader = tagReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					b.closeErr = closeCause
				}
				var calls, hooks atomic.Int32
				client := tagClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return tagWire(op.status, b), nil })
				s := tagScope(t, client, tagParent)
				if mode == "source after Close" {
					b.closeErr = nil
					b.onClose = func() { client.ResourceBase = "https://glance.invalid/replaced/v2/" }
				} else if strings.Contains(mode, "cancel") {
					b.onClose = func() { cancel(cancelCause) }
				}
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return errors.New("accepted replay")
				}
				v, e := tagCall(s, ctx, op.name, tagName, tagOptions{})
				proof := tagProof(t, e, op.status, raw)
				if calls.Load() != 1 || hooks.Load() != 0 || b.closes.Load() != 1 || strings.Contains(mode, "Read") && !errors.Is(e, readCause) || mode != "source after Close" && strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled)) || mode == "source after Close" && !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(v, e, calls.Load(), hooks.Load(), b.closes.Load())
				}
				if op.status == 204 {
					if v == nil || v.ack == nil || v.ack.Namespace != tagParent || v.ack.StatusCode != 204 || !bytes.Equal(v.ack.Body, raw) || op.named && (v.ack.Name == nil || *v.ack.Name != tagName) || !op.named && v.ack.Name != nil {
						t.Fatal(v, e)
					}
					proof.Body[0] = '!'
					proof.Header.Set("X-Request-Id", "proof changed")
					if v.ack.Body[0] != 'a' || v.ack.Header.Get("X-Request-Id") != "actual-tag" {
						t.Fatal("ack aliases proof")
					}
				} else if v != nil {
					t.Fatal("partial typed result", v, e)
				}
			})
		}
	}
	for _, op := range []string{"Delete", "DeleteAll"} {
		t.Run(op+" opaque204 and strict404 hooks", func(t *testing.T) {
			raw := []byte{255, 0, 1}
			b := &tagBody{Reader: bytes.NewReader(raw)}
			wire := tagWire(204, b)
			var calls atomic.Int32
			client := tagClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != "DELETE" || r.Body != nil || r.URL.RawQuery != "" {
					t.Error(r.Method, r.URL, r.Body)
				}
				return wire, nil
			})
			s := tagScope(t, client, tagParent)
			v, e := tagCall(s, context.Background(), op, tagName, tagOptions{})
			if e != nil || v == nil || v.ack == nil || !bytes.Equal(v.ack.Body, raw) || calls.Load() != 1 || b.closes.Load() != 1 {
				t.Fatal(v, e)
			}
			raw[0] = 1
			wire.Header.Set("X-Request-Id", "late")
			if v.ack.Body[0] != 255 || v.ack.Header.Get("X-Request-Id") != "actual-tag" {
				t.Fatal("borrowed ack alias")
			}
			var hooks atomic.Int32
			cause := errors.New("strict404 hook")
			rejected := &tagBody{Reader: strings.NewReader("actual404")}
			client.HTTPClient.Transport = tagTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return tagWire(404, rejected), nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return cause
			}
			v, e = tagCall(s, context.Background(), op, tagName, tagOptions{})
			if v != nil || !errors.Is(e, cause) || !gophercloud.ResponseCodeIs(e, 404) || hooks.Load() != 1 || calls.Load() != 2 || rejected.closes.Load() != 1 {
				t.Fatal(v, e, hooks.Load(), calls.Load(), rejected.closes.Load())
			}
		})
	}
	for _, op := range tagOperations {
		for _, code := range []int{202, 403, 404, 409} {
			t.Run(fmt.Sprintf("%s strict%d", op.name, code), func(t *testing.T) {
				raw := []byte("actual server rejection")
				b := &tagBody{Reader: bytes.NewReader(raw)}
				var calls atomic.Int32
				client := tagClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return tagWire(code, b), nil })
				v, e := tagCall(tagScope(t, client, tagParent), context.Background(), op.name, tagName, tagOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(e, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{op.status}) || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Request-Id") != "actual-tag" || calls.Load() != 1 || b.closes.Load() != 1 {
					t.Fatal(v, e, native, calls.Load(), b.closes.Load())
				}
			})
		}
	}
	for _, mode := range []string{"transport nested404", "callback nested404", "reauth nested404"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("unrelated404 cause")
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{204}, Method: "DELETE", URL: "https://unrelated.invalid", Body: []byte("unrelated404")}
			var calls, hooks atomic.Int32
			b := &tagBody{Reader: strings.NewReader("original failure")}
			client := tagClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport nested404" {
					return nil, errors.Join(cause, nested)
				}
				status := 503
				if mode == "reauth nested404" {
					status = 401
				}
				return tagWire(status, b), nil
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
			v, e := tagScope(t, client, tagParent).Delete(context.Background(), tagName)
			if v != nil || e == nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			if mode == "reauth nested404" {
				var reauth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(e, &reauth) || !gophercloud.ResponseCodeIs(reauth.ErrOriginal, 401) || !errors.Is(reauth.ErrReauth, cause) || !gophercloud.ResponseCodeIs(reauth.ErrReauth, 404) || hooks.Load() != 1 {
					t.Fatal(e, reauth, hooks.Load())
				}
			} else if !errors.Is(e, cause) {
				t.Fatal(e)
			}
			if mode != "transport nested404" && b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		})
	}
	t.Run("configured prebody hooks fixed source and live original auth", func(t *testing.T) {
		var calls, reauth, backoff, retries atomic.Int32
		var bodies []*tagBody
		transportCause := errors.New("prebody transport")
		client := tagClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); client.SetToken("reauth"); return nil }
		client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff.Add(1)
			client.SetToken("backoff")
			return nil
		}
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
			retries.Add(1)
			raw, _ := json.Marshal(o.JSONBody)
			if !o.KeepResponseBody || o.JSONResponse != nil || o.RawBody != nil || string(raw) != `{"tags":[{"name":"CPU Limits"}]}` {
				t.Error(o, string(raw))
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
		provider, retry := client.ProviderClient, reflect.ValueOf(client.RetryFunc).Pointer()
		s := tagScope(t, client, tagParent)
		client.HTTPClient.Transport = tagTransport(func(r *http.Request) (*http.Response, error) {
			i := int(calls.Add(1)) - 1
			codes := []int{401, 429, 503, 0, 201}
			tokens := []string{"initial", "reauth", "backoff", "retry503", "retrytransport"}
			if i >= len(codes) {
				return nil, errors.New("replay")
			}
			raw, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || r.URL.String() != tagBase+tagCollection || string(raw) != `{"tags":[{"name":"CPU Limits"}]}` || r.Header.Get("X-OpenStack-Append") != "False" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[i] {
				t.Error(r.Method, r.URL, string(raw), r.Header)
			}
			if codes[i] == 0 {
				return nil, transportCause
			}
			b := &tagBody{Reader: strings.NewReader(tagListJSON)}
			bodies = append(bodies, b)
			return tagWire(codes[i], b), nil
		})
		v, e := s.Set(context.Background(), []string{tagName})
		if v == nil || e != nil || calls.Load() != 5 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 2 || client.ProviderClient != provider || reflect.ValueOf(client.RetryFunc).Pointer() != retry {
			t.Fatal(v, e, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
		}
		for _, b := range bodies {
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		}
	})
	for _, change := range []string{"in-place RawMessage", "changed JSON", "JSON nil", "JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSONBody"} {
		t.Run("Set retry guard "+change, func(t *testing.T) {
			callbackCause := errors.New("retry callback")
			var calls, hooks, borrowedReads atomic.Int32
			b := &tagBody{Reader: strings.NewReader("original503")}
			borrowed := &tagBody{Reader: tagReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			client := tagClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != `{"tags":[{"name":"CPU Limits"}]}` {
					t.Error(string(raw))
				}
				return tagWire(503, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					t.Error(e)
				}
				switch change {
				case "in-place RawMessage":
					raw, ok := o.JSONBody.(json.RawMessage)
					if !ok {
						t.Error(o.JSONBody)
					} else {
						i := bytes.Index(raw, []byte("CPU"))
						if i < 0 {
							t.Error(string(raw))
						} else {
							raw[i] = 'X'
						}
					}
				case "changed JSON":
					o.JSONBody = map[string]any{"tags": []map[string]string{{"name": "other"}}}
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
			v, e := tagScope(t, client, tagParent).Set(context.Background(), []string{tagName})
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, callbackCause) || !errors.As(e, &native) || native.Actual != 503 || string(native.Body) != "original503" || calls.Load() != 1 || hooks.Load() != 1 || b.closes.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 {
				t.Fatal(v, e, native, calls.Load(), hooks.Load())
			}
			if change == "unsupported JSONBody" {
				var encoding *json.UnsupportedTypeError
				if !errors.As(e, &encoding) {
					t.Fatal("encoding cause lost", e)
				}
			}
		})
	}
	for _, op := range []string{"Create", "DeleteAll"} {
		t.Run(op+" absent body differs from null", func(t *testing.T) {
			var calls atomic.Int32
			b := &tagBody{Reader: strings.NewReader("original503")}
			client := tagClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Body != nil {
					t.Error(r.Body)
				}
				return tagWire(503, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				o.JSONBody = json.RawMessage("null")
				return nil
			}
			v, e := tagCall(tagScope(t, client, tagParent), context.Background(), op, tagName, tagOptions{})
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 1 || b.closes.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	for _, op := range []string{"Update", "Set"} {
		t.Run(op+" same serialized replacement and caller mutation", func(t *testing.T) {
			var calls, hooks atomic.Int32
			var bodies []*tagBody
			input := []string{tagName}
			client := tagClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				want := `{"name":"CPU Limits"}`
				if op == "Set" {
					want = `{"tags":[{"name":"CPU Limits"}]}`
				}
				if string(raw) != want {
					t.Error(string(raw), want)
				}
				status, reply := 503, "original503"
				if n == 2 {
					status, reply = 200, tagJSON
					if op == "Set" {
						status, reply = 201, tagListJSON
					}
				}
				b := &tagBody{Reader: strings.NewReader(reply)}
				bodies = append(bodies, b)
				return tagWire(status, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks.Add(1)
				input[0] = "caller mutation"
				if op == "Update" {
					o.JSONBody = map[string]string{"name": tagName}
				} else {
					o.JSONBody = map[string]any{"tags": []map[string]string{{"name": tagName}}}
				}
				return nil
			}
			s := tagScope(t, client, tagParent)
			var e error
			if op == "Set" {
				v, err := s.Set(context.Background(), input)
				e = err
				if v == nil {
					t.Fatal(v, e)
				}
			} else {
				v, err := s.Update(context.Background(), tagName)
				e = err
				if v == nil {
					t.Fatal(v, e)
				}
			}
			if e != nil || calls.Load() != 2 || hooks.Load() != 1 {
				t.Fatal(e, calls.Load(), hooks.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	t.Run("explicit native retry MoreHeaders policy remains available", func(t *testing.T) {
		var calls, hooks atomic.Int32
		var bodies []*tagBody
		client := tagClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			want := "False"
			code, reply := 503, "retry"
			if n == 2 {
				want = "True"
				code, reply = 201, tagListJSON
			}
			if r.Header.Get("X-OpenStack-Append") != want {
				t.Error(r.Header)
			}
			b := &tagBody{Reader: strings.NewReader(reply)}
			bodies = append(bodies, b)
			return tagWire(code, b), nil
		})
		client.MoreHeaders = map[string]string{"X-Ordinary": "original"}
		provider := client.ProviderClient
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			hooks.Add(1)
			o.MoreHeaders = map[string]string{"X-OpenStack-Append": "True"}
			return nil
		}
		v, e := tagScope(t, client, tagParent).Set(context.Background(), []string{tagName})
		if e != nil || v == nil || calls.Load() != 2 || hooks.Load() != 1 || client.ProviderClient != provider || len(client.MoreHeaders) != 1 || client.MoreHeaders["X-Ordinary"] != "original" {
			t.Fatal(v, e, calls.Load(), hooks.Load(), client.MoreHeaders)
		}
		for _, b := range bodies {
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		}
	})
	for _, op := range tagOperations {
		t.Run(op.name+" expanded codes actual gate", func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("unexpected Read"), errors.New("unexpected Close"), errors.New("unexpected cause")
			var calls, hooks atomic.Int32
			var bodies []*tagBody
			client := tagClient(func(*http.Request) (*http.Response, error) {
				n := calls.Add(1)
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 202, "unexpected actual202"
				}
				b := &tagBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.Reader = tagReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return tagWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					return e
				}
				o.OkCodes = []int{202}
				return nil
			}
			v, e := tagCall(tagScope(t, client, tagParent), ctx, op.name, tagName, tagOptions{})
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			target := tagBase + tagCollection
			if op.named {
				target += "/" + url.PathEscape(tagName)
			}
			if v != nil || !errors.As(e, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{op.status}) || native.Method != op.method || native.URL != target || string(native.Body) != "unexpected actual202" || native.ResponseHeader.Get("X-Request-Id") != "actual-tag" || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled) || errors.As(e, &proof) || calls.Load() != 2 || hooks.Load() != 1 {
				t.Fatal(v, e, native, calls.Load(), hooks.Load())
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
			var bodies []*tagBody
			client := tagClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				b := &tagBody{Reader: strings.NewReader(tagListJSON)}
				bodies = append(bodies, b)
				if n == 2 {
					raw, _ := io.ReadAll(r.Body)
					if r.Method != "POST" || string(raw) != `{"tags":[{"name":"CPU Limits"}]}` {
						t.Error(r.Method, string(raw))
					}
					return tagWire(201, b), nil
				}
				code, target := 307, tagBase+tagCollection
				switch redirect {
				case "foreign origin":
					target = "https://foreign.invalid/tags"
				case "changed path":
					target = tagBase + "metadefs/namespaces/other/tags"
				case "changed query":
					target += "?marker=other"
				case "changed method":
					code = 303
				}
				wire := tagWire(code, b)
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			v, e := tagScope(t, client, tagParent).Set(context.Background(), []string{tagName})
			if redirect == "same target" {
				if v == nil || e != nil || calls.Load() != 2 || redirects.Load() != 1 {
					t.Fatal(v, e, calls.Load(), redirects.Load())
				}
			} else if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
}

func TestMetadefTagsCanonicalModelsAndRawOwnership(t *testing.T) {
	for _, op := range []string{"Create", "Get", "Update", "Set"} {
		for _, raw := range []string{`{}`, `{"name":null,"created_at":null,"updated_at":null}`, `{"name":"","created_at":"","updated_at":"not-a-date","Name":42,"id":true,"links":false,"number":9007199254740993,"decimal":1.2300e+60}`} {
			t.Run(op+" "+raw, func(t *testing.T) {
				reply := raw
				if op == "Set" {
					reply = `{"tags":[` + raw + `],"unknown":null,"name":42,"created_at":"root-date"}`
				}
				status := 200
				if op == "Create" || op == "Set" {
					status = 201
				}
				b := &tagBody{Reader: strings.NewReader(reply)}
				wire := tagWire(status, b)
				client := tagClient(func(*http.Request) (*http.Response, error) { return wire, nil })
				result, e := tagCall(tagScope(t, client, tagParent), context.Background(), op, tagName, tagOptions{})
				if e != nil || result == nil || b.closes.Load() != 1 {
					t.Fatal(result, e, b.closes.Load())
				}
				v := result.value
				if op == "Set" {
					v = result.set.Tags[0]
					if string(result.set.Body["created_at"]) != `"root-date"` || result.set.Links != nil {
						t.Fatal(result.set)
					}
				}
				if v.Links != nil || v.Header.Get("X-Request-Id") != "actual-tag" || v.StatusCode != status {
					t.Fatal(v)
				}
				var fields map[string]json.RawMessage
				_ = json.Unmarshal([]byte(raw), &fields)
				if !reflect.DeepEqual(v.Body, fields) {
					t.Fatal(v.Body, fields)
				}
				if raw == `{}` || strings.Contains(raw, `"name":null`) {
					if v.Name != nil || v.CreatedAt != nil || v.UpdatedAt != nil {
						t.Fatal(v)
					}
				} else if v.Name == nil || *v.Name != "" || v.CreatedAt == nil || *v.CreatedAt != "" || v.UpdatedAt == nil || *v.UpdatedAt != "not-a-date" || string(v.Body["number"]) != "9007199254740993" || string(v.Body["decimal"]) != "1.2300e+60" {
					t.Fatal(v)
				}
				wire.Header.Set("X-Request-Id", "borrowed changed")
				if v.Header.Get("X-Request-Id") != "actual-tag" {
					t.Fatal("header alias")
				}
				if op == "Set" {
					before := append([]byte(nil), result.set.Body["tags"]...)
					v.Body["name"] = json.RawMessage(`"row changed"`)
					v.Header.Set("X-Request-Id", "row changed")
					if !bytes.Equal(result.set.Body["tags"], before) || result.set.Header.Get("X-Request-Id") != "actual-tag" {
						t.Fatal("Set row aliases root")
					}
				}
			})
		}
	}
	for _, op := range []string{"Create", "Get", "Update", "Set", "All"} {
		for _, bad := range []string{"", `null`, `[]`, `42`, `{"name":42}`, `{"created_at":false}`, `{"updated_at":{}}`, "{\"name\":\"\xff\"}"} {
			t.Run(op+" invalid "+bad, func(t *testing.T) {
				reply := bad
				if (op == "Set" || op == "All") && bad != "" {
					reply = `{"tags":[` + bad + `]}`
				}
				status := 200
				if op == "Create" || op == "Set" {
					status = 201
				}
				b := &tagBody{Reader: strings.NewReader(reply)}
				var calls atomic.Int32
				client := tagClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return tagWire(status, b), nil })
				v, e := tagCall(tagScope(t, client, tagParent), context.Background(), op, tagName, tagOptions{})
				tagProof(t, e, status, []byte(reply))
				if v != nil || calls.Load() != 1 || b.closes.Load() != 1 {
					t.Fatal(v, e, calls.Load(), b.closes.Load())
				}
			})
		}
	}
	for _, op := range []string{"Set", "All"} {
		for _, raw := range []string{`{}`, `{"tags":null}`, `{"tags":{}}`, `{"Tags":[]}`, `{"tags":[]}`, `{"tags":[{},null]}`} {
			t.Run(op+" envelope "+raw, func(t *testing.T) {
				status := 200
				if op == "Set" {
					status = 201
				}
				b := &tagBody{Reader: strings.NewReader(raw)}
				client := tagClient(func(*http.Request) (*http.Response, error) { return tagWire(status, b), nil })
				v, e := tagCall(tagScope(t, client, tagParent), context.Background(), op, tagName, tagOptions{})
				if raw == `{"tags":[]}` {
					if e != nil || v == nil || op == "Set" && v.set.Tags == nil || op == "All" && v.rows == nil {
						t.Fatal(v, e)
					}
				} else {
					tagProof(t, e, status, []byte(raw))
					if v != nil {
						t.Fatal("partial typed result", v)
					}
				}
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			})
		}
	}
	t.Run("atomic Set rows and decoder reuse", func(t *testing.T) {
		var value tags.Tag
		if e := json.Unmarshal([]byte(tagJSON), &value); e != nil {
			t.Fatal(e)
		}
		name := *value.Name
		before := append([]byte(nil), value.Body["name"]...)
		if e := json.Unmarshal([]byte(`{"name":"new","updated_at":42}`), &value); e == nil || *value.Name != name || !bytes.Equal(value.Body["name"], before) {
			t.Fatal(value, e)
		}
		var set tags.SetResult
		if e := json.Unmarshal([]byte(`{"tags":[{"name":"old"}]}`), &set); e != nil {
			t.Fatal(e)
		}
		if e := json.Unmarshal([]byte(`{"tags":[{"name":"new"},null]}`), &set); e == nil || len(set.Tags) != 1 || *set.Tags[0].Name != "old" {
			t.Fatal(set, e)
		}
	})
}

func TestMetadefTagsListAndPagingControls(t *testing.T) {
	for _, limit := range []*int{nil, tagInt(0)} {
		for _, cap := range []int{0, 1, 3, 20} {
			t.Run(fmt.Sprintf("default limit %v cap%d", limit, cap), func(t *testing.T) {
				var calls atomic.Int32
				reply := `{"tags":[{"name":"first"},{"name":"last"}],"next":"https://foreign.invalid","first":42}`
				if limit != nil {
					reply = `{"tags":[]}`
				}
				client := tagClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					want := ""
					if limit != nil {
						want = "limit=0"
					}
					if r.URL.RawQuery != want {
						t.Error(r.URL)
					}
					wire := tagWire(200, &tagBody{Reader: strings.NewReader(reply)})
					wire.Header.Set("Link", `<https://foreign.invalid>; rel="next"`)
					return wire, nil
				})
				v, e := tagScope(t, client, tagParent).All(context.Background(), tags.WithListOpts(tags.ListOpts{Limit: limit, MaxItems: cap}))
				want := 2
				if cap == 1 {
					want = 1
				}
				if limit != nil {
					want = 0
				}
				if e != nil || v == nil || len(v) != want || calls.Load() != 1 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	for _, mode := range []string{"short", "exact then empty", "oversized", "raw marker before caller mutation"} {
		t.Run(mode, func(t *testing.T) {
			pages := []string{`{"tags":[{"name":"a"},{"name":"wire:空 /?#%"}]}`, `{"tags":[{"name":"b"}]}`}
			if mode == "exact then empty" {
				pages[1] = `{"tags":[]}`
			}
			if mode == "oversized" {
				pages[0] = `{"tags":[{"name":"a"},{"name":"b"},{"name":"wire:空 /?#%"}]}`
			}
			var calls atomic.Int32
			client := tagClient(func(r *http.Request) (*http.Response, error) {
				n := int(calls.Add(1)) - 1
				if n >= len(pages) {
					t.Error("extra request", r.URL)
					return nil, errors.New("extra request")
				}
				q := r.URL.Query()
				wantMarker := "initial:?"
				if n == 1 {
					wantMarker = "wire:空 /?#%"
				}
				if q.Get("limit") != "2" || q.Get("marker") != wantMarker || q.Get("sort_key") != "name" || q.Get("sort_dir") != "asc" || len(q) != 4 || r.URL.EscapedPath() != tagPrefix+tagCollection {
					t.Error(r.URL, q)
				}
				return tagWire(200, &tagBody{Reader: strings.NewReader(pages[n])}), nil
			})
			s := tagScope(t, client, tagParent)
			seen := 0
			for v, e := range s.List(context.Background(), tags.WithListLimit(2), tags.WithListMarker("initial:?"), tags.WithListSortKey("name"), tags.WithListSortDir("asc")) {
				if e != nil {
					t.Fatal(e)
				}
				seen++
				if mode == "raw marker before caller mutation" {
					*v.Name = "caller changed"
					v.Body["name"] = json.RawMessage(`"caller changed raw"`)
				}
			}
			want := 3
			if mode == "exact then empty" {
				want = 2
			}
			if mode == "oversized" {
				want = 4
			}
			if seen != want || calls.Load() != 2 {
				t.Fatal(seen, calls.Load())
			}
		})
	}
	t.Run("preserve explicit empty query values", func(t *testing.T) {
		client := tagClient(func(r *http.Request) (*http.Response, error) {
			q := r.URL.Query()
			if q.Get("limit") != "0" || !reflect.DeepEqual(q["marker"], []string{""}) || !reflect.DeepEqual(q["sort_key"], []string{""}) || len(q) != 3 {
				t.Error(r.URL, q)
			}
			return tagWire(200, &tagBody{Reader: strings.NewReader(`{"tags":[]}`)}), nil
		})
		v, e := tagScope(t, client, tagParent).All(context.Background(), tags.WithListMarker(""), tags.WithListSortKey(""), tags.WithListLimit(0))
		if e != nil || v == nil {
			t.Fatal(v, e)
		}
	})
	for _, last := range []string{`{}`, `{"name":null}`, `{"name":""}`, `{"name":"line\ncontrol"}`, `{"name":"initial"}`} {
		for _, mode := range []string{"All", "cap", "break"} {
			t.Run(mode+" final marker "+last, func(t *testing.T) {
				var calls atomic.Int32
				raw := `{"tags":[` + last + `]}`
				client := tagClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return tagWire(200, &tagBody{Reader: strings.NewReader(raw)}), nil
				})
				s := tagScope(t, client, tagParent)
				options := []tags.ListOption{tags.WithListLimit(1), tags.WithListMarker("initial")}
				if mode == "cap" {
					options = append(options, tags.WithListMaxItems(1))
				}
				if mode == "break" {
					for v, e := range s.List(context.Background(), options...) {
						if e != nil || v == nil {
							t.Fatal(v, e)
						}
						break
					}
				} else {
					v, e := s.All(context.Background(), options...)
					if mode == "cap" {
						if e != nil || len(v) != 1 {
							t.Fatal(v, e)
						}
					} else {
						if v != nil || e == nil {
							t.Fatal(v, e)
						}
						tagProof(t, e, 200, []byte(raw))
					}
				}
				if calls.Load() != 1 {
					t.Fatal("invalid marker followed", calls.Load())
				}
			})
		}
	}
	t.Run("previous marker cycle detected after second page", func(t *testing.T) {
		var calls atomic.Int32
		raw := `{"tags":[{"name":"a"}]}`
		client := tagClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			if n > 2 {
				t.Error("cycle followed")
			}
			if n == 2 && r.URL.Query().Get("marker") != "a" {
				t.Error(r.URL)
			}
			return tagWire(200, &tagBody{Reader: strings.NewReader(raw)}), nil
		})
		v, e := tagScope(t, client, tagParent).All(context.Background(), tags.WithListLimit(1))
		if v != nil || e == nil || calls.Load() != 2 {
			t.Fatal(v, e, calls.Load())
		}
		tagProof(t, e, 200, []byte(raw))
	})
	for _, mode := range []string{"All", "cap", "break"} {
		t.Run(mode+" consumed rows", func(t *testing.T) {
			raw := `{"tags":[{"name":"good"},null],"next":42}`
			var calls atomic.Int32
			client := tagClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return tagWire(200, &tagBody{Reader: strings.NewReader(raw)}), nil
			})
			s := tagScope(t, client, tagParent)
			if mode == "break" {
				for v, e := range s.List(context.Background()) {
					if e != nil || v == nil {
						t.Fatal(v, e)
					}
					break
				}
			} else {
				options := []tags.ListOption{}
				if mode == "cap" {
					options = append(options, tags.WithListMaxItems(1))
				}
				v, e := s.All(context.Background(), options...)
				if mode == "cap" {
					if e != nil || len(v) != 1 {
						t.Fatal(v, e)
					}
				} else {
					if v != nil {
						t.Fatal(v, e)
					}
					tagProof(t, e, 200, []byte(raw))
				}
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
	for _, raw := range []string{`{"tags":[{"name":"good"},]}`, "{\"tags\":[{\"name\":\"good\"}],\"unknown\":\"\xff\"}"} {
		t.Run("full syntax before rows "+raw, func(t *testing.T) {
			client := tagClient(func(*http.Request) (*http.Response, error) {
				return tagWire(200, &tagBody{Reader: strings.NewReader(raw)}), nil
			})
			seen := 0
			for v, e := range tagScope(t, client, tagParent).List(context.Background(), tags.WithListMaxItems(1)) {
				if e == nil || v != nil {
					t.Fatal(v, e)
				}
				tagProof(t, e, 200, []byte(raw))
				seen++
			}
			if seen != 1 {
				t.Fatal(seen)
			}
		})
	}
	t.Run("lazy parallel reusable iterator", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		headers := map[string]string{"X-Own": "initial"}
		limit, marker, key, dir := 0, "", "", "asc"
		full := tags.WithListOpts(tags.ListOpts{Headers: headers, Limit: &limit, Marker: &marker, SortKey: &key, SortDir: &dir})
		headers["X-Own"] = "late"
		limit = 42
		marker = "late"
		key = "late"
		dir = "desc"
		client := tagClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			q := r.URL.Query()
			if r.Header.Get("X-Own") != "initial" || q.Get("limit") != "0" || q.Get("marker") != "" || q.Get("sort_key") != "" || q.Get("sort_dir") != "asc" {
				t.Error(r.URL, r.Header)
			}
			return tagWire(200, &tagBody{Reader: strings.NewReader(`{"tags":[]}`)}), nil
		})
		options := []tags.ListOption{full, func(o *tags.ListOpts) error { callbacks.Add(1); return nil }}
		seq := tagScope(t, client, tagParent).List(context.Background(), options...)
		options[0] = nil
		options[1] = nil
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("not lazy")
		}
		var wg sync.WaitGroup
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for v, e := range seq {
					if e != nil || v != nil {
						t.Error(v, e)
					}
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 2 || callbacks.Load() != 2 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"source between rows", "cancel between rows", "new headers next page"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("between rows")
			var calls atomic.Int32
			client := tagClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				raw := `{"tags":[{"name":"a"},{"name":"b"}]}`
				if mode == "new headers next page" {
					raw = `{"tags":[{"name":"a"}]}`
					if n == 2 {
						raw = `{"tags":[]}`
						if r.Header.Get("X-Source") != "next" || r.Header.Get("X-Auth-Token") != "new-token" {
							t.Error(r.Header)
						}
					}
				}
				return tagWire(200, &tagBody{Reader: strings.NewReader(raw)}), nil
			})
			client.MoreHeaders = map[string]string{"X-Source": "first"}
			s := tagScope(t, client, tagParent)
			options := []tags.ListOption{}
			if mode == "new headers next page" {
				options = append(options, tags.WithListLimit(1))
			}
			seen, failures := 0, 0
			for v, e := range s.List(ctx, options...) {
				if e != nil {
					failures++
					if mode == "source between rows" && !errors.Is(e, resource.ErrInvalidOption) || mode == "cancel between rows" && (!errors.Is(e, cause) || !errors.Is(e, context.Canceled)) {
						t.Fatal(e)
					}
					continue
				}
				seen++
				if v == nil {
					t.Fatal(v)
				}
				switch mode {
				case "source between rows":
					client.Type = "compute"
				case "cancel between rows":
					cancel(cause)
				case "new headers next page":
					client.MoreHeaders = map[string]string{"X-Source": "next"}
					client.SetToken("new-token")
				}
			}
			if mode == "new headers next page" {
				if seen != 1 || failures != 0 || calls.Load() != 2 {
					t.Fatal(seen, failures, calls.Load())
				}
			} else if seen != 1 || failures != 1 || calls.Load() != 1 {
				t.Fatal(seen, failures, calls.Load())
			}
		})
	}
}
