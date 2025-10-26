package db

import "time"

type Message struct {
    MessageID   int64
    ChatID      int64
    UserID      int64
    Payload     []byte
    PayloadHash []byte
    CreatedAt   time.Time
    BatchID     *int64
}

type MerkleBatch struct {
    BatchID       int64
    ChatID        int64
    RootHash      []byte
    FromMessageID int64
    ToMessageID   int64
    CreatedAt     time.Time
}
