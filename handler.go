package main

import (
	"encoding/json"
	log "github.com/asiainfoLDP/datahub/utils/clog"
	"github.com/asiainfoLDP/datahub_alarm/ds"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"time"
)

func rootHandler(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	JsonResult(rw, http.StatusOK, ds.ResultOK, "Service is running.", nil)
}

func sendMessageHandler(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Info("Begin send message")
	defer log.Info("End send message")

	msg := ds.AlarmEvent{}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("ReadAll err:", err)
		JsonResult(rw, http.StatusBadRequest, ds.ErrorReadAll, err.Error(), nil)
	}

	err = json.Unmarshal(body, &msg)
	if err != nil {
		log.Error("Unmarshal err:", err)
		JsonResult(rw, http.StatusBadRequest, ds.ErrorUnmarshal, err.Error(), nil)
	}

	event := ds.AlarmEvent{Sender: msg.Sender, Content: msg.Content, Send_time: time.Now()}
	b, err := json.Marshal(&event)
	if err != nil {
		log.Error("Marshal err:", err)
		JsonResult(rw, http.StatusBadRequest, ds.ErrorMarshal, err.Error(), nil)
	}

	_, _, err = AlarmMQ.SendSyncMessage(MQ_TOPIC_TO_ALARM, []byte("alarm to weixin"), b)
	if err != nil {
		log.Errorf("SendAsyncMessage error: %s", err.Error())
		JsonResult(rw, http.StatusBadRequest, ds.ErrorSendAsyncMessage, err.Error(), nil)
	}

	JsonResult(rw, http.StatusOK, ds.ResultOK, "OK.", nil)
}

func JsonResult(w http.ResponseWriter, statusCode int, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	result := ds.Result{Code: code, Msg: msg, Data: data}
	jsondata, err := json.Marshal(&result)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(getJsonBuildingErrorJson()))
	} else {
		w.WriteHeader(statusCode)
		w.Write(jsondata)
	}
}

func getJsonBuildingErrorJson() []byte {

	return []byte(log.Infof(`{"code": %d, "msg": %s}`, ds.ErrorMarshal, "Json building error"))

}
