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
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func imageRecordWaitSeed(status json.RawMessage) *ImageRecord {
	fields := make(map[string]json.RawMessage)
	for key, raw := range imageRecordDefaults(`{"seeded":true}`) {
		fields[key] = json.RawMessage(raw)
	}
	fields["id"], fields["status"], fields["checksum"] = json.RawMessage(`"original"`), append(json.RawMessage(nil), status...), json.RawMessage(`"seed checksum"`)
	header := http.Header{"X-Task-Proof": {"seed"}, "X-Seed": {"original receipt"}}
	return &ImageRecord{
		Resource: &resource.RawResource{Metadata: resource.Metadata{Body: fields, Header: header.Clone(), StatusCode: 201}},
		Wire:     &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"original"`), "status": append(json.RawMessage(nil), status...), "wire_only": json.RawMessage(`900719925474099312345`)}, Header: header.Clone(), StatusCode: 201}},
		Envelope: json.RawMessage(`{"seed":900719925474099312345}`), Header: header.Clone(), StatusCode: 201, ImportMethods: []string{"seed method"},
	}
}

func TestImageRecordWaitStatusInitialTargetReturnsIndependentSeed(t *testing.T) {
	for _, mode := range []string{"ordinary", "zero timeout", "target is default failure", "custom attribute", "empty target", "negative duration ignored after initial match"} {
		t.Run(mode, func(t *testing.T) {
			seed := imageRecordWaitSeed(json.RawMessage(`"AcTiVe"`))
			target := "ACTIVE"
			options := []ImageRecordWaitOption{}
			switch mode {
			case "zero timeout":
				options = append(options, WithImageRecordWaitTimeout(0))
			case "empty target":
				seed.Resource.Body["status"] = json.RawMessage(`""`)
				target = ""
			case "negative duration ignored after initial match":
				options = append(options, WithImageRecordWaitOpts(ImageRecordWaitOpts{Timeout: taskOptionPointer(-time.Second), FailureStates: []string{string([]byte{0xff})}}))
			case "target is default failure":
				seed.Resource.Body["status"] = json.RawMessage(`"ERROR"`)
				target = "error"
			case "custom attribute":
				seed.Resource.Body["name"] = json.RawMessage(`"Ready"`)
				target = "READY"
				options = append(options, WithImageRecordWaitAttribute("name"))
			}
			locations, callbacks, calls := 0, 0, 0
			client := taskCoreClient(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("initial target performed GET")
				return nil, nil
			})
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{}, nil }})
			options = append(options, WithImageRecordWaitCallback(func(int) { callbacks++ }))
			got, err := service.WaitForImageRecordStatus(context.Background(), seed, target, options...)
			if got == nil || err != nil || got == seed || got.Resource == seed.Resource || got.Wire == seed.Wire || calls != 0 || callbacks != 0 || locations != 1 || !reflect.DeepEqual(got, seed) {
				t.Fatal(got, err, calls, callbacks, locations)
			}
			got.Resource.Body["id"][1] = 'X'
			got.Resource.Header.Set("X-Seed", "view changed")
			got.Wire.Body["wire_only"][0] = '0'
			got.Wire.Header.Set("X-Seed", "wire changed")
			got.Header.Set("X-Seed", "record changed")
			got.Envelope[0] = '!'
			got.ImportMethods[0] = "changed"
			if string(seed.Resource.Body["id"]) != `"original"` || string(seed.Wire.Body["wire_only"]) != "900719925474099312345" || seed.Header.Get("X-Seed") != "original receipt" || seed.Resource.Header.Get("X-Seed") != "original receipt" || seed.Wire.Header.Get("X-Seed") != "original receipt" || string(seed.Envelope) != `{"seed":900719925474099312345}` || seed.ImportMethods[0] != "seed method" {
				t.Fatal("short circuit aliases seed", seed)
			}
		})
	}
}

func TestImageRecordWaitStatusFreshSelectionFailureAndPending(t *testing.T) {
	for _, test := range []struct {
		name, initial, target, first string
		options                      []ImageRecordWaitOption
		failure, invalid             bool
		wantCalls, wantCallbacks     int
	}{
		{"initial ERROR gets fresh ACTIVE", `"ERROR"`, "active", `{"status":"ACTIVE"}`, nil, false, false, 1, 0},
		{"fresh failure", `"pending"`, "active", `{"status":"eRrOr"}`, nil, true, false, 1, 0},
		{"target before failure", `"pending"`, "ERROR", `{"status":"error"}`, nil, false, false, 1, 0},
		{"empty failures disabled", `"pending"`, "active", `{"status":"ERROR"}`, []ImageRecordWaitOption{WithImageRecordWaitFailureStates()}, false, false, 2, 1},
		{"replacement failure", `"pending"`, "active", `{"status":"broken"}`, []ImageRecordWaitOption{WithImageRecordWaitFailureStates("BROKEN")}, true, false, 1, 0},
		{"failure exact not prefix", `"pending"`, "active", `{"status":"error-extra"}`, nil, false, false, 2, 1},
		{"null selected state pending", `"pending"`, "active", `{"status":null}`, nil, false, false, 2, 1},
		{"initial null selected state", `null`, "active", `{"status":"ACTIVE"}`, nil, false, false, 1, 0},
		{"nonnull numeric selected state", `"pending"`, "active", `{"status":false}`, nil, false, true, 1, 0},
		{"missing selected field overlays seed", `"pending"`, "active", `{"name":"fresh"}`, nil, false, false, 2, 1},
		{"custom canonical attribute", `"pending"`, "active", `{"visibility":"ACTIVE"}`, []ImageRecordWaitOption{WithImageRecordWaitAttribute("visibility")}, false, false, 1, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			seed := imageRecordWaitSeed(json.RawMessage(test.initial))
			calls, callbacks := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/images/original" || req.URL.RawQuery != "" {
					t.Fatal(req.Method, req.URL)
				}
				if _, present := req.Context().Deadline(); present {
					t.Fatal("default status waiter added deadline")
				}
				if calls == 1 {
					return taskCoreJSON(req, 203, test.first), nil
				}
				if calls != 2 {
					t.Fatal("unexpected poll", calls)
				}
				return taskCoreJSON(req, 200, `{"status":"ACTIVE"}`), nil
			})
			options := append([]ImageRecordWaitOption{WithImageRecordWaitPollInterval(time.Nanosecond), WithImageRecordWaitCallback(func(progress int) { callbacks++; th.AssertEquals(t, 0, progress) })}, test.options...)
			got, err := New(client).WaitForImageRecordStatus(context.Background(), seed, test.target, options...)
			if calls != test.wantCalls || callbacks != test.wantCallbacks {
				t.Fatal(calls, callbacks, test.wantCalls, test.wantCallbacks)
			}
			if test.failure || test.invalid {
				if got == nil || err == nil || got.StatusCode != 203 {
					t.Fatal(got, err)
				}
				if test.failure && !errors.Is(err, resource.ErrFailedState) {
					t.Fatal(err)
				}
				if test.invalid && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				taskCoreProof(t, err, 203, test.first)
			} else {
				if got == nil || err != nil || string(got.Resource.Body["checksum"]) != `"seed checksum"` {
					t.Fatal(got, err)
				}
				th.AssertEquals(t, `{"seeded":true}`, string(seed.Resource.Body["properties"]))
				if test.wantCalls == 2 && test.name == "missing selected field overlays seed" {
					th.AssertEquals(t, `"fresh"`, string(got.Resource.Body["name"]))
				}
			}
		})
	}
}

func TestImageRecordWaitDeleteFreshFetchAnd404OwnedReturn(t *testing.T) {
	for _, mode := range []string{"immediate404", "later404", "fresh deleted", "initial deleted still fetches", "clean404 retry hook"} {
		t.Run(mode, func(t *testing.T) {
			seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
			if mode == "initial deleted still fetches" {
				seed.Resource.Body["status"] = json.RawMessage(`"deleted"`)
			}
			calls, callbacks := 0, 0
			var deadline time.Time
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/original" || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("deletion waiter sent mutation/retarget", req.Method, req.URL)
				}
				selected, present := req.Context().Deadline()
				if !present || time.Until(selected) < 118*time.Second || time.Until(selected) > 121*time.Second || calls > 1 && !selected.Equal(deadline) {
					t.Fatal("delete 120s single budget", selected)
				}
				deadline = selected
				if mode == "fresh deleted" || mode == "initial deleted still fetches" {
					return taskCoreJSON(req, 299, `{"id":"response id","status":"DeLeTeD","name":"latest"}`), nil
				}
				if mode == "later404" && calls == 1 {
					response := taskCoreJSON(req, 201, `{"id":"response id","status":"pending","name":"latest"}`)
					response.Header.Set("OpenStack-image-import-methods", "fresh, fresh")
					return response, nil
				}
				return taskCoreJSON(req, 404, `{"message":"missing"}`), nil
			})
			hooks := 0
			if mode == "clean404 retry hook" {
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					hooks++
					return err
				}
			}
			got, err := New(client).WaitForImageRecordDelete(context.Background(), seed, WithImageRecordWaitPollInterval(time.Nanosecond), WithImageRecordWaitCallback(func(progress int) { callbacks++; th.AssertEquals(t, 0, progress) }))
			wantCalls, wantCallbacks := 1, 0
			if mode == "later404" {
				wantCalls, wantCallbacks = 2, 1
			}
			if got == nil || err != nil || got == seed || calls != wantCalls || callbacks != wantCallbacks {
				t.Fatal(got, err, calls, callbacks)
			}
			if mode == "clean404 retry hook" && hooks != 1 {
				t.Fatal(hooks)
			}
			if mode == "immediate404" || mode == "clean404 retry hook" {
				if !reflect.DeepEqual(got, seed) {
					t.Fatal("absence fabricated fetched receipt", got, seed)
				}
				got.Envelope[0] = '!'
				got.ImportMethods[0] = "changed"
				got.Resource.Body["checksum"][1] = 'X'
				if string(seed.Envelope) != `{"seed":900719925474099312345}` || seed.ImportMethods[0] != "seed method" || string(seed.Resource.Body["checksum"]) != `"seed checksum"` {
					t.Fatal(seed)
				}
			} else {
				wantCode := 299
				if mode == "later404" {
					wantCode = 201
				}
				if got.StatusCode != wantCode || got.Header.Get("X-Task-Proof") != "actual" || string(got.Resource.Body["id"]) != `"response id"` || string(got.Resource.Body["name"]) != `"latest"` || string(got.Resource.Body["checksum"]) != `"seed checksum"` {
					t.Fatal("latest private overlay lost", got)
				}
				if mode == "later404" {
					th.CheckDeepEquals(t, []string{"fresh", " fresh"}, got.ImportMethods)
					th.AssertEquals(t, `{"id":"response id","status":"pending","name":"latest"}`, string(got.Envelope))
				}
			}
			th.AssertEquals(t, `"original"`, string(seed.Resource.Body["id"]))
		})
	}
}

func TestImageRecordWaitStatusOwnsOneBudgetLocationHeadersAndLiteralID(t *testing.T) {
	seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
	seed.Resource.Body["id"] = json.RawMessage(`"a /한:%?\\b"`)
	cloud := "captured"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}
	headers := map[string]string{"X-Option": "factory snapshot"}
	failures := []string{"ERROR"}
	timeout := time.Second
	interval := time.Nanosecond
	option := WithImageRecordWaitOpts(ImageRecordWaitOpts{Headers: headers, FailureStates: failures, Timeout: &timeout, PollInterval: &interval})
	headers["X-Option"] = "caller changed"
	failures[0] = "PENDING"
	timeout = 0
	interval = time.Hour
	calls, locations, optionsApplied, callbacks := 0, 0, 0, 0
	var retained *ImageRecordWaitOpts
	var client *gophercloud.ServiceClient
	var deadline time.Time
	client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape("a /한:%?\\b") || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory snapshot" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != map[bool]string{true: "live before", false: "live after"}[calls == 1] || req.Context().Value("image wait proof") != "context value" {
			t.Fatal(req.URL, req.Header)
		}
		selected, present := req.Context().Deadline()
		if !present || calls > 1 && !selected.Equal(deadline) {
			t.Fatal("poll budget changed", selected, deadline)
		}
		deadline = selected
		retained.Headers["X-Option"] = "retained changed"
		retained.FailureStates[0] = "PENDING"
		*retained.Timeout = 0
		*retained.PollInterval = time.Hour
		if calls == 1 {
			return taskCoreJSON(req, 201, `{"id":"response decoy","status":"pending","name":"first name"}`), nil
		}
		if calls != 2 {
			t.Fatal("extra poll", calls)
		}
		return taskCoreJSON(req, 203, `{"id":null,"status":"ACTIVE"}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "getter changed"
		seed.Resource.Body["id"][1] = 'X'
		return facts, nil
	}})
	parent, cancel := context.WithTimeout(context.WithValue(context.Background(), "image wait proof", "context value"), 5*time.Second)
	defer cancel()
	started := time.Now()
	got, err := service.WaitForImageRecordStatus(parent, seed, "active", option, func(value *ImageRecordWaitOpts) error {
		optionsApplied++
		retained = value
		client.SetToken("live before")
		cloud = "option changed"
		facts.Project.ID[1] = 'X'
		value.Callback = func(progress int) { callbacks++; th.AssertEquals(t, 0, progress); client.SetToken("live after") }
		return WithImageRecordWaitHeader("X-Final", "yes")(value)
	})
	if got == nil || err != nil || calls != 2 || locations != 1 || optionsApplied != 1 || callbacks != 1 || deadline.Sub(started) < 900*time.Millisecond || deadline.Sub(started) > 1100*time.Millisecond {
		t.Fatal(got, err, calls, locations, optionsApplied, callbacks, deadline.Sub(started))
	}
	th.AssertEquals(t, "null", string(got.Resource.Body["id"]))
	th.AssertEquals(t, `"first name"`, string(got.Resource.Body["name"]))
	var location resource.CloudLocation
	if err := json.Unmarshal(got.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured" || string(location.Project.ID) != `"token project"` {
		t.Fatal(location, err)
	}
}

