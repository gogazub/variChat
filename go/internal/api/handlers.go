package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"order-orchestration/go/internal/cgobridge"
)

// PostMerkleHandler обрабатывает POST /merkle
func PostMerkleHandler(w http.ResponseWriter, r *http.Request) {
    var msgs []string
    if err := json.NewDecoder(r.Body).Decode(&msgs); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprintf(w, "invalid input: %v", err)
        return
    }

	// TODO: вынести лишнюю логику в service
	
    // Преобразуем в [][]byte
    byteMsgs := make([][]byte, len(msgs))
    for i, m := range msgs {
        byteMsgs[i] = []byte(m)
    }

    // Вызов C++ модуля через bridge
    root, err := cgobridge.MerkleRoot(byteMsgs)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        fmt.Fprintf(w, "error: %v", err)
        return
    }

    // Ответ JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "merkle_root": fmt.Sprintf("%x", root),
    })
}
