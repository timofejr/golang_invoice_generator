package dto

type CreateInvoiceResponse struct {
	Invoice [][]string `json:"invoice"`
	Title   string     `json:"title"`
}
