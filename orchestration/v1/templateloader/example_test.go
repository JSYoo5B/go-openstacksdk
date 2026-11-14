package templateloader_test

import (
	"context"
	"fmt"
	"log"

	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/templateloader"
)

func ExampleLoader_ReadEnvAndTemplates() {
	ctx := context.Background()
	loader, err := templateloader.New()
	if err != nil {
		log.Fatal(err)
	}
	attrs, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{
		Template:         templateloader.Source{File: "stack.yaml"},
		EnvironmentFiles: []string{"base-env.yaml", "site-env.yaml"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(attrs.Template, attrs.Environment, attrs.Files)
}
