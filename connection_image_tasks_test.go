package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageTaskFacadeSharesClientAndOwnsInput(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("task-0")
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "작업% ?#"
	const accepted = `{"id":"server-id","type":"future","status":"pending","input":{"n":9007199254740993},"result":{},"self":"https://foreign.test/task"}`
	calls := 0
	var source *gophercloud.ServiceClient
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		wantMethod, wantPath := http.MethodGet, endpoint+"tasks"
		if n == 0 || n == 5 {
			wantMethod = http.MethodPost
		} else if n == 1 || n == 6 {
			wantPath += "/" + url.PathEscape(id)
		}
		if r.Method != wantMethod || r.URL.Scheme+"://"+r.URL.Host+r.URL.EscapedPath() != wantPath || r.Header.Get("X-Auth-Token") != fmt.Sprintf("task-%d", n) || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", n) {
			t.Fatalf("request%d: %s %s headers=%v", n, r.Method, r.URL, r.Header)
		}
		code, body := http.StatusOK, accepted
		switch n {
		case 0, 5:
			raw, err := io.ReadAll(r.Body)
			var input struct {
				Type  string                     `json:"type"`
				Input map[string]json.RawMessage `json:"input"`
			}
			if err != nil || json.Unmarshal(raw, &input) != nil || input.Type != "future" || input.Input == nil {
				t.Fatalf("create body=%s err=%v", raw, err)
			}
			if n == 0 && string(input.Input["n"]) != "9007199254740993" || n == 5 && len(input.Input) != 0 {
				t.Fatalf("owned create input=%s", raw)
			}
			code = http.StatusCreated
		case 1:
			// Return the same actual object for the requested literal ID.
		case 2, 3:
			q := r.URL.Query()
			if q.Get("type") != "future" || q.Get("limit") != "0" || q.Get("status") != "future" || r.Header.Get("X-Call") != "stable" || len(q) != 3+n-2 {
				t.Fatalf("task filters/headers=%s %v", r.URL, r.Header)
			}
			if n == 2 {
				body = `{"tasks":[{"id":"first","status":"pending"}],"next":"/v2/tasks?limit=0&status=future&type=future&marker=next"}`
			} else {
				if q.Get("marker") != "next" {
					t.Fatal("canonical continuation lost marker")
				}
				body = `{"tasks":[{"id":"second"}]}`
			}
		case 4:
			body = `{"tasks":[]}`
		case 6:
			code, body = http.StatusNotFound, "task hidden or missing"
		default:
			t.Fatalf("unexpected hidden task operation: %s %s", r.Method, r.URL)
		}
		provider.SetToken(fmt.Sprintf("task-%d", calls))
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		return &http.Response{StatusCode: code, Header: http.Header{"X-Proof": {fmt.Sprint(n)}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	upper, err := conn.Image(ctx)
	if err != nil {
		t.Fatal(err)
	}
	version, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	source = upper.RawClient()
	if version.RawClient() != source || upper.API.Tasks.RawClient() != source || version.Tasks.RawClient() != source || source.ProviderClient != provider {
		t.Fatal("Task facade replaced the shared image client")
	}
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	raw := json.RawMessage(`9007199254740993`)
	values := map[string]any{"n": raw}
	option := image.WithCreateTaskInput(values)
	raw[0] = '1'
	delete(values, "n")
	created, err := upper.CreateTask(ctx, "future", option)
	if err != nil || created == nil || created.ID == nil || *created.ID != "server-id" || created.StatusCode != 201 || string(created.Input["n"]) != "9007199254740993" || len(created.Result) != 0 || created.Result == nil {
		t.Fatalf("create response=%+v err=%v", created, err)
	}
	created.Input["n"][0] = '1'
	if string(created.Body["input"]) != `{"n":9007199254740993}` {
		t.Fatal("typed input aliases actual response body")
	}
	fetched, err := upper.GetTask(ctx, id)
	if err != nil || fetched == nil || fetched.ID == nil || *fetched.ID != "server-id" || fetched.StatusCode != 200 {
		t.Fatalf("GET must retain actual response ID: %+v %v", fetched, err)
	}
	var ids []string
	for task, err := range upper.Tasks(ctx, image.WithListTasksType("future"), image.WithListTasksStatus("future"), image.WithListTasksLimit(0), image.WithListTasksHeader("X-Call", "stable")) {
		if err != nil || task == nil || task.ID == nil || task.Input != nil || task.Result != nil {
			t.Fatalf("list stub=%+v err=%v", task, err)
		}
		ids = append(ids, *task.ID)
	}
	if strings.Join(ids, ",") != "first,second" || calls != 4 {
		t.Fatalf("pagination ids=%v calls=%d", ids, calls)
	}
	empty, err := upper.AllTasks(ctx)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty AllTasks=%v err=%v", empty, err)
	}
	if task, err := upper.CreateTask(ctx, "future"); err != nil || task == nil {
		t.Fatalf("default input: %+v %v", task, err)
	}
	missing, err := upper.GetTask(ctx, id)
	var native gophercloud.ErrUnexpectedResponseCode
	if missing != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "task hidden or missing" || calls != 7 {
		t.Fatalf("strict 404=%+v %v calls=%d", missing, err, calls)
	}
	invalid, err := upper.CreateTask(ctx, "future", image.WithCreateTaskInput(map[string]any{string([]byte{0xff}): true}))
	if invalid != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 7 {
		t.Fatalf("invalid original key reached HTTP: %+v %v calls=%d", invalid, err, calls)
	}
}
