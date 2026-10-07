package blockstorage_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func vuiZero(value blockstorage.VolumeImageUploadOpts) bool {
	return value.Force == nil && value.DiskFormat == nil && value.ContainerFormat == nil && value.Visibility == nil && value.Protected == nil
}

func TestVolumeImageUploadOptionFactoriesOwnPointersDefaultsAndFullReplacement(t *testing.T) {
	force, protected := true, false
	disk, container, visibility := "", "container /?%#\n\x00", ""
	input := blockstorage.VolumeImageUploadOpts{Force: &force, DiskFormat: &disk, ContainerFormat: &container, Visibility: &visibility, Protected: &protected}
	factory := blockstorage.WithVolumeImageUploadOptions(input)
	force, protected, disk, container, visibility = false, true, "caller disk", "caller container", "caller visibility"
	first, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), factory)
	if err != nil || first.Force == nil || !*first.Force || first.DiskFormat == nil || *first.DiskFormat != "" || first.ContainerFormat == nil || *first.ContainerFormat != "container /?%#\n\x00" || first.Visibility == nil || *first.Visibility != "" || first.Protected == nil || *first.Protected {
		t.Fatal(first, err)
	}
	if first.Force == input.Force || first.DiskFormat == input.DiskFormat || first.ContainerFormat == input.ContainerFormat || first.Visibility == input.Visibility || first.Protected == input.Protected {
		t.Fatal("factory retained caller pointers", first, input)
	}
	*first.Force, *first.Protected = false, true
	*first.DiskFormat, *first.ContainerFormat, *first.Visibility = "returned disk", "returned container", "returned visibility"
	again, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), factory)
	if err != nil || !*again.Force || *again.Protected || *again.DiskFormat != "" || *again.ContainerFormat != "container /?%#\n\x00" || *again.Visibility != "" || again.Force == first.Force || again.Protected == first.Protected || again.DiskFormat == first.DiskFormat || again.ContainerFormat == first.ContainerFormat || again.Visibility == first.Visibility {
		t.Fatal("factory reuse borrowed returned pointers", first, again, err)
	}
	defaults, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t))
	if err != nil || defaults.Force == nil || *defaults.Force || defaults.DiskFormat != nil || defaults.ContainerFormat != nil || defaults.Visibility != nil || defaults.Protected != nil {
		t.Fatal("unknown format or protected defaults invented", defaults, err)
	}
	secondDefault, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t))
	if err != nil || secondDefault.Force == nil || secondDefault.Force == defaults.Force || *secondDefault.Force {
		t.Fatal("default pointer is shared", defaults, secondDefault, err)
	}
	replaced, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), factory, blockstorage.WithVolumeImageUploadOptions(blockstorage.VolumeImageUploadOpts{}))
	if err != nil || replaced.Force == nil || *replaced.Force || replaced.DiskFormat != nil || replaced.ContainerFormat != nil || replaced.Visibility != nil || replaced.Protected != nil {
		t.Fatal("full replacement retained omitted fields", replaced, err)
	}
	ordered, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t),
		blockstorage.WithVolumeImageUploadForce(true),
		blockstorage.WithVolumeImageUploadOptions(blockstorage.VolumeImageUploadOpts{}),
		blockstorage.WithVolumeImageUploadForce(false),
		blockstorage.WithVolumeImageUploadDiskFormat(""),
		blockstorage.WithVolumeImageUploadContainerFormat(""),
		blockstorage.WithVolumeImageUploadVisibility(""),
		blockstorage.WithVolumeImageUploadProtected(false))
	if err != nil || ordered.Force == nil || *ordered.Force || ordered.DiskFormat == nil || *ordered.DiskFormat != "" || ordered.ContainerFormat == nil || *ordered.ContainerFormat != "" || ordered.Visibility == nil || *ordered.Visibility != "" || ordered.Protected == nil || *ordered.Protected {
		t.Fatal("explicit empty or false presence lost", ordered, err)
	}
	factories := []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadForce(false), blockstorage.WithVolumeImageUploadDiskFormat(""), blockstorage.WithVolumeImageUploadContainerFormat(""), blockstorage.WithVolumeImageUploadVisibility(""), blockstorage.WithVolumeImageUploadProtected(false)}
	reused, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), factories...)
	if err != nil {
		t.Fatal(err)
	}
	*reused.Force, *reused.Protected = true, true
	*reused.DiskFormat, *reused.ContainerFormat, *reused.Visibility = "changed disk", "changed container", "changed visibility"
	independent, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), factories...)
	if err != nil || *independent.Force || *independent.Protected || *independent.DiskFormat != "" || *independent.ContainerFormat != "" || *independent.Visibility != "" || independent.Force == reused.Force || independent.Protected == reused.Protected || independent.DiskFormat == reused.DiskFormat || independent.ContainerFormat == reused.ContainerFormat || independent.Visibility == reused.Visibility {
		t.Fatal("individual factory reuse retained returned pointer mutation", reused, independent, err)
	}
}

