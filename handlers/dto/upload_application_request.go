package dto

type UploadApplicationRequest struct {
	ManufactureType string `form:"manufacture_type" binding:"required,oneof=bread kond"`
}