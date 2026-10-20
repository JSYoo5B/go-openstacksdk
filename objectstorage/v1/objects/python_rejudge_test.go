package objects_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

type pythonRejudgeObjectReply struct {
	status  int
	headers map[string]string
	body    string
}

// pythonRejudgeObjectAPI serves replies in order and records method, URI,
// Content-Type and body of each request.
func pythonRejudgeObjectAPI(t *testing.T, replies ...pythonRejudgeObjectReply) (*objects.API, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	var seen []string
	cloud.Mux.HandleFunc("/v1/AUTH_project/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		seen = append(seen, fmt.Sprintf("%s %s ct=%s body=%q", r.Method, r.URL.RequestURI(), r.Header.Get("Content-Type"), raw))
		if len(seen) > len(replies) {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		reply := replies[len(seen)-1]
		for name, value := range reply.headers {
			w.Header().Set(name, value)
		}
		w.WriteHeader(reply.status)
		_, _ = io.WriteString(w, reply.body)
	})
	return objects.New(cloud.Client("object-store", "/v1/AUTH_project/")), &seen
}

func pythonRejudgeObjectSequence(t *testing.T, seen []string, want ...string) {
	t.Helper()
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Fatalf("requests = %q, want %q", seen, want)
	}
}

// Python download_object GETs the object and returns its bytes. Go escapes a
// slash inside the object name, which Swift decodes to the same path.
func TestPythonRejudgeObjectDownloadRequest(t *testing.T) {
	api, seen := pythonRejudgeObjectAPI(t, pythonRejudgeObjectReply{200, map[string]string{"Content-Type": "application/octet-stream", "Etag": "abc"}, "payload"})
	var out bytes.Buffer
	value, err := api.DownloadObject(context.Background(), "c1", "dir/obj", &out)
	if err != nil || out.String() != "payload" || value.BytesWritten != 7 || value.StatusCode != 200 {
		t.Fatal(value, err)
	}
	pythonRejudgeObjectSequence(t, *seen, `GET /v1/AUTH_project/c1/dir%2Fobj ct= body=""`)
}

// Python delete_object HEADs to learn X-Static-Large-Object, then DELETEs,
// adding multipart-manifest=delete for SLO manifests. ignore_missing=True
// turns a 404 into a quiet return.
func TestPythonRejudgeObjectDeleteRequests(t *testing.T) {
	ctx := context.Background()
	t.Run("ordinary object", func(t *testing.T) {
		api, seen := pythonRejudgeObjectAPI(t, pythonRejudgeObjectReply{200, nil, ""}, pythonRejudgeObjectReply{204, nil, ""})
		value, err := api.DeleteObject(ctx, "c1", "obj")
		if err != nil || value.StaticLargeObject == nil || *value.StaticLargeObject || value.IgnoredMissing {
			t.Fatal(value, err)
		}
		pythonRejudgeObjectSequence(t, *seen, `HEAD /v1/AUTH_project/c1/obj ct= body=""`, `DELETE /v1/AUTH_project/c1/obj ct= body=""`)
	})
	t.Run("static large object", func(t *testing.T) {
		report := `{"Number Deleted":3,"Number Not Found":0,"Response Body":"","Response Status":"200 OK","Errors":[]}`
		api, seen := pythonRejudgeObjectAPI(t, pythonRejudgeObjectReply{200, map[string]string{"X-Static-Large-Object": "True"}, ""}, pythonRejudgeObjectReply{200, map[string]string{"Content-Type": "application/json"}, report})
		value, err := api.DeleteObject(ctx, "c1", "obj")
		if err != nil || value.StaticLargeObject == nil || !*value.StaticLargeObject || value.Bulk == nil {
			t.Fatal(value, err)
		}
		pythonRejudgeObjectSequence(t, *seen, `HEAD /v1/AUTH_project/c1/obj ct= body=""`, `DELETE /v1/AUTH_project/c1/obj?multipart-manifest=delete ct= body=""`)
	})
	t.Run("missing object", func(t *testing.T) {
		api, seen := pythonRejudgeObjectAPI(t, pythonRejudgeObjectReply{404, nil, ""})
		value, err := api.DeleteObject(ctx, "c1", "obj")
		if err != nil || !value.IgnoredMissing {
			t.Fatal(value, err)
		}
		pythonRejudgeObjectSequence(t, *seen, `HEAD /v1/AUTH_project/c1/obj ct= body=""`)
	})
}

// Python is_object_stale HEADs the object and compares x-sdk-md5/x-sdk-sha256
// metadata with the local hashes; a missing object is stale.
func TestPythonRejudgeObjectStaleRequests(t *testing.T) {
	ctx := context.Background()
	data := []byte("local file")
	filename := filepath.Join(t.TempDir(), "local")
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		t.Fatal(err)
	}
	md5sum, sha := md5.Sum(data), sha256.Sum256(data)
	localMD5, localSHA := hex.EncodeToString(md5sum[:]), hex.EncodeToString(sha[:])
	cases := []struct {
		name  string
		reply pythonRejudgeObjectReply
		stale bool
	}{
		{"matching hashes", pythonRejudgeObjectReply{200, map[string]string{"X-Object-Meta-X-Sdk-Md5": localMD5, "X-Object-Meta-X-Sdk-Sha256": localSHA}, ""}, false},
		{"different hash", pythonRejudgeObjectReply{200, map[string]string{"X-Object-Meta-X-Sdk-Md5": "0", "X-Object-Meta-X-Sdk-Sha256": localSHA}, ""}, true},
		{"missing object", pythonRejudgeObjectReply{404, nil, ""}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api, seen := pythonRejudgeObjectAPI(t, tc.reply)
			value, err := api.IsObjectStale(ctx, "c1", "obj", filename)
			if err != nil || value.Stale == nil || *value.Stale != tc.stale {
				t.Fatal(value, err)
			}
			pythonRejudgeObjectSequence(t, *seen, `HEAD /v1/AUTH_project/c1/obj ct= body=""`)
		})
	}
}

// The Python cloud create_directory_marker_object PUTs an empty body with
// content-type application/directory and no checksum metadata.
func TestPythonRejudgeObjectDirectoryMarkerRequest(t *testing.T) {
	api, seen := pythonRejudgeObjectAPI(t, pythonRejudgeObjectReply{201, map[string]string{"Etag": "d41d8cd98f00b204e9800998ecf8427e"}, ""})
	value, err := api.CreateDirectoryMarkerObject(context.Background(), "c1", "dir")
	if err != nil || value.Size != 0 || value.Mode != "ordinary" {
		t.Fatal(value, err)
	}
	pythonRejudgeObjectSequence(t, *seen, `PUT /v1/AUTH_project/c1/dir ct=application/directory body=""`)
}
