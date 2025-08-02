package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	ddlambda "github.com/DataDog/datadog-lambda-go"
	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type InvokerResult struct {
	Message           string      `json:"message"`
	RequestID         string      `json:"requestId"`
	InvocationResult  interface{} `json:"invocationResult"`
	ResponseTime      string      `json:"responseTime"`
	HTTPStatusCode    int         `json:"httpStatusCode"`
	APIGatewayURL     string      `json:"apiGatewayUrl"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	fmt.Printf("Invoker Lambda processing request: %s\n", request.RequestContext.RequestID)

	// Get the API Gateway URL from environment variable
	apiURL := os.Getenv("TARGET_API_URL")
	if apiURL == "" {
		return Response{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"error": "TARGET_API_URL environment variable not set"}`,
		}, fmt.Errorf("TARGET_API_URL not configured")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	client = httptrace.WrapClient(client)
	// Record start time for response time measurement
	startTime := time.Now()

	// Make HTTP GET request to the hello-world lambda via API Gateway
	resp, err := client.Get(apiURL)
	if err != nil {
		return Response{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: fmt.Sprintf(`{"error": "Failed to invoke target lambda: %s"}`, err.Error()),
		}, err
	}
	defer resp.Body.Close()

	// Calculate response time
	responseTime := time.Since(startTime)

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: fmt.Sprintf(`{"error": "Failed to read response: %s"}`, err.Error()),
		}, err
	}

	// Parse the response from hello-world lambda
	var helloWorldResponse interface{}
	if err := json.Unmarshal(bodyBytes, &helloWorldResponse); err != nil {
		// If JSON parsing fails, use raw string
		helloWorldResponse = string(bodyBytes)
	}

	// Create our response
	result := InvokerResult{
		Message:           "Successfully invoked hello-world lambda via HTTP",
		RequestID:         request.RequestContext.RequestID,
		InvocationResult:  helloWorldResponse,
		ResponseTime:      responseTime.String(),
		HTTPStatusCode:    resp.StatusCode,
		APIGatewayURL:     apiURL,
	}

	resultBytes, err := json.Marshal(result)
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
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
			"X-Instrumented-By":          "Datadog-Orchestrion",
			"X-Invoked-Target":           "hello-world-lambda",
		},
		Body: string(resultBytes),
	}

	return response, nil
}

func main() {
	lambda.Start(ddlambda.WrapFunction(handler, nil))
}
