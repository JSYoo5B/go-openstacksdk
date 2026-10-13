package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSwiftChunkStreamDiscoversLazilyAndSharesCachedFacade(t *testing.T) {
	cloud := testcloud.New(t)
	locates, requests, callbacks := 0, 0, 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return cloud.Server.URL + "/swift/v1/AUTH_p/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /swift/v1/AUTH_p/bucket/object", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.RawQuery != "" || r.Header.Get("X-Newest") != "" {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("X-Request-Id", "swift-chunks")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("abcde"))
	})
	options := []objectstorage.ObjectStreamOption{objectstorage.WithObjectStreamChunkSize(2), objectstorage.WithObjectStreamQueryString("ignored\nquery"), func(*objectstorage.ObjectStreamOpts) error { callbacks++; return nil }}
	stream := conn.StreamObjectChunks(context.Background(), "bucket", "object", options...)
	options[0] = objectstorage.WithObjectStreamChunkSize(4)
	if locates != 0 || requests != 0 || callbacks != 0 || stream.Receipt() != nil || stream.Err() != nil {
		t.Fatal(locates, requests, callbacks, stream.Receipt(), stream.Err())
	}
	var data []byte
	var chunks [][]byte
	for chunk, err := range stream.Chunks() {
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, chunk...)
		chunks = append(chunks, chunk)
	}
	if string(data) != "abcde" || len(chunks) != 3 || string(chunks[0]) != "ab" || string(chunks[1]) != "cd" || string(chunks[2]) != "e" || requests != 1 || locates != 1 || callbacks != 1 || stream.Err() != nil {
		t.Fatal(string(data), chunks, requests, locates, callbacks, stream.Err())
	}
	proof := stream.Receipt()
	if proof == nil || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "swift-chunks" {
		t.Fatal(proof)
	}
	proof.Header.Set("X-Request-Id", "forged")
	if stream.Receipt().Header.Get("X-Request-Id") != "swift-chunks" {
		t.Fatal(stream.Receipt())
	}
	for range stream.Chunks() {
		t.Fatal("stream is single-use")
	}
	service, err := conn.ObjectStorageV1(context.Background())
	if err != nil || service.Cloud == nil || locates != 1 {
		t.Fatal(service, err, locates)
	}
	next := conn.StreamObjectChunks(context.Background(), "bucket", "object", objectstorage.WithObjectStreamChunkSize(1))
	for chunk, err := range next.Chunks() {
		if err != nil || string(chunk) != "a" {
			t.Fatal(chunk, err)
		}
		break
	}
	if err := next.Close(); err != nil || next.Err() != nil || locates != 1 || requests != 2 {
		t.Fatal(err, next.Err(), locates, requests)
	}
}

func TestConnectionSwiftChunkStreamPreflightAndClosedHandleNeverDiscover(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return "", gophercloud.ErrEndpointNotFound{}
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	closed := conn.StreamObjectChunks(context.Background(), "bucket", "object")
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	for range closed.Chunks() {
		t.Fatal("closed before start must stay empty")
	}
	if locates != 0 || closed.Receipt() != nil {
		t.Fatal(locates, closed.Receipt())
	}
	for _, mode := range []string{"nil context", "nil connection", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			current := conn
			ctx := context.Background()
			want := resource.ErrInvalidOption
			cause := errors.New("stream caller cancellation")
			if mode == "nil context" {
				ctx = nil
			}
			if mode == "nil connection" {
				current = nil
			}
			if mode == "cancelled" {
				next, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = next
				want = context.Canceled
			}
			stream := current.StreamObjectChunks(ctx, "bucket", "object")
			seen := 0
			for data, err := range stream.Chunks() {
				seen++
				if data != nil || !errors.Is(err, want) {
					t.Fatal(data, err)
				}
				if mode == "cancelled" && !errors.Is(err, cause) {
					t.Fatal(err)
				}
			}
			var proof *resource.ResponseError
			if seen != 1 || !errors.Is(stream.Err(), want) || stream.Receipt() != nil || errors.As(stream.Err(), &proof) || locates != 0 {
				t.Fatal(seen, stream.Err(), stream.Receipt(), proof, locates)
			}
		})
	}
	stream := conn.StreamObjectChunks(context.Background(), "bucket", "object")
	seen := 0
	for _, err := range stream.Chunks() {
		seen++
		var absent gophercloud.ErrEndpointNotFound
		if !errors.As(err, &absent) {
			t.Fatal(err)
		}
	}
	if seen != 1 || locates != 1 || stream.Receipt() != nil {
		t.Fatal(seen, locates, stream.Receipt())
	}
}

func TestConnectionSwiftChunkStreamMissingReturnsActualReceipt(t *testing.T) {
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.ObjectStorage, cloud.Server.URL+"/swift/v1/AUTH_p/"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /swift/v1/AUTH_p/bucket/missing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "actual-missing")
		w.WriteHeader(404)
		_, _ = w.Write([]byte("missing"))
	})
	stream := conn.StreamObjectChunks(context.Background(), "bucket", "missing")
	for data, err := range stream.Chunks() {
		t.Fatal(data, err)
	}
	proof := stream.Receipt()
	if stream.Err() != nil || proof == nil || proof.StatusCode != 404 || proof.Header.Get("X-Request-Id") != "actual-missing" {
		t.Fatal(stream.Err(), proof)
	}
}

func TestConnectionSwiftChunkStreamCleanupErrorsKeepPrivateProof(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("swift-token")
	closeCause := errors.New("physical stream close")
	body := &connectionObjectReadBody{Reader: strings.NewReader("abcd"), closeErr: closeCause}
	requests := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{Request: r, StatusCode: 200, Header: http.Header{"X-Proof": {"actual-cleanup"}}, Body: body, ContentLength: 4}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpoint(sdk.ObjectStorage, "https://cloud.test/swift/v1/AUTH_p/"))
	if err != nil {
		t.Fatal(err)
	}
	stream := conn.StreamObjectChunks(context.Background(), "bucket", "object", objectstorage.WithObjectStreamChunkSize(2))
	for data, err := range stream.Chunks() {
		if err != nil || string(data) != "ab" {
			t.Fatal(data, err)
		}
		break
	}
	err = stream.Err()
	var proof *resource.ResponseError
	if !errors.Is(err, closeCause) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Proof") != "actual-cleanup" || body.closes != 1 || requests != 1 {
		t.Fatal(err, proof, body.closes, requests)
	}
	proof.Header.Set("X-Proof", "forged")
	proof.Body = []byte("forged")
	var fresh *resource.ResponseError
	if !errors.As(stream.Err(), &fresh) || fresh.Header.Get("X-Proof") != "actual-cleanup" || string(fresh.Body) == "forged" || stream.Receipt().Header.Get("X-Proof") != "actual-cleanup" {
		t.Fatal(stream.Err(), fresh, stream.Receipt())
	}
	if !errors.Is(stream.Close(), closeCause) || body.closes != 1 {
		t.Fatal(stream.Close(), body.closes)
	}
}
