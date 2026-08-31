package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())
	engine.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Ranvex API is healthy", "data": gin.H{"service": "ranvex-api"}})
	})

	log.Println("Ranvex API listening on :8080")
	if err := engine.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
