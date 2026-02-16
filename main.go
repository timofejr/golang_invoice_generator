package main

import (
	"os"
	"timofejr/invoice_generator/handlers"
	"timofejr/invoice_generator/middlewares"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.Static("/static", "./static")
	router.LoadHTMLGlob("templates/*")

	router.Use(middlewares.IPMiddleware())

	router.GET("/kond", handlers.KondPage)
	router.POST("/kond/upload_file", handlers.UploadApplicationFile)
	router.POST("/kond/create_invoice", handlers.CreateInvoice)

	router.GET("/bread", handlers.BreadPage)
	router.POST("/bread/upload_file", handlers.UploadApplicationFile)
	router.POST("/bread/create_invoice", handlers.CreateInvoice)

	router.POST("/delete_file", handlers.DeleteFile)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router.Run("0.0.0.0:" + port)
}
