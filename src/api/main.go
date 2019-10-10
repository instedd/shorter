package main

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var (
	db        *dynamodb.DynamoDB
	tableName *string
)

type entry struct {
	Key    string `json:"key" dynamodbav:"entry_key"`
	URL    string `json:"url" dynamodbav:"entry_url"`
	APIKey string `json:"-" dynamodbav:"api_key"`
}

type entryKey struct {
	Key string `dynamodbav:"entry_key"`
}

func toDynamoDb(e interface{}) map[string]*dynamodb.AttributeValue {
	d, err := dynamodbattribute.MarshalMap(e)
	if err != nil {
		panic(err)
	}

	return d
}

func fromDynamoDb(d map[string]*dynamodb.AttributeValue) entry {
	var entry entry
	err := dynamodbattribute.UnmarshalMap(d, &entry)
	if err != nil {
		panic(err)
	}

	return entry
}

func main() {
	region := getEnv("AWS_REGION", "us-east-1")
	config := aws.NewConfig().WithRegion(region)
	session := session.New(config)
	db = dynamodb.New(session)

	tableName = aws.String(os.Getenv("TABLE_NAME"))

	lambda.Start(handleRequest)
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if req.HTTPMethod == "GET" && req.Resource == "/{key}" {
		return handleGetKey(req)
	}

	if req.HTTPMethod == "POST" && req.Resource == "/api/v1/links" {
		return handleCreateLink(req)
	}

	return notFound()
}

func handleGetKey(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	key := req.PathParameters["key"]
	item, err := db.GetItem(&dynamodb.GetItemInput{
		TableName: tableName,
		Key:       toDynamoDb(entryKey{Key: key}),
	})

	if err != nil {
		panic(err)
	}

	entry := fromDynamoDb(item.Item)
	if entry.URL == "" {
		return notFound()
	}

	return redirect(entry.URL)
}

func handleCreateLink(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	url := req.QueryStringParameters["url"]

	if len(url) == 0 {
		return badRequest()
	}

	entry := entry{Key: generateKey(), URL: url, APIKey: req.RequestContext.Authorizer["principalId"].(string)}
	_, err := db.PutItem(&dynamodb.PutItemInput{
		TableName:           tableName,
		ConditionExpression: aws.String("attribute_not_exists(entry_key)"),
		Item:                toDynamoDb(entry),
	})

	if err != nil {
		panic(err)
	}

	return jsonData(entry)
}

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var alphabetLen = big.NewInt(int64(len(alphabet)))

func generateKey() string {
	b := make([]byte, 0, 6)
	for i := 0; i < 6; i++ {
		pos, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			panic(err)
		}

		b = append(b, alphabet[pos.Int64()])
	}

	return string(b)
}

func notFound() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       "",
		StatusCode: 404,
	}, nil
}

func badRequest() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       "",
		StatusCode: 400,
	}, nil
}

func redirect(location string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Location": location,
		},
		StatusCode: 302,
	}, nil
}

func jsonData(data interface{}) (events.APIGatewayProxyResponse, error) {
	json, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return events.APIGatewayProxyResponse{
		Body: string(json),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		StatusCode: 200,
	}, nil
}