func TestVolumeImageUploadOriginalsOwnCallbackSliceAndIntermediatePointers(t *testing.T) {
	force, protected := true, false
	disk, container, visibility := "disk", "container", "visibility"
	var first, second, last *blockstorage.VolumeImageUploadOpts
	var order []int
	var callbacks []blockstorage.VolumeImageUploadOption
	callbacks = []blockstorage.VolumeImageUploadOption{
		func(next *blockstorage.VolumeImageUploadOpts) error {
			order = append(order, 1)
			next.Force, next.DiskFormat = &force, &disk
			first = next
			callbacks[1] = func(*blockstorage.VolumeImageUploadOpts) error {
				t.Error("caller slice changed callback order")
				return errors.New("replacement callback")
			}
			return nil
		},
		func(next *blockstorage.VolumeImageUploadOpts) error {
			order = append(order, 2)
			if next.Force == first.Force || next.DiskFormat == first.DiskFormat {
				t.Fatal("callback shares prior pointers", next, first)
			}
			*first.Force, *first.DiskFormat = false, "old callback mutation"
			if !*next.Force || *next.DiskFormat != "disk" {
				t.Fatal("prior callback changed current value", next)
			}
			next.ContainerFormat, next.Visibility, next.Protected = &container, &visibility, &protected
			second = next
			return nil
		},
		func(next *blockstorage.VolumeImageUploadOpts) error {
			order = append(order, 3)
			last = next
			if next.Force == second.Force || next.DiskFormat == second.DiskFormat || next.ContainerFormat == second.ContainerFormat || next.Visibility == second.Visibility || next.Protected == second.Protected {
				t.Fatal("later callback borrows pointers", next, second)
			}
			*second.Force, *second.Protected = false, true
			*second.DiskFormat, *second.ContainerFormat, *second.Visibility = "bad disk", "bad container", "bad visibility"
			if !*next.Force || *next.Protected || *next.DiskFormat != "disk" || *next.ContainerFormat != "container" || *next.Visibility != "visibility" {
				t.Fatal("retained intermediate values leaked", next)
			}
			return nil
		},
	}
	value, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), callbacks...)
	if err != nil || !reflect.DeepEqual(order, []int{1, 2, 3}) || value.Force == nil || !*value.Force || value.Protected == nil || *value.Protected || *value.DiskFormat != "disk" || *value.ContainerFormat != "container" || *value.Visibility != "visibility" {
		t.Fatal(order, value, err)
	}
	if value.Force == last.Force || value.DiskFormat == last.DiskFormat || value.ContainerFormat == last.ContainerFormat || value.Visibility == last.Visibility || value.Protected == last.Protected {
		t.Fatal("returned options borrowed final callback pointers", value, last)
	}
	*last.Force, *last.Protected = false, true
	*last.DiskFormat, *last.ContainerFormat, *last.Visibility = "last disk", "last container", "last visibility"
	if !*value.Force || *value.Protected || *value.DiskFormat != "disk" || *value.ContainerFormat != "container" || *value.Visibility != "visibility" {
		t.Fatal("retained final callback changed prepared policy", value, last)
	}
	*value.Force, *value.Protected = true, false
	*value.DiskFormat, *value.ContainerFormat, *value.Visibility = "returned disk", "returned container", "returned visibility"
	if *last.Force || !*last.Protected || *last.DiskFormat != "last disk" || *last.ContainerFormat != "last container" || *last.Visibility != "last visibility" {
		t.Fatal("returned mutation reached final callback", last)
	}
}

