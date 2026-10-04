package blockstorage_test

import (
	"context"
	"errors"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSearchVolumesContinuationAuthorityCyclesAndEmptySourceEOF(t *testing.T) {
	for _, tc := range []struct {
		name, next   string
		cycle, empty bool
	}{
		{"other authority", "https://foreign.example/search/cinder/v3/p/volumes/detail", false, false}, {"different collection", "/search/cinder/v3/p/volumes", false, false}, {"fragment", "?marker=2#leak", false, false}, {"userinfo", "https://user:password@foreign.example/path", false, false}, {"cycle", "?a=1&b=2", true, false}, {"empty EOF ignores next", "https://foreign.example/path", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				rows := `[{"id":"id"}]`
				if tc.empty {
					rows = "[]"
				}
				next := tc.next
				if tc.cycle && n == 2 {
					next = "?b=2&a=1"
				}
				w.Header().Set("X-Proof", "origin")
				testcloud.JSON(w, 200, volumeSearchTestPage(rows, next))
			})
			result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{})
			want := int32(1)
			if tc.cycle {
				want = 2
			}
			if result == nil || calls.Load() != want || len(result.Pages) != int(want) {
				t.Fatal(result, err, calls.Load())
			}
			if tc.empty {
				if err != nil || string(result.Value) != "[]" || result.Volumes == nil || len(result.Volumes) != 0 {
					t.Fatal(result, err)
				}
				return
			}
			var response *resource.ResponseError
			if err == nil || !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Proof") != "origin" || result.Value != nil || result.Volumes != nil {
				t.Fatal(result, err)
			}
			if tc.cycle {
				var cycle *resource.PaginationCycleError
				if !errors.As(err, &cycle) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestSearchVolumesAcceptedReadCloseEnvelopeAndUTF8ErrorsKeepPhysicalProof(t *testing.T) {
	for _, kind := range []string{"read", "close", "missing envelope", "null array", "malformed JSON", "invalid UTF8"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			sentinel := errors.New("accepted body failure")
			body := `{"volumes":[{"id":"id"}]}`
			reader := &getVolumesContractReader{data: strings.NewReader(body)}
			switch kind {
			case "read":
				reader.readError = sentinel
			case "close":
				reader.closeError = sentinel
			case "missing envelope":
				body = `{"Volumes":[]}`
			case "null array":
				body = `{"volumes":null}`
			case "malformed JSON":
				body = `{"volumes":[`
			case "invalid UTF8":
				body = string([]byte{'{', '"', 'v', 'o', 'l', 'u', 'm', 'e', 's', '"', ':', '[', '{', '"', 'i', 'd', '"', ':', '"', 0xff, '"', '}', ']', '}'})
			}
			reader.data = strings.NewReader(body)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return getVolumesContractResponse(r, 200, reader, "physical"), nil
			})
			result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{})
			var response *resource.ResponseError
			if result == nil || calls.Load() != 1 || reader.closes.Load() != 1 || len(result.Pages) != 1 || result.Value != nil || result.Volumes != nil || !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Proof") != "physical" || string(result.Pages[0].Body) != body {
				t.Fatal(result, err, calls.Load(), reader.closes.Load())
			}
			if (kind == "read" || kind == "close") && !errors.Is(err, sentinel) {
				t.Fatal(err)
			}
		})
	}
}

func TestSearchVolumesLaterNativeAndTransportErrorsNeverBorrowAcceptedPage(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(map[bool]string{false: "transport", true: "HTTP rejection"}[reject], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			sentinel := errors.New("later transport failed")
			first := volumeSearchTestPage(`[{"id":"id"}]`, `?marker=2`)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(first)), "accepted"), nil
				}
				if !reject {
					return nil, sentinel
				}
				return getVolumesContractResponse(r, 403, io.NopCloser(strings.NewReader(`{"rejected":"actual"}`)), "rejected"), nil
			})
			result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{})
			var response *resource.ResponseError
			if result == nil || calls.Load() != 2 || len(result.Pages) != 1 || result.Pages[0].Header.Get("X-Proof") != "accepted" || result.Value != nil || result.Volumes != nil || err == nil || errors.As(err, &response) {
				t.Fatal(result, err, calls.Load())
			}
			if reject {
				var native gophercloud.ErrUnexpectedResponseCode
				if !gophercloud.ResponseCodeIs(err, 403) || !errors.As(err, &native) || string(native.Body) != `{"rejected":"actual"}` || native.ResponseHeader.Get("X-Proof") != "rejected" {
					t.Fatal(err)
				}
			} else if !errors.Is(err, sentinel) {
				t.Fatal(err)
			}
		})
	}
}

