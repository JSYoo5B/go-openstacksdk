package compute_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func backupServerWire(status int, body, stage string) *http.Response {
	return testcloud.Response(status, body, http.Header{"Content-Type": {"application/json"}, "X-Backup-Stage": {stage}})
}

type backupServerFixture struct {
	service      *compute.Service
	image        *image.Service
	nova, glance *gophercloud.ServiceClient
	cloud        *testcloud.Cloud
	calls        []string
	imageLoads   int
}

func newBackupServerFixture(t *testing.T, reply func(*http.Request) *http.Response) *backupServerFixture {
	t.Helper()
	cloud := testcloud.New(t)
	f := &backupServerFixture{cloud: cloud}
	f.nova = cloud.Client("compute", "/nova/v2.1/")
	f.nova.Microversion = "2.96"
	f.glance = cloud.Client("image", "/glance/v2/")
	f.image = image.New(f.glance)
	f.service = compute.New(f.nova, compute.Dependencies{BackupImages: func(ctx context.Context) (*image.Service, error) {
		f.imageLoads++
		f.calls = append(f.calls, "load-image")
		return f.image, nil
	}})
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		f.calls = append(f.calls, req.Method+" "+req.URL.Path)
		if req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		if req.URL.Path == "/nova/v2.1/" {
			if req.Header.Get("X-OpenStack-Nova-API-Version") != "" {
				t.Error("discovery was versioned", req.Header)
			}
			return backupServerWire(200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.100","links":[{"rel":"self","href":"https://advertised.invalid/v2.1/"}]}}`, "discovery"), nil
		}
		if req.Method == http.MethodPost {
			data, _ := io.ReadAll(req.Body)
			if string(data) != `{"createBackup":{"backup_type":"daily","name":"backup","rotation":3}}` {
				t.Error(string(data))
			}
			if req.Header.Get("X-OpenStack-Nova-API-Version") != "2.45" || req.Header.Get("Accept") != "" {
				t.Error(req.Header)
			}
			return reply(req), nil
		}
		return reply(req), nil
	}))
	return f
}

var backupServerInput = compute.BackupServerRequest{Name: "backup", BackupType: "daily", Rotation: 3}

func assertBackupStage(t *testing.T, f *backupServerFixture, result *compute.BackupServerResult, err error) {
	t.Helper()
	if err != nil || result == nil || result.Action == nil || result.Action.ImageID != "img1" || result.Action.Microversion != "2.45" || result.Action.StatusCode != 202 || result.Action.Header.Get("X-Backup-Stage") != "action" || len(result.Action.Discovery.Responses) != 1 || result.Action.Discovery.Responses[0].Header.Get("X-Backup-Stage") != "discovery" {
		t.Fatal(result, err)
	}
	if f.nova.Microversion != "2.96" {
		t.Fatal("shared selected version changed", f.nova.Microversion)
	}
}

func TestBackupServerDefaultLookupAndConcreteOverride(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(map[bool]string{false: "lazy", true: "concrete"}[override], func(t *testing.T) {
			f := newBackupServerFixture(t, func(req *http.Request) *http.Response {
				if req.Method == http.MethodPost {
					return backupServerWire(202, `{"image_id":"img1"}`, "action")
				}
				return backupServerWire(200, `{"id":"img1","name":"backup","status":"queued","x_extra":7}`, "lookup")
			})
			var options []compute.BackupServerOption
			if override {
				options = append(options, compute.WithBackupImageService(f.image))
			}
			result, err := f.service.BackupServer(context.Background(), "s1", backupServerInput, options...)
			assertBackupStage(t, f, result, err)
			if result.Lookup == nil || result.Image == nil || result.Image.StatusCode != 200 || result.Image.Header.Get("X-Backup-Stage") != "lookup" || string(result.Image.Resource.Body["status"]) != `"queued"` || result.Wait != nil {
				t.Fatal(result)
			}
			want := []string{"GET /nova/v2.1/", "POST /nova/v2.1/servers/s1/action", "load-image", "GET /glance/v2/images/img1"}
			if override {
				want = []string{want[0], want[1], want[3]}
			}
			if strings.Join(f.calls, "|") != strings.Join(want, "|") {
				t.Fatal(f.calls, want)
			}
			if f.imageLoads != map[bool]int{false: 1, true: 0}[override] {
				t.Fatal(f.imageLoads)
			}
		})
	}
}

