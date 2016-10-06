package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"net/http"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/PuerkitoBio/goquery"
	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

const (
	WORKER_COUNT  = 3
	DB_COLLECTION = "TrackingClients"
	PHANTOMJS     = "./phantomjs"
	TRACKERJS     = "page_tracker.js"
)

type TrackingRequest struct {
	Clients []TrackingClients `json:"clients" binding:"required"`
}

type TrackingClients struct {
	ClientId string `json:"id" binding:"required"`
	Tags     []Tag  `json:"tags" binding:"required"`
	docId    int
}

type Tag struct {
	Device string `json:"device"`
	Part   string `json:"part"`
	Url    string `json:"url" binding:"required"`
	result chan [2]interface{}
}

var inspectChannel chan TrackingClients
var tsdb *db.Col

var logAccess *logrus.Logger
var logError *logrus.Logger

func setupWorkers(worker_count int) {
	if worker_count < WORKER_COUNT {
		worker_count = WORKER_COUNT
	}

	for i := 0; i < worker_count; i++ {
		go mainWorker()
	}
}

func mainWorker() {
	for {
		var wc sync.WaitGroup

		cl := <-inspectChannel

		resultChannel := make(chan [2]interface{})
		for _, tag := range cl.Tags {
			tag.result = resultChannel
			wc.Add(1)
			go subWorker(tag, &wc)
		}

		go func() {
			wc.Wait()
			close(resultChannel)
		}()

		for result := range resultChannel {
			if result[1].(bool) {

			} else {

			}

			// 결과 저장
			// client_id, device, part, url, status, result, latest_update_datetime
		}

	}
}

func subWorker(tag Tag, wc *sync.WaitGroup) {
	defer func() {
		if r := recover(); r != nil {
			err, _ := r.(error)
			logError.Error(err)
			tag.result <- [2]interface{}{err.Error(), false}
		}

		wc.Done()
	}()

	cmd := exec.Command(PHANTOMJS, TRACKERJS, tag.Url)
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	doc, _ := goquery.NewDocumentFromReader(stdout)
	finder := doc.Find("div#wp_tg_cts iframe")

	if len(finder.Nodes) > 0 {
		finder.Each(func(_ int, s *goquery.Selection) {
			trackingResult, exist := s.Attr("src")
			if exist == false {
				err := errors.New("Not render tracking tag")
				panic(err)
			}

			tag.result <- [2]interface{}{trackingResult, true}
		})
	} else {
		err := errors.New("Nothing tracking tag")
		panic(err)
	}
}

// Register new client information
func setNewClientHandler(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, _ := r.(error)
			logError.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Can not create new client"})
		}
	}()

	var cl struct {
		client_id   int
		client_name string
	}

	// TODO: 이부분을 미들웨어에서 공통적으로 처리할 수 없을까?
	if err := c.BindJSON(&cl); err != nil {
		logError.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request data format"})
	}

	expr := map[string]interface{}{"in": []interface{}{"client_id"}, "limit": 1}
	queryResult := make(map[int]struct{})
	if err := db.Lookup(cl.client_id, expr, tsdb, &queryResult); err != nil {
		panic(err)
	}

	if len(queryResult) > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "Duplicated client"})
	}

	_, err := tsdb.Insert(map[string]interface{}{"client_id": cl.client_id, "client_name": cl.client_name})
	if err != nil {
		panic(err)
	}
}

func setTrackingHandler(c *gin.Context) {
	var rq TrackingRequest
	if err := c.BindJSON(&rq); err != nil {
		logError.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request data format"})
	}

	for _, cl := range rq.Clients {
		// DB에 존재하는 클라이언트 인지 확인, 없으면 에러로 판단
		// 실행중인 클라이언트 인지 확인
		// latest_start_datetime 갱신
		go func() {
			inspectChannel <- cl
		}()
	}
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

	logAccess.Out, _ = os.OpenFile("log/tracking-station.access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	logError.Out, _ = os.OpenFile("log/tracking-station.error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
}

func setupDatabase() {
	odb, err := db.OpenDB("tsdb")
	if err != nil {
		panic(err)
	}

	existCollection := (func(cols []string) bool {
		for _, col := range cols {
			if col == DB_COLLECTION {
				return true
			}
		}
		return false
	})(odb.AllCols())

	if existCollection == false {
		if err := odb.Create(DB_COLLECTION); err != nil {
			panic(err)
		}
	}

	tsdb = odb.Use(DB_COLLECTION)

	// Create index
	tsdb.Index([]string{"client_id"})
}

func main() {
	router := gin.New()

	inspectChannel = make(chan TrackingClients)

	setupLogger()
	setupDatabase()
	setupWorkers(3)

	router.Use(LogMiddleware())

	router.GET("/clients", func(c *gin.Context) {})
	router.GET("/clients/:client_id", func(c *gin.Context) {})
	router.POST("/clients", setNewClientHandler)

	router.GET("/tracking/:client_id", func(c *gin.Context) {})
	router.POST("/tracking", setTrackingHandler)

	router.Run(":8585")
}
