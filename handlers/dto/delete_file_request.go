package dto

type DeleteFileRequest struct {
	InvoiceID string `json:"invoice_id" binding:"required"`
}
