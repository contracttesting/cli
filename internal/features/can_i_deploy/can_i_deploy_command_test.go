package can_i_deploy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatNotDeployableReportWalksTheTree(t *testing.T) {
	version := "3.1.0"
	results := map[string]CanIDeployResult{
		"payments-api": {
			Deployable:         false,
			ParticipantVersion: &version,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/payments/*": {
					"get": {
						"request": {
							{Reason: "property_missing_in_provider", Details: map[string]string{
								"property":     "currency",
								"propertyType": "string",
								"consumerName": "payments-web",
								"providerName": "payments-api",
							}},
						},
						"200": {
							// out of order on purpose: the report sorts break lines within an interaction
							{Reason: "property_missing_in_provider", Details: map[string]string{
								"property":     "items",
								"propertyType": "array<object>",
								"consumerName": "payments-web",
								"providerName": "payments-api",
							}},
							{Reason: "property_type_mismatch", Details: map[string]string{
								"property":             "amount",
								"consumerPropertyType": "string",
								"providerPropertyType": "number",
							}},
						},
					},
				},
			},
		},
		"users": {
			Deployable:         true,
			ParticipantVersion: &version,
			Endpoints:          map[string]map[string]map[string][]ContractBreak{},
		},
	}

	report := formatNotDeployableReport("payments-web", "production", results)

	assert.Equal(t, `❌ payments-web cannot be deployed to production

payments-api (3.1.0, latest deployed version in "production"):
  GET /payments/*
    request:
      - property "currency":string required in payments-web (consumer) absent in payments-api (provider)
    response 200:
      - property "amount" type mismatch — consumer has string, provider has number
      - property "items":array<object> required in payments-web (consumer) absent in payments-api (provider)
`, report)
}

func TestFormatNotDeployableReportOmitsVersionWhenNull(t *testing.T) {
	results := map[string]CanIDeployResult{
		"payments-api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/payments/*": {
					"get": {
						"200": {
							{Reason: "provider_resource_not_found"},
						},
					},
				},
			},
		},
	}

	report := formatNotDeployableReport("payments-web", "production", results)

	assert.Equal(t, `❌ payments-web cannot be deployed to production

payments-api:
  GET /payments/*
    response 200:
      - no matching resource in provider
`, report)
}

func TestFormatBreakLine(t *testing.T) {
	t.Run("provider_resource_not_found", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason: "provider_resource_not_found",
		})

		assert.Equal(t, `no matching resource in provider`, line)
	})

	t.Run("provider_resource_not_deployed_in_environment", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason:  "provider_resource_not_deployed_in_environment",
			Details: map[string]string{"deployedEnvironments": "staging, qa"},
		})

		assert.Equal(t, `provider is not deployed in "production" (deployed in: staging, qa)`, line)
	})

	t.Run("provider_resource_not_deployed_in_environment without deployedEnvironments", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason: "provider_resource_not_deployed_in_environment",
		})

		assert.Equal(t, `provider is not deployed in "production"`, line)
	})

	t.Run("property_missing_in_provider", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason: "property_missing_in_provider",
			Details: map[string]string{
				"property":     "amount",
				"propertyType": "number",
				"consumerName": "payments-web",
				"providerName": "payments-api",
			},
		})

		assert.Equal(t, `property "amount":number required in payments-web (consumer) absent in payments-api (provider)`, line)
	})

	t.Run("property_missing_in_provider renders array item types", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason: "property_missing_in_provider",
			Details: map[string]string{
				"property":     "items",
				"propertyType": "array<string>",
				"consumerName": "payments-web",
				"providerName": "payments-api",
			},
		})

		assert.Equal(t, `property "items":array<string> required in payments-web (consumer) absent in payments-api (provider)`, line)
	})

	t.Run("property_missing_in_consumer", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason: "property_missing_in_consumer",
			Details: map[string]string{
				"property":     "amount",
				"propertyType": "number",
				"consumerName": "payments-web",
				"providerName": "payments-api",
			},
		})

		assert.Equal(t, `property "amount":number required in payments-api (provider) absent in payments-web (consumer)`, line)
	})

	t.Run("property_optional_in_provider_required_in_consumer", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason:  "property_optional_in_provider_required_in_consumer",
			Details: map[string]string{"property": "amount", "propertyType": "number"},
		})

		assert.Equal(t, `property "amount":number is optional in provider but required in consumer`, line)
	})

	t.Run("property_optional_in_consumer_required_in_provider", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason:  "property_optional_in_consumer_required_in_provider",
			Details: map[string]string{"property": "amount", "propertyType": "number"},
		})

		assert.Equal(t, `property "amount":number is optional in consumer but required in provider`, line)
	})

	t.Run("property_type_mismatch uses the role-resolved types", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason: "property_type_mismatch",
			Details: map[string]string{
				"property":             "amount",
				"consumerPropertyType": "string",
				"providerPropertyType": "number",
			},
		})

		assert.Equal(t, `property "amount" type mismatch — consumer has string, provider has number`, line)
	})

	t.Run("property_not_matching_any_variant in a response blames the provider variant", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason: "property_not_matching_any_variant",
			Details: map[string]string{
				"property":             "$.petId#1",
				"consumerPropertyType": "anyOf<number, string>",
				"providerPropertyType": "boolean",
			},
		})

		assert.Equal(t, `property "$.petId":boolean may be returned by provider but not accepted by consumer (consumer accepts anyOf<number, string>)`, line)
	})

	t.Run("property_not_matching_any_variant in a request blames the consumer variant", func(t *testing.T) {
		line := formatBreakLine("production", "request", ContractBreak{
			Reason: "property_not_matching_any_variant",
			Details: map[string]string{
				"property":             "$.petId#1",
				"consumerPropertyType": "boolean",
				"providerPropertyType": "anyOf<number, string>",
			},
		})

		assert.Equal(t, `property "$.petId":boolean may be sent by consumer but not accepted by provider (provider accepts anyOf<number, string>)`, line)
	})

	t.Run("property_not_matching_any_variant keeps a plain variant path as written", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason: "property_not_matching_any_variant",
			Details: map[string]string{
				"property":             "$.pets[]",
				"consumerPropertyType": "anyOf<object, object>",
				"providerPropertyType": "object",
			},
		})

		assert.Equal(t, `property "$.pets[]":object may be returned by provider but not accepted by consumer (consumer accepts anyOf<object, object>)`, line)
	})

	t.Run("unknown reason falls back to the verbatim reason with details", func(t *testing.T) {
		line := formatBreakLine("production", "200", ContractBreak{
			Reason:  "some_future_reason",
			Details: map[string]string{"property": "amount", "extra": "x"},
		})

		assert.Equal(t, `some_future_reason (extra: x, property: amount)`, line)
	})
}
