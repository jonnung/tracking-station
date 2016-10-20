package tracking_station

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"encoding/json"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/gin-gonic/gin"
)

// lookupClient look up client ID in clients document
func LookupClient(clientId int) map[int]struct{} {
	clientsCol := DB.Use("clients")

	expr := map[string]interface{}{
		"in":    []interface{}{"client_id"},
		"limit": 1,
	}

	lookup := make(map[int]struct{})
	if err := db.Lookup(clientId, expr, clientsCol, &lookup); err != nil {
		panic(err)
	}

	return lookup
}

// SetClientHandler register new client information
func SetClientHandler(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, _ := r.(error)
			LogError.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Can not create new client"})
		}
	}()

	var cl struct {
		ClientId   int    `json:"client_id" binding:"required"`
		ClientName string `json:"client_name" binding:"required"`
	}

	// TODO: 이부분을 미들웨어에서 공통적으로 처리할 수 없을까?
	if err := c.BindJSON(&cl); err != nil {
		LogError.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request data format"})
		return
	}

	if existClient := LookupClient(cl.ClientId); len(existClient) > 0 {
		LogError.Error(errors.New(fmt.Sprintf("Already exist client ID (%d)", cl.ClientId)))
		c.JSON(http.StatusConflict, gin.H{"message": "Duplicated client"})
		return
	}

	newClient := map[string]interface{}{
		"client_id":     cl.ClientId,
		"client_name":   cl.ClientName,
		"register_date": fmt.Sprint(time.Now().UTC().Format("2006-01-02 15:04:05")),
	}

	clientsCol := DB.Use("clients")
	_, err := clientsCol.Insert(newClient)
	if err != nil {
		panic(err)
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Created new client"})
}

// ClientsHandler fetch all clients
func ClientsHandler(c *gin.Context) {
	type client map[string]interface{}

	rc := map[string]interface{}{
		"message": "success",
		"clients": make([]client, 0),
	}

	clientsCol := DB.Use("clients")

	clientsCol.ForEachDoc(func(docId int, docContent []byte) (willMoveOn bool) {
		var dat client
		if err := json.Unmarshal(docContent, &dat); err != nil {
			panic(err)
		}

		rc["clients"] = append(rc["clients"].([]client), dat)
		return true
	})

	if len(rc["clients"].([]client)) > 0 {
		c.JSON(http.StatusOK, rc)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"message": "Empty clients"})
	}
}

// ClientOneHandler fetch client specified for client ID
func ClientOneHandler(c *gin.Context) {
	clientId := c.MustGet("clientId").(int)

	query := map[string]interface{}{
		"eq":    clientId,
		"in":    []interface{}{"client_id"},
		"limit": 1,
	}
	queryResult := make(map[int]struct{})
	clientsCol := DB.Use("clients")
	if err := db.EvalQuery(query, clientsCol, &queryResult); err != nil {
		panic(err) // todo: panic 처리 필요
	}

	if len(queryResult) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found client"})
	}

	for docId := range queryResult {
		if readBack, err := clientsCol.Read(docId); err != nil {
			panic(err) // todo: panic 처리 필요
		} else if len(readBack) > 0 {
			c.JSON(http.StatusOK, gin.H{"message": "success", "result": readBack})
		}
	}
}

// ClientDeleteHandler permanently remove a client
func ClientDeleteHandler(c *gin.Context) {
	clientDocIds := c.MustGet("clientDocIds").(map[int]struct{})

	clientsCol := DB.Use("clients")
	for docId, _ := range clientDocIds {
		if err := clientsCol.Delete(docId); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Could not delete from specified client ID"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"message": "Deteled client"})
	}
}

// SetClientTagsHandler add client url to tracking
func SetClientTagsHandler(c *gin.Context) {
	clientDocIds := c.MustGet("clientDocIds").(map[int]struct{})

	var tags Tags
	if err := c.BindJSON(&tags); err != nil {
		LogError.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request data format"})
	}

	clientCol := DB.Use("clients")
	for docId, _ := range clientDocIds {
		if readClient, err := clientCol.Read(docId); err != nil {
			panic(err) // todo: panic 처리 필요
		} else {
			readClient["tags"] = tags
			err := clientCol.Update(docId, readClient)

			if err != nil {
				panic(err) // todo: panic 처리 필요
			}

			c.JSON(http.StatusOK, gin.H{"message": "success"})
		}
	}

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
