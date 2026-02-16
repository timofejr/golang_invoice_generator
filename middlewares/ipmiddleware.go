package middlewares

import (
	"net"
	"net/http"
	"log"
	
	"github.com/gin-gonic/gin"
)

func IPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		remoteIP := net.ParseIP(c.Request.RemoteAddr)
		if remoteIP == nil {
			host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
			if err == nil {
				remoteIP = net.ParseIP(host)
			}
		}

		if remoteIP == nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		
		log.Print(remoteIP)
		
			
		c.Next()
	}
}