func TestImageRecordWaitZeroIntervalsTimeoutAndParentBudget(t *testing.T) {
	t.Run("zero interval unlimited uses100ms", func(t *testing.T) {
		calls := 0
		var first time.Time
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if _, present := req.Context().Deadline(); present {
				t.Fatal("unlimited SDK deadline")
			}
			if calls == 1 {
				first = time.Now()
				return taskCoreJSON(req, 200, `{"status":"pending"}`), nil
			}
			if time.Since(first) < 90*time.Millisecond {
				t.Fatal("zero interval busy loop", time.Since(first))
			}
			return taskCoreJSON(req, 200, `{"status":"active"}`), nil
		})
		got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitUnlimited(), WithImageRecordWaitPollInterval(0))
		if got == nil || err != nil || calls != 2 {
			t.Fatal(got, err, calls)
		}
	})
	for _, deleteWait := range []bool{false, true} {
		t.Run(fmt.Sprintf("zero timeout delete=%v", deleteWait), func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			service := New(client)
			seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
			var got *ImageRecord
			var err error
			if deleteWait {
				got, err = service.WaitForImageRecordDelete(context.Background(), seed, WithImageRecordWaitTimeout(0))
			} else {
				got, err = service.WaitForImageRecordStatus(context.Background(), seed, "active", WithImageRecordWaitTimeout(0))
			}
			if got != nil || !errors.Is(err, context.DeadlineExceeded) || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("zero interval bounded by shorter timeout", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return taskCoreJSON(req, 200, `{"status":"pending"}`), nil
		})
		got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitTimeout(5*time.Millisecond), WithImageRecordWaitPollInterval(0), WithImageRecordWaitCallback(func(int) { callbacks++ }))
		if got == nil || !errors.Is(err, context.DeadlineExceeded) || calls != 1 || callbacks != 1 || got.StatusCode != 200 {
			t.Fatal(got, err, calls, callbacks)
		}
	})
	t.Run("parent budget wins", func(t *testing.T) {
		parent, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		parentDeadline, _ := parent.Deadline()
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			deadline, present := req.Context().Deadline()
			if !present || !deadline.Equal(parentDeadline) {
				t.Fatal(deadline, parentDeadline)
			}
			return taskCoreJSON(req, 200, `{"status":"active"}`), nil
		})
		got, err := New(client).WaitForImageRecordStatus(parent, imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitTimeout(time.Hour))
		if got == nil || err != nil {
			t.Fatal(got, err)
		}
	})
}

