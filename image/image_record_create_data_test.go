package image

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageRecordCreateDataFactoriesPreserveNoneEmptyAndTruthiness(t *testing.T) {
	for _, test := range []struct {
		name            string
		data            ImageRecordCreateData
		present, truthy bool
		body            string
	}{
		{"zero None", ImageRecordCreateData{}, false, false, ""}, {"nil Reader None", ImageRecordCreateReader(nil), false, false, ""},
		{"nil bytes present", ImageRecordCreateBytes(nil), true, false, ""}, {"empty bytes present", ImageRecordCreateBytes([]byte{}), true, false, ""},
		{"binary bytes", ImageRecordCreateBytes([]byte{255, 0}), true, true, string([]byte{255, 0})}, {"empty text present", ImageRecordCreateText(""), true, false, ""},
		{"Unicode text", ImageRecordCreateText("한글"), true, true, "한글"}, {"empty Reader truthy", ImageRecordCreateReader(strings.NewReader("")), true, true, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := captureImageRecordCreateData(context.Background(), nil, test.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.present() != test.present || got.truthy() != test.truthy {
				t.Fatal(got, test)
			}
			reader := got.openReader()
			if !test.present {
				if reader != nil {
					t.Fatal(reader)
				}
				return
			}
			if reader == nil {
				t.Fatal("present empty value lost")
			}
			body, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(body) != test.body {
				t.Fatalf("string(body) = %v, want %v", string(body), test.body)
			}
		})
	}
}

func TestImageRecordCreateDataOwnsBytesAndBorrowsReaderWithoutCaptureIO(t *testing.T) {
	original := []byte("before")
	factory := ImageRecordCreateBytes(original)
	original[0] = '!'
	captured, err := captureImageRecordCreateData(context.Background(), nil, factory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	captured.data[1] = '!'
	again, err := captureImageRecordCreateData(context.Background(), nil, factory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err := io.ReadAll(again.openReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "before" {
		t.Fatalf("string(body) = %v, want %v", string(body), "before")
	}
	body, err = io.ReadAll(factory.openReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "before" {
		t.Fatalf("string(body) = %v, want %v", string(body), "before")
	}
	reader := &imageUploadCoreReader{reader: strings.NewReader("skip-body")}
	_, err = reader.reader.(*strings.Reader).Seek(5, io.SeekStart)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	borrowed := ImageRecordCreateReader(reader)
	got, err := captureImageRecordCreateData(context.Background(), nil, borrowed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.openReader() != reader || got.reader != reader || reader.reads.Load() != 0 || reader.seeks.Load() != 0 || reader.closes.Load() != 0 {
		t.Fatal(got, reader)
	}
	body, err = io.ReadAll(got.openReader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "body" {
		t.Fatalf("string(body) = %v, want %v", string(body), "body")
	}
	if reader.closes.Load() != 0 || reader.seeks.Load() != 0 {
		t.Fatal("borrowed lifetime changed", reader)
	}
}

func TestImageRecordCreateDataCaptureRejectsTypedNilTextAndCanceledSource(t *testing.T) {
	var typedNil *strings.Reader
	for _, mode := range []string{"typed nil Reader", "invalid UTF8 text", "canceled context", "changed source"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			marker := errors.New("create data capture")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			client := taskCoreClient(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("foundation dispatched")
				return nil, nil
			})
			p, err := New(client).captureImageRecord(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			value := ImageRecordCreateBytes([]byte("data"))
			switch mode {
			case "typed nil Reader":
				value = ImageRecordCreateReader(typedNil)
			case "invalid UTF8 text":
				value = ImageRecordCreateText(string([]byte{255}))
			case "canceled context":
				cancel(marker)
			case "changed source":
				client.Endpoint = "https://foreign.test/"
			}
			_, err = captureImageRecordCreateData(p.ctx, p.check, value)
			expected := resource.ErrInvalidOption
			if mode == "canceled context" {
				expected = marker
			}
			if !errors.Is(err, expected) || calls != 0 {
				t.Fatal(err, calls)
			}
		})
	}
}