func TestBackupServerWaitUsesCloudCaseAndFreshLookups(t *testing.T) {
	statuses := []string{"active", "ACTIVE", "queued", "active"}
	f := newBackupServerFixture(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return backupServerWire(202, `{"image_id":"img1"}`, "action")
		}
		status := statuses[0]
		statuses = statuses[1:]
		return backupServerWire(200, `{"id":"img1","status":"`+status+`"}`, "image")
	})
	result, err := f.service.BackupServer(context.Background(), "s1", backupServerInput, compute.WithBackupServerWait(true), compute.WithBackupServerPollInterval(time.Nanosecond))
	assertBackupStage(t, f, result, err)
	if result.Wait == nil || result.Wait.Lookups != 3 || result.Wait.Image == nil || result.Image != result.Wait.Image || string(result.Image.Resource.Body["status"]) != `"active"` || len(statuses) != 0 {
		t.Fatal(result, statuses)
	}
	if len(f.calls) != 7 || f.imageLoads != 1 {
		t.Fatal(f.calls, f.imageLoads)
	}
}

func TestBackupServerWaitFailureTimeoutAndMissingSeed(t *testing.T) {
	for _, scenario := range []string{"error", "zero timeout", "missing no wait", "missing wait"} {
		t.Run(scenario, func(t *testing.T) {
			getCount := 0
			f := newBackupServerFixture(t, func(req *http.Request) *http.Response {
				if req.Method == http.MethodPost {
					return backupServerWire(202, `{"image_id":"img1"}`, "action")
				}
				if strings.Contains(scenario, "missing") {
					if req.URL.Path == "/glance/v2/images/img1" {
						return backupServerWire(404, `{"message":"gone"}`, "missing")
					}
					return backupServerWire(200, `{"images":[]}`, "empty-list")
				}
				getCount++
				status := "queued"
				if getCount > 1 {
					status = "error"
				}
				return backupServerWire(200, `{"id":"img1","status":"`+status+`"}`, "image")
			})
			options := []compute.BackupServerOption{}
			if scenario != "missing no wait" {
				options = append(options, compute.WithBackupServerWait(true), compute.WithBackupServerPollInterval(time.Nanosecond))
			}
			if scenario == "zero timeout" {
				options = append(options, compute.WithBackupServerTimeout(0))
			}
			result, err := f.service.BackupServer(context.Background(), "s1", backupServerInput, options...)
			if result == nil || result.Action == nil || result.Action.StatusCode != 202 || result.Lookup == nil {
				t.Fatal(result, err)
			}
			switch scenario {
			case "error":
				if !errors.Is(err, resource.ErrFailedState) || result.Wait == nil || result.Wait.Lookups != 1 || result.Image == nil || string(result.Image.Resource.Body["status"]) != `"error"` {
					t.Fatal(result, err)
				}
			case "zero timeout":
				if !errors.Is(err, context.DeadlineExceeded) || result.Wait == nil || result.Wait.Lookups != 0 || getCount != 1 {
					t.Fatal(result, err, getCount)
				}
			case "missing no wait":
				if err != nil || result.Image != nil || result.Wait != nil {
					t.Fatal(result, err)
				}
			case "missing wait":
				if !errors.Is(err, resource.ErrInvalidOption) || result.Image != nil || result.Wait != nil {
					t.Fatal(result, err)
				}
			}
		})
	}
}

