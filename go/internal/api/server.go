package api

import (
	"fmt"
	"net/http"
)

func NewServer(addr string) *http.Server {
    mux := http.NewServeMux()
    mux.HandleFunc("/merkle", PostMerkleHandler)

    srv := &http.Server{
        Addr:    addr,
        Handler: mux,
    }

    fmt.Printf("API server listening on %s\n", addr)
    return srv
}
