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

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const imageTaskContractObject = `{"id":"response-task","image_id":"response-parent","type":"future","status":"future-state","owner":"owner","request_id":"canonical-request","user_id":"canonical-user","message":"","self":"https://passive.invalid/self","schema":"https://passive.invalid/schema","created_at":"literal-created","updated_at":"literal-updated","expires_at":"literal-expiry","deleted_at":"literal-deleted","deleted":false,"input":{"number":9007199254740993,"huge":1e1000},"result":{},"request-id":17,"user":false,"links":42,"x-number":9007199254740995}`
const imageTaskContractList = `{"tasks":[` + imageTaskContractObject + `],"next":17,"first":"https://foreign.invalid/tasks","schema":false}`

func imageTaskContractRows(s *image.Service, ctx context.Context, ref resource.Ref, iterator bool, options ...image.ListImageTasksOption) ([]*image.ImageTaskInfo, error) {
	if !iterator {
		return s.AllImageTasks(ctx, ref, options...)
	}
	rows := make([]*image.ImageTaskInfo, 0)
	for row, err := range s.ImageTasks(ctx, ref, options...) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func TestImageTaskContractsFixedRouteAndParentIdentity(t *testing.T) {
	for _, iterator := range []bool{false, true} {
		for _, version := range []string{"", "2.12", "2.18"} {
			t.Run(fmt.Sprintf("direct finite iterator%t version%s", iterator, version), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("image", "/unused/catalog/")
				client.ResourceBase = cloud.Server.URL + taskContractsPrefix
				client.Microversion = version
				client.MoreHeaders = map[string]string{"X-Source": "original"}
				cloud.Provider.SetToken("live")
				id := "literal:%2F ?#한글"
				var calls atomic.Int32
				cloud.Mux.HandleFunc(taskContractsPrefix, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != "GET" || r.URL.EscapedPath() != taskContractsPrefix+"images/"+url.PathEscape(id)+"/tasks" || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "original" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "last" || r.Header.Get("X-Auth-Token") != "live" {
						t.Error(r.Method, r.URL, r.Header)
					}
					body, _ := io.ReadAll(r.Body)
					if len(body) != 0 {
						t.Error(string(body))
					}
					wantVersion := ""
					if version != "" {
						wantVersion = "image " + version
					}
					if r.Header.Get("OpenStack-API-Version") != wantVersion {
						t.Error(r.Header)
					}
					w.Header().Set("X-Request-Id", "actual-task")
					w.Header().Set("Link", `<https://foreign.invalid/tasks>; rel="next"`)
					w.Header().Set("Location", "https://foreign.invalid/tasks")
					_, _ = io.WriteString(w, imageTaskContractList)
				})
				rows, err := imageTaskContractRows(image.New(client), context.Background(), resource.ID(id), iterator, image.WithListImageTasksHeaders(map[string]string{"X-Option": "owned"}), image.WithListImageTasksHeader("X-Final", "last"))
				if err != nil || len(rows) != 1 || *rows[0].ID != "response-task" || *rows[0].ImageID != "response-parent" || rows[0].StatusCode != 200 || calls.Load() != 1 {
					t.Fatal(rows, err, calls.Load())
				}
			})
		}
	}
	t.Run("long literal ID has no local length or UUID gate", func(t *testing.T) {
		id := strings.Repeat("한", 300) + ":% ?#"
		var calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.EscapedPath() != taskContractsPrefix+"images/"+url.PathEscape(id)+"/tasks" || r.URL.RawQuery != "" || r.Body != nil {
				t.Error(r.URL, r.Body)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[]}`)}), nil
		})
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID(id))
		if rows == nil || err != nil || calls.Load() != 1 {
			t.Fatal(rows, err, calls.Load())
		}
	})
	for _, id := range []string{"", ".", "..", "a/b", "a\\b", "line\n", string([]byte{0xff})} {
		t.Run("unsafe literal ID "+id, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("unexpected request")
			})
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID(id), func(*image.ListImageTasksOpts) error { callbacks.Add(1); return nil })
			if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(rows, err, calls.Load(), callbacks.Load())
			}
		})
	}
	t.Run("exact Name completes all pages without parent GET", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", taskContractsPrefix)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc(taskContractsPrefix, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			page := calls.Add(1)
			if r.Method != "GET" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" {
				t.Error(r.Method, r.URL, r.Header)
			}
			switch page {
			case 1:
				if r.URL.Path != taskContractsPrefix+"images" || r.URL.Query().Get("name") != "exact" {
					t.Error(r.URL)
				}
				cloud.Provider.SetToken("refreshed")
				_, _ = io.WriteString(w, `{"images":[{"id":"chosen","name":"exact"},{"id":"prefix","name":"exact suffix"}],"next":"/v2/images?name=exact&marker=second"}`)
			case 2:
				if r.URL.Path != taskContractsPrefix+"images" || r.URL.Query().Get("marker") != "second" || r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Error(r.URL, r.Header)
				}
				_, _ = io.WriteString(w, `{"images":[{"id":"other","name":"other"}]}`)
			case 3:
				if r.URL.Path != taskContractsPrefix+"images/chosen/tasks" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Error(r.URL, r.Header)
				}
				_, _ = io.WriteString(w, imageTaskContractList)
			default:
				t.Error("unexpected lookup/task replay", page)
			}
		})
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.Name("exact"), func(o *image.ListImageTasksOpts) error {
			callbacks.Add(1)
			o.Headers["X-Option"] = "owned"
			client.MoreHeaders["X-Source"] = "after callback"
			return nil
		})
		if err != nil || len(rows) != 1 || calls.Load() != 3 || callbacks.Load() != 1 {
			t.Fatal(rows, err, calls.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"missing", "ambiguous", "late HTTP", "late native model", "unsafe chosen ID"} {
		t.Run("Name failure "+mode, func(t *testing.T) {
			var calls, taskGET atomic.Int32
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.URL.Path != taskContractsPrefix+"images" {
					taskGET.Add(1)
					t.Error(r.URL)
				}
				code, raw := 200, `{"images":[]}`
				switch mode {
				case "ambiguous":
					raw = `{"images":[{"id":"one","name":"exact"},{"id":"two","name":"exact"}]}`
				case "unsafe chosen ID":
					raw = `{"images":[{"id":"../bad","name":"exact"}]}`
				case "late HTTP", "late native model":
					if n == 1 {
						raw = `{"images":[{"id":"chosen","name":"exact"}],"next":"/v2/images?marker=next"}`
					} else if mode == "late HTTP" {
						code, raw = 503, "late lookup error"
					} else {
						raw = `{"images":[{"id":"other","name":"other","created_at":"not-a-native-time"}]}`
					}
				}
				return taskContractsWire(code, &taskContractsBody{Reader: strings.NewReader(raw)}), nil
			})
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.Name("exact"))
			if rows != nil || err == nil || taskGET.Load() != 0 {
				t.Fatal(rows, err, calls.Load(), taskGET.Load())
			}
			if mode == "missing" && !errors.Is(err, resource.ErrNotFound) || mode == "ambiguous" && !errors.Is(err, resource.ErrAmbiguous) || mode == "unsafe chosen ID" && !errors.Is(err, resource.ErrInvalidOption) || mode == "late HTTP" && !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageTaskContractsCanonicalDetailAndRawOwnership(t *testing.T) {
	t.Run("explicit deletion decoder and literal detail ownership", func(t *testing.T) {
		raw := []byte(imageTaskContractList)
		wire := taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)})
		client := taskContractsClient(func(*http.Request) (*http.Response, error) { return wire, nil })
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
		v := rows[0]
		if v.Deleted == nil || *v.Deleted || v.DeletedAt == nil || *v.DeletedAt != "literal-deleted" || *v.CreatedAt != "literal-created" || *v.UpdatedAt != "literal-updated" || *v.ExpiresAt != "literal-expiry" || *v.RequestID != "canonical-request" || *v.UserID != "canonical-user" || *v.Type != "future" || *v.Status != "future-state" || *v.Message != "" || v.Links != nil || v.Result == nil || string(v.Input["number"]) != "9007199254740993" || string(v.Input["huge"]) != "1e1000" || string(v.Body["request-id"]) != "17" || string(v.Body["user"]) != "false" {
			t.Fatal(v)
		}
		raw[0] = '!'
		wire.Header.Set("X-Request-Id", "late")
		v.Input["number"][0] = '1'
		*v.ImageID = "typed parent"
		*v.Deleted = true
		if string(v.Body["deleted"]) != "false" || string(v.Body["image_id"]) != `"response-parent"` || !strings.Contains(string(v.Body["input"]), "9007199254740993") || v.Header.Get("X-Request-Id") != "actual-task" {
			t.Fatal("borrowed/typed/raw alias", v)
		}
		v.Body["result"] = json.RawMessage(`{"changed":true}`)
		if len(v.Result) != 0 {
			t.Fatal("Result aliases raw")
		}
	})
	for _, raw := range []string{`{}`, `{"deleted":null,"deleted_at":null,"input":null,"result":null}`, `{"Deleted":17,"DeletedAt":false,"request-id":{},"user":[]}`, `{"deleted":false,"deleted_at":"","input":{},"result":{}}`, `{"deleted":true,"deleted_at":"literal-not-a-date"}`} {
		t.Run("nullable deletion "+raw, func(t *testing.T) {
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[` + raw + `]}`)}), nil
			})
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
			if err != nil || len(rows) != 1 {
				t.Fatal(rows, err)
			}
			v := rows[0]
			if strings.Contains(raw, `"deleted":false`) {
				if v.Deleted == nil || *v.Deleted || v.DeletedAt == nil || *v.DeletedAt != "" || v.Input == nil || v.Result == nil {
					t.Fatal(v)
				}
			} else if strings.Contains(raw, `"deleted":true`) {
				if v.Deleted == nil || !*v.Deleted || *v.DeletedAt != "literal-not-a-date" {
					t.Fatal(v)
				}
			} else if v.Deleted != nil || v.DeletedAt != nil || v.ID != nil || v.ImageID != nil || v.RequestID != nil || v.UserID != nil {
				t.Fatal("null/decoy/seeding", v)
			}
		})
	}
	for _, row := range []string{`{"deleted":0}`, `{"deleted":"false"}`, `{"deleted_at":false}`, `{"input":[]}`, `{"result":true}`, `{"image_id":17}`, `{"created_at":[]}`} {
		t.Run("wrong canonical "+row, func(t *testing.T) {
			raw := []byte(`{"tasks":[` + row + `]}`)
			body := &taskContractsBody{Reader: bytes.NewReader(raw)}
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { return taskContractsWire(200, body), nil })
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
			taskContractsProof(t, err, 200, raw)
			if rows != nil || body.closes.Load() != 1 {
				t.Fatal(rows, err, body.closes.Load())
			}
		})
	}
	t.Run("public composite decoder is atomic", func(t *testing.T) {
		var value image.ImageTaskInfo
		if err := json.Unmarshal([]byte(imageTaskContractObject), &value); err != nil {
			t.Fatal(err)
		}
		before, _ := json.Marshal(value.Body)
		for _, bad := range []string{`{"id":"replaced","deleted":17}`, `{"id":"replaced","deleted_at":[]}`, `{"id":"replaced","input":[]}`} {
			if err := json.Unmarshal([]byte(bad), &value); err == nil {
				t.Fatal("accepted bad field", bad)
			}
			after, _ := json.Marshal(value.Body)
			if *value.ID != "response-task" || value.Deleted == nil || *value.Deleted || *value.DeletedAt != "literal-deleted" || !bytes.Equal(before, after) {
				t.Fatal("partial composite mutation", value)
			}
		}
	})
}

