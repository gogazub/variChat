package main

import (
	"encoding/json"
	"log"
	"net/http"
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
  http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) })
  http.HandleFunc("/orders", createOrder)
  err := http.ListenAndServe(":8081", nil)
  if err != nil{
	log.Printf(err.Error())
  }
  
}
