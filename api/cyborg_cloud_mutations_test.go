package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const cyborgCloudProfileBody = `{"device_profile":{"uuid":"` + cyborgCloudProfileUUID + `","name":"fpga","description":"","groups":[{"resources:FPGA":"1"}]}}`

func TestCyborgCloudProfileCreateSendsOneElementArrayAndKeepsEmptyDescription(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/device_profiles", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := string(cyborgCloudDecode(t, r)); got != `[{"description":"","groups":[{"resources:FPGA":"1"}],"name":"fpga","vendor":false}]` {
			t.Errorf("body=%s", got)
		}
		testcloud.JSON(w, 201, cyborgCloudProfileBody)
	})
	empty := ""
	value, err := deviceprofiles.New(cloud.Client("accelerator", "/v2")).Create(context.Background(),
		deviceprofiles.CreateOpts{Name: "fpga", Groups: []deviceprofiles.Group{{"resources:FPGA": "1"}}, Description: &empty},
		deviceprofiles.WithCreateField("vendor", false))
	if err != nil || value == nil || value.UUID != cyborgCloudProfileUUID || value.Description == nil || *value.Description != "" || value.StatusCode != 201 || calls.Load() != 1 {
		t.Fatalf("value=%+v err=%v calls=%d", value, err, calls.Load())
	}
}

func TestCyborgCloudProfileCreateOmitsNilDescription(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/device_profiles", func(w http.ResponseWriter, r *http.Request) {
		if got := string(cyborgCloudDecode(t, r)); got != `[{"groups":[{}],"name":"fpga"}]` {
			t.Errorf("body=%s", got)
		}
		testcloud.JSON(w, 201, cyborgCloudProfileBody)
	})
	if _, err := deviceprofiles.New(cloud.Client("accelerator", "/v2")).Create(context.Background(),
		deviceprofiles.CreateOpts{Name: "fpga", Groups: []deviceprofiles.Group{{}}}); err != nil {
		t.Fatal(err)
	}
}

func TestCyborgCloudProfileCreateAcceptsOnly201AndNeverReplays(t *testing.T) {
	for _, status := range []int{200, 202, 204, 500} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/device_profiles", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			testcloud.JSON(w, status, cyborgCloudProfileBody)
		})
		value, err := deviceprofiles.New(cloud.Client("accelerator", "/v2")).Create(context.Background(),
			deviceprofiles.CreateOpts{Name: "fpga", Groups: []deviceprofiles.Group{{}}})
		if value != nil || !cyborgCloudIsStatus(err, status) || calls.Load() != 1 {
			t.Fatalf("status %d value=%+v err=%v calls=%d", status, value, err, calls.Load())
		}
	}
}

func TestCyborgCloudProfileCreateRejectsInvalidInputBeforeHTTP(t *testing.T) {
	cloud, calls := cyborgCloudNoHTTP(t)
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	ctx := context.Background()
	good := deviceprofiles.CreateOpts{Name: "fpga", Groups: []deviceprofiles.Group{{}}}
	bad := "not-a-uuid"
	for name, run := range map[string]func() error{
		"nil option": func() error { _, err := a.Create(ctx, good, nil); return err },
		"name collision": func() error {
			_, err := a.Create(ctx, good, deviceprofiles.WithCreateField("name", "other"))
			return err
		},
		"groups override": func() error {
			_, err := a.Create(ctx, good, deviceprofiles.WithCreateField("groups", []int{}))
			return err
		},
		"auth header": func() error {
			_, err := a.Create(ctx, good, deviceprofiles.WithCreateHeader("X-Auth-Token", "t"))
			return err
		},
		"empty name": func() error { _, err := a.Create(ctx, deviceprofiles.CreateOpts{Groups: good.Groups}); return err },
		"no groups":  func() error { _, err := a.Create(ctx, deviceprofiles.CreateOpts{Name: "fpga"}); return err },
		"null group": func() error {
			_, err := a.Create(ctx, deviceprofiles.CreateOpts{Name: "fpga", Groups: []deviceprofiles.Group{nil}})
			return err
		},
		"bad uuid": func() error {
			_, err := a.Create(ctx, deviceprofiles.CreateOpts{Name: "fpga", Groups: good.Groups, UUID: &bad})
			return err
		},
	} {
		if err := run(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := a.Create(cyborgCloudCanceled(), good); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight sent HTTP")
	}
}

func TestCyborgCloudProfileCreateDeadlineKeepsCauseWithoutReplay(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cyborgCloudHang(cloud, "POST /v2/device_profiles", &calls)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := deviceprofiles.New(cloud.Client("accelerator", "/v2")).Create(ctx, deviceprofiles.CreateOpts{Name: "fpga", Groups: []deviceprofiles.Group{{}}})
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestCyborgCloudProfileDeleteByIDMissingPolicyAndStatusCodes(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	status := 204
	cloud.Mux.HandleFunc("DELETE /v2/device_profiles/"+cyborgCloudProfileUUID, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.ContentLength > 0 || r.URL.RawQuery != "" {
			t.Errorf("delete has body/query: %s", r.URL)
		}
		testcloud.JSON(w, status, `{}`)
	})
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	ctx := context.Background()
	if err := a.Delete(ctx, resource.ID(cyborgCloudProfileUUID)); err != nil {
		t.Fatal(err)
	}
	status = 404
	if err := a.Delete(ctx, resource.ID(cyborgCloudProfileUUID)); err != nil {
		t.Fatalf("default ignores missing: %v", err)
	}
	err := a.Delete(ctx, resource.ID(cyborgCloudProfileUUID), resource.WithMissingError())
	if !errors.Is(err, resource.ErrNotFound) || !cyborgCloudIsStatus(err, 404) {
		t.Fatalf("missing error: %v", err)
	}
	for _, code := range []int{200, 202, 500} {
		status = code
		before := calls.Load()
		if err := a.Delete(ctx, resource.ID(cyborgCloudProfileUUID)); !cyborgCloudIsStatus(err, code) || calls.Load() != before+1 {
			t.Fatalf("status %d: %v calls=%d", code, err, calls.Load()-before)
		}
	}
}

func TestCyborgCloudProfileDeleteByNameResolvesExactNameToUUIDEndpoint(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/device_profiles", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("name lookup query %q", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"device_profiles":[{"uuid":"`+cyborgCloudProfileUUID+`","name":"fpga","groups":[]},{"uuid":"22222222-2222-4222-8222-222222222222","name":"fpga-2","groups":[]}]}`)
	})
	cloud.Mux.HandleFunc("DELETE /v2/device_profiles/"+cyborgCloudProfileUUID, func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.WriteHeader(204)
	})
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	if err := a.Delete(context.Background(), resource.Name("fpga")); err != nil || lists.Load() != 1 || deletes.Load() != 1 {
		t.Fatalf("err=%v lists=%d deletes=%d", err, lists.Load(), deletes.Load())
	}
	if err := a.Delete(context.Background(), resource.Name("absent")); err != nil || deletes.Load() != 1 {
		t.Fatalf("absent name: %v deletes=%d", err, deletes.Load())
	}
}

