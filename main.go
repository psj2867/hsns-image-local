package main

import (
	"net/http"

	localhsns "github.com/psj2867/hsns-image-local/local_hsns"
)

func main() {
	http.ListenAndServe("0.0.0.0:8081",
		localhsns.Default("./temp"))
}