func TestImageRecordWaitAcceptedJSONToleranceAndBodyStateErrors(t *testing.T) {
	for _, test := range []struct {
		code             int
		raw              string
		pending, invalid bool
	}{
		{201, `{"status":"ACTIVE"}`, false, false}, {299, `{"status":"active"}`, false, false}, {300, `{"status":"active"}`, false, false}, {399, `{"status":"active"}`, false, false},
		{204, "", true, false}, {202, "not JSON", true, false}, {304, `{"broken":`, true, false},
		{200, `null`, false, true}, {200, `[]`, false, true}, {200, `{"status":false}`, false, true},
	} {
		t.Run(fmt.Sprintf("%d/%q", test.code, test.raw), func(t *testing.T) {
			calls, retries, callbacks := 0, 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, test.code, test.raw), nil
				}
				return taskCoreJSON(req, 200, `{"status":"active"}`), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitPollInterval(time.Nanosecond), WithImageRecordWaitCallback(func(int) { callbacks++ }))
			if test.invalid {
				projected := test.raw == `{"status":false}`
				if (got != nil) != projected || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || callbacks != 0 {
					t.Fatal(got, err, calls, callbacks)
				}
				taskCoreProof(t, err, test.code, test.raw)
			} else {
				wantCalls := 1
				if test.pending {
					wantCalls = 2
				}
				if got == nil || err != nil || calls != wantCalls || callbacks != wantCalls-1 {
					t.Fatal(got, err, calls, callbacks)
				}
			}
			if retries != 0 {
				t.Fatal("accepted response replay", retries)
			}
		})
	}
	for _, raw := range []string{`{"status":null}`, `{"status":false}`, `{"status":["deleted"]}`} {
		t.Run("delete nonstring/"+raw, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, raw), nil })
			got, err := New(client).WaitForImageRecordDelete(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)))
			if got == nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || got.StatusCode != 200 {
				t.Fatal(got, err, calls)
			}
			taskCoreProof(t, err, 200, raw)
		})
	}
}

