package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func rootHandler(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	fmt.Println("here...")
}
