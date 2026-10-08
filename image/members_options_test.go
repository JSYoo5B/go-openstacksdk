package image

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageMembersOptionsSnapshotsReplacementAndLastWins(t *testing.T) {
	input := map[string]string{"X-A": "snapshot"}
	full, merge := WithImageMemberOpts(ImageMemberOpts{Headers: input}), WithImageMemberHeaders(input)
	input["X-A"] = "changed"
	for _, option := range []ImageMemberOption{full, merge} {
		value, err := parseImageMemberOptions([]ImageMemberOption{option, WithImageMemberHeader("x-a", "last")})
		if err != nil || value.Headers["X-A"] != "last" {
			t.Fatal(value, err)
		}
		value, err = parseImageMemberOptions([]ImageMemberOption{option})
		if err != nil || value.Headers["X-A"] != "snapshot" {
			t.Fatal(value, err)
		}
	}
	value, err := parseImageMemberOptions([]ImageMemberOption{WithImageMemberHeader("X-Discard", "old"), WithImageMemberOpts(ImageMemberOpts{})})
	if err != nil || value.Headers == nil || len(value.Headers) != 0 {
		t.Fatal(value, err)
	}
	ignore := false
	remove := WithRemoveImageMemberOpts(RemoveImageMemberOpts{Headers: map[string]string{"X-A": "old"}, IgnoreMissing: &ignore})
	find := WithFindImageMemberOpts(FindImageMemberOpts{Headers: map[string]string{"X-A": "old"}, IgnoreMissing: &ignore})
	ignore = true
	r, err := parseRemoveImageMemberOptions([]RemoveImageMemberOption{remove, WithRemoveImageMemberHeader("x-a", "last"), WithRemoveImageMemberHeaders(map[string]string{"X-B": "merged"})})
	if err != nil || *r.IgnoreMissing || r.Headers["X-A"] != "last" || r.Headers["X-B"] != "merged" {
		t.Fatal(r, err)
	}
	f, err := parseFindImageMemberOptions([]FindImageMemberOption{find, WithFindImageMemberHeader("x-a", "last"), WithFindImageMemberHeaders(map[string]string{"X-B": "merged"})})
	if err != nil || *f.IgnoreMissing || f.Headers["X-A"] != "last" || f.Headers["X-B"] != "merged" {
		t.Fatal(f, err)
	}
	r, err = parseRemoveImageMemberOptions([]RemoveImageMemberOption{WithRemoveImageMemberIgnoreMissing(false), WithRemoveImageMemberOpts(RemoveImageMemberOpts{})})
	if err != nil || r.IgnoreMissing != nil || r.Headers == nil {
		t.Fatal(r, err)
	}
	f, err = parseFindImageMemberOptions([]FindImageMemberOption{WithFindImageMemberIgnoreMissing(false), WithFindImageMemberOpts(FindImageMemberOpts{})})
	if err != nil || f.IgnoreMissing != nil || f.Headers == nil {
		t.Fatal(f, err)
	}
	listFull := WithListImageMembersOpts(ListImageMembersOpts{Headers: map[string]string{"X-A": "old"}, MaxItems: 2})
	l, err := parseListImageMembersOptions([]ListImageMembersOption{listFull, WithListImageMembersHeader("x-a", "last"), WithListImageMembersHeaders(map[string]string{"X-B": "merged"}), WithListImageMembersMaxItems(3)})
	if err != nil || l.MaxItems != 3 || l.Headers["X-A"] != "last" || l.Headers["X-B"] != "merged" {
		t.Fatal(l, err)
	}
	l, err = parseListImageMembersOptions([]ListImageMembersOption{WithListImageMembersMaxItems(7), WithListImageMembersOpts(ListImageMembersOpts{})})
	if err != nil || l.MaxItems != 0 || l.Headers == nil {
		t.Fatal(l, err)
	}
}