func TestImageRecordWaitHTTPFailuresAnd404HandlingFaultsAreTerminal(t *testing.T) {
	for _, deleteWait := range []bool{false, true} {
		for _, code := range []int{403, 500, 503} {
			t.Run(fmt.Sprintf("native%d/delete=%v", code, deleteWait), func(t *testing.T) {
				calls := 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return taskCoreJSON(req, code, `{"message":"rejected"}`), nil
				})
				service := New(client)
				seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
				var got *ImageRecord
				var err error
				if deleteWait {
					got, err = service.WaitForImageRecordDelete(context.Background(), seed)
				} else {
					got, err = service.WaitForImageRecordStatus(context.Background(), seed, "active")
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodGet || string(native.Body) != `{"message":"rejected"}` || native.ResponseHeader.Get("X-Task-Proof") != "actual" || calls != 1 {
					t.Fatal(got, err, native, calls)
				}
			})
		}
	}
	for _, mode := range []string{"later native503", "later projector invalidshape"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"status":"pending","name":"last successful"}`), nil
				}
				if mode == "later native503" {
					return taskCoreJSON(req, 503, `{"message":"late rejection"}`), nil
				}
				return taskCoreJSON(req, 200, `null`), nil
			})
			got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitPollInterval(time.Nanosecond))
			if got == nil || err == nil || calls != 2 || got.StatusCode != 203 || string(got.Resource.Body["name"]) != `"last successful"` || string(got.Envelope) != `{"status":"pending","name":"last successful"}` {
				t.Fatal(got, err, calls)
			}
			if mode == "later native503" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"message":"late rejection"}` {
					t.Fatal(err, native)
				}
			} else {
				taskCoreProof(t, err, 200, `null`)
			}
		})
	}
	t.Run("status clean404 is error", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return taskCoreJSON(req, 404, `{"message":"missing"}`), nil
		})
		got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active")
		var native gophercloud.ErrUnexpectedResponseCode
		if got != nil || !errors.As(err, &native) || native.Actual != 404 || calls != 1 {
			t.Fatal(got, err, calls)
		}
	})
	for _, mode := range []string{"read", "wrappedEOF", "close", "retry hook", "transport nested404"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("rejected deletion handling failure")
			calls := 0
			body := &taskCoreBody{reader: strings.NewReader(`{"message":"missing"}`)}
			switch mode {
			case "read":
				body.reader = &taskCoreReader{body: `{"message":"missing"}`, err: marker}
			case "wrappedEOF":
				body.reader = &taskCoreReader{body: `{"message":"missing"}`, err: errors.Join(io.EOF, marker)}
			case "close":
				body.closeErr = marker
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if mode == "transport nested404" {
					return nil, errors.Join(marker, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
				}
				return taskCoreHTTP(req, 404, body), nil
			})
			if mode == "retry hook" {
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					return errors.Join(err, marker)
				}
			}
			got, err := New(client).WaitForImageRecordDelete(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)))
			if got != nil || !errors.Is(err, marker) || calls != 1 {
				t.Fatal("faulty404 treated as deletion", got, err, calls)
			}
			if mode != "transport nested404" && body.closes != 1 {
				t.Fatal(body.closes)
			}
		})
	}
}

