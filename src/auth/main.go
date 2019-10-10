package main

import (
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
)

var (
	apigw *apigateway.APIGateway
)

func main() {
	region := getEnv("AWS_REGION", "us-east-1")
	config := aws.NewConfig().WithRegion(region)
	session := session.New(config)
	apigw = apigateway.New(session)

	lambda.Start(handleRequest)
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func handleRequest(req events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	var key *apigateway.ApiKey
	input := apigateway.GetApiKeysInput{IncludeValues: aws.Bool(true)}
	err := apigw.GetApiKeysPages(&input, func(page *apigateway.GetApiKeysOutput, lastPage bool) bool {
		for i := 0; i < len(page.Items); i++ {
			if *page.Items[i].Value == req.AuthorizationToken {
				key = page.Items[i]
				return false
			}
		}

		return true
	})

	if err != nil {
		panic(err)
	}

	if key != nil {
		return events.APIGatewayCustomAuthorizerResponse{
			PrincipalID:    *key.Name,
			PolicyDocument: createPolicy("Allow", req.MethodArn),
		}, nil
	}

	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID:    "unknown",
		PolicyDocument: createPolicy("Deny", req.MethodArn),
	}, nil
}

func createPolicy(effect string, methodArn string) events.APIGatewayCustomAuthorizerPolicy {
	return events.APIGatewayCustomAuthorizerPolicy{
		Version: "2012-10-17",
		Statement: []events.IAMPolicyStatement{
			events.IAMPolicyStatement{
				Action:   []string{"execute-api:Invoke"},
				Effect:   effect,
				Resource: []string{methodArn},
			},
		},
	}
}
