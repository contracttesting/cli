package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/contracttesting/cli/internal/components"
	"github.com/contracttesting/cli/internal/features/record_deployment"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordDeploymentCommand(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "api"
		version     = "v1"
		environment = "production"
		endpoint    = brokerURL + "/api/deployments"
	)

	t.Run("records a deployment, posts participant+version+environment, prints the message, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"success":true,"message":"deployment recorded"}`), nil
			})

		command := record_deployment.NewRecordDeploymentCommand(
			record_deployment.NewRecordDeploymentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, `{"participant":"api","version":"v1","environment":"production","force":false}`, string(capturedBody))
		assert.Contains(t, out.String(), "deployment recorded")
		assert.Empty(t, errOut.String())
	})

	t.Run("--force-record sends force=true in the request body", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"message":"deployment recorded despite a not deployable verdict"}`), nil
			})

		command := record_deployment.NewRecordDeploymentCommand(
			record_deployment.NewRecordDeploymentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment, "--force-record"})

		err := command.Execute()

		require.NoError(t, err)
		assert.JSONEq(t, `{"participant":"api","version":"v1","environment":"production","force":true}`, string(capturedBody))
		assert.Contains(t, out.String(), "deployment recorded despite a not deployable verdict")
		assert.Empty(t, errOut.String())
	})

	t.Run("409 compatibility_check_required tells the user to run can-i-deploy and lists the drift", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusConflict, `{
				"message": "run can-i-deploy for api v1 against production first",
				"reason": "compatibility_check_required",
				"results": {
					"payments": { "checkedVersion": "v1", "deployedVersion": "v2" },
					"billing":  { "checkedVersion": null, "deployedVersion": "v1" },
					"users":    { "checkedVersion": "v3", "deployedVersion": null }
				}
			}`))

		command := record_deployment.NewRecordDeploymentCommand(
			record_deployment.NewRecordDeploymentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		require.ErrorIs(t, err, components.ErrSilent)
		assert.Contains(t, out.String(), "a fresh compatibility check is required")
		assert.Contains(t, out.String(), "payments: checked against v1, but v2 is deployed now")
		assert.Contains(t, out.String(), "billing: checked as not deployed, but v1 is deployed now")
		assert.Contains(t, out.String(), "users: checked against v3, but it is no longer deployed")
		assert.Contains(t, out.String(), "ctio can-i-deploy api --version v1 --environment production")
		assert.NotContains(t, errOut.String(), "Error:")
	})

	t.Run("409 not_deployable lists each failing counterpart with version and reason", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusConflict, `{
				"message": "api v1 is not deployable to production",
				"reason": "not_deployable",
				"results": {
					"payments": { "counterpartVersion": "v1", "reason": "property_type_mismatch" },
					"billing":  { "counterpartVersion": null, "reason": "provider_resource_not_deployed_in_environment" }
				}
			}`))

		command := record_deployment.NewRecordDeploymentCommand(
			record_deployment.NewRecordDeploymentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		require.ErrorIs(t, err, components.ErrSilent)
		assert.Contains(t, out.String(), "api v1 is not deployable to production")
		assert.Contains(t, out.String(), `payments (v1): property type mismatch`)
		assert.Contains(t, out.String(), `billing: provider is not deployed in "production"`)
		assert.Contains(t, out.String(), "ctio can-i-deploy api --version v1 --environment production")
		assert.NotContains(t, errOut.String(), "Error:")
	})

	t.Run("missing --version fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		command := record_deployment.NewRecordDeploymentCommand(
			record_deployment.NewRecordDeploymentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})

	t.Run("missing --environment fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		command := record_deployment.NewRecordDeploymentCommand(
			record_deployment.NewRecordDeploymentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version})

		err := command.Execute()

		require.Error(t, err)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})

	t.Run("non-2xx response surfaces the broker body and exits non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusBadRequest, `{"success":false,"message":"version not found"}`))

		command := record_deployment.NewRecordDeploymentCommand(
			record_deployment.NewRecordDeploymentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "version not found")
	})
}