func TestImageRecordWaitAcceptedAndRejectedStickyPhysicalGuards(t *testing.T) {
	for _, code := range []int{201, 404} {
		for _, mode := range []string{"read", "close", "cancel", "source restored on Close", "outer restored on Close"} {
			t.Run(fmt.Sprintf("%d/%s", code, mode), func(t *testing.T) {
				const raw = `{"status":"active"}`
				marker := errors.New("wait physical boundary")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls, retries := 0, 0
				var client *gophercloud.ServiceClient
				invalidOuter := false
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
					if invalidOuter {
						return marker
					}
					return nil
				})
				body := &taskCoreBody{reader: strings.NewReader(raw)}
				action := func() {}
				switch mode {
				case "read":
					body.reader = &taskCoreReader{body: raw, err: marker}
				case "close":
					body.closeErr = marker
				case "cancel":
					action = func() { cancel(marker) }
				case "source restored on Close":
					action = func() { client.Endpoint = "https://foreign.test/" }
				case "outer restored on Close":
					action = func() { invalidOuter = true }
				}
				if mode != "read" {
					body.reader = &taskCoreReader{body: raw, err: io.EOF, action: action}
				}
				selected := io.ReadCloser(body)
				if strings.Contains(mode, "restored") {
					selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; invalidOuter = false }}
				}
				client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return taskCoreHTTP(req, code, selected), nil
				})
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				got, err := New(client).WaitForImageRecordDelete(ctx, imageRecordWaitSeed(json.RawMessage(`"pending"`)))
				if got != nil || err == nil || calls != 1 || body.closes != 1 || code == 201 && retries != 0 {
					t.Fatal(got, err, calls, body.closes, retries)
				}
				if mode == "source restored on Close" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal("cause lost", err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if code == 201 {
					taskCoreProof(t, err, code, raw)
				}
			})
		}
	}
}

