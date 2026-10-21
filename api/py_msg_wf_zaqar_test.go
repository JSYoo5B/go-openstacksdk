package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/messaging/v2/claims"
	"github.com/JSYoo5B/go-openstacksdk/messaging/v2/messages"
	"github.com/JSYoo5B/go-openstacksdk/messaging/v2/queues"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const pyMsgWfJobsPath = "/zaqar/v2/queues/jobs"

func pyMsgWfClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("messaging", "/zaqar")
	client.MoreHeaders = map[string]string{"Client-ID": "client-1", "X-PROJECT-ID": "project-1"}
	return client
}

func pyMsgWfRequireIdentity(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("Client-ID") != "client-1" || r.Header.Get("X-PROJECT-ID") != "project-1" {
		t.Errorf("%s %s identity headers=%v", r.Method, r.URL.Path, r.Header)
	}
}

// Python MessageResource sends Client-ID and X-PROJECT-ID on every call; Go
// takes both from the configured service client for every Zaqar operation.
func TestPyMsgWfConfiguredIdentityHeadersAreSentOnEveryZaqarCall(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		pyMsgWfRequireIdentity(t, r)
		switch {
		case r.Method == "PUT" && r.URL.Path == "/zaqar/v2/queues/jobs":
			w.WriteHeader(201)
		case r.Method == "GET" && r.URL.Path == "/zaqar/v2/queues/jobs":
			testcloud.JSON(w, 200, `{"_default_message_ttl":60}`)
		case r.Method == "GET" && r.URL.Path == "/zaqar/v2/queues":
			if r.URL.Query().Get("marker") != "" {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, 200, `{"queues":[{"name":"jobs"}],"links":[{"rel":"next","href":"/v2/queues?marker=jobs"}]}`)
		case r.Method == "POST" && r.URL.Path == "/zaqar/v2/queues/jobs/messages":
			testcloud.JSON(w, 201, `{"resources":["/v2/queues/jobs/messages/m1"]}`)
		case r.Method == "GET" && r.URL.Path == "/zaqar/v2/queues/jobs/messages/m1":
			testcloud.JSON(w, 200, `{"href":"/v2/queues/jobs/messages/m1","ttl":30,"age":1,"body":{}}`)
		case r.Method == "DELETE":
			w.WriteHeader(204)
		case r.Method == "POST" && r.URL.Path == "/zaqar/v2/queues/jobs/claims":
			w.Header().Set("Location", "/v2/queues/jobs/claims/c1")
			testcloud.JSON(w, 201, `{"messages":[]}`)
		case r.Method == "GET" && r.URL.Path == "/zaqar/v2/queues/jobs/claims/c1":
			testcloud.JSON(w, 200, `{"ttl":60,"age":1,"messages":[]}`)
		case r.Method == "PATCH" && r.URL.Path == "/zaqar/v2/queues/jobs/claims/c1":
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	ctx := context.Background()
	client := pyMsgWfClient(cloud)
	q, m, c := queues.New(client), messages.New(client), claims.New(client)
	if err := q.Create(ctx, queues.CreateOpts{QueueName: "jobs"}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Get(ctx, "jobs"); err != nil {
		t.Fatal(err)
	}
	for _, err := range q.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.Create(ctx, "jobs", messages.BatchCreateOpts{{TTL: 30, Body: map[string]any{}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(ctx, "jobs", "m1"); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(ctx, "jobs", "m1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateClaim(ctx, "jobs", claims.CreateOpts{TTL: 60, Grace: 60}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "jobs", "c1"); err != nil {
		t.Fatal(err)
	}
	if err := c.Update(ctx, "jobs", "c1", claims.UpdateOpts{TTL: 120}); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(ctx, "jobs", "c1"); err != nil {
		t.Fatal(err)
	}
	if err := q.Delete(ctx, "jobs"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 12 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

// delete_queue/delete_message/delete_claim default to ignore_missing=True.
// Go keeps the native strict Delete and composes the same default with
// resource.IgnoreMissing, which suppresses only an actual HTTP 404.
func TestPyMsgWfDeleteIgnoreMissingDefaultComposesWithNativeDelete(t *testing.T) {
	for _, status := range []int{404, 500} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method != "DELETE" {
				t.Errorf("unexpected %s %s", r.Method, r.URL)
			}
			testcloud.JSON(w, status, `{}`)
		})
		ctx := context.Background()
		client := pyMsgWfClient(cloud)
		results := []error{
			queues.New(client).Delete(ctx, "jobs"),
			messages.New(client).Delete(ctx, "jobs", "m1"),
			claims.New(client).Delete(ctx, "jobs", "c1"),
		}
		for i, err := range results {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != status {
				t.Fatalf("native delete %d status %d: %v", i, status, err)
			}
			got := resource.IgnoreMissing(err)
			if status == 404 && got != nil {
				t.Fatalf("delete %d must ignore a missing resource: %v", i, got)
			}
			if status == 500 && got == nil {
				t.Fatalf("delete %d must not hide a server error", i)
			}
		}
		if calls.Load() != 3 {
			t.Fatalf("status %d calls=%d, delete must not be retried", status, calls.Load())
		}
	}
}

// Python appends ?claim_id=... only when the claim is truthy.
func TestPyMsgWfMessageDeleteSendsClaimIDOnlyWhenPresent(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []string
	cloud.Mux.HandleFunc("DELETE "+pyMsgWfJobsPath+"/messages/m1", func(w http.ResponseWriter, r *http.Request) {
		pyMsgWfRequireIdentity(t, r)
		queries = append(queries, r.URL.RawQuery)
		w.WriteHeader(204)
	})
	api := messages.New(pyMsgWfClient(cloud))
	ctx := context.Background()
	if err := api.Delete(ctx, "jobs", "m1"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "jobs", "m1", messages.WithDeleteOptions(messages.DeleteOpts{ClaimID: ""})); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "jobs", "m1", messages.WithDeleteOptions(messages.DeleteOpts{ClaimID: "c1"})); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 3 || queries[0] != "" || queries[1] != "" || queries[2] != "claim_id=c1" {
		t.Fatalf("queries=%q", queries)
	}
}

// Python Claim._translate_response derives the claim ID from the Location
// header after a successful create, and returns early on 204.
func TestPyMsgWfClaimCreateReturnsIDFromLocation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		location string
		body     string
		wantID   string
		wantLoc  string
		wantMsgs int
	}{
		{"location", 201, "/v2/queues/jobs/claims/c1", `{"messages":[{"href":"/v2/queues/jobs/messages/m1?claim_id=c1","ttl":10,"age":1,"body":{"a":false}}]}`, "c1", "/v2/queues/jobs/claims/c1", 1},
		{"no marker", 201, "/elsewhere/c2", `{"messages":[]}`, "", "/elsewhere/c2", 0},
		{"no header", 201, "", `{"messages":[]}`, "", "", 0},
		{"nothing to claim", 204, "/v2/queues/jobs/claims/ignored", "", "", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST "+pyMsgWfJobsPath+"/claims", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				pyMsgWfRequireIdentity(t, r)
				if tc.location != "" {
					w.Header().Set("Location", tc.location)
				}
				if tc.status == 204 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, tc.status, tc.body)
			})
			got, err := claims.New(pyMsgWfClient(cloud)).CreateClaim(context.Background(), "jobs", claims.CreateOpts{TTL: 60, Grace: 60, Limit: 5})
			if err != nil || calls.Load() != 1 {
				t.Fatalf("error=%v calls=%d", err, calls.Load())
			}
			if got.ID != tc.wantID || got.Location != tc.wantLoc || len(got.Messages) != tc.wantMsgs {
				t.Fatalf("claim=%+v", got)
			}
			if tc.wantMsgs == 1 && got.Messages[0].Body["a"] != false {
				t.Fatalf("message body=%v", got.Messages[0].Body)
			}
		})
	}
}

