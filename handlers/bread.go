package handlers

import (
	"github.com/gin-gonic/gin"
)

func BreadPage(c *gin.Context) {
	c.HTML(200, "bread.html", nil)
}
