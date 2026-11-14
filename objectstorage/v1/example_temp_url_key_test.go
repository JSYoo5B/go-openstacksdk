package v1_test

import (
	"context"
	"fmt"

	objectstorage "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func tempURLKeysTempUrlKey(ctx context.Context, service *objectstorage.Service, name, accountSecret, containerSecret string) error {
	accountOptions := []accounts.SetTempURLKeyOption{
		accounts.WithSetTempURLKeyOpts(accounts.SetTempURLKeyOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		accounts.WithSetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "account-key"}),
		request.Header("X-Request-Label", "account"),
		request.Custom(func(c *accounts.SetTempURLKeyOpts) error { c.Secondary = false; return nil }),
	}
	accountAck, err := service.Accounts.SetTempURLKey(ctx, accountSecret, accountOptions...)
	if accountAck != nil {
		fmt.Println(accountAck.StatusCode)
	}
	if err != nil {
		return err
	}

	containerOptions := []containers.SetTempURLKeyOption{
		containers.WithSetTempURLKeyOpts(containers.SetTempURLKeyOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		containers.WithSetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "container-key"}),
		request.Header("X-Request-Label", "container"),
		request.Custom(func(c *containers.SetTempURLKeyOpts) error { c.Secondary = true; return nil }),
	}
	containerAck, err := service.Containers.SetTempURLKey(ctx, name, containerSecret, containerOptions...)
	if containerAck != nil {
		fmt.Println(containerAck.StatusCode)
	}
	if err != nil {
		return err
	}

	getOptions := []objectstorage.GetTempURLKeyOption{
		objectstorage.WithGetTempURLKeyOpts(objectstorage.GetTempURLKeyOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		request.Custom(func(c *objectstorage.GetTempURLKeyOpts) error { c.Container = name; return nil }),
		objectstorage.WithGetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "key-discovery"}),
		request.Header("X-Request-Label", "read"),
		request.Custom(func(c *objectstorage.GetTempURLKeyOpts) error { v := true; c.Newest = &v; return nil }),
		request.Custom(func(c *objectstorage.GetTempURLKeyOpts) error { c.Newest = nil; return nil }),
		request.Custom(func(c *objectstorage.GetTempURLKeyOpts) error { v := false; c.Newest = &v; return nil }),
	}
	selected, err := service.GetTempURLKey(ctx, getOptions...)
	if selected != nil {
		if selected.Container != nil {
			fmt.Println(selected.Container.StatusCode)
		}
		if selected.Account != nil {
			fmt.Println(selected.Account.StatusCode)
		}
	}
	if err != nil {
		return err
	}
	fmt.Println(selected.Key != nil, selected.FromContainer, selected.Secondary)
	return nil
}

func Example_tempUrlKey() {}