func TestImageRecordWaitCallbacksGuardBeforePauseOrNextGET(t *testing.T) {
	for _, mode := range []string{"cancel", "source drift", "service binding", "outer drift", "timeout while paused"} {
		t.Run(mode, func(t *testing.T) {
			const raw = `{"status":"pending","progress":77}`
			marker := errors.New("wait callback cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return marker
				}
				return nil
			})
			calls, callbacks := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 203, raw), nil })
			service := New(client)
			options := []ImageRecordWaitOption{WithImageRecordWaitPollInterval(time.Hour), WithImageRecordWaitCallback(func(progress int) {
				callbacks++
				th.AssertEquals(t, 0, progress)
				switch mode {
				case "cancel":
					cancel(marker)
				case "source drift":
					client.ResourceBase = "https://foreign.test/"
				case "service binding":
					service.API = nil
				case "outer drift":
					outerInvalid = true
				}
			})}
			if mode == "timeout while paused" {
				options = append(options, WithImageRecordWaitTimeout(5*time.Millisecond))
			}
			got, err := service.WaitForImageRecordStatus(ctx, imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", options...)
			if got == nil || err == nil || calls != 1 || callbacks != 1 || got.StatusCode != 203 {
				t.Fatal(got, err, calls, callbacks)
			}
			switch mode {
			case "source drift", "service binding":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "timeout while paused":
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal(err)
				}
			default:
				if !errors.Is(err, marker) {
					t.Fatal(err)
				}
			}
			taskCoreProof(t, err, 203, raw)
		})
	}
}