func TestVolumeImageUploadPrepareValidatesOnlyFinalUTF8AndPreservesLiteralStrings(t *testing.T) {
	bad := string([]byte{0xff})
	fields := []struct {
		name   string
		option func(string) blockstorage.VolumeImageUploadOption
		get    func(blockstorage.VolumeImageUploadOpts) *string
	}{
		{"disk", blockstorage.WithVolumeImageUploadDiskFormat, func(v blockstorage.VolumeImageUploadOpts) *string { return v.DiskFormat }},
		{"container", blockstorage.WithVolumeImageUploadContainerFormat, func(v blockstorage.VolumeImageUploadOpts) *string { return v.ContainerFormat }},
		{"visibility", blockstorage.WithVolumeImageUploadVisibility, func(v blockstorage.VolumeImageUploadOpts) *string { return v.Visibility }},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			invalid, err := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), field.option(bad))
			if !errors.Is(err, resource.ErrInvalidOption) || !vuiZero(invalid) {
				t.Fatal("invalid UTF-8 was coerced or returned partial options", invalid, err)
			}
			for _, literal := range []string{"", "literal /?%#\n\x00 한글"} {
				repaired, cause := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), field.option(bad), field.option(literal))
				selected := field.get(repaired)
				if cause != nil || selected == nil || *selected != literal || repaired.Force == nil || *repaired.Force {
					t.Fatal("final repair or literal preservation failed", repaired, cause)
				}
			}
			cleared, cause := blockstorage.PrepareVolumeImageUploadOptions(vsaContext(t), field.option(bad), blockstorage.WithVolumeImageUploadOptions(blockstorage.VolumeImageUploadOpts{}))
			if cause != nil || field.get(cleared) != nil || cleared.Force == nil || *cleared.Force {
				t.Fatal("overwritten invalid input was validated", cleared, cause)
			}
		})
	}
}

func TestVolumeImageUploadPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause(t *testing.T) {
	for _, kind := range []string{"nil context", "already canceled", "nil option", "callback error", "cancel", "callback error and cancel"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(vsaContext(t))
			defer cancel(nil)
			var selected context.Context = ctx
			callbackCause, cancelCause := errors.New("upload option callback"), errors.New("upload custom cancel")
			first, later := 0, 0
			options := []blockstorage.VolumeImageUploadOption{
				func(next *blockstorage.VolumeImageUploadOpts) error {
					first++
					truth := true
					next.Force, next.Protected = &truth, &truth
					if kind == "cancel" || kind == "callback error and cancel" {
						cancel(cancelCause)
					}
					if kind == "callback error" || kind == "callback error and cancel" {
						return callbackCause
					}
					return nil
				},
				func(*blockstorage.VolumeImageUploadOpts) error { later++; return nil },
			}
			wantFirst := 1
			switch kind {
			case "nil context":
				selected, wantFirst = nil, 0
			case "already canceled":
				cancel(cancelCause)
				wantFirst = 0
			case "nil option":
				options[0], wantFirst = nil, 0
			}
			value, err := blockstorage.PrepareVolumeImageUploadOptions(selected, options...)
			if err == nil || !vuiZero(value) || first != wantFirst || later != 0 {
				t.Fatal(value, err, first, later)
			}
			if kind == "nil context" || kind == "nil option" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			if kind == "callback error" || kind == "callback error and cancel" {
				if !errors.Is(err, callbackCause) {
					t.Fatal("callback cause lost", err)
				}
			}
			if kind == "cancel" || kind == "already canceled" || kind == "callback error and cancel" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
					t.Fatal("custom context cause lost", err)
				}
			}
		})
	}
}
