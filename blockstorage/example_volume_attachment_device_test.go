package blockstorage_test

import (
	"encoding/json"
	"errors"
	"fmt"

	volumesv2 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v2/volumes"
	volumesv3 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func textVolumeAttachmentDevice(value string) *string { return &value }

func deviceVolumeAttachmentDevice(value *string, err error) string {
	if err != nil {
		panic(err)
	}
	if value == nil {
		return "nil"
	}
	return fmt.Sprintf("%q", *value)
}

func rawDeviceVolumeAttachmentDevice(value json.RawMessage, err error) string {
	if err != nil {
		panic(err)
	}
	if value == nil {
		return "nil"
	}
	return string(value)
}

func invalidVolumeAttachmentDevice(label string, err error) {
	var operation *resource.OperationError
	if !errors.As(err, &operation) || !errors.Is(err, resource.ErrInvalidOption) {
		panic(fmt.Sprintf("unexpected error: %v", err))
	}
	fmt.Printf("%s %s %s %t\n", label, operation.Operation,
		operation.Resource, errors.Is(err, resource.ErrInvalidOption))
}

func Example_volumeAttachmentDevice() {
	owned := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{
		{ServerID: textVolumeAttachmentDevice("server"), Device: textVolumeAttachmentDevice("")},
		{ServerID: textVolumeAttachmentDevice("server"), Device: textVolumeAttachmentDevice("/dev/vdb")},
	}}
	fmt.Println("owned first", deviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDevice(owned, "server")))
	fmt.Println("owned no match", deviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDevice(owned, "SERVER")))
	empty := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{}}
	fmt.Println("owned empty", deviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDevice(empty, "server")))
	null := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{
		{ServerID: textVolumeAttachmentDevice("server"), Device: nil}, // Manual Body:nil means explicit null.
	}}
	fmt.Println("owned null", deviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDevice(null, "server")))

	var current blockstorage.AttachVolumeObservation
	if err := json.Unmarshal([]byte(`{"attachments":[{"server_id":"server","deviceVolumeAttachmentDevice":"/dev/original"}]}`), &current); err != nil {
		panic(err)
	}
	current.Attachments[0].Device = textVolumeAttachmentDevice("/dev/current")
	fmt.Println("typed current", deviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDevice(&current, "server")))
	fmt.Println("raw original", rawDeviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDeviceFields(current.Body, "server")))

	nativeV2 := &volumesv2.Volume{Attachments: []volumesv2.Attachment{
		{ServerID: "server", Device: ""},
	}}
	nativeV3 := &volumesv3.Volume{Attachments: []volumesv3.Attachment{
		{ServerID: "server", Device: "/dev/vdc"},
	}}
	fmt.Println("native v2", deviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDevice(nativeV2, "server")))
	fmt.Println("native v3", deviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDevice(nativeV3, "server")))

	raw := map[string]json.RawMessage{
		"attachments": json.RawMessage(`[{"server_id":"server","deviceVolumeAttachmentDevice":false}]`),
		"status":      json.RawMessage("unrelated invalidVolumeAttachmentDevice JSON"), // This field is not consumed.
	}
	fmt.Println("raw false", rawDeviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
	raw["attachments"] = json.RawMessage(`[{"server_id":"server","deviceVolumeAttachmentDevice":9007199254740993123456789}]`)
	fmt.Println("raw number", rawDeviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
	raw["attachments"] = json.RawMessage(`[{"server_id":"server","deviceVolumeAttachmentDevice":{"path":"/dev/vdb"}}]`)
	fmt.Println("raw object", rawDeviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
	raw["attachments"] = json.RawMessage(`[{"server_id":"server","deviceVolumeAttachmentDevice":null}]`)
	fmt.Println("raw null", rawDeviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))

	tail := []byte(`{"attachments":[{"server_id":"server","deviceVolumeAttachmentDevice":"/dev/vdb"},null]}`)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(tail, &fields); err != nil {
		panic(err)
	}
	fmt.Println("raw short circuit", rawDeviceVolumeAttachmentDevice(blockstorage.GetVolumeAttachDeviceFields(fields, "server")))
	var whole blockstorage.VolumeInfo
	fmt.Println("whole decoder rejects tail", json.Unmarshal(tail, &whole) != nil)

	_, err := blockstorage.GetVolumeAttachDevice((*blockstorage.VolumeInfo)(nil), "server")
	invalidVolumeAttachmentDevice("typed nil error", err)
	fields["attachments"] = json.RawMessage(`[{"server_id":"server"}]`)
	_, err = blockstorage.GetVolumeAttachDeviceFields(fields, "server")
	invalidVolumeAttachmentDevice("raw missing error", err)
}
