package main

import (
	"context"
	"encoding/json"
	"fmt"

	ddlambda "github.com/DataDog/datadog-lambda-go"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	fmt.Printf("Processing request data for request %s.\n", request.RequestContext.RequestID)
	fmt.Printf("Body size = %d.\n", len(request.Body))

	body := map[string]interface{}{
		"message":   "Hello World from Go Lambda with Orchestrion!",
		"requestId": request.RequestContext.RequestID,
		"timestamp": request.RequestContext.RequestTime,
		"tracing":   "Auto-instrumented with Datadog Orchestrion",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return Response{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"error": "Failed to marshal response"}`,
		}, err
	}

	response := Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"X-Instrumented-By":           "Datadog-Orchestrion",
		},
		Body: string(bodyBytes),
	}

	return response, nil
}

func main() {
	// This lambda.Start call will be automatically instrumented by orchestrion
	// No manual wrapping needed!
	lambda.Start(ddlambda.WrapFunction(handler, nil))
}