func TestPyMsgWfClaimCreateFailuresAreNotReplayedAndKeepNativeStatus(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST "+pyMsgWfJobsPath+"/claims", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 500, `{}`)
	})
	api := claims.New(pyMsgWfClient(cloud))
	got, err := api.CreateClaim(context.Background(), "jobs", claims.CreateOpts{TTL: 60, Grace: 60})
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if got != nil || !errors.As(err, &unexpected) || unexpected.Actual != 500 || calls.Load() != 1 {
		t.Fatalf("claim=%v error=%v calls=%d", got, err, calls.Load())
	}
	if _, err := api.CreateClaim(context.Background(), "jobs", claims.CreateOpts{TTL: 60, Grace: 60}, claims.WithCreateField("ttl", 1)); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatalf("core-field extension must fail before HTTP: %v calls=%d", err, calls.Load())
	}
	if _, err := api.CreateClaim(context.Background(), "jobs", claims.CreateOpts{TTL: 60, Grace: 60}, nil); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatalf("nil option must fail before HTTP: %v calls=%d", err, calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.CreateClaim(ctx, "jobs", claims.CreateOpts{TTL: 60, Grace: 60}); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("canceled create: %v calls=%d", err, calls.Load())
	}
}

// The ID extracted from Location feeds the later Get/Update/Delete routes.
func TestPyMsgWfClaimIDFromCreateAddressesLaterOperations(t *testing.T) {
	cloud := testcloud.New(t)
	var seen []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch r.Method {
		case "POST":
			w.Header().Set("Location", "/v2/queues/jobs/claims/c9")
			testcloud.JSON(w, 201, `{"messages":[]}`)
		case "GET":
			testcloud.JSON(w, 200, `{"ttl":60,"age":2,"messages":[]}`)
		default:
			w.WriteHeader(204)
		}
	})
	api := claims.New(pyMsgWfClient(cloud))
	ctx := context.Background()
	claimed, err := api.CreateClaim(ctx, "jobs", claims.CreateOpts{TTL: 60, Grace: 60})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.Get(ctx, "jobs", claimed.ID); err != nil {
		t.Fatal(err)
	}
	if err := api.Update(ctx, "jobs", claimed.ID, claims.UpdateOpts{TTL: 90}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "jobs", claimed.ID); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST " + pyMsgWfJobsPath + "/claims", "GET " + pyMsgWfJobsPath + "/claims/c9", "PATCH " + pyMsgWfJobsPath + "/claims/c9", "DELETE " + pyMsgWfJobsPath + "/claims/c9"}
	if len(seen) != len(want) {
		t.Fatalf("requests=%v", seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("requests=%v", seen)
		}
	}
}

