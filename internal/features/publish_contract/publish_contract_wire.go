package publish_contract

type ContractFragment struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

type PublishContractRequestBody struct {
	Participant string             `json:"participant"`
	Version     string             `json:"version"`
	Contracts   []ContractFragment `json:"contracts"`
}

type PublishContractResponseBody struct {
	Message    string   `json:"message"`
	Violations []string `json:"violations"`
}
