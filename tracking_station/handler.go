package tracking_station

import (
	"fmt"
	"net/http"
	"time"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/gin-gonic/gin"
)

func SetClientTagsHandler(c *gin.Context) {

}

// Register new client information
func SetClientHandler(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, _ := r.(error)
			LogError.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Can not create new client"})
		}
	}()

	var cl struct {
		client_id   int
		client_name string
	}

	// TODO: 이부분을 미들웨어에서 공통적으로 처리할 수 없을까?
	if err := c.BindJSON(&cl); err != nil {
		LogError.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request data format"})
	}

	clientsCol := DB.Use("clients")

	expr := map[string]interface{}{"in": []interface{}{"client_id"}, "limit": 1}
	queryResult := make(map[int]struct{})
	if err := db.Lookup(cl.client_id, expr, clientsCol, &queryResult); err != nil {
		panic(err)
	}

	if len(queryResult) > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "Duplicated client"})
	}

	newClient := map[string]interface{}{
		"client_id":     cl.client_id,
		"client_name":   cl.client_name,
		"register_date": fmt.Sprint(time.Now().UTC().Format("2006-01-02 15:04:05")),
	}

	_, err := clientsCol.Insert(newClient)
	if err != nil {
		panic(err)
	}
}

func ClientsHandler(c *gin.Context) {

}

func ClientOneHandler(c *gin.Context) {

}

func TrackingClientHandler(c *gin.Context) {

}

func SetTrackingHandler(c *gin.Context) {
	var rq TrackingRequest
	if err := c.BindJSON(&rq); err != nil {
		LogError.Error(err)
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
