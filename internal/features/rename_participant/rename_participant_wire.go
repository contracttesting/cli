package rename_participant

type RenameParticipantRequestBody struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

type RenameParticipantResponseBody struct {
	Message string `json:"message"`
}
