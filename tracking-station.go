package main

import (
	"github.com/gin-gonic/gin"
	"sync"
	"log"
)

const (
	TRACKING_NEW    = "/job/new"
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


func setupWorker(worker_count int) {
	if worker_count < WORKER_COUNT {
		worker_count = WORKER_COUNT
	}

	for i := 0; i < worker_count; i++ {
		go startWorker()
	}
}


func startWorker() {
	for {
		cl := <-inspectQueue
		log.Println(cl)
		var wc sync.WaitGroup
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


func newTrackingHandler(c *gin.Context) {
	var rq RequestInspector
	if err := c.BindJSON(&rq); err != nil {
		log.Fatal(err)
	}

	requestQueue <- rq
}


func main() {
	router := gin.New()

	requestQueue = make(chan RequestInspector, QUEUE_SIZE)

	setupWorker(3)
	distributionWork()

	router.POST(TRACKING_NEW, newTrackingHandler)
	router.GET(TRACKING_STATUS, func(c *gin.Context){})
	router.GET(TRACKING_RESULT, func(c *gin.Context){})

	router.Run(":8585")
}
