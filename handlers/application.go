package handlers

import (
	"net/http"
	"strings"

	"timofejr/invoice_generator/handlers/dto"
	"timofejr/invoice_generator/spreadsheets"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UploadApplicationFile(c *gin.Context) {
	var req dto.UploadApplicationRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "файл не передан"})
		return
	}

	tempFileExtension := strings.Split(file.Filename, ".")

	if file_extension := tempFileExtension[len(tempFileExtension)-1]; file_extension != "xlsx" {
		c.JSON(400, gin.H{"error": "файл не верного формата"})
		return
	}

	fileId := uuid.NewString()

	err = c.SaveUploadedFile(file, "./uploads/"+fileId+".xlsx")
	if err != nil {
		c.JSON(500, gin.H{"error": "не удалось сохранить файл"})
		return
	}

	applicationType := strings.Split(file.Filename, " ")[0]

	switch applicationType {
	case "Магаз.":
		applicationType = "store"
	case "ОПТ":
		applicationType = "wholesale"
	default:
		c.JSON(500, gin.H{"error": "неизвестный файл"})
		return
	}

	contragents, err := spreadsheets.GetContragents(fileId, applicationType)
	if err != nil {
		c.JSON(500, gin.H{"error": "не удалось получить список контр-агентов"})
		return
	}

	worksheets, err := spreadsheets.GetWorksheets(req.ManufactureType)
	if err != nil {
		c.JSON(500, gin.H{"error": "не удалось получить список листов"})
		return
	}

	daytimes, err := spreadsheets.GetDaytimes(applicationType)
	if err != nil {
		c.JSON(500, gin.H{"error": "не удалось получить времен заявок"})
		return
	}

	c.JSON(200, dto.UploadApplicationResponse{
		ID:              fileId,
		ContrAgents:     contragents,
		Daytimes:        daytimes,
		WorkSheets:      worksheets,
		ApplicationType: applicationType,
	})
}

func CreateInvoice(c *gin.Context) {
	var req dto.CreateInvoiceRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoice, err := spreadsheets.GetInvoiceNew(req.InvoiceID, req.ContrAgent, req.Daytime, req.Worksheets, req.ApplicationType)

	if err != nil {
		c.JSON(500, gin.H{"error": "не удалось создать накладную"})
		return
	}

	var title strings.Builder
	title.WriteString(req.ContrAgent + ": ")

	for i, worksheet := range req.Worksheets {
		if req.Daytime != nil && *req.Daytime != "" {
			switch *req.Daytime {
			case "Утро":
				title.WriteString(worksheet + " У")
			case "День":
				title.WriteString(worksheet + " Д")
			case "Дозавоз":
				title.WriteString(worksheet + " Доз.")
			default:
				title.WriteString(worksheet)
			}
		} else {
			title.WriteString(worksheet)
		}

		if i == len(req.Worksheets)-1 {
			title.WriteString(".")
		} else {
			title.WriteString(", ")
		}
	}

	c.JSON(200, dto.CreateInvoiceResponse{
		Invoice: invoice,
		Title:   title.String(),
	})
}

func DeleteFile(c *gin.Context) {
	var req dto.DeleteFileRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := spreadsheets.DeleteFile(req.InvoiceID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{})
}
