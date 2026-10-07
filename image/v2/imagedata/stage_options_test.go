package imagedata

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestStageOptionsOwnedSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	size := int64(0)
	value := StageOpts{Size: &size, Headers: map[string]string{"x-note": "before"}}
	option := WithStageOpts(value)
	size, value.Headers["x-note"] = 99, "after"
	var captured *StageOpts
	parsed, err := parseStageOpts([]StageOption{option, func(config *StageOpts) error { captured = config; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	*captured.Size, captured.Headers["x-note"] = 42, "captured"
	if parsed.Size == nil || *parsed.Size != 0 || parsed.Headers["X-Note"] != "before" {
		t.Fatalf("snapshot lost: %#v", parsed)
	}
	replaced, err := parseStageOpts([]StageOption{option, WithStageOpts(StageOpts{}), WithStageSize(-1), WithStageSize(7)})
	if err != nil || replaced.Size == nil || *replaced.Size != 7 || len(replaced.Headers) != 0 {
		t.Fatalf("replacement/last wins: %#v %v", replaced, err)
	}
	var workers sync.WaitGroup
	failures := make(chan error, 10)
	for range 10 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			config, err := parseStageOpts([]StageOption{option})
			if err != nil || config.Size == nil || *config.Size != 0 || config.Headers["X-Note"] != "before" {
				failures <- fmt.Errorf("reuse: %#v %v", config, err)
				return
			}
			*config.Size, config.Headers["X-Note"] = 123, "private"
		}()
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func TestStageOptionsHeaderAuthorityAndCanonicalLastWins(t *testing.T) {
	input := map[string]string{"x-note": "first", "X-NOTE": "first"}
	option := WithStageHeaders(input)
	input["x-note"], input["X-NOTE"] = "after", "after"
	parsed, err := parseStageOpts([]StageOption{option, WithStageHeader("x-NOTE", "last")})
	if err != nil || len(parsed.Headers) != 1 || parsed.Headers["X-Note"] != "last" {
		t.Fatalf("header ownership: %#v %v", parsed, err)
	}
	for _, key := range []string{"Accept", "Content-Type", "X-OpenStack-Image-Size", "Authorization", "X-Auth-Token", "X-Service-Token", "Cookie", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "OpenStack-API-Version", "X-OpenStack-Glance-API-Version"} {
		if _, err := parseStageOpts([]StageOption{WithStageHeader(key, "blocked")}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("header %s: %v", key, err)
		}
	}
	for _, headers := range []map[string]string{{"X-Note": "one", "x-note": "two"}, {"invalid key": "one"}, {"X-Note": "one\nother"}, {"X-Note": string([]byte{255})}} {
		if _, err := parseStageOpts([]StageOption{WithStageHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("invalid headers %#v: %v", headers, err)
		}
	}
	if _, err := stageHeaders(map[string]string{"accept": "metadata", "content-type": "metadata", "openstack-api-version": "image 2.8"}, true, "2.8"); err != nil {
		t.Fatal(err)
	}
	if _, err := stageHeaders(map[string]string{"x-openstack-image-size": "7"}, true, ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := stageHeaders(map[string]string{"OpenStack-API-Version": "image 2.9"}, true, "2.8"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestStageOptionsSizeNilZeroAndInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		options []StageOption
		want    *int64
	}{
		{nil, nil}, {[]StageOption{WithStageSize(0)}, stageTestSize(0)}, {[]StageOption{WithStageSize(12)}, stageTestSize(12)},
		{[]StageOption{WithStageSize(12), WithStageOpts(StageOpts{})}, nil},
	} {
		parsed, err := parseStageOpts(test.options)
		if err != nil || (parsed.Size == nil) != (test.want == nil) || parsed.Size != nil && *parsed.Size != *test.want {
			t.Fatalf("size %#v want%v err%v", parsed.Size, test.want, err)
		}
	}
	for _, options := range [][]StageOption{{nil}, {WithStageSize(-1)}, {WithStageOpts(StageOpts{Size: stageTestSize(-1)})}} {
		if _, err := parseStageOpts(options); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid configuration: %v", err)
		}
	}
}

func stageTestSize(value int64) *int64 { return &value }
