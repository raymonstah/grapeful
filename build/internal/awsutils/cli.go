package awsutils

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli/v2"
)

func LambdaCommand() *cli.Command {
	return &cli.Command{
		Name:  "lambda",
		Usage: "upload zipped lambdas from a path to s3",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "bucket",
				Usage:    "location of where the zipped lambdas should be stored",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "target-path",
				Usage:    "path to artifact resources (zip files for lambdas)",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			var (
				ctx   = c.Context
				s     = session.Must(session.NewSession())
				s3API = s3.New(s)
			)

			if err := UploadLambdas(ctx, s3API, c.String("target-path"), c.String("lambdas-bucket")); err != nil {
				return fmt.Errorf("unable to upload lambdas: %w", err)
			}

			return nil
		},
	}
}

func CloudformationCommand() *cli.Command {
	return &cli.Command{
		Name:  "cloudformation",
		Usage: "deploy cloudformation stack",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "path",
				Usage:    "path to cloudformation template",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "stack-name",
				Usage:    "name of the cloudformation stack",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "lambdas-bucket",
				Usage: "optional -- location of the zipped lambdas",
			},
		},
		Action: func(c *cli.Context) error {
			var (
				ctx   = c.Context
				s     = session.Must(session.NewSession())
				s3API = s3.New(s)
				cfAPI = cloudformation.New(s)
			)

			err := CreateOrUpdateCloudformationStack(ctx, c.String("lambdas-bucket"), c.String("path"), c.String("stack-name"), cfAPI, s3API)
			if err != nil {
				return fmt.Errorf("unable to create or update cloudformation stack: %w", err)
			}
			return nil
		},
	}
}
