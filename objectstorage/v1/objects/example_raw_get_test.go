package objects_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func runRawGet(ctx context.Context, cloud, container, object, query string, stream bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	swift, err := conn.ObjectStorageV1(ctx)
	if err != nil {
		return err
	}
	response, requestErr := swift.Objects.GetObjectRaw(ctx, container, object,
		func(c *objects.RawObjectOpts) error { c.QueryString = query; return nil },
		func(c *objects.RawObjectOpts) error { c.Stream = stream; return nil },
	)
	if response == nil {
		return requestErr
	}
	fmt.Fprintf(os.Stderr, "status=%d\n", response.StatusCode)
	_, readErr := io.Copy(os.Stdout, response.Body)
	closeErr := response.Body.Close()
	return errors.Join(requestErr, readErr, closeErr)
}

func Example_rawGet() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	container := flag.String("container", "assets", "literal container name")
	object := flag.String("object", "logs/part.bin", "literal object name; empty selects container")
	query := flag.String("query", "", "already encoded query string")
	stream := flag.Bool("stream", false, "transfer a guarded stream")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runRawGet(ctx, *cloud, *container, *object, *query, *stream); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
