package image

import (
	"errors"
	"sync"
	"testing"

	"gophercloudsdk/resource"
)

func TestDeleteOptionsDefaultsAndExplicitStorePreflight(t *testing.T) {
	defaults, err := parseDeleteImageOptions(nil)
	if err != nil || defaults.StoreID != nil || defaults.IgnoreMissing == nil || !*defaults.IgnoreMissing {
		t.Fatalf("defaults=%+v err=%v", defaults, err)
	}
	for _, store := range []string{"", " ", ".", "..", "a/b", "a\\b", "a?b", "a#b", "a%b", "a:b", "a\nb", "a\x00b", "a\u0085b", "a\u00a0b", string([]byte{255})} {
		_, err := parseDeleteImageOptions([]DeleteImageOption{WithDeleteImageStore(store)})
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("store=%q err=%v", store, err)
		}
	}
	for _, store := range []string{"fast", "store&=fast", "저장소"} {
		value, err := parseDeleteImageOptions([]DeleteImageOption{WithDeleteImageStore(store), WithDeleteImageIgnoreMissing(false)})
		if err != nil || value.StoreID == nil || *value.StoreID != store || *value.IgnoreMissing {
			t.Errorf("store=%q value=%+v err=%v", store, value, err)
		}
	}
	_, err = parseDeleteImageOptions([]DeleteImageOption{nil})
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cause := errors.New("caller delete policy failed")
	_, err = parseDeleteImageOptions([]DeleteImageOption{func(*DeleteImageOpts) error { return cause }})
	if !errors.Is(err, cause) {
		t.Fatal("caller cause lost", err)
	}
}

func TestDeleteOptionsFullReplacementAndConcurrentReuse(t *testing.T) {
	store, ignore := "fast", false
	option := WithDeleteImageOpts(DeleteImageOpts{StoreID: &store, IgnoreMissing: &ignore})
	store, ignore = "changed", true
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			value, err := parseDeleteImageOptions([]DeleteImageOption{WithDeleteImageStore(""), option})
			if err != nil || value.StoreID == nil || *value.StoreID != "fast" || *value.IgnoreMissing {
				t.Errorf("value=%+v err=%v", value, err)
				return
			}
			*value.StoreID, *value.IgnoreMissing = "local mutation", true
		})
	}
	group.Wait()
	defaults, err := parseDeleteImageOptions([]DeleteImageOption{option, WithDeleteImageOpts(DeleteImageOpts{})})
	if err != nil || defaults.StoreID != nil || !*defaults.IgnoreMissing {
		t.Fatalf("replacement=%+v err=%v", defaults, err)
	}
	value, err := parseDeleteImageOptions([]DeleteImageOption{option, WithDeleteImageStore("last"), WithDeleteImageIgnoreMissing(true)})
	if err != nil || *value.StoreID != "last" || !*value.IgnoreMissing {
		t.Fatalf("last option=%+v err=%v", value, err)
	}
}

func TestDeleteOptionsCallbacksOnceAndIndependentRetainedHandles(t *testing.T) {
	var retained []*DeleteImageOpts
	var calls int
	first := func(value *DeleteImageOpts) error {
		calls++
		store, ignore := "fast", false
		value.StoreID, value.IgnoreMissing = &store, &ignore
		retained = append(retained, value)
		return nil
	}
	second := func(value *DeleteImageOpts) error {
		calls++
		*retained[0].StoreID, *retained[0].IgnoreMissing = "retained mutation", true
		if *value.StoreID != "fast" || *value.IgnoreMissing {
			t.Fatal("earlier retained handle changed the next callback")
		}
		retained = append(retained, value)
		return nil
	}
	value, err := parseDeleteImageOptions([]DeleteImageOption{first, second})
	if err != nil {
		t.Fatal(err)
	}
	for _, config := range retained {
		*config.StoreID, *config.IgnoreMissing = "after preflight", true
	}
	if calls != 2 || *value.StoreID != "fast" || *value.IgnoreMissing || retained[0] == retained[1] {
		t.Fatalf("value=%+v calls=%d", value, calls)
	}
}
