package broker

import (
	"github.com/gin-gonic/gin"
)

func Start() {
	router := gin.Default()
	router.POST("/message", appendMessages)

	router.Run("localhost:8080")
}
