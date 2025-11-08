package main

import (
	"log"

	gintrace "github.com/DataDog/dd-trace-go/contrib/gin-gonic/gin/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/gin-gonic/gin"
)

func main() {
	// Start Datadog tracer
	tracer.Start(
		tracer.WithService("go-api-demo"),
		tracer.WithEnv("dev"),
		tracer.WithServiceVersion("1.0.0"),
	)
	defer tracer.Stop()

	// Create a Gin router
	r := gin.Default()

	// Add DataDog tracing middleware
	r.Use(gintrace.Middleware("go-api-demo"))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	log.Println("Server running on :8080")
	r.Run(":8080")
}
