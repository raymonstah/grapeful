package awsutils

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"
	"time"
)

func CreateOrUpdateCloudformationStack(ctx context.Context, lambdasBucket, pathToTemplate, stackname string, cfAPI cloudformationiface.CloudFormationAPI, s3API s3iface.S3API) error {
	bytes, err := ioutil.ReadFile(pathToTemplate)
	if err != nil {
		return fmt.Errorf("error reading file %v: %w", pathToTemplate, err)
	}

	versions, err := getVersions(ctx, s3API, lambdasBucket)
	if err != nil {
		return fmt.Errorf("unable to get versions: %w", err)
	}

	for _, version := range versions {
		fmt.Printf("zipped lambda found: %v, version %v\n", *version.Key, *version.VersionId)
	}

	parameters, err := makeParams(string(bytes), versions, lambdasBucket)
	_, err = cfAPI.DescribeStacksWithContext(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackname)},
	)
	if err != nil {
		var awsErr awserr.Error
		if ok := errors.As(err, &awsErr); ok && awsErr.Code() == "ValidationError" {
			_, err = cfAPI.CreateStackWithContext(ctx, &cloudformation.CreateStackInput{
				Capabilities: []*string{
					aws.String(cloudformation.CapabilityCapabilityNamedIam),
					aws.String(cloudformation.CapabilityCapabilityAutoExpand),
				},
				Parameters:   parameters,
				StackName:    aws.String(stackname),
				TemplateBody: aws.String(string(bytes)),
			})
			if err != nil {
				return fmt.Errorf("unable to create stack %v: %w", stackname, err)
			}
			fmt.Printf("created stack %v\n", stackname)
			go printEvents(ctx, cfAPI, stackname)
			err := cfAPI.WaitUntilStackCreateCompleteWithContext(ctx, &cloudformation.DescribeStacksInput{
				StackName: aws.String(stackname),
			})
			if err != nil {
				return fmt.Errorf("unable to wait until stack create complete: %w", err)
			}
			return nil

		}
		return fmt.Errorf("unable to describe stack %v: %w", stackname, err)
	}

	_, err = cfAPI.UpdateStackWithContext(ctx, &cloudformation.UpdateStackInput{
		Capabilities: []*string{
			aws.String(cloudformation.CapabilityCapabilityNamedIam),
			aws.String(cloudformation.CapabilityCapabilityAutoExpand),
		},
		Parameters:   parameters,
		StackName:    aws.String(stackname),
		TemplateBody: aws.String(string(bytes)),
	})
	if err != nil {
		var awsErr awserr.Error
		if ok := errors.As(err, &awsErr); ok && awsErr.Code() == "ValidationError" {
			// delete the stack and try again
			_, err = cfAPI.DeleteStackWithContext(ctx, &cloudformation.DeleteStackInput{
				StackName: aws.String(stackname),
			})
			if err != nil {
				return fmt.Errorf("unable to delete stack %v: %w", stackname, err)
			}
			fmt.Println("deleting stack.. please try again")
			return nil

		}
		return fmt.Errorf("unable to update stack: %w", err)
	}

	go printEvents(ctx, cfAPI, stackname)
	err = cfAPI.WaitUntilStackUpdateCompleteWithContext(ctx, &cloudformation.DescribeStacksInput{StackName: aws.String(stackname)})
	if err != nil {
		return fmt.Errorf("error waiting for stack to complete: %w", err)
	}
	return nil
}


func isErrShown(err error) bool {
	v, ok := err.(awserr.Error)
	if !ok {
		return true
	}

	if code := v.Code(); code == "RequestCanceled" || code == "ValidationError" {
		return false
	}

	return true
}

