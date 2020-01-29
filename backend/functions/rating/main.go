package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/raymonstah/grapeful/backend/domain/happinessrating"
	"github.com/urfave/cli/v2"
)

var now bool

// Handler handles requests
type Handler struct {
	ratingMutator happinessrating.Mutator
	ratingFinder  happinessrating.Finder
}

func (h *Handler) handle(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		fmt.Println("got GET")
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Hello there!",
	}, nil
}

func main() {
	app := cli.NewApp()
	app.Usage = "a REST API for recording happiness levels"
	app.EnableBashCompletion = true
	app.HideVersion = true
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "now",
			Usage:       "set to run locally (not part of lambda)",
			Destination: &now,
		},
	}
	app.Action = action
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func action(c *cli.Context) error {
	h := Handler{}
	
	if now {
		response, err := h.handle(c.Context, events.APIGatewayProxyRequest{})
		if err != nil {
			return fmt.Errorf("unable to handle request: %w", err)
		}
		fmt.Printf("%+v\n", response)
		return nil
	}

	lambda.Start(h.handle)
	return nil
}