type pyMsgWfWaitCase struct {
	name string
	path string
	wait func(ctx context.Context, client *gophercloud.ServiceClient, options ...resource.WaitOption) error
}

func pyMsgWfWaitCases() []pyMsgWfWaitCase {
	return []pyMsgWfWaitCase{
		{"queue", pyMsgWfJobsPath, func(ctx context.Context, c *gophercloud.ServiceClient, o ...resource.WaitOption) error {
			return queues.New(c).WaitForDeletion(ctx, "jobs", o...)
		}},
		{"message", pyMsgWfJobsPath + "/messages/m1", func(ctx context.Context, c *gophercloud.ServiceClient, o ...resource.WaitOption) error {
			return messages.New(c).WaitForDeletion(ctx, "jobs", "m1", o...)
		}},
		{"claim", pyMsgWfJobsPath + "/claims/c1", func(ctx context.Context, c *gophercloud.ServiceClient, o ...resource.WaitOption) error {
			return claims.New(c).WaitForDeletion(ctx, "jobs", "c1", o...)
		}},
	}
}

const pyMsgWfOne = `{"href":"/x","ttl":1,"age":1,"body":{},"messages":[],"_default_message_ttl":1}`

// wait_for_delete polls GET until a 404; present resources keep the wait going.
func TestPyMsgWfWaitForDeletionPollsUntilNotFound(t *testing.T) {
	t.Parallel()
	for _, tc := range pyMsgWfWaitCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				pyMsgWfRequireIdentity(t, r)
				if calls.Add(1) < 3 {
					testcloud.JSON(w, 200, pyMsgWfOne)
					return
				}
				testcloud.JSON(w, 404, `{}`)
			})
			err := tc.wait(context.Background(), pyMsgWfClient(cloud), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5*time.Second))
			if err != nil || calls.Load() != 3 {
				t.Fatalf("error=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestPyMsgWfWaitForDeletionAlreadyMissingServerErrorsAndTimeout(t *testing.T) {
	t.Parallel()
	for _, tc := range pyMsgWfWaitCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status := atomic.Int32{}
			status.Store(404)
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if status.Load() == 200 {
					testcloud.JSON(w, 200, pyMsgWfOne)
					return
				}
				testcloud.JSON(w, int(status.Load()), `{}`)
			})
			ctx := context.Background()
			fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(100 * time.Millisecond)}
			if err := tc.wait(ctx, pyMsgWfClient(cloud), fast...); err != nil || calls.Load() != 1 {
				t.Fatalf("already missing: error=%v calls=%d", err, calls.Load())
			}
			status.Store(500)
			if err := tc.wait(ctx, pyMsgWfClient(cloud), fast...); err == nil || errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("a server error must fail the wait: %v", err)
			}
			status.Store(200)
			if err := tc.wait(ctx, pyMsgWfClient(cloud), fast...); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("a resource that stays present must time out: %v", err)
			}
		})
	}
}

