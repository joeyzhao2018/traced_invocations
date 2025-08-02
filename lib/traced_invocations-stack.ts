import * as cdk from 'aws-cdk-lib';
import * as lambda from "aws-cdk-lib/aws-lambda";
import * as apigateway from "aws-cdk-lib/aws-apigateway";
import { Construct } from "constructs";
import { DockerImage } from "aws-cdk-lib";

export class TracedInvocationsStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Create the Go Lambda function
    const helloWorldFunction = new lambda.Function(this, "HelloWorldFunction", {
      runtime: lambda.Runtime.PROVIDED_AL2023,
      handler: "bootstrap",
      code: lambda.Code.fromAsset("../", {
        bundling: {
          image: DockerImage.fromRegistry("golang:1.24"),
          command: [
            "bash",
            "-c",
            `cp -r /asset-input/datadog-lambda-go /tmp/datadog-lambda-go && cd /asset-input/traced_invocations/lambda/hello-world && GOCACHE=/tmp go mod tidy && GOCACHE=/tmp GOARCH=amd64 GOOS=linux  go build -tags lambda.norpc -o /asset-output/bootstrap main.go`,
            // `GOCACHE=/tmp go mod tidy && GOCACHE=/tmp GOARCH=amd64 GOOS=linux go build -tags lambda.norpc -o /asset-output/bootstrap main.go`,
          ],
          user: "root",
        },
      }),
      timeout: cdk.Duration.seconds(30),
      memorySize: 128,
      environment: {
        DD_API_KEY: process.env.DD_API_KEY || "your_datadog_api_key_here",
        DD_SERVICE: "traced-invocations",
        DD_SITE: "datadoghq.com",
        DD_TRACE_ENABLED: "true",
        DD_TRACE_MANAGED_SERVICES: "true",
        DD_COLD_START_TRACING: "false",
        DD_ENV: "joe",
      },
    });

    const layers = [
      lambda.LayerVersion.fromLayerVersionArn(
        this,
        "extension",
        "arn:aws:lambda:us-west-2:464622532012:layer:Datadog-Extension:83"
      ),
    ];
    helloWorldFunction.addLayers(...layers);

    // Create API Gateway
    const api = new apigateway.RestApi(this, "tracedInvocationsAPI", {
      restApiName: "Traced Invocations Service",
      description: "This service serves traced invocations.",
    });

    // Create Lambda integration
    const tracedInvocationsIntegration = new apigateway.LambdaIntegration(
      helloWorldFunction
    );

    // Add method to API Gateway
    api.root.addMethod("GET", tracedInvocationsIntegration);

    // Create the Invoker Go Lambda function
    const invokerFunction = new lambda.Function(this, "InvokerFunction", {
      runtime: lambda.Runtime.PROVIDED_AL2023,
      handler: "bootstrap",
      code: lambda.Code.fromAsset("../", {
        bundling: {
          image: DockerImage.fromRegistry("golang:1.24"),
          command: [
            "bash",
            "-c",
            `cp -r /asset-input/datadog-lambda-go /tmp/datadog-lambda-go && cd /asset-input/traced_invocations/lambda/invoker1 && GOCACHE=/tmp go mod tidy && GOCACHE=/tmp GOARCH=amd64 GOOS=linux  go build -tags lambda.norpc -o /asset-output/bootstrap main.go`,
          ],
          user: "root",
        },
      }),
      timeout: cdk.Duration.seconds(30),
      memorySize: 128,
      environment: {
        DD_API_KEY: process.env.DD_API_KEY || "your_datadog_api_key_here",
        DD_SERVICE: "traced-invocations-invoker",
        DD_SITE: "datadoghq.com",
        DD_TRACE_ENABLED: "true",
        DD_TRACE_MANAGED_SERVICES: "true",
        DD_COLD_START_TRACING: "false",
        DD_ENV: "joe",
        TARGET_API_URL: api.url, // Pass the API Gateway URL to the invoker
      },
    });

    invokerFunction.addLayers(...layers);

    // Output the API Gateway URL
    new cdk.CfnOutput(this, "ApiUrl", {
      value: api.url,
      description: "URL of the API Gateway",
    });

    new cdk.CfnOutput(this, "InvokerUrl", {
      value: `${api.url}invoke`,
      description: "URL to invoke the invoker lambda",
    });
  }
}

// App
const app = new cdk.App();
new TracedInvocationsStack(app, "TracedInvocationsStack", {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION,
  },
});
