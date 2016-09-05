package main

import (
	"encoding/json"
	log "github.com/asiainfoLDP/datahub/utils/clog"
	"github.com/asiainfoLDP/datahub_alarm/ds"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func rootHandler(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	JsonResult(rw, http.StatusOK, ds.ResultOK, "Service is running.", nil)
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
