package accounts_test

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func changeAccountMetadataAccounts(ctx context.Context, api *accounts.API) error {
	getOptions := []accounts.GetMetadataOption{
		accounts.WithGetMetadataOpts(accounts.GetMetadataOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		accounts.WithGetMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
		request.Header("X-Request-Label", "account-metadata"),
		request.Custom(func(c *accounts.GetMetadataOpts) error { v := false; c.Newest = &v; return nil }),
	}
	before, err := api.GetMetadata(ctx, getOptions...)
	if err != nil {
		return err
	}
	if before.Metadata.BytesUsed != nil {
		fmt.Println(*before.Metadata.BytesUsed)
	}

	mutationOptions := []accounts.MetadataOption{
		accounts.WithMetadataOpts(accounts.MetadataOpts{
			Headers: map[string]string{"X-Request-Label": "initial"},
		}),
		accounts.WithMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
		request.Header("X-Request-Label", "metadata-mutation"),
	}
	acknowledgement, err := api.SetMetadata(ctx, map[string]string{
		"book": "Moby Dick", "note": "literal %2F 雪",
	}, mutationOptions...)
	if acknowledgement != nil {
		fmt.Println(acknowledgement.StatusCode)
	}
	if err != nil {
		return err
	}

	refreshOptions := append([]accounts.GetMetadataOption(nil), getOptions...)
	refreshOptions = append(refreshOptions, request.Custom(func(c *accounts.GetMetadataOpts) error { c.Newest = nil; return nil }))
	refreshed, err := api.GetMetadata(ctx, refreshOptions...)
	if err != nil {
		return err
	}
	fmt.Println(refreshed.Metadata.Values["book"])

	removed, err := api.DeleteMetadata(ctx, []string{"note"}, mutationOptions...)
	if removed != nil {
		fmt.Println(removed.StatusCode)
	}
	return err
}

func Example_accounts() {}
