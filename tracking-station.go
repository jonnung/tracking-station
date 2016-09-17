package main

import (
	"github.com/gin-gonic/gin"
	"sync"
	"log"
	"github.com/Sirupsen/logrus"
	"os"
	"fmt"
)

const (
	TRACKING_ADD = "/job/new"
	TRACKING_STATUS = "/job/status"
	TRACKING_RESULT = "/job/result"
	WORKER_COUNT = 3
	QUEUE_SIZE = 5
)

type RequestInspector struct {
	Clients []TrackingClients `json:"clients" binding:"required"`
}

type TrackingClients struct {
	ClientId string `json:"id" binding:"required"`
	Tags []Tag `json:"tags" binding:"required"`
}

type Tag struct {
	Part string `json:"part" binding:"required"`
	Url string `json:"url" binding:"required"`
}

var requestQueue chan RequestInspector
var inspectQueue chan TrackingClients

var logAccess *logrus.Logger
var logError *logrus.Logger

func setupWorkers(worker_count int) {
	if worker_count < WORKER_COUNT {
		worker_count = WORKER_COUNT
	}

	for i := 0; i < worker_count; i++ {
		go startWorker()
	}
}


func startWorker() {
	for {
		var wc sync.WaitGroup

		cl := <-inspectQueue

		for _, tag := range cl.Tags {
			wc.Add(1)
			go startSubWorker(tag.Part, tag.Url, &wc)
		}
		wc.Wait()
	}
}


func startSubWorker(part string, url string, wc *sync.WaitGroup) {
	wc.Done()
}


func distributionWork() {
	go func() {
		rq := <-requestQueue
		for _, cl := range rq.Clients {
			// 클라이언트 정보 저장
			// 요청 시간 저장
			inspectQueue <- cl
		}
	}()
}


func addTrackingHandler(c *gin.Context) {
	var rq RequestInspector
	if err := c.BindJSON(&rq); err != nil {
		log.Fatal(err)
	}

	requestQueue <- rq
}

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
		logAccess.Info(output)
	}
}

func setupLogger() {
	logAccess = logrus.New()
	logError = logrus.New()

	logAccess.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     false,
		FullTimestamp:   true,
	}
	logError.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     false,
		FullTimestamp:   true,
	}

	logAccess.Level = logrus.InfoLevel
	logError.Level = logrus.ErrorLevel

	logAccess.Out, _ = os.OpenFile("log/tracking-station.access.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0755)
	logError.Out, _ = os.OpenFile("log/tracking-station.error.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0755)
}

func main() {
	router := gin.New()

	router.Use(LogMiddleware())

	requestQueue = make(chan RequestInspector, QUEUE_SIZE)

	setupLogger()
	setupWorkers(3)
	distributionWork()

	router.POST(TRACKING_ADD, addTrackingHandler)
	router.GET(TRACKING_STATUS, func(c *gin.Context){})
	router.GET(TRACKING_RESULT, func(c *gin.Context){})

	router.Run(":8585")
}
