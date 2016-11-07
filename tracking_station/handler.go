package tracking_station

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"encoding/json"

	"reflect"

	"github.com/HouzuoGuo/tiedot/db"
	"github.com/gin-gonic/gin"
)

type TrackingResponse struct {
	status_code int
	message     string
}

// lookupClient look up client ID in clients document
func LookupClient(clientId int) (int, error) {
	clientsCol := DB.Use("clients")

	expr := map[string]interface{}{
		"in":    []interface{}{"client_id"},
		"limit": 1,
	}

	clientDocIds := make(map[int]struct{})
	if err := db.Lookup(clientId, expr, clientsCol, &clientDocIds); err != nil {
		return 0, errors.New("Not found client")
	}

	var docId int
	for docId, _ = range clientDocIds {
	}
	return docId, nil
}

// isRunning check exist job running for client
func isRunning(clientId int) bool {
	var query interface{}
	var queryResult map[int]struct{}
	var docId int

	json.Unmarshal(
		[]byte(fmt.Sprintf(`{"n": [{"eq": "%d", "in": ["client_id"]}, {"eq": "running", "in": ["status"]}]}`, clientId)),
		&query,
	)

	trackingCol := DB.Use("tracking")
	queryResult = make(map[int]struct{})
	if err := db.EvalQuery(query, trackingCol, &queryResult); err != nil {
		panic(err)
	}

	for docId = range queryResult {
	}

	return docId != 0
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

	if _, err := LookupClient(cl.ClientId); err != nil {
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
	clientDocId := c.MustGet("clientDocId").(int)

	clientsCol := DB.Use("clients")
	if err := clientsCol.Delete(clientDocId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Could not delete from specified client ID"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "Deteled client"})
}

// SetClientTagsHandler add client url to tracking
func SetClientTagsHandler(c *gin.Context) {
	clientDocId := c.MustGet("clientDocId").(int)

	var rq struct {
		Tags `json:"tags" binding:"required"`
	}

	if err := c.BindJSON(&rq); err != nil {
		LogError.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request data format"})
		return
	}

	clientCol := DB.Use("clients")
	if readClient, err := clientCol.Read(clientDocId); err != nil {
		panic(err) // todo: panic 처리 필요
	} else {
		readClient["tags"] = rq.Tags
		err := clientCol.Update(clientDocId, readClient)

		if err != nil {
			panic(err) // todo: panic 처리 필요
		}

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}

}

// TrackingClientHandler return job currently running
func TrackingClientHandler(c *gin.Context) {
	clientId := c.MustGet("clientId").(int)

	var query interface{}
	json.Unmarshal([]byte(fmt.Sprintf(`[{"eq": "%d", "in": ["client_id"]}]`, clientId)), &query)
	//json.Unmarshal([]byte(fmt.Sprintf(`{"n": [{"eq": "%d", "in": ["client_id"]}, {"eq": "running", "in": ["status"]}]}`, clientId)), &query)

	queryResult := make(map[int]struct{})

	trackingCol := DB.Use("tracking")
	if err := db.EvalQuery(query, trackingCol, &queryResult); err != nil {
		panic(err)
	}

	var trackingDocid int
	for trackingDocid = range queryResult {
	}
	readBack, _ := trackingCol.Read(trackingDocid)

	c.JSON(http.StatusOK, gin.H{"message": "success", "result": readBack})
}

// SetTrackingHandler register new tracking job for client
func SetTrackingHandler(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case error:
				LogError.Error(r)
				c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprint(r)})
			case TrackingResponse:
				rsp := r.(TrackingResponse)
				c.JSON(rsp.status_code, rsp.message)
			}
		}
	}()

	var rq TrackingClients

	if err := c.BindJSON(&rq); err != nil {
		panic(err)
	}

	if docId, err := LookupClient(rq.ClientId); err != nil {
		panic(err)
	} else {
		if isRunning(rq.ClientId) {
			panic(TrackingResponse{
				status_code: http.StatusNotAcceptable,
				message:     "Already exists for running proccess",
			})
		}

		clientCol := DB.Use("clients")

		tcDoc, _ := clientCol.Read(docId)
		fmt.Println(reflect.TypeOf(tcDoc["tags"]))
		for _, tcDocTag := range tcDoc["tags"].([]interface{}) {
			vMap := reflect.ValueOf(tcDocTag)
			for _, rqTag := range rq.Tags {
				if vMap.MapIndex(reflect.ValueOf("device")).String() == rqTag.Device && vMap.MapIndex(reflect.ValueOf("part")).String() == rqTag.Device {
					rqTag.Url = vMap.MapIndex(reflect.ValueOf("url")).String()
				}
			}
		}
	}

	trackingCol := DB.Use("tracking")
	clientDocId, err := trackingCol.Insert(map[string]interface{}{
		"client_id": rq.ClientId,
		"status":    "wait",
		"tags":      rq.Tags,
	})

	if err != nil {
		panic(err)
	}

	rq.docId = clientDocId

	//go func() {
	//	inspectChannel <- rq
	//}()

	c.JSON(http.StatusCreated, gin.H{"message": "success", "result": clientDocId})
}