func TestPyMsgWfWaitForDeletionRejectsInvalidOptionsAndCancellationBeforeHTTP(t *testing.T) {
	t.Parallel()
	for _, tc := range pyMsgWfWaitCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cloud := testcloud.New(t)
			zaqarQueueNoRequests(t, cloud)
			ctx := context.Background()
			if err := tc.wait(ctx, pyMsgWfClient(cloud), resource.WithTimeout(0)); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("zero timeout: %v", err)
			}
			if err := tc.wait(ctx, pyMsgWfClient(cloud), nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("nil option: %v", err)
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if err := tc.wait(canceled, pyMsgWfClient(cloud)); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled: %v", err)
			}
		})
	}
}

func TestPyMsgWfSubscriptionWaitForDeletionPollsUntilNotFound(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	status := atomic.Int32{}
	status.Store(200)
	cloud.Mux.HandleFunc("GET "+subscriptionsPrefix+"/queues/jobs/subscriptions/s1", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Client-ID") != subscriptionsClientID {
			t.Errorf("Client-ID=%q", r.Header.Get("Client-ID"))
		}
		if n := calls.Add(1); n >= 3 && status.Load() == 200 {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, int(status.Load()), `{"id":"s1","subscriber":"http://x","ttl":60}`)
	})
	scope := subscriptionScope(t, subscriptionClient(cloud), "jobs")
	ctx := context.Background()
	fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
	if err := scope.WaitForDeletion(ctx, "s1", fast...); err != nil || calls.Load() != 3 {
		t.Fatalf("error=%v calls=%d", err, calls.Load())
	}
	status.Store(500)
	if err := scope.WaitForDeletion(ctx, "s1", fast...); err == nil || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("a server error must fail the wait: %v", err)
	}
	status.Store(200)
	calls.Store(-1 << 20)
	if err := scope.WaitForDeletion(ctx, "s1", resource.WithPollInterval(time.Millisecond), resource.WithTimeout(50*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a present subscription must time out: %v", err)
	}
	if err := scope.WaitForDeletion(ctx, "s1", resource.WithTimeout(0)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("invalid option: %v", err)
	}
}
