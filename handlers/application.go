package handlers

import (
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"timofejr/invoice_generator/handlers/dto"
	"timofejr/invoice_generator/spreadsheets"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func logRequestError(c *gin.Context, operation string, err error) {
	log.Printf(
		"error operation=%s method=%s path=%s ip=%s err=%v",
		operation,
		c.Request.Method,
		c.FullPath(),
		c.ClientIP(),
		err,
	)
}

func saveUploadedFileNoChmod(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func UploadApplicationFile(c *gin.Context) {
	var req dto.UploadApplicationRequest
	if err := c.ShouldBind(&req); err != nil {
		logRequestError(c, "upload.bind_request", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		logRequestError(c, "upload.form_file", err)
		c.JSON(400, gin.H{"error": "файл не передан"})
		return
	}

	tempFileExtension := strings.Split(file.Filename, ".")

	if file_extension := tempFileExtension[len(tempFileExtension)-1]; file_extension != "xlsx" {
		log.Printf(
			"error operation=upload.invalid_extension method=%s path=%s ip=%s filename=%q extension=%q",
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			file.Filename,
			file_extension,
		)
		c.JSON(400, gin.H{"error": "файл не верного формата"})
		return
	}

	fileId := uuid.NewString()

	err = saveUploadedFileNoChmod(file, "./uploads/"+fileId+".xlsx")
	if err != nil {
		log.Printf(
			"error operation=upload.save_file method=%s path=%s ip=%s file_id=%s filename=%q err=%v",
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			fileId,
			file.Filename,
			err,
		)
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
		log.Printf(
			"error operation=upload.unknown_application_type method=%s path=%s ip=%s filename=%q parsed_type=%q",
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			file.Filename,
			applicationType,
		)
		c.JSON(500, gin.H{"error": "неизвестный файл"})
		return
	}

	contragents, err := spreadsheets.GetContragents(fileId, applicationType)
	if err != nil {
		log.Printf(
			"error operation=upload.get_contragents method=%s path=%s ip=%s file_id=%s application_type=%s err=%v",
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			fileId,
			applicationType,
			err,
		)
		c.JSON(500, gin.H{"error": "не удалось получить список контр-агентов"})
		return
	}

	worksheets, err := spreadsheets.GetWorksheets(req.ManufactureType)
	if err != nil {
		log.Printf(
			"error operation=upload.get_worksheets method=%s path=%s ip=%s manufacture_type=%s err=%v",
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			req.ManufactureType,
			err,
		)
		c.JSON(500, gin.H{"error": "не удалось получить список листов"})
		return
	}

	daytimes, err := spreadsheets.GetDaytimes(applicationType)
	if err != nil {
		log.Printf(
			"error operation=upload.get_daytimes method=%s path=%s ip=%s application_type=%s err=%v",
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			applicationType,
			err,
		)
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
		logRequestError(c, "invoice.bind_request", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoice, err := spreadsheets.GetInvoiceNew(req.InvoiceID, req.ContrAgent, req.Daytime, req.Worksheets, req.ApplicationType)

	if err != nil {
		log.Printf(
			"error operation=invoice.create method=%s path=%s ip=%s invoice_id=%s contragent=%q application_type=%s worksheets=%d err=%v",
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			req.InvoiceID,
			req.ContrAgent,
			req.ApplicationType,
			len(req.Worksheets),
			err,
		)
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
		logRequestError(c, "delete_file.bind_request", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := spreadsheets.DeleteFile(req.InvoiceID)

	if err != nil {
		log.Printf(
			"error operation=delete_file.remove method=%s path=%s ip=%s invoice_id=%s err=%v",
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			req.InvoiceID,
			err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка удаления файла"})
		return
	}

	c.JSON(200, gin.H{})
}
