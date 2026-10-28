package testcloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestRecorderRecordsAndLinksRequests(t *testing.T) {
	cloud := New(t)
	rec := NewRecorder(func(r *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(r.Body)
		return Response(201, string(raw), http.Header{"X-Proof": {"kept"}}), nil
	})
	cloud.SetTransport(rec)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, cloud.Server.URL+"/a%2Fb?x=1", strings.NewReader(`{"k":1}`))
			req.Header.Set("X-Custom", "value")
			resp, err := cloud.Provider.HTTPClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			raw, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != 201 || string(raw) != `{"k":1}` || resp.Header.Get("X-Proof") != "kept" || resp.Request == nil {
				t.Error(resp.StatusCode, string(raw), resp.Header, resp.Request)
			}
		})
	}
	wg.Wait()
	calls := rec.Calls()
	if rec.Len() != 4 || len(calls) != 4 {
		t.Fatal(calls)
	}
	for _, call := range calls {
		if call.Method != http.MethodPost || call.Path != "/a/b" || call.EscapedPath != "/a%2Fb" || call.RawQuery != "x=1" || call.Body != `{"k":1}` || call.Header.Get("X-Custom") != "value" {
			t.Fatalf("%+v", call)
		}
	}
	rec.Reset()
	if rec.Len() != 0 || len(rec.Calls()) != 0 || len(calls) != 4 {
		t.Fatal("Reset must clear only the recorder", rec.Calls(), calls)
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, cloud.Server.URL+"/after", strings.NewReader("next"))
	resp, err := cloud.Provider.HTTPClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if rec.Len() != 1 || rec.Calls()[0].Path != "/after" || rec.Calls()[0].Body != "next" {
		t.Fatal(rec.Calls())
	}
}

func TestRoundTripperFuncIsExact(t *testing.T) {
	want := &http.Response{StatusCode: 204}
	got, err := RoundTripperFunc(func(*http.Request) (*http.Response, error) { return want, nil }).RoundTrip(&http.Request{})
	if err != nil || got != want || got.Request != nil {
		t.Fatal(got, err)
	}
	other := &http.Request{Method: "OTHER"}
	linked, _ := LinkRequest(func(*http.Request) (*http.Response, error) { return &http.Response{Request: other}, nil }).RoundTrip(&http.Request{})
	if linked.Request != other {
		t.Fatal("LinkRequest replaced an existing Response.Request")
	}
	if resp := Response(200, "", nil); resp.Header != nil || resp.Status != "" || resp.ContentLength != 0 {
		t.Fatalf("%+v", resp)
	}
}

func TestBodyFaults(t *testing.T) {
	cause, closeCause := errors.New("read"), errors.New("close")
	var onRead, onClose int
	body := &Body{Data: `{"partial":`, ReadErr: cause, CloseErr: closeCause, OnRead: func() { onRead++ }, OnClose: func() { onClose++ }}
	buf := make([]byte, 4)
	n, err := body.Read(buf)
	if n != 4 || err != nil || string(buf) != `{"pa` {
		t.Fatal(n, err)
	}
	raw, err := io.ReadAll(body)
	if string(raw) != `rtial":` || !errors.Is(err, cause) {
		t.Fatal(string(raw), err)
	}
	if n, err := body.Read(buf); n != 0 || !errors.Is(err, cause) {
		t.Fatal(n, err)
	}
	if err := body.Close(); !errors.Is(err, closeCause) {
		t.Fatal(err)
	}
	_ = body.Close()
	if onRead != 1 || onClose != 2 || body.Closes() != 2 || body.Reads() < 3 {
		t.Fatal(onRead, onClose, body.Closes(), body.Reads())
	}
	clean := &Body{}
	if n, err := clean.Read(buf); n != 0 || err != io.EOF || clean.Close() != nil {
		t.Fatal(n, err)
	}
}