func TestBackupServerStopsAfterActionOrImageErrorsAndNeverCleansUp(t *testing.T) {
	for _, phase := range []string{"action", "image", "dependency"} {
		t.Run(phase, func(t *testing.T) {
			cause := errors.New("Glance unavailable")
			f := newBackupServerFixture(t, func(req *http.Request) *http.Response {
				if req.Method == http.MethodPost {
					if phase == "action" {
						return backupServerWire(500, "action-failure", "action-error")
					}
					return backupServerWire(202, `{"image_id":"img1"}`, "action")
				}
				return backupServerWire(500, "image-failure", "image-error")
			})
			if phase == "dependency" {
				f.service = compute.New(f.nova, compute.Dependencies{BackupImages: func(context.Context) (*image.Service, error) { f.imageLoads++; return nil, cause }})
			}
			result, err := f.service.BackupServer(context.Background(), "s1", backupServerInput, compute.WithBackupServerWait(true))
			if err == nil || result == nil || result.Action == nil || result.Wait != nil {
				t.Fatal(result, err)
			}
			if phase == "dependency" {
				if !errors.Is(err, cause) || result.Action.StatusCode != 202 || f.imageLoads != 1 {
					t.Fatal(result, err, f.imageLoads)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 500 {
					t.Fatal(err)
				}
				if phase == "action" && f.imageLoads != 0 {
					t.Fatal(f.imageLoads)
				}
				if phase == "image" && result.Action.StatusCode != 202 {
					t.Fatal(result)
				}
			}
			for _, call := range f.calls {
				if strings.HasPrefix(call, "DELETE") {
					t.Fatal("unexpected cleanup", f.calls)
				}
			}
		})
	}
}

type backupServerFaultBody struct {
	reader                io.Reader
	readErr, closeErr     error
	afterRead, afterClose func()
}

func (b *backupServerFaultBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if b.afterRead != nil {
		b.afterRead()
		b.afterRead = nil
	}
	if b.readErr != nil {
		err = b.readErr
		b.readErr = nil
	}
	return n, err
}
func (b *backupServerFaultBody) Close() error {
	if b.afterClose != nil {
		b.afterClose()
	}
	return b.closeErr
}

func TestBackupServerCrossServicePhysicalFaultsKeepActionProof(t *testing.T) {
	for _, phase := range []string{"action", "image"} {
		t.Run(phase, func(t *testing.T) {
			readErr, closeErr, cancelErr := errors.New("physical read"), errors.New("physical close"), errors.New("caller cancellation")
			ctx, cancel := context.WithCancelCause(context.Background())
			var f *backupServerFixture
			f = newBackupServerFixture(t, func(req *http.Request) *http.Response {
				if phase == "image" && req.Method == http.MethodPost {
					return backupServerWire(202, `{"image_id":"img1"}`, "action")
				}
				status, body, stage := 200, `{"id":"img1","status":"active"}`, "image"
				if phase == "action" {
					status, body, stage = 202, `{"image_id":"img1"}`, "action"
				}
				response := backupServerWire(status, "", stage)
				original := f.nova.ResourceBase
				response.Body = &backupServerFaultBody{reader: strings.NewReader(body), readErr: readErr, closeErr: closeErr, afterRead: func() { f.nova.ResourceBase = "https://changed.invalid/"; cancel(cancelErr) }, afterClose: func() { f.nova.ResourceBase = original }}
				return response
			})
			result, err := f.service.BackupServer(ctx, "s1", backupServerInput, compute.WithBackupServerWait(true))
			for _, cause := range []error{readErr, closeErr, cancelErr, context.Canceled, resource.ErrInvalidOption} {
				if !errors.Is(err, cause) {
					t.Fatal(cause, err)
				}
			}
			var receipt *resource.ResponseError
			if !errors.As(err, &receipt) || receipt.StatusCode != map[string]int{"action": 202, "image": 200}[phase] || receipt.Header.Get("X-Backup-Stage") != phase {
				t.Fatal(err, receipt)
			}
			if result == nil || result.Action == nil || result.Action.StatusCode != 202 || result.Wait != nil {
				t.Fatal(result, err)
			}
			if phase == "action" && f.imageLoads != 0 {
				t.Fatal(f.imageLoads)
			}
			actionCalls := 0
			for _, call := range f.calls {
				if strings.HasPrefix(call, "POST") {
					actionCalls++
				}
				if strings.HasPrefix(call, "DELETE") {
					t.Fatal(f.calls)
				}
			}
			if actionCalls != 1 {
				t.Fatal(f.calls)
			}
		})
	}
}

func TestBackupServerSourceAndOptionsFailBeforeWork(t *testing.T) {
	f := newBackupServerFixture(t, func(*http.Request) *http.Response { return backupServerWire(202, `{"image_id":"img1"}`, "action") })
	_, err := f.service.BackupServer(context.Background(), "s1", backupServerInput, func(*compute.BackupServerOpts) error { f.service.API.Servers = nil; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(f.calls) != 0 || f.imageLoads != 0 {
		t.Fatal(err, f.calls, f.imageLoads)
	}
	// Build a fresh facade; nested leaf callbacks must obey the same outer guard.
	f.service = compute.New(f.nova, compute.Dependencies{BackupImages: func(context.Context) (*image.Service, error) { f.imageLoads++; return f.image, nil }})
	_, err = f.service.BackupServer(context.Background(), "s1", backupServerInput, compute.WithBackupServerActionOptions(func(*request.Config[servers.BackupActionRequest]) error { f.service.API.Servers = nil; return nil }))
	if !errors.Is(err, resource.ErrInvalidOption) || len(f.calls) != 0 || f.imageLoads != 0 {
		t.Fatal(err, f.calls, f.imageLoads)
	}
	f.service = compute.New(f.nova, compute.Dependencies{})
	calls := 0
	cause := errors.New("pre-cancelled")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	_, err = f.service.BackupServer(ctx, "s1", backupServerInput, func(*compute.BackupServerOpts) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 0 || len(f.calls) != 0 {
		t.Fatal(err, calls, f.calls)
	}
	_, err = f.service.BackupServer(context.Background(), "s1", backupServerInput, nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(f.calls) != 0 {
		t.Fatal(err, f.calls)
	}
}
