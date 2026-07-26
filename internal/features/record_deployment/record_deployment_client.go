package record_deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/contracttesting/cli/internal/components"
)

type RecordDeploymentClient struct {
	httpClient *components.HTTPClient
}

func NewRecordDeploymentClient(httpClient *components.HTTPClient) *RecordDeploymentClient {
	return &RecordDeploymentClient{httpClient: httpClient}
}

func (c *RecordDeploymentClient) Record(ctx context.Context, requestBody *RecordDeploymentRequestBody) (RecordDeploymentResponseBody, error) {
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return RecordDeploymentResponseBody{}, fmt.Errorf("cannot serialize deployment to JSON: %w", err)
	}

	response, err := c.httpClient.Post(ctx, "/api/deployments", bodyJSON)
	if err != nil {
		return RecordDeploymentResponseBody{}, fmt.Errorf("cannot post deployment to broker: %w", err)
	}

	var responseBody RecordDeploymentResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return RecordDeploymentResponseBody{}, fmt.Errorf("cannot parse deployment response: %w", err)
	}

	// 409 carries a structured rejection (reason + results) the command renders
	if response.StatusCode() != http.StatusOK && response.StatusCode() != http.StatusConflict {
		return RecordDeploymentResponseBody{}, fmt.Errorf("cannot post deployment to broker: %s", responseBody.Message)
	}

	return responseBody, nil
}
