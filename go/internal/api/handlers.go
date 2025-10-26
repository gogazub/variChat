package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"veriChat/go/internal/cgobridge"
	"veriChat/go/internal/service"
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

type postMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	UserID    int64  `json:"user_id"`
	Payload   string `json:"payload"`
	IdempKey  string `json:"idempotency_key,omitempty"`
}

type postMessageResponse struct {
	MessageID int64  `json:"message_id"`
	Status    string `json:"status"`
}

func makePostMessageHandler(svc *service.MessageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req postMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid input: %v", err), http.StatusBadRequest)
			return
		}

		id, err := svc.SubmitMessage(r.Context(), req.ChatID, req.UserID, []byte(req.Payload), req.IdempKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed: %v", err), http.StatusInternalServerError)
			return
		}

		resp := postMessageResponse{
			MessageID: id,
			Status:    "accepted",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
