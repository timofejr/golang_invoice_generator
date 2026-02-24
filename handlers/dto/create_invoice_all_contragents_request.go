package dto

type CreateInvoiceAllContragentsRequest struct {
	InvoiceID       string   `json:"invoice_id" binding:"required"`
	Daytime         *string  `json:"daytime"`
	Worksheets      []string `json:"worksheets" binding:"required"`
	ApplicationType string   `json:"application_type" binding:"required"`
}