func printEvents(ctx context.Context, api cloudformationiface.CloudFormationAPI, stackName string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	now := time.Now().Add(-20 * time.Second)
	seen := map[string]struct{}{} // keep track of events we've seen

outer:
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		input := cloudformation.DescribeStackEventsInput{
			StackName: aws.String(stackName),
		}

		var nextToken *string
		for {
			input.NextToken = nextToken

			output, err := api.DescribeStackEventsWithContext(ctx, &input)
			if err != nil {
				if isErrShown(err) {
					fmt.Printf("describe stack events failed for stack, %v - %v\n", stackName, err)
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
					continue outer
				}
			}

			var events []*cloudformation.StackEvent
			for _, event := range output.StackEvents {
				if event.Timestamp.Before(now) {
					break
				}

				if _, ok := seen[*event.EventId]; ok {
					continue
				}
				seen[*event.EventId] = struct{}{}

				events = append(events, event)
			}

			for i := len(events) - 1; i >= 0; i-- {
				event := events[i]
				text := fmt.Sprintf("%s  %-25s %-35s %-35s %s\n",
					event.Timestamp.In(time.Local).Format("15:04:05.000"),
					aws.StringValue(event.LogicalResourceId),
					aws.StringValue(event.ResourceType),
					aws.StringValue(event.ResourceStatus),
					aws.StringValue(event.ResourceStatusReason),
				)

				if status := aws.StringValue(event.ResourceStatus); strings.Contains(status, "FAILED") || strings.Contains(status, "DELETE") {
					color.Red(text)
				} else if strings.Contains(status, "UPDATE") {
					color.Yellow(text)
				} else if strings.Contains(status, "CREATE") {
					color.Green(text)
				} else {
					color.Blue(text)
				}
			}

			nextToken = output.NextToken
			if nextToken == nil {
				break
			}
		}
	}
}

// makeParams for cloudformation
func makeParams(templateBody string, versions []*s3.ObjectVersion, lambdasBucket string) ([]*cloudformation.Parameter, error) {
	var parameters []*cloudformation.Parameter

	keys, err := extractParameterNames(templateBody)
	if err != nil {
		return nil, fmt.Errorf("unable to extract parameter names: %w", err)
	}
	for _, key := range keys {
		switch key {
		case "LambdaBucket":
			parameters = appendParameter(parameters, key, lambdasBucket)
		default:
			// if nothing matched so far, assume the parameter is for lambda
			for _, v := range versions {
				// strip symbols
				reg, err := regexp.Compile("[^a-zA-Z0-9]+")
				if err != nil {
					return nil, err
				}
				k := reg.ReplaceAllString(*v.Key, "")
				if k == strings.ToLower(key) {
					parameters = appendParameter(parameters, key, *v.VersionId)
					continue
				}
			}
		}
	}

	return parameters, nil
}

// extractParameterNames extracts the names of the parameters from the
// cloudformation template body provided. May be either YAML or JSON.
func extractParameterNames(templateBody string) ([]string, error) {
	var (
		data     = []byte(templateBody)
		template struct {
			Parameters map[string]interface{} `yaml:"Parameters"`
		}
	)

	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("unable to unmarshal template body: %w", err)
	}

	var names []string
	for name := range template.Parameters {
		names = append(names, name)
	}
	sort.Strings(names)

	return names, nil
}

func appendParameter(parameters []*cloudformation.Parameter, name, value string) []*cloudformation.Parameter {
	if name == "" {
		panic(fmt.Errorf("illegal attempt to set template parameter with blank name"))
	}
	if value == "" {
		panic(fmt.Errorf("illegal attempt to set template parameter, %v, with blank value", name))
	}

	fmt.Printf("parameter %v: %v\n", name, value)
	return append(parameters, &cloudformation.Parameter{
		ParameterKey:   aws.String(name),
		ParameterValue: aws.String(value),
	})
}

// getVersions of the latest objects in a s3 bucket
func getVersions(ctx context.Context, s3API s3iface.S3API, bucketName string) (allLatestVersions []*s3.ObjectVersion, err error) {
	if bucketName == "" {
		return
	}
	var nextKeyMarker *string
	for {
		versionOutput, err := s3API.ListObjectVersionsWithContext(ctx, &s3.ListObjectVersionsInput{
			Bucket:    aws.String(bucketName),
			KeyMarker: nextKeyMarker,
			MaxKeys:   aws.Int64(100),
		})
		if err != nil {
			return nil, fmt.Errorf("unable to list object versions: %w", err)
		}
		for _, version := range versionOutput.Versions {
			if *version.IsLatest {
				allLatestVersions = append(allLatestVersions, version)
			}
		}
		if !*versionOutput.IsTruncated {
			break
		}
		nextKeyMarker = versionOutput.NextKeyMarker
	}

	return
}
