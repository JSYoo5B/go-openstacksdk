package image

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestDownloadOptionsDefaultsAndBoundedPreflight(t *testing.T) {
	defaults, err := parseDownloadImageOptions(nil)
	if err != nil || *defaults.ChunkSize != 1024*1024 || !*defaults.VerifyChecksum || defaults.StorePreferences != nil {
		t.Fatalf("defaults=%+v err=%v", defaults, err)
	}
	for _, size := range []int{-1, 0, 64*1024*1024 + 1} {
		_, err := parseDownloadImageOptions([]DownloadImageOption{WithDownloadChunkSize(size)})
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("size=%d err=%v", size, err)
		}
	}
	for _, store := range []string{"", " ", "fast,slow", "bad\nstore", "bad\x7fstore", "bad\u0085store", string([]byte{255})} {
		_, err := parseDownloadImageOptions([]DownloadImageOption{WithDownloadStorePreferences(store)})
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("store=%q err=%v", store, err)
		}
	}
	for _, size := range []int{1, 64 * 1024 * 1024} {
		_, err := parseDownloadImageOptions([]DownloadImageOption{WithDownloadChunkSize(size), WithDownloadStorePreferences("a &/?=", "a &/?="), WithDownloadChecksumVerification(false)})
		if err != nil {
			t.Errorf("valid boundary size=%d err=%v", size, err)
		}
	}
	_, err = parseDownloadImageOptions([]DownloadImageOption{nil})
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestDownloadOptionsOwnFullReplacementAndConcurrentReuse(t *testing.T) {
	size, verify := 3, false
	stores := []string{"fast", "fast", "slow"}
	option := WithDownloadImageOpts(DownloadImageOpts{ChunkSize: &size, VerifyChecksum: &verify, StorePreferences: stores})
	size, verify, stores[0] = -1, true, "changed"
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			prepared, err := parseDownloadImageOptions([]DownloadImageOption{WithDownloadChunkSize(0), option})
			if err != nil || *prepared.ChunkSize != 3 || *prepared.VerifyChecksum || !reflect.DeepEqual(prepared.StorePreferences, []string{"fast", "fast", "slow"}) {
				t.Errorf("prepared=%+v err=%v", prepared, err)
				return
			}
			*prepared.ChunkSize, *prepared.VerifyChecksum, prepared.StorePreferences[0] = 7, true, "owned mutation"
		})
	}
	group.Wait()
	replaced, err := parseDownloadImageOptions([]DownloadImageOption{option, WithDownloadImageOpts(DownloadImageOpts{})})
	if err != nil || *replaced.ChunkSize != 1024*1024 || !*replaced.VerifyChecksum || replaced.StorePreferences != nil {
		t.Fatalf("replacement=%+v err=%v", replaced, err)
	}
	empty, err := parseDownloadImageOptions([]DownloadImageOption{WithDownloadStorePreferences()})
	if err != nil || empty.StorePreferences == nil || len(empty.StorePreferences) != 0 {
		t.Fatalf("explicit empty=%+v err=%v", empty, err)
	}
}

func TestDownloadOptionsCustomCallbacksOnceAndOwnedPointers(t *testing.T) {
	var retained *DownloadImageOpts
	var calls int
	option := func(value *DownloadImageOpts) error {
		calls++
		retained = value
		size, verify := 3, false
		value.ChunkSize, value.VerifyChecksum, value.StorePreferences = &size, &verify, []string{"fast"}
		return nil
	}
	prepared, err := parseDownloadImageOptions([]DownloadImageOption{option})
	if err != nil {
		t.Fatal(err)
	}
	*retained.ChunkSize, *retained.VerifyChecksum, retained.StorePreferences[0] = 0, true, "changed"
	if calls != 1 || *prepared.ChunkSize != 3 || *prepared.VerifyChecksum || prepared.StorePreferences[0] != "fast" {
		t.Fatalf("prepared=%+v callbacks=%d", prepared, calls)
	}
	cause := errors.New("caller option failed")
	_, err = parseDownloadImageOptions([]DownloadImageOption{func(*DownloadImageOpts) error { return cause }})
	if !errors.Is(err, cause) {
		t.Fatal("caller cause lost", err)
	}
}