func TestImageMembersOptionsCallbacksPointersAndSliceOwnership(t *testing.T) {
	var old *RemoveImageMemberOpts
	second := 0
	options := make([]RemoveImageMemberOption, 2)
	options[0] = func(config *RemoveImageMemberOpts) error {
		if config.Headers == nil {
			t.Fatal("headers not initialized")
		}
		old = config
		config.Headers["X-A"] = "owned"
		ignore := false
		config.IgnoreMissing = &ignore
		options[1] = func(*RemoveImageMemberOpts) error { t.Fatal("caller slice mutation executed"); return nil }
		return nil
	}
	options[1] = func(config *RemoveImageMemberOpts) error {
		second++
		old.Headers["X-A"] = "late"
		*old.IgnoreMissing = true
		if config.Headers["X-A"] != "owned" || *config.IgnoreMissing {
			t.Fatal(config)
		}
		old = config
		return nil
	}
	client := deleteCoreClient(func(r *http.Request) (*http.Response, error) {
		old.Headers["X-A"] = "wire"
		*old.IgnoreMissing = true
		if r.Header.Get("X-A") != "owned" {
			t.Fatal(r.Header)
		}
		return deleteCoreHTTP(204, io.NopCloser(strings.NewReader("")), nil), nil
	})
	ack, err := New(client).RemoveImageMember(context.Background(), resource.ID("image"), "member", options...)
	if ack == nil || err != nil || second != 1 {
		t.Fatal(ack, err, second)
	}
	var retained *FindImageMemberOpts
	f, err := parseFindImageMemberOptions([]FindImageMemberOption{func(config *FindImageMemberOpts) error {
		retained = config
		config.Headers["X-A"] = "owned"
		ignore := false
		config.IgnoreMissing = &ignore
		return nil
	}, func(config *FindImageMemberOpts) error {
		retained.Headers["X-A"] = "late"
		*retained.IgnoreMissing = true
		if config.Headers["X-A"] != "owned" || *config.IgnoreMissing {
			t.Fatal(config)
		}
		return nil
	}})
	if err != nil || *f.IgnoreMissing || f.Headers["X-A"] != "owned" {
		t.Fatal(f, err)
	}
	listOptions := []ListImageMembersOption{WithListImageMembersMaxItems(1)}
	client = deleteCoreClient(func(*http.Request) (*http.Response, error) {
		return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"members":[{}, {"member_id":7}]}`)), nil), nil
	})
	seq := New(client).ListImageMembers(context.Background(), resource.ID("image"), listOptions...)
	listOptions[0] = WithListImageMembersMaxItems(0)
	count := 0
	for row, failure := range seq {
		if row == nil || failure != nil {
			t.Fatal(row, failure)
		}
		count++
	}
	if count != 1 {
		t.Fatal(count)
	}
}

func TestImageMembersOptionsValidationAndDefaults(t *testing.T) {
	for _, headers := range []map[string]string{{"Authorization": "x"}, {"X-Auth-Token": "x"}, {"Host": "foreign.test"}, {"Cookie": "x"}, {"Content-Length": "3"}, {"Content-Type": "x"}, {"Accept": "x"}, {"OpenStack-API-Version": "image 2.1"}, {"X-A": "one", "x-a": "two"}, {"Bad Key": "x"}, {"X-A": "bad\nvalue"}} {
		if _, err := parseImageMemberOptions([]ImageMemberOption{WithImageMemberHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
		if _, err := parseRemoveImageMemberOptions([]RemoveImageMemberOption{WithRemoveImageMemberHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
		if _, err := parseFindImageMemberOptions([]FindImageMemberOption{WithFindImageMemberHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
		if _, err := parseListImageMembersOptions([]ListImageMembersOption{WithListImageMembersHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
	}
	if _, err := parseImageMemberOptions([]ImageMemberOption{nil}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := parseRemoveImageMemberOptions([]RemoveImageMemberOption{nil}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := parseFindImageMemberOptions([]FindImageMemberOption{nil}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := parseListImageMembersOptions([]ListImageMembersOption{nil}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := parseListImageMembersOptions([]ListImageMembersOption{WithListImageMembersMaxItems(-1)}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cause := errors.New("callback cause")
	client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("HTTP after option failure"); return nil, nil })
	if value, err := New(client).GetImageMember(context.Background(), resource.Name("parent"), "member", func(*ImageMemberOpts) error { return cause }); value != nil || !errors.Is(err, cause) {
		t.Fatal(value, err)
	}
	if value, err := New(client).FindImageMember(context.Background(), resource.Name("parent"), "member", func(*FindImageMemberOpts) error { return cause }); value != nil || !errors.Is(err, cause) {
		t.Fatal(value, err)
	}
}

func TestImageMembersOptionsParallelReusableHelpers(t *testing.T) {
	option := WithImageMemberHeaders(map[string]string{"X-A": "snapshot"})
	remove := WithRemoveImageMemberIgnoreMissing(false)
	find := WithFindImageMemberIgnoreMissing(false)
	list := WithListImageMembersOpts(ListImageMembersOpts{Headers: map[string]string{"X-A": "snapshot"}, MaxItems: 2})
	var workers sync.WaitGroup
	for i := 0; i < 12; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			v, err := parseImageMemberOptions([]ImageMemberOption{option})
			if err != nil || v.Headers["X-A"] != "snapshot" {
				t.Error(v, err)
			}
			v.Headers["X-A"] = "local"
			r, err := parseRemoveImageMemberOptions([]RemoveImageMemberOption{remove})
			if err != nil || *r.IgnoreMissing {
				t.Error(r, err)
			}
			*r.IgnoreMissing = true
			f, err := parseFindImageMemberOptions([]FindImageMemberOption{find})
			if err != nil || *f.IgnoreMissing {
				t.Error(f, err)
			}
			*f.IgnoreMissing = true
			l, err := parseListImageMembersOptions([]ListImageMembersOption{list})
			if err != nil || l.Headers["X-A"] != "snapshot" || l.MaxItems != 2 {
				t.Error(l, err)
			}
			l.Headers["X-A"] = "local"
		}()
	}
	workers.Wait()
}
