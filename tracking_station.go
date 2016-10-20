package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"strconv"

	ts "github.com/jonnung/tracking_station/tracking_station"
)

func LogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		output := fmt.Sprintf("%s %s %s %s %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			c.ContentType(),
			c.Request.Header.Get("User-Agenct"),
		)
		ts.LogAccess.Info(output)
	}
}

func ResourceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var clientId int
		var noParamErr error

		if clientId, noParamErr = strconv.Atoi(c.Param("client_id")); noParamErr != nil {
			if existClient := ts.LookupClient(clientId); len(existClient) > 0 {
				c.Set("clientId", clientId)
				c.Set("clientDocIds", existClient)
			}
		}

		c.Next()
	}
}

func main() {
	router := gin.New()

	ts.SetupLogger()
	ts.SetupDatabase()
	ts.SetupWorkers(3)

	router.Use(LogMiddleware())
	router.Use(ResourceMiddleware())

	router.GET("/clients", ts.ClientsHandler)
	router.GET("/clients/:client_id", ts.ClientOneHandler)
	router.DELETE("/clients/:client_id", ts.ClientDeleteHandler)
	router.POST("/clients/:client_id/tags", ts.SetClientTagsHandler)
	router.POST("/clients", ts.SetClientHandler)

	router.GET("/tracking/:client_id", ts.TrackingClientHandler)
	router.POST("/tracking", ts.SetTrackingHandler)

	router.Run(":8585")
}
