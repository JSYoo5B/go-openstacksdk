package objects_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func showWaitWait(result *objects.ObjectWaitResult, err error) {
	if result != nil {
		status := "<absent>"
		if result.Status != nil {
			status = *result.Status
		}
		fmt.Printf("polls=%d complete=%v deleted=%v status=%q\n",
			result.Polls, result.Complete, result.Deleted, status)
		if result.Last != nil {
			for _, attempt := range result.Last.Attempts {
				fmt.Printf("physical=%d error=%v\n", attempt.PhysicalAttempt, attempt.Error)
				if attempt.Response != nil {
					fmt.Printf("HTTP%d headers=%v body=%q\n",
						attempt.Response.StatusCode, attempt.Response.Header, attempt.Response.Body)
				}
			}
			if result.Last.Acknowledgement != nil {
				fmt.Printf("last acknowledgement HTTP%d\n", result.Last.Acknowledgement.StatusCode)
			}
		}
	}
	if err != nil {
		fmt.Println(err)
		if errors.Is(err, resource.ErrUnsupported) {
			fmt.Println("unsupported status source")
		}
		var proof *resource.ResponseError
		if errors.As(err, &proof) {
			fmt.Printf("error proof HTTP%d headers=%v body=%q\n",
				proof.StatusCode, proof.Header, proof.Body)
		}
	}
}

func waitExamplesWait(ctx context.Context, conn *sdk.Connection) {
	service, err := conn.ObjectStorageV1(ctx)
	if err != nil {
		showWaitWait(nil, err)
		return
	}
	policy := []objects.ObjectWaitOption{
		objects.WithObjectWaitOpts(objects.ObjectWaitOpts{
			Headers: map[string]string{"X-Client": "object-wait"},
			Wait:    []resource.WaitOption{resource.WithPollInterval(2 * time.Second)},
		}),
		request.Header("X-Trace", "initial"),
		objects.WithObjectWaitHeaders(map[string]string{"x-trace": "last"}),
		objects.WithObjectWait(
			resource.WithTimeout(time.Minute),
			resource.WithProgressCallback(func(progress int) error {
				fmt.Printf("nonterminal observation, progress=%d\n", progress)
				return nil
			}),
		),
	}
	statusOptions := append([]objects.ObjectWaitOption{}, policy...)
	statusOptions = append(statusOptions,
		objects.WithObjectWaitStatusHeader("X-Object-Meta-State"),
		objects.WithObjectWait(resource.WithFailureStates("ERROR", "FAILED")),
	)
	result, err := service.Objects.WaitForStatus(ctx, "backups", "archive.tar", "READY", statusOptions...)
	showWaitWait(result, err)

	// A known string attribute can also be observed, such as the MIME type.
	attributeOptions := []objects.ObjectWaitOption{
		objects.WithObjectWait(resource.WithStatusAttribute("content_type"), resource.WithFailureStates()),
	}
	result, err = service.Objects.WaitForStatus(ctx, "backups", "archive.tar", "application/octet-stream", attributeOptions...)
	showWaitWait(result, err)

	// UnlimitedWait removes the SDK deadline; this caller still has 30 seconds.
	deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	deleteOptions := []objects.ObjectWaitOption{objects.WithObjectWait(resource.WithUnlimitedWait())}
	result, err = service.Objects.WaitForDelete(deleteCtx, "backups", "already-deleted.bin", deleteOptions...)
	showWaitWait(result, err)
}

func Example_wait() {}
