package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type ApiReqPayload struct {
	Phone    string
	Msg      string
	Pattern  string
	Callback string
}

type httpResponse struct {
	Status string
	Msg    string
	Code   int
}

func SendTextMsg(w http.ResponseWriter, r *http.Request) {
	var p ApiReqPayload
	_ = json.NewDecoder(r.Body).Decode(&p)
	resp := httpResponse{
		Status: "OK",
		Msg:    "",
		Code:   0,
	}
	sendTask := sendTextTask{
		ToPhone: p.Phone,
		Msg:     p.Msg,
	}
	err := pushTaskToQ(sendTask)
	if err != nil {
		resp.Status = "Failed"
		resp.Msg = err.Error()
		resp.Code = 4002
		json.NewEncoder(w).Encode(resp)
		return
	}
	json.NewEncoder(w).Encode(resp)
	return
}

func apiWorker() {
	router := mux.NewRouter()
	router.HandleFunc("/", SendTextMsg).Methods("POST")
	//router.HandleFunc("/oacard", SendMsg).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))
}

func main() {
	go tokenWorker()
	go sendTextWorker()
	apiWorker()
}
