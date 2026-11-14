package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_memberRecordMutations() {
	ownerCloud := flag.String("owner-cloud", os.Getenv("OPENSTACKSDK_OWNER_CLOUD"), "owner clouds.yaml cloud")
	memberCloud := flag.String("member-cloud", os.Getenv("OPENSTACKSDK_MEMBER_CLOUD"), "shared recipient clouds.yaml cloud")
	imageID := flag.String("image-id", os.Getenv("OPENSTACKSDK_IMAGE_ID"), "literal parent image ID")
	memberID := flag.String("member-id", os.Getenv("OPENSTACKSDK_MEMBER_ID"), "literal recipient project ID")
	status := flag.String("status", "accepted", "requested member status")
	flag.Parse()
	if *ownerCloud == "" || *memberCloud == "" || *imageID == "" || *memberID == "" {
		log.Fatal("owner-cloud, member-cloud, image-id and member-id are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runMemberRecordMutations(ctx, *ownerCloud, *memberCloud, *imageID, *memberID, *status); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func showRecordMemberRecordMutations(operation string, record *image.ImageMemberRecord, callErr error) error {
	value := map[string]any{"operation": operation}
	if record != nil {
		value["resource"] = record.Resource
		value["wire"] = record.Wire
		value["image_id"] = record.ImageID
		value["status_code"] = record.StatusCode
		value["envelope_bytes"] = len(record.Envelope)
	}
	if callErr != nil {
		value["error"] = callErr.Error()
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return callErr
}

func showAcknowledgementMemberRecordMutations(ack *image.ImageMemberAcknowledgement, callErr error) error {
	value := map[string]any{"operation": "remove"}
	if ack != nil {
		value["image_id"] = ack.ImageID
		value["member_id"] = ack.MemberID
		value["status_code"] = ack.StatusCode
		value["response_bytes"] = len(ack.Body)
	}
	if callErr != nil {
		value["error"] = callErr.Error()
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return callErr
}

func runMemberRecordMutations(ctx context.Context, ownerCloud, memberCloud, imageID, memberID, status string) error {
	owner, err := openstack.Connect(ctx, openstack.WithCloud(ownerCloud))
	if err != nil {
		return err
	}
	recipient, err := openstack.Connect(ctx, openstack.WithCloud(memberCloud))
	if err != nil {
		return err
	}
	parent := resource.ID(imageID)
	input := image.ImageMemberRecordRequest{ID: memberID}
	created, err := owner.AddImageMemberRecord(ctx, parent,
		image.WithImageMemberRecordWriteAttribute("member_id", memberID),
		request.Header("X-Request-Source", "member-record-example"))
	if err := showRecordMemberRecordMutations("add", created, err); err != nil {
		return err
	}
	fetched, err := owner.GetImageMemberRecord(ctx, parent, input,
		request.Header("X-Request-Source", "member-record-example"))
	if err := showRecordMemberRecordMutations("get", fetched, err); err != nil {
		return err
	}
	updated, err := recipient.UpdateImageMemberRecord(ctx, parent,
		image.ImageMemberRecordRequest{Record: fetched},
		image.WithImageMemberRecordWriteAttribute("status", status))
	if err := showRecordMemberRecordMutations("update", updated, err); err != nil {
		return err
	}
	service, err := recipient.Image(ctx)
	if err != nil {
		return err
	}
	repeated, err := service.UpdateImageMemberRecord(ctx, parent,
		image.ImageMemberRecordRequest{Record: updated},
		image.WithImageMemberRecordWriteAttribute("status", status))
	if err := showRecordMemberRecordMutations("update-again", repeated, err); err != nil {
		return err
	}
	ack, err := owner.RemoveImageMemberRecord(ctx, parent, input,
		request.IgnoreMissing(false),
		request.Header("X-Request-Source", "member-record-example"))
	return showAcknowledgementMemberRecordMutations(ack, err)
}
