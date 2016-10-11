package tracking_station

import (
	"errors"
	"os/exec"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

const (
	WORKER_COUNT = 3
	PHANTOMJS    = "vendor/bin/phantomjs"
	TRACKERJS    = "script/page_tracker.js"
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

func SetupWorkers(worker_count int) {
	inspectChannel = make(chan TrackingClients)

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
			LogError.Error(err)
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
