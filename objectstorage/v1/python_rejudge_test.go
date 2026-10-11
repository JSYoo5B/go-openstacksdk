package v1_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	objectstorage "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

type pythonRejudgeSwiftReply struct {
	status  int
	headers map[string]string
	body    string
}

// pythonRejudgeSwiftService answers requests by "METHOD path" and records the
// order, mirroring the HEAD/GET sequence of the pinned Python proxy.
func pythonRejudgeSwiftService(t *testing.T, replies map[string]pythonRejudgeSwiftReply) (*objectstorage.Service, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	var seen []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.RequestURI()
		seen = append(seen, key)
		reply, ok := replies[key]
		if !ok {
			t.Errorf("unexpected request %s", key)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for name, value := range reply.headers {
			w.Header().Set(name, value)
		}
		if reply.body == "" {
			w.WriteHeader(reply.status)
			return
		}
		testcloud.JSON(w, reply.status, reply.body)
	})
	return objectstorage.New(cloud.Client("object-store", "/v1/AUTH_project/")), &seen
}

func pythonRejudgeSwiftSequence(t *testing.T, seen []string, want ...string) {
	t.Helper()
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Fatalf("requests = %q, want %q", seen, want)
	}
}

func pythonRejudgeSwiftHMAC(key, body string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

// Python get_temp_url_key prefers container Temp-URL-Key-2, then Temp-URL-Key,
// and only then reads the account in the same order.
func TestPythonRejudgeSwiftTempURLKeySelection(t *testing.T) {
	ctx := context.Background()
	t.Run("account only", func(t *testing.T) {
		service, seen := pythonRejudgeSwiftService(t, map[string]pythonRejudgeSwiftReply{
			"HEAD /v1/AUTH_project/": {204, map[string]string{"X-Account-Meta-Temp-Url-Key": "primary", "X-Account-Meta-Temp-Url-Key-2": "secondary"}, ""},
		})
		value, err := service.GetTempURLKey(ctx)
		if err != nil || string(value.Key) != "secondary" || value.FromContainer || !value.Secondary {
			t.Fatal(value, err)
		}
		pythonRejudgeSwiftSequence(t, *seen, "HEAD /v1/AUTH_project/")
	})
	t.Run("container key wins", func(t *testing.T) {
		service, seen := pythonRejudgeSwiftService(t, map[string]pythonRejudgeSwiftReply{
			"HEAD /v1/AUTH_project/c1": {204, map[string]string{"X-Container-Meta-Temp-Url-Key": "container"}, ""},
		})
		value, err := service.GetTempURLKey(ctx, objectstorage.WithGetTempURLKeyContainer("c1"))
		if err != nil || string(value.Key) != "container" || !value.FromContainer || value.Secondary {
			t.Fatal(value, err)
		}
		pythonRejudgeSwiftSequence(t, *seen, "HEAD /v1/AUTH_project/c1")
	})
	t.Run("container falls back to account", func(t *testing.T) {
		service, seen := pythonRejudgeSwiftService(t, map[string]pythonRejudgeSwiftReply{
			"HEAD /v1/AUTH_project/c1": {204, nil, ""},
			"HEAD /v1/AUTH_project/":   {204, map[string]string{"X-Account-Meta-Temp-Url-Key": "account"}, ""},
		})
		value, err := service.GetTempURLKey(ctx, objectstorage.WithGetTempURLKeyContainer("c1"))
		if err != nil || string(value.Key) != "account" || value.FromContainer {
			t.Fatal(value, err)
		}
		pythonRejudgeSwiftSequence(t, *seen, "HEAD /v1/AUTH_project/c1", "HEAD /v1/AUTH_project/")
	})
}

// Python generate_temp_url signs "METHOD\nexpires\npath" with HMAC-SHA1 and,
// without temp_url_key, reads only the account key.
func TestPythonRejudgeSwiftGenerateTempURL(t *testing.T) {
	now := time.Unix(1700000000, 0)
	path := "/v1/AUTH_project/c1/obj"
	want := pythonRejudgeSwiftHMAC("secret", fmt.Sprintf("GET\n%d\n%s", now.Unix()+60, path))
	t.Run("explicit key sends nothing", func(t *testing.T) {
		service, seen := pythonRejudgeSwiftService(t, nil)
		value, err := service.GenerateTempURL(context.Background(), path, 60, "GET", objectstorage.WithGenerateTempURLKey([]byte("secret")), objectstorage.WithGenerateTempURLTimestamp(now))
		if err != nil || value.Signature != want || value.Expires != now.Unix()+60 || value.Path != path {
			t.Fatal(value, err)
		}
		pythonRejudgeSwiftSequence(t, *seen)
	})
	t.Run("discovered account key", func(t *testing.T) {
		service, seen := pythonRejudgeSwiftService(t, map[string]pythonRejudgeSwiftReply{
			"HEAD /v1/AUTH_project/": {204, map[string]string{"X-Account-Meta-Temp-Url-Key": "secret"}, ""},
		})
		value, err := service.GenerateTempURL(context.Background(), path, 60, "GET", objectstorage.WithGenerateTempURLTimestamp(now))
		if err != nil || value.Signature != want {
			t.Fatal(value, err)
		}
		pythonRejudgeSwiftSequence(t, *seen, "HEAD /v1/AUTH_project/")
	})
}

// Python generate_form_signature signs the endpoint path joined with container
// and prefix, reading the container key before the account key.
func TestPythonRejudgeSwiftGenerateFormSignature(t *testing.T) {
	now := time.Unix(1700000000, 0)
	service, seen := pythonRejudgeSwiftService(t, map[string]pythonRejudgeSwiftReply{
		"HEAD /v1/AUTH_project/c1": {204, nil, ""},
		"HEAD /v1/AUTH_project/":   {204, map[string]string{"X-Account-Meta-Temp-Url-Key": "secret"}, ""},
	})
	input := objectstorage.FormSignatureInput{ObjectPrefix: "uploads/", RedirectURL: "https://example.invalid/done", MaxFileSize: 1024, MaxUploadCount: 3, Timeout: 600}
	value, err := service.GenerateFormSignature(context.Background(), "c1", input, objectstorage.WithGenerateFormSignatureTimestamp(now))
	expires := now.Unix() + 600
	want := pythonRejudgeSwiftHMAC("secret", fmt.Sprintf("/v1/AUTH_project/c1/uploads/\nhttps://example.invalid/done\n1024\n3\n%d", expires))
	if err != nil || value.Signature != want || value.Expires != expires {
		t.Fatal(value, err)
	}
	pythonRejudgeSwiftSequence(t, *seen, "HEAD /v1/AUTH_project/c1", "HEAD /v1/AUTH_project/")
}

// Python get_info reads /info beside the versioned endpoint, and
// get_object_segment_size(None) clamps 1 GiB to the published bounds or falls
// back to 2684354561 when /info is 404.
func TestPythonRejudgeSwiftInfoAndSegmentSize(t *testing.T) {
	ctx := context.Background()
	info := `{"swift":{"max_file_size":5368709122,"version":"2.30.0"},"slo":{"min_segment_size":1048576}}`
	t.Run("get_info", func(t *testing.T) {
		service, seen := pythonRejudgeSwiftService(t, map[string]pythonRejudgeSwiftReply{"GET /info": {200, nil, info}})
		value, err := service.GetInfo(ctx)
		if err != nil || string(value.Swift["max_file_size"]) != "5368709122" || string(value.SLO["min_segment_size"]) != "1048576" {
			t.Fatal(value, err)
		}
		pythonRejudgeSwiftSequence(t, *seen, "GET /info")
	})
	t.Run("default segment size", func(t *testing.T) {
		service, seen := pythonRejudgeSwiftService(t, map[string]pythonRejudgeSwiftReply{"GET /info": {200, nil, info}})
		value, err := service.GetObjectSegmentSize(ctx)
		if err != nil || value.Size != 1073741824 || value.UsedFallback {
			t.Fatal(value, err)
		}
		pythonRejudgeSwiftSequence(t, *seen, "GET /info")
	})
	t.Run("missing info falls back", func(t *testing.T) {
		service, seen := pythonRejudgeSwiftService(t, map[string]pythonRejudgeSwiftReply{"GET /info": {404, nil, ""}})
		value, err := service.GetObjectSegmentSize(ctx, objectstorage.WithObjectSegmentSize(4294967296))
		if err != nil || value.Size != 2684354561 || !value.UsedFallback {
			t.Fatal(value, err)
		}
		pythonRejudgeSwiftSequence(t, *seen, "GET /info")
	})
}

// Python set_account_temp_url_key and set_container_temp_url_key POST the
// Temp-URL-Key metadata header (Temp-URL-Key-2 when secondary). Python then
// sends a HEAD to refresh a Resource it discards; Go sends only the POST.
func TestPythonRejudgeSwiftSetTempURLKeyRequests(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var seen []string
	cloud.Mux.HandleFunc("/v1/AUTH_project/", func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, fmt.Sprintf("%s %s account=%q account2=%q container=%q container2=%q", r.Method, r.URL.RequestURI(),
			r.Header.Get("X-Account-Meta-Temp-Url-Key"), r.Header.Get("X-Account-Meta-Temp-Url-Key-2"),
			r.Header.Get("X-Container-Meta-Temp-Url-Key"), r.Header.Get("X-Container-Meta-Temp-Url-Key-2")))
		w.WriteHeader(http.StatusNoContent)
	})
	client := cloud.Client("object-store", "/v1/AUTH_project/")
	if _, err := accounts.New(client).SetTempURLKey(ctx, "k1"); err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.New(client).SetTempURLKey(ctx, "k2", accounts.WithSetTempURLKeySecondary(true)); err != nil {
		t.Fatal(err)
	}
	if _, err := containers.New(client).SetTempURLKey(ctx, "c1", "k3"); err != nil {
		t.Fatal(err)
	}
	if _, err := containers.New(client).SetTempURLKey(ctx, "c1", "k4", containers.WithSetTempURLKeySecondary(true)); err != nil {
		t.Fatal(err)
	}
	pythonRejudgeSwiftSequence(t, seen,
		`POST /v1/AUTH_project/ account="k1" account2="" container="" container2=""`,
		`POST /v1/AUTH_project/ account="" account2="k2" container="" container2=""`,
		`POST /v1/AUTH_project/c1 account="" account2="" container="k3" container2=""`,
		`POST /v1/AUTH_project/c1 account="" account2="" container="" container2="k4"`)
}