func TestSearchVolumesBlockedContinuationCancellationKeepsPriorPageAndCause(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeSearchTestClient(cloud)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("cancel blocked volume search")
	entered := make(chan struct{})
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(volumeSearchTestPage(`[{"id":"id"}]`, `?marker=2`))), "prior"), nil
		}
		close(entered)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	type outcome struct {
		result *blockstorage.SearchVolumesResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := blockstorage.SearchVolumes(ctx, client, blockstorage.SearchVolumesRequest{})
		done <- outcome{result, err}
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("second physical request never entered")
	}
	cancel(cause)
	select {
	case got := <-done:
		var response *resource.ResponseError
		if got.result == nil || len(got.result.Pages) != 1 || got.result.Pages[0].Header.Get("X-Proof") != "prior" || got.result.Value != nil || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || errors.As(got.err, &response) || calls.Load() != 2 {
			t.Fatal(got.result, got.err, calls.Load())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancellation did not release search")
	}
}

func TestSearchVolumesPostResponseSourceMutationIsTerminalWithCurrentEvidence(t *testing.T) {
	for _, field := range []string{"provider", "endpoint", "base", "type", "microversion"} {
		t.Run(field, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var calls atomic.Int32
			reader := &getVolumesContractReader{data: strings.NewReader(volumeSearchTestPage(`[{"id":"id"}]`, `?marker=2`))}
			reader.onClose = func() {
				switch field {
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "endpoint":
					client.Endpoint = cloud.Server.URL + "/different/"
				case "base":
					client.ResourceBase = cloud.Server.URL + "/different/"
				case "type":
					client.Type = "volume"
				case "microversion":
					client.Microversion = "3.60"
				}
			}
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return getVolumesContractResponse(r, 200, reader, "current"), nil
			})
			result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{})
			var response *resource.ResponseError
			if result == nil || calls.Load() != 1 || len(result.Pages) != 1 || result.Value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &response) || response.Header.Get("X-Proof") != "current" {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestGetVolumeAcceptedMemberErrorsKeepObservedAndNeverReplayList(t *testing.T) {
	for _, kind := range []string{"envelope", "descriptor", "close", "source", "cancellation"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			sentinel := errors.New("member phase failed")
			body := `{"volume":{"id":"id"}}`
			if kind == "envelope" {
				body = `{"Volume":{"id":"id"}}`
			}
			if kind == "descriptor" {
				body = `{"volume":{"id":"id","bootable":"bad"}}`
			}
			reader := &getVolumesContractReader{data: strings.NewReader(body)}
			if kind == "close" {
				reader.closeError = sentinel
			}
			if kind == "source" {
				reader.onClose = func() { client.Endpoint = cloud.Server.URL + "/changed/" }
			}
			if kind == "cancellation" {
				reader.onClose = func() { cancel(sentinel) }
			}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.Path != volumeSearchTestBase+"volumes/id" {
					t.Error("member error replayed list", r.URL)
				}
				return getVolumesContractResponse(r, 200, reader, "member"), nil
			})
			result, err := blockstorage.GetVolume(ctx, client, blockstorage.GetVolumeRequest{NameOrID: "id"})
			var response *resource.ResponseError
			if result == nil || calls.Load() != 1 || reader.closes.Load() != 1 || result.Observed == nil || result.Observed.StatusCode != 200 || result.Observed.Header.Get("X-Proof") != "member" || string(result.Observed.Body) != body || len(result.Pages) != 0 || result.Volume != nil || result.Value != nil || !errors.As(err, &response) || response.StatusCode != 200 {
				t.Fatal(result, err, calls.Load())
			}
			if (kind == "close" || kind == "cancellation") && !errors.Is(err, sentinel) {
				t.Fatal(err)
			}
			if kind == "cancellation" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
