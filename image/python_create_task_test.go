package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonCreateTaskTransport func(*http.Request) (*http.Response, error)

func (f pythonCreateTaskTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func pythonCreateTaskService(t *testing.T, bodies *[]string, status int, reply string) *image.Service {
	t.Helper()
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: pythonCreateTaskTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/glance/v2/tasks" {
			t.Errorf("%s %s", req.Method, req.URL)
		}
		data, _ := io.ReadAll(req.Body)
		*bodies = append(*bodies, string(data))
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(reply)), Request: req}, nil
	})}}
	provider.UseTokenLock()
	provider.SetToken("token")
	return image.New(&gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "https://glance.example/glance/v2/", Type: "image"})
}

func pythonCreateTaskSameJSON(t *testing.T, got, want string) bool {
	t.Helper()
	var a, b any
	if err := json.Unmarshal([]byte(got), &a); err != nil {
		t.Fatal(got, err)
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		t.Fatal(want, err)
	}
	encodedA, _ := json.Marshal(a)
	encodedB, _ := json.Marshal(b)
	return string(encodedA) == string(encodedB)
}

func TestPythonCreateTaskSendsNullInput(t *testing.T) {
	ctx := context.Background()
	var bodies []string
	service := pythonCreateTaskService(t, &bodies, http.StatusCreated, `{"id":"t1","type":"import","status":"pending","input":null}`)
	// create_task(type="import", input=None)
	got, err := service.CreateTask(ctx, "import", image.WithCreateTaskNullInput())
	if err != nil || got.StatusCode != 201 || got.ID == nil || *got.ID != "t1" || got.Input != nil || string(got.Body["input"]) != "null" {
		t.Fatal(got, err)
	}
	// A later null clears an earlier object; a later object replaces null;
	// CreateTaskOpts.NullInput selects null too.
	steps := [][]image.CreateTaskOption{
		{image.WithCreateTaskInput(map[string]any{"a": 1}), image.WithCreateTaskNullInput()},
		{image.WithCreateTaskNullInput(), image.WithCreateTaskInput(map[string]any{"a": 1})},
		{image.WithCreateTaskOpts(image.CreateTaskOpts{NullInput: true})},
		{},
	}
	for _, options := range steps {
		if _, err := service.CreateTask(ctx, "import", options...); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{
		`{"type":"import","input":null}`,
		`{"type":"import","input":null}`,
		`{"type":"import","input":{"a":1}}`,
		`{"type":"import","input":null}`,
		`{"type":"import","input":{}}`,
	}
	if len(bodies) != len(want) {
		t.Fatal(bodies)
	}
	for i := range want {
		if !pythonCreateTaskSameJSON(t, bodies[i], want[i]) {
			t.Fatalf("call %d: %s", i, bodies[i])
		}
	}
}

func TestPythonCreateTaskNullInputRejections(t *testing.T) {
	ctx := context.Background()
	var bodies []string
	service := pythonCreateTaskService(t, &bodies, http.StatusBadRequest, `{"message":"Input is required"}`)
	var operation *resource.OperationError
	_, err := service.CreateTask(ctx, "import", image.WithCreateTaskOpts(image.CreateTaskOpts{NullInput: true, Input: map[string]json.RawMessage{"a": json.RawMessage("1")}}))
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "CreateTask" || len(bodies) != 0 {
		t.Fatal(err, bodies)
	}
	_, err = service.CreateTask(ctx, "import", image.WithCreateTaskNullInput())
	if !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) || !errors.As(err, &operation) || operation.Operation != "CreateTask" || len(bodies) != 1 || !pythonCreateTaskSameJSON(t, bodies[0], `{"type":"import","input":null}`) {
		t.Fatal(err, bodies)
	}
}
