package record_deployment

const ReasonCompatibilityCheckRequired = "compatibility_check_required"
const ReasonNotDeployable = "not_deployable"

type RecordDeploymentRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Force       bool   `json:"force"`
}

type RecordDeploymentResponseBody struct {
	Message string                            `json:"message"`
	Reason  string                            `json:"reason"`
	Results map[string]RecordDeploymentResult `json:"results"`
}

// RecordDeploymentResult carries the per-counterpart rejection details: the
// checked/deployed pair on compatibility_check_required, the version+reason
// pair on not_deployable.
type RecordDeploymentResult struct {
	CheckedVersion     *string `json:"checkedVersion"`
	DeployedVersion    *string `json:"deployedVersion"`
	CounterpartVersion *string `json:"counterpartVersion"`
	Reason             string  `json:"reason"`
}
