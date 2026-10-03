package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/image"
	"gophercloudsdk/resource"
)

type imageUploadConnectionReader struct {
	reader                        *strings.Reader
	reads, closes, seeks, lengths int
}

func (r *imageUploadConnectionReader) Read(p []byte) (int, error) { r.reads++; return r.reader.Read(p) }
func (r *imageUploadConnectionReader) Close() error               { r.closes++; return nil }
func (r *imageUploadConnectionReader) Seek(n int64, w int) (int64, error) {
	r.seeks++
	return r.reader.Seek(n, w)
}
func (r *imageUploadConnectionReader) Len() int { r.lengths++; return r.reader.Len() }

func TestConnectionImageUploadFacadeOwnsPhases(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("upload-0")
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "이미지% ?#"
	const created = `{"id":"이미지% ?#","name":"actual-name","status":"queued","size":9007199254740995,"protected":false,"tags":[],"x-number":1e1000}`
	calls, hooks := 0, 0
	var source *gophercloud.ServiceClient
	provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		hooks++
		return errors.New("binary must not invoke shared retry")
	}
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		method, target := http.MethodPost, endpoint+"images"
		if n == 1 {
			method, target = http.MethodPut, endpoint+"images/"+url.PathEscape(id)+"/file"
		}
		if n == 4 {
			method, target = http.MethodPut, endpoint+"images/later/file"
		}
		expectedSource := fmt.Sprintf("source-%d", n)
		if n == 1 {
			expectedSource = "source-0"
		}
		if n == 4 {
			expectedSource = "source-3"
		}
		if r.Method != method || r.URL.Scheme+"://"+r.URL.Host+r.URL.EscapedPath() != target || r.URL.RawQuery != "" || r.Header.Get("X-Source") != expectedSource || r.Header.Get("X-Auth-Token") != fmt.Sprintf("upload-%d", n) {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		raw, e := io.ReadAll(r.Body)
		if e != nil {
			t.Fatal(e)
		}
		if method == http.MethodPost {
			if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" || r.Header.Get("X-OpenStack-Image-Size") != "" {
				t.Fatal(r.Header)
			}
			var fields map[string]json.RawMessage
			if e = json.Unmarshal(raw, &fields); e != nil || string(fields["name"]) != `"requested-name"` || string(fields["disk_format"]) != `"qcow2"` || string(fields["container_format"]) != `"bare"` || string(fields["visibility"]) != `"private"` {
				t.Fatal(string(raw), e)
			}
			if n == 0 && (string(fields["protected"]) != "false" || string(fields["min_disk"]) != "0" || string(fields["tags"]) != "[]" || string(fields["null"]) != "null" || string(fields["exact"]) != "9007199254740995" || r.Header.Get("X-Call") != "owned") {
				t.Fatal(string(raw), r.Header)
			}
		} else {
			if r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Accept") != "" || string(raw) != "binary-data" {
				t.Fatal(string(raw), r.Header)
			}
			if n == 1 && r.Header.Get("X-OpenStack-Image-Size") != "0" || n == 4 && r.Header.Get("X-OpenStack-Image-Size") != "" {
				t.Fatal(r.Header)
			}
		}
		code, body := 201, created
		switch n {
		case 1:
			code, body = 204, "opaque acknowledgement"
		case 2:
			body = `{"name":"missing-id"}`
		case 3:
			body = `{"id":"later","status":"queued"}`
		case 4:
			code, body = 503, "binary unavailable"
		case 5:
			body = `{"size":1.5}`
		}
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		provider.SetToken(fmt.Sprintf("upload-%d", calls))
		return &http.Response{Request: r, StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {fmt.Sprint(n)}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, e := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint))
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	upper, e := conn.Image(ctx)
	if e != nil {
		t.Fatal(e)
	}
	native, e := conn.ImageV2(ctx)
	if e != nil {
		t.Fatal(e)
	}
	source = upper.RawClient()
	if native.RawClient() != source || upper.API.Images.RawClient() != source || source.ProviderClient != provider {
		t.Fatal("upload facade replaced shared client")
	}
	source.MoreHeaders = map[string]string{"X-Source": "source-0", "Content-Type": "source/type", "Accept": "source/accept"}
	headers := map[string]string{"X-Call": "owned"}
	header := image.WithImageUploadHeaders(headers)
	headers["X-Call"] = "changed"
	fields := map[string]json.RawMessage{"null": json.RawMessage(`null`), "exact": json.RawMessage(`9007199254740995`)}
	field := image.WithImageUploadFields(fields)
	fields["exact"][0] = '1'
	reader := &imageUploadConnectionReader{reader: strings.NewReader("binary-data")}
	result, e := upper.UploadImage(ctx, image.UploadImageRequest{Name: "requested-name", Data: reader}, header, field, image.WithImageUploadSize(0), image.WithImageUploadProtected(false), image.WithImageUploadMinDisk(0), image.WithImageUploadTags())
	if e != nil || result == nil || result.Image == nil || result.ImageID != id || *result.Image.Name != "actual-name" || *result.Image.Size != 9007199254740995 || result.Image.StatusCode != 201 || result.Metadata.StatusCode != 201 || string(result.Metadata.Body) != created || result.Acknowledgement.StatusCode != 204 || string(result.Acknowledgement.Body) != "opaque acknowledgement" || calls != 2 {
		t.Fatal(result, e, calls)
	}
	result.Metadata.Header.Set("X-Proof", "late")
	result.Metadata.Body[0] = '!'
	result.Image.Body["x-number"][0] = '2'
	if result.Image.Header.Get("X-Proof") != "0" || result.Acknowledgement.Header.Get("X-Proof") != "1" || string(result.Image.Properties["x-number"]) != "1e1000" {
		t.Fatal("phase evidence aliases", result)
	}
	if reader.reads == 0 || reader.closes != 0 || reader.seeks != 0 || reader.lengths != 0 {
		t.Fatal(reader)
	}
	untouched := &imageUploadConnectionReader{reader: strings.NewReader("binary-data")}
	missing, e := upper.UploadImage(ctx, image.UploadImageRequest{Name: "requested-name", Data: untouched})
	var proof *resource.ResponseError
	if missing == nil || missing.Image == nil || missing.Metadata == nil || missing.Acknowledgement != nil || !errors.Is(e, resource.ErrInvalidOption) || !errors.As(e, &proof) || proof.StatusCode != 201 || string(proof.Body) != `{"name":"missing-id"}` || untouched.reads != 0 || calls != 3 {
		t.Fatal(missing, e, calls)
	}
	laterReader := &imageUploadConnectionReader{reader: strings.NewReader("binary-data")}
	later, e := upper.UploadImage(ctx, image.UploadImageRequest{Name: "requested-name", Data: laterReader})
	var failure gophercloud.ErrUnexpectedResponseCode
	if later == nil || later.ImageID != "later" || later.Metadata == nil || later.Image == nil || later.Acknowledgement != nil || !errors.As(e, &failure) || failure.Actual != 503 || string(failure.Body) != "binary unavailable" || hooks != 0 || calls != 5 || laterReader.closes != 0 || laterReader.seeks != 0 {
		t.Fatal(later, e, hooks, calls)
	}
	malformed, e := upper.UploadImage(ctx, image.UploadImageRequest{Name: "requested-name", Data: untouched})
	proof = nil
	if malformed == nil || malformed.Image != nil || malformed.Acknowledgement != nil || malformed.Metadata == nil || !errors.As(e, &proof) || proof.StatusCode != 201 || string(proof.Body) != `{"size":1.5}` || calls != 6 || untouched.reads != 0 {
		t.Fatal(malformed, e, calls)
	}
	if source.MoreHeaders["Content-Type"] != "source/type" || source.MoreHeaders["Accept"] != "source/accept" || source.ProviderClient != provider {
		t.Fatal("SDK changed source media/provider")
	}
}
