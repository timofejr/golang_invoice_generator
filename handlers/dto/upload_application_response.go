package dto

type UploadApplicationResponse struct {
	ID string `json:"id"`
	ContrAgents []string `json:"contr_agents"`
	Daytimes []string `json:"daytimes"`
	WorkSheets []string `json:"worksheets"`
	ApplicationType string `json:"application_type"`
}