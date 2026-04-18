package broker

import (
	"github.com/gin-gonic/gin"
)

func Start() {
	// Register routes and run the server
	router := gin.Default()
	router.POST("/message", appendMessages)
	if err := router.Run("localhost:8080"); err != nil {
		panic(err)
	}
}