func TestCyborgCloudProfileDeleteRejectsUnsafeIDAndCanceledContextBeforeHTTP(t *testing.T) {
	cloud, calls := cyborgCloudNoHTTP(t)
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	for _, id := range []string{"", "not-a-uuid", cyborgCloudProfileUUID + ",x", "../x"} {
		if err := a.Delete(context.Background(), resource.ID(id)); err == nil {
			t.Errorf("id %q accepted", id)
		}
	}
	if err := a.Delete(cyborgCloudCanceled(), resource.ID(cyborgCloudProfileUUID)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight sent HTTP")
	}
}

func TestCyborgCloudRequestCreateBodyAcceptsOnly201AndIsAcknowledgementOnly(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	status := 201
	cloud.Mux.HandleFunc("POST /v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := string(cyborgCloudDecode(t, r)); got != `{"device_profile_name":"fpga","extra":false}` {
			t.Errorf("body=%s", got)
		}
		testcloud.JSON(w, status, `{"arqs":[{"uuid":"a1","state":"Initial","hostname":null}]}`)
	})
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	v, err := a.Create(context.Background(), acceleratorrequests.CreateOpts{DeviceProfileName: "fpga"}, acceleratorrequests.WithCreateField("extra", false))
	if err != nil || len(v.Requests) != 1 || v.Requests[0].State != "Initial" || calls.Load() != 1 {
		t.Fatalf("v=%+v err=%v calls=%d", v, err, calls.Load())
	}
	for _, code := range []int{200, 202, 500} {
		status = code
		v, err := a.Create(context.Background(), acceleratorrequests.CreateOpts{DeviceProfileName: "fpga"}, acceleratorrequests.WithCreateField("extra", false))
		if v != nil || !cyborgCloudIsStatus(err, code) {
			t.Fatalf("status %d: %+v %v", code, v, err)
		}
	}
	if calls.Load() != 4 {
		t.Fatalf("calls=%d; each failure must be one attempt", calls.Load())
	}
}

func TestCyborgCloudRequestCreateDeadlineKeepsCauseWithoutReplay(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cyborgCloudHang(cloud, "POST /v2/accelerator_requests", &calls)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := acceleratorrequests.New(cloud.Client("accelerator", "/v2")).Create(ctx, acceleratorrequests.CreateOpts{DeviceProfileName: "fpga"})
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestCyborgCloudRequestDeleteSelectorsStatusCodesAndNoReplay(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	status := 204
	cloud.Mux.HandleFunc("DELETE /v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "arqs=a1" || r.ContentLength > 0 {
			t.Errorf("delete %s", r.URL)
		}
		testcloud.JSON(w, status, `{}`)
	})
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	ctx := context.Background()
	if err := a.Delete(ctx, resource.ID("a1")); err != nil {
		t.Fatal(err)
	}
	status = 404
	if err := a.Delete(ctx, resource.ID("a1")); err != nil {
		t.Fatalf("default ignores missing: %v", err)
	}
	if err := a.Delete(ctx, resource.ID("a1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	for _, code := range []int{200, 202, 500} {
		status = code
		before := calls.Load()
		if err := a.Delete(ctx, resource.ID("a1")); !cyborgCloudIsStatus(err, code) || calls.Load() != before+1 {
			t.Fatalf("status %d: %v", code, err)
		}
	}
}

func TestCyborgCloudRequestDeleteRejectsInvalidInputBeforeHTTP(t *testing.T) {
	cloud, calls := cyborgCloudNoHTTP(t)
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	ctx := context.Background()
	if err := a.Delete(ctx, resource.ID("a1,a2")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := a.Delete(ctx, resource.ID("")); err == nil {
		t.Fatal("empty ID accepted")
	}
	if err := a.Delete(cyborgCloudCanceled(), resource.ID("a1")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := a.DeleteMany(cyborgCloudCanceled(), []string{"a1"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := a.DeleteMany(ctx, []string{"a1"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight sent HTTP")
	}
}

func TestCyborgCloudRequestDeleteDeadlineKeepsCauseWithoutReplay(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cyborgCloudHang(cloud, "DELETE /v2/accelerator_requests", &calls)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := acceleratorrequests.New(cloud.Client("accelerator", "/v2")).Delete(ctx, resource.ID("a1"))
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}
