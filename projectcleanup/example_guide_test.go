package projectcleanup_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"time"
)

func Example() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	plan, err := conn.PlanProjectCleanup(ctx,
		projectcleanup.WithCreatedBefore(time.Now().Add(-30*24*time.Hour)),
		projectcleanup.WithSkip("compute.server"),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, stage := range plan.Stages {
		for _, c := range stage.Candidates {
			fmt.Println(stage.Service, stage.Kind, c.ID, c.Name, c.Reason, c.DependsOn)
		}
	}
	for _, e := range plan.Excluded {
		fmt.Println("kept", e.Kind, e.ID, e.Reason)
	}
	for _, u := range plan.Unavailable {
		fmt.Println("not examined", u.Service, u.Reason)
	}
	result, err := conn.ExecuteProjectCleanup(ctx, plan,
		projectcleanup.WithAbsenceWait(resource.WithTimeout(5*time.Minute)),
		projectcleanup.WithProgressCallback(func(e projectcleanup.Event) error {
			log.Println(e.Outcome.Candidate.Ref, e.Outcome.State)
			return nil
		}),
	)
	for _, o := range result.Outcomes {
		fmt.Println(o.Candidate.Ref, o.State, o.Reason, o.Err)
	}
	if err != nil {
		log.Fatal(err)
	}
}