func TestImageRecordWaitCompletePreflightAndConcreteReplacement(t *testing.T) {
	for _, test := range []struct {
		name    string
		alter   func(*ImageRecord)
		target  string
		options []ImageRecordWaitOption
	}{
		{"missing id", func(seed *ImageRecord) { delete(seed.Resource.Body, "id") }, "active", nil},
		{"null id", func(seed *ImageRecord) { seed.Resource.Body["id"] = json.RawMessage(`null`) }, "active", nil},
		{"nonstring id", func(seed *ImageRecord) { seed.Resource.Body["id"] = json.RawMessage(`false`) }, "active", nil},
		{"control id", func(seed *ImageRecord) { seed.Resource.Body["id"] = json.RawMessage(`"bad\n"`) }, "active", nil},
		{"missing Resource", func(seed *ImageRecord) { seed.Resource = nil }, "active", nil},
		{"nonstring initial selected state", func(seed *ImageRecord) { seed.Resource.Body["status"] = json.RawMessage(`false`) }, "active", nil},
		{"invalid UTF8 target", nil, string([]byte{0xff}), nil},
		{"nil option", nil, "active", []ImageRecordWaitOption{nil}},
		{"negative timeout", nil, "active", []ImageRecordWaitOption{WithImageRecordWaitTimeout(-time.Nanosecond)}},
		{"negative interval", nil, "active", []ImageRecordWaitOption{WithImageRecordWaitPollInterval(-time.Nanosecond)}},
		{"unlimited with timeout", nil, "active", []ImageRecordWaitOption{WithImageRecordWaitOpts(ImageRecordWaitOpts{Unlimited: true, Timeout: taskOptionPointer(time.Second)})}},
		{"remote attr alias", nil, "active", []ImageRecordWaitOption{WithImageRecordWaitAttribute("os_hidden")}},
		{"attr case alias", nil, "active", []ImageRecordWaitOption{WithImageRecordWaitAttribute("Status")}},
		{"unknown attr", nil, "active", []ImageRecordWaitOption{WithImageRecordWaitAttribute("unknown")}},
		{"owned token", nil, "active", []ImageRecordWaitOption{WithImageRecordWaitHeader("X-Auth-Token", "foreign")}},
		{"header newline", nil, "active", []ImageRecordWaitOption{WithImageRecordWaitHeaders(map[string]string{"X-Extra": "\n"})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
			if test.alter != nil {
				test.alter(seed)
			}
			got, err := New(client).WaitForImageRecordStatus(context.Background(), seed, test.target, test.options...)
			if got != nil || err == nil || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		calls, callbacks := 0, 0
		service := New(taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }))
		got, err := service.WaitForImageRecordStatus(ctx, imageRecordWaitSeed(json.RawMessage(`"active"`)), "active", func(*ImageRecordWaitOpts) error { callbacks++; return nil })
		if got != nil || err == nil || calls != 0 || callbacks != 0 {
			t.Fatal(got, err, calls, callbacks)
		}
	}
	service := New(taskCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("nil seed reached HTTP"); return nil, nil }))
	if got, err := service.WaitForImageRecordStatus(context.Background(), nil, "active"); got != nil || err == nil {
		t.Fatal(got, err)
	}
	if got, err := service.WaitForImageRecordDelete(context.Background(), nil); got != nil || err == nil {
		t.Fatal(got, err)
	}
	var nilService *Service
	if got, err := nilService.WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"active"`)), "active"); got != nil || err == nil {
		t.Fatal(got, err)
	}
	t.Run("later full opts replace prior configuration", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("X-Discard") != "" || req.Header.Get("X-Final") != "yes" {
				t.Fatal(req.Header)
			}
			return taskCoreJSON(req, 200, `{"status":"ACTIVE","name":"ERROR"}`), nil
		})
		options := []ImageRecordWaitOption{WithImageRecordWaitAttribute("name"), WithImageRecordWaitFailureStates("ACTIVE"), WithImageRecordWaitHeader("X-Discard", "old"), WithImageRecordWaitOpts(ImageRecordWaitOpts{Headers: map[string]string{"X-Final": "yes"}})}
		got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", options...)
		if got == nil || err != nil || calls != 1 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestImageRecordWaitUnicodeStatusComparison(t *testing.T) {
	// Python str.lower uses full mappings and contextual final sigma. These
	// binding cases distinguish the explicit Unicode16 runtime contract from
	// Go's single-rune lowercase operation without duplicating its table tests.
	for _, test := range []struct {
		name, initial, target string
		wantCalls             int
	}{
		{"dotted capital I does not equal plain i", "İ", "i", 1},
		{"dotted capital I equals expanded lowercase", "İ", "i\u0307", 0},
		{"Greek final sigma context", "ΟΣ", "ος", 0},
		{"Greek final sigma does not equal ordinary sigma", "ΟΣ", "οσ", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			initial, err := json.Marshal(test.initial)
			if err != nil {
				t.Fatal(err)
			}
			seed := imageRecordWaitSeed(initial)
			calls, callbacks := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/images/original" || calls != 1 {
					t.Fatal(req.Method, req.URL, calls)
				}
				status, err := json.Marshal(test.target)
				if err != nil {
					t.Fatal(err)
				}
				return taskCoreJSON(req, 203, `{"status":`+string(status)+`}`), nil
			})
			got, err := New(client).WaitForImageRecordStatus(context.Background(), seed, test.target, WithImageRecordWaitPollInterval(time.Nanosecond), WithImageRecordWaitCallback(func(progress int) { callbacks++; th.AssertEquals(t, 0, progress) }))
			if got == nil || err != nil || calls != test.wantCalls || callbacks != 0 {
				t.Fatal(got, err, calls, callbacks)
			}
			if test.wantCalls == 0 {
				if got == seed || !reflect.DeepEqual(got, seed) {
					t.Fatal("Unicode initial match lost seed clone", got, seed)
				}
			} else if got.StatusCode != 203 || taskCoreText(t, got.Resource.Body["status"]) != test.target {
				t.Fatal("fresh exact target not returned", got)
			}
		})
	}
	for _, test := range []struct {
		name, failure, target    string
		failed                   bool
		wantCalls, wantCallbacks int
	}{
		{"expanded lowercase is not plain-i failure", "i", "active", false, 2, 1},
		{"expanded lowercase matches selected failure", "i\u0307", "active", true, 1, 0},
		{"Unicode target wins before same failure", "i\u0307", "i\u0307", false, 1, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			const first = `{"status":"İ","name":"Unicode observation"}`
			calls, callbacks := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/images/original" {
					t.Fatal(req.Method, req.URL)
				}
				if calls == 1 {
					return taskCoreJSON(req, 203, first), nil
				}
				if calls != 2 {
					t.Fatal("extra Unicode failure poll", calls)
				}
				return taskCoreJSON(req, 200, `{"status":"active"}`), nil
			})
			got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), test.target, WithImageRecordWaitFailureStates(test.failure), WithImageRecordWaitPollInterval(time.Nanosecond), WithImageRecordWaitCallback(func(progress int) { callbacks++; th.AssertEquals(t, 0, progress) }))
			if got == nil || calls != test.wantCalls || callbacks != test.wantCallbacks || errors.Is(err, resource.ErrFailedState) != test.failed || err != nil && !test.failed {
				t.Fatal(got, err, calls, callbacks)
			}
			if test.failed {
				if got.StatusCode != 203 || string(got.Envelope) != first || taskCoreText(t, got.Resource.Body["status"]) != "İ" {
					t.Fatal("Unicode failure lost physical observation", got)
				}
				taskCoreProof(t, err, 203, first)
			}
		})
	}
}