func TestImageTaskContractsLazyConsumptionAndLocalCaps(t *testing.T) {
	for _, mode := range []string{"unlimited", "cap", "break"} {
		t.Run("finite rows passive continuation "+mode, func(t *testing.T) {
			var calls atomic.Int32
			raw := `{"tasks":[{"id":"first"},{"id":"second"}],"next":"https://foreign.invalid/tasks","first":17,"schema":false}`
			if mode != "unlimited" {
				raw = `{"tasks":[{"id":"first"},null,{"deleted":17}],"next":17}`
			}
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.RawQuery != "" || r.Body != nil {
					t.Error(r.URL, r.Body)
				}
				wire := taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(raw)})
				wire.Header.Set("Link", `<https://foreign.invalid/tasks>; rel="next"`)
				return wire, nil
			})
			s := image.New(client)
			count := 0
			if mode == "break" {
				for row, err := range s.ImageTasks(context.Background(), resource.ID("chosen")) {
					if err != nil || row == nil || *row.ID != "first" {
						t.Fatal(row, err)
					}
					count++
					break
				}
			} else {
				option := image.WithListImageTasksMaxItems(0)
				if mode == "cap" {
					option = image.WithListImageTasksMaxItems(1)
				}
				rows, err := s.AllImageTasks(context.Background(), resource.ID("chosen"), option)
				if err != nil || rows == nil {
					t.Fatal(rows, err)
				}
				count = len(rows)
				if count > 1 {
					rows[0].Header.Set("X-Request-Id", "first changed")
					if rows[1].Header.Get("X-Request-Id") != "actual-task" {
						t.Fatal("rows share headers")
					}
				}
			}
			want := 1
			if mode == "unlimited" {
				want = 2
			}
			if count != want || calls.Load() != 1 {
				t.Fatal(count, calls.Load())
			}
		})
	}
	for _, raw := range [][]byte{[]byte(`{}`), []byte(`null`), []byte(`[]`), []byte(`{"Tasks":[]}`), []byte(`{"tasks":null}`), []byte(`{"tasks":{}}`), []byte(`{"tasks":[{},null]}`), []byte(`{"tasks":[{}]`), []byte(`{"tasks":[{}],"unused":"` + string([]byte{0xff}) + `"}`)} {
		t.Run(fmt.Sprintf("whole envelope%x", raw), func(t *testing.T) {
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
			})
			options := []image.ListImageTasksOption{}
			if !bytes.Contains(raw, []byte("null]")) {
				options = append(options, image.WithListImageTasksMaxItems(1))
			}
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"), options...)
			taskContractsProof(t, err, 200, raw)
			if rows != nil {
				t.Fatal(rows, err)
			}
		})
	}
	t.Run("empty result nonnil despite invalid passive next", func(t *testing.T) {
		var calls atomic.Int32
		client := taskContractsClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[],"next":{},"links":false}`)}), nil
		})
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
		if err != nil || rows == nil || len(rows) != 0 || calls.Load() != 1 {
			t.Fatal(rows, err, calls.Load())
		}
	})
	t.Run("later consumed error yields nil row and whole proof", func(t *testing.T) {
		raw := []byte(`{"tasks":[{"id":"first"},{"deleted_at":17}]}`)
		client := taskContractsClient(func(*http.Request) (*http.Response, error) {
			return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
		})
		count := 0
		var terminal error
		for row, err := range image.New(client).ImageTasks(context.Background(), resource.ID("chosen")) {
			if err != nil {
				if row != nil {
					t.Fatal(row, err)
				}
				terminal = err
				continue
			}
			count++
			if *row.ID != "first" {
				t.Fatal(row)
			}
		}
		taskContractsProof(t, terminal, 200, raw)
		if count != 1 {
			t.Fatal(count)
		}
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
		taskContractsProof(t, err, 200, raw)
		if rows != nil {
			t.Fatal(rows, err)
		}
	})
	t.Run("copied option slice lazy and parallel reusable", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Choice") != "captured" {
				t.Error(r.Header)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(imageTaskContractList)}), nil
		})
		options := []image.ListImageTasksOption{func(o *image.ListImageTasksOpts) error {
			callbacks.Add(1)
			o.Headers["X-Choice"] = "captured"
			return nil
		}}
		seq := image.New(client).ImageTasks(context.Background(), resource.ID("chosen"), options...)
		options[0] = image.WithListImageTasksHeader("X-Choice", "late")
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("eager iterator")
		}
		var wg sync.WaitGroup
		for range 3 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				n := 0
				for row, err := range seq {
					if row == nil || err != nil {
						t.Error(row, err)
					}
					n++
				}
				if n != 1 {
					t.Error(n)
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 3 || callbacks.Load() != 3 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
}

func TestImageTaskContractsOwnedOptionsAndSource(t *testing.T) {
	t.Run("factory snapshot full replacement and retained callback isolation", func(t *testing.T) {
		headers := map[string]string{"X-Choice": "factory"}
		option := image.WithListImageTasksOpts(image.ListImageTasksOpts{Headers: headers, MaxItems: 1})
		headers["X-Choice"] = "late"
		var retained *image.ListImageTasksOpts
		var callbacks, calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Choice") != "factory" || r.Header.Get("X-Discard") != "" || r.Header.Get("X-Final") != "last" || r.URL.RawQuery != "" {
				t.Error(r.URL, r.Header)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(`{"tasks":[{},null]}`)}), nil
		})
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"), image.WithListImageTasksHeader("X-Discard", "removed"), option, image.WithListImageTasksHeader("X-Final", "first"), image.WithListImageTasksHeader("X-Final", "last"), func(o *image.ListImageTasksOpts) error { callbacks.Add(1); retained = o; return nil }, func(*image.ListImageTasksOpts) error {
			callbacks.Add(1)
			retained.Headers["X-Choice"] = "retained late"
			retained.MaxItems = 0
			return nil
		})
		if err != nil || len(rows) != 1 || callbacks.Load() != 2 || calls.Load() != 1 {
			t.Fatal(rows, err, callbacks.Load(), calls.Load())
		}
	})
	for _, mode := range []string{"nil context", "canceled context", "nil service", "nil provider", "wrong service", "bad endpoint", "source header", "unsafe Ref", "bad UTF8 Ref"} {
		t.Run("zero-work preflight "+mode, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			cause := errors.New("context cause")
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("unexpected request")
			})
			s := image.New(client)
			ctx := context.Background()
			ref := resource.ID("chosen")
			want := error(resource.ErrInvalidOption)
			switch mode {
			case "nil context":
				ctx = nil
			case "canceled context":
				c, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx, want = c, cause
			case "nil service":
				s = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong service":
				client.Type = "compute"
				want = resource.ErrUnsupported
			case "bad endpoint":
				client.Endpoint = "https://glance.invalid/v2/?q=1"
			case "source header":
				client.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
			case "unsafe Ref":
				ref = resource.ID("../unsafe")
			case "bad UTF8 Ref":
				ref = resource.Name(string([]byte{0xff}))
			}
			rows, err := s.AllImageTasks(ctx, ref, func(*image.ListImageTasksOpts) error { callbacks.Add(1); return nil })
			if rows != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(rows, err, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, option := range []image.ListImageTasksOption{nil, image.WithListImageTasksMaxItems(-1), image.WithListImageTasksHeader("Authorization", "forged"), image.WithListImageTasksHeader("Content-Type", "text/plain"), image.WithListImageTasksHeaders(map[string]string{"X-A": "first", "x-a": "second"}), image.WithListImageTasksHeader("X-A", "bad\nvalue")} {
		t.Run("invalid option precedes Name HTTP", func(t *testing.T) {
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected lookup") })
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.Name("exact"), option)
			if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(rows, err, calls.Load())
			}
		})
	}
	t.Run("callback failure keeps cause and stops later callback", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		cause := errors.New("option callback")
		client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.Name("exact"), func(*image.ListImageTasksOpts) error { callbacks.Add(1); return cause }, func(*image.ListImageTasksOpts) error { callbacks.Add(100); return nil })
		if rows != nil || !errors.Is(err, cause) || calls.Load() != 0 || callbacks.Load() != 1 {
			t.Fatal(rows, err, calls.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"provider", "endpoint with stable base", "base", "type", "microversion"} {
		t.Run("fixed target during callback "+mode, func(t *testing.T) {
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected wire") })
			client.ResourceBase = taskContractsBase
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.Name("exact"), func(*image.ListImageTasksOpts) error {
				switch mode {
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "endpoint with stable base":
					client.Endpoint = "https://other.invalid/catalog/"
				case "base":
					client.ResourceBase = "https://other.invalid/v2/"
				case "type":
					client.Type = "compute"
				case "microversion":
					client.Microversion = "2.19"
				}
				return nil
			})
			if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(rows, err, calls.Load())
			}
		})
	}
	t.Run("Name accepted source drift and original lookup error both retained", func(t *testing.T) {
		var calls atomic.Int32
		var client *gophercloud.ServiceClient
		client = taskContractsClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			body := &taskContractsBody{Reader: strings.NewReader("lookup unavailable"), onClose: func() { client.Microversion = "2.19" }}
			return taskContractsWire(503, body), nil
		})
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.Name("exact"))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 {
			t.Fatal(rows, err, calls.Load())
		}
	})
	t.Run("source snapshot before callback and live token", func(t *testing.T) {
		var calls atomic.Int32
		client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Source") != "before" || r.Header.Get("X-Auth-Token") != "latest" {
				t.Error(r.Header)
			}
			return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(imageTaskContractList)}), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "before"}
		rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"), func(*image.ListImageTasksOpts) error {
			client.MoreHeaders["X-Source"] = "after"
			client.SetToken("latest")
			return nil
		})
		if err != nil || len(rows) != 1 || calls.Load() != 1 {
			t.Fatal(rows, err, calls.Load())
		}
	})
	t.Run("source checked after yielded row but break stops immediately", func(t *testing.T) {
		for _, breakNow := range []bool{false, true} {
			raw := []byte(`{"tasks":[{"id":"first"},{"id":"second"}]}`)
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
			})
			count := 0
			var terminal error
			for row, err := range image.New(client).ImageTasks(context.Background(), resource.ID("chosen")) {
				if err != nil {
					if row != nil {
						t.Fatal(row, err)
					}
					terminal = err
					continue
				}
				count++
				client.Endpoint = "https://other.invalid/v2/"
				if breakNow {
					break
				}
			}
			if count != 1 {
				t.Fatal(count)
			}
			if breakNow {
				if terminal != nil {
					t.Fatal(terminal)
				}
			} else {
				taskContractsProof(t, terminal, 200, raw)
				if !errors.Is(terminal, resource.ErrInvalidOption) {
					t.Fatal(terminal)
				}
			}
		}
	})
}

func TestImageTaskContractsResponseOwnershipAndNativeHooks(t *testing.T) {
	for _, iterator := range []bool{false, true} {
		for _, failure := range []string{"read", "close", "context", "combined opaque", "source"} {
			t.Run(fmt.Sprintf("accepted iterator%t %s", iterator, failure), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("read cause"), errors.New("Close cause"), errors.New("context custom cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				raw := []byte(imageTaskContractList)
				if failure == "read" {
					raw = []byte("partial-before-read-error")
				}
				if failure == "combined opaque" {
					raw = []byte{'o', 'p', 'a', 'q', 'u', 'e', 0xff}
				}
				body := &taskContractsBody{Reader: bytes.NewReader(raw)}
				if failure == "read" || failure == "combined opaque" {
					body.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if failure == "close" || failure == "combined opaque" {
					body.closeErr = closeCause
				}
				var calls, hooks atomic.Int32
				var client *gophercloud.ServiceClient
				body.onClose = func() {
					if failure == "context" || failure == "combined opaque" {
						cancel(cancelCause)
					}
					if failure == "source" {
						client.Microversion = "2.19"
					}
				}
				client = taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(200, body), nil })
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return nil
				}
				rows, err := imageTaskContractRows(image.New(client), ctx, resource.ID("chosen"), iterator)
				taskContractsProof(t, err, 200, raw)
				if rows != nil || calls.Load() != 1 || hooks.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(rows, err, calls.Load(), hooks.Load(), body.closes.Load())
				}
				if (failure == "read" || failure == "combined opaque") && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if (failure == "close" || failure == "combined opaque") && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if (failure == "context" || failure == "combined opaque") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal(err)
				}
				if failure == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
	for _, code := range []int{201, 202, 204, 403, 404} {
		t.Run(fmt.Sprintf("strict actual %d", code), func(t *testing.T) {
			var calls atomic.Int32
			body := &taskContractsBody{Reader: strings.NewReader("actual failure")}
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(code, body), nil })
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if rows != nil || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != "GET" || native.URL != taskContractsBase+"images/chosen/tasks" || string(native.Body) != "actual failure" || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || errors.As(err, &accepted) || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(rows, err, native, calls.Load(), body.closes.Load())
			}
		})
	}
	t.Run("prebody retry preserves original provider and bodyless route", func(t *testing.T) {
		for _, mode := range []string{"503", "transport", "reauth"} {
			t.Run(mode, func(t *testing.T) {
				cause := errors.New("transport cause")
				var calls, hooks atomic.Int32
				var bodies []*taskContractsBody
				var client *gophercloud.ServiceClient
				client = taskContractsClient(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					if r.Method != "GET" || r.URL.String() != taskContractsBase+"images/chosen/tasks" || r.Body != nil {
						t.Error(r.Method, r.URL, r.Body)
					}
					if n == 1 && mode == "transport" {
						return nil, cause
					}
					code, raw := 200, imageTaskContractList
					if n == 1 {
						code, raw = 503, "original503"
						if mode == "reauth" {
							code = 401
						}
					}
					if n == 2 && (r.Header.Get("X-Auth-Token") != "refreshed" || mode != "reauth" && r.Header.Get("X-Hook") != "advanced") {
						t.Error(r.Header)
					}
					b := &taskContractsBody{Reader: strings.NewReader(raw)}
					bodies = append(bodies, b)
					return taskContractsWire(code, b), nil
				})
				if mode == "reauth" {
					client.ReauthFunc = func(context.Context) error { hooks.Add(1); client.SetToken("refreshed"); return nil }
				} else {
					client.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
						hooks.Add(1)
						if method != "GET" || target != taskContractsBase+"images/chosen/tasks" || o.JSONBody != nil || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody || mode == "503" && !gophercloud.ResponseCodeIs(original, 503) || mode == "transport" && !errors.Is(original, cause) {
							t.Error(original, o)
						}
						o.MoreHeaders = map[string]string{"X-Hook": "advanced"}
						client.SetToken("refreshed")
						return nil
					}
				}
				provider := client.ProviderClient
				rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
				if err != nil || len(rows) != 1 || calls.Load() != 2 || hooks.Load() != 1 || client.ProviderClient != provider {
					t.Fatal(rows, err, calls.Load(), hooks.Load())
				}
				for _, b := range bodies {
					if b.closes.Load() != 1 {
						t.Fatal(b.closes.Load())
					}
				}
			})
		}
	})
	for _, change := range []string{"JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSON", "expanded OkCodes"} {
		t.Run("shared request guard "+change, func(t *testing.T) {
			var calls, hooks, borrowedReads atomic.Int32
			callbackCause, readCause, closeCause, cancelCause := errors.New("hook cause"), errors.New("expanded Read cause"), errors.New("expanded Close cause"), errors.New("expanded context cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			borrowed := &taskContractsBody{Reader: taskContractsReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			var bodies []*taskContractsBody
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Body != nil {
					t.Error("request acquired body", r.Body)
				}
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 204, "private204"
				}
				b := &taskContractsBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return taskContractsWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) {
					t.Error(original)
				}
				switch change {
				case "JSON null":
					o.JSONBody = json.RawMessage(`null`)
				case "KeepResponseBody":
					o.KeepResponseBody = false
				case "JSONResponse":
					o.JSONResponse = new(any)
				case "RawBody":
					o.RawBody = borrowed
				case "unsupported JSON":
					o.JSONBody = make(chan int)
				case "expanded OkCodes":
					o.OkCodes = []int{200, 204}
					return nil
				}
				return callbackCause
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			rows, err := image.New(client).AllImageTasks(ctx, resource.ID("chosen"))
			if change == "expanded OkCodes" {
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if rows != nil || !errors.As(err, &native) || native.Actual != 204 || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != "GET" || native.URL != taskContractsBase+"images/chosen/tasks" || string(native.Body) != "private204" || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || errors.As(err, &accepted) || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || calls.Load() != 2 {
					t.Fatal(rows, err, native, calls.Load())
				}
			} else if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
			if change == "unsupported JSON" {
				var typed *json.UnsupportedTypeError
				if !errors.As(err, &typed) {
					t.Fatal(err)
				}
			}
			if hooks.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatal(hooks.Load(), borrowedReads.Load(), borrowed.closes.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, mode := range []string{"transport", "callback", "reauth"} {
		t.Run("no nested missing suppression "+mode, func(t *testing.T) {
			cause := errors.New("nested cause")
			nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}}
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth" {
					code = 401
				}
				return taskContractsWire(code, &taskContractsBody{Reader: strings.NewReader("native failure")}), nil
			})
			if mode == "callback" {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth" {
				client.ReauthFunc = func(context.Context) error { return errors.Join(cause, nested) }
			}
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
			if rows != nil || err == nil || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
			if mode == "reauth" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) || !errors.Is(native.ErrReauth, nested) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) || !errors.Is(err, nested) || mode == "callback" && !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal(err)
			}
		})
	}
	for _, mode := range []string{"same target", "foreign target", "other query", "changed method"} {
		t.Run("native redirect policy "+mode, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*taskContractsBody
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Method != "GET" || r.URL.String() != taskContractsBase+"images/chosen/tasks" || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				b := &taskContractsBody{Reader: strings.NewReader(imageTaskContractList)}
				bodies = append(bodies, b)
				if n == 2 {
					return taskContractsWire(200, b), nil
				}
				wire := taskContractsWire(307, b)
				target := taskContractsBase + "images/chosen/tasks"
				if mode == "foreign target" {
					target = "https://foreign.invalid/images/chosen/tasks"
				}
				if mode == "other query" {
					target += "?limit=1"
				}
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				if mode == "changed method" {
					r.Method = "POST"
				}
				return nil
			}
			rows, err := image.New(client).AllImageTasks(context.Background(), resource.ID("chosen"))
			if mode == "same target" {
				if err != nil || len(rows) != 1 || calls.Load() != 2 {
					t.Fatal(rows, err, calls.Load())
				}
			} else if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
			if redirects.Load() != 1 {
				t.Fatal(redirects.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
}
