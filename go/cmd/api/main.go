package main

import "C"
import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"order-orchestration/go/internal/cgobridge"
	"sync"
)

var (
  mu sync.Mutex
  store = map[string]map[string]interface{}{}
)

func createOrder(w http.ResponseWriter, r *http.Request) {
  var o map[string]interface{}
  json.NewDecoder(r.Body).Decode(&o)
  id := "o1" // временно фикс
  mu.Lock(); store[id]=o; mu.Unlock()
  w.WriteHeader(201)
  json.NewEncoder(w).Encode(map[string]string{"order_id": id})
}

func main() {
	result := cgobridge.Hello()
	fmt.Println(result)
  http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) })
  http.HandleFunc("/orders", createOrder)
  err := http.ListenAndServe(":8081", nil)
  if err != nil{
	log.Printf(err.Error())
  }
  
}
