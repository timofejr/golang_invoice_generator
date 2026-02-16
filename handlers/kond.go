package handlers

import (
	"github.com/gin-gonic/gin"
)

func KondPage(c *gin.Context) {
	c.HTML(200, "kond.html", nil)
}