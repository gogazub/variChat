package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func InsertMessage(ctx context.Context, msg *Message) (int64, error) {
    res, err := DB.ExecContext(ctx,
        `INSERT INTO messages (chat_id, user_id, payload, payload_hash, batch_id)
         VALUES (?, ?, ?, ?, ?)`,
        msg.ChatID, msg.UserID, msg.Payload, msg.PayloadHash, msg.BatchID,
    )
    if err != nil {
        return 0, fmt.Errorf("insert message failed: %w", err)
    }
    return res.LastInsertId()
}

func InsertMerkleBatch(ctx context.Context, batch *MerkleBatch) (int64, error) {
    res, err := DB.ExecContext(ctx,
        `INSERT INTO merkle_batches (chat_id, root_hash, from_message_id, to_message_id)
         VALUES (?, ?, ?, ?)`,
        batch.ChatID, batch.RootHash, batch.FromMessageID, batch.ToMessageID,
    )
    if err != nil {
        return 0, fmt.Errorf("insert merkle batch failed: %w", err)
    }
    return res.LastInsertId()
}


// GetMessagePayloads возвращает payloads и их хешы по списку message_id.
// Возвращает slice в том порядке, в котором были переданы ids (если id не найден nil в соответствующей позиции).
func GetMessagePayloads(ctx context.Context, ids []int64) ([][]byte, [][]byte, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT message_id, payload, payload_hash FROM messages WHERE message_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("GetMessagePayloads query: %w", err)
	}
	defer rows.Close()

	// Собираем в мапу, потом распакуем в порядок ids
	payloadMap := make(map[int64][]byte)
	hashMap := make(map[int64][]byte)
	for rows.Next() {
		var id int64
		var payload []byte
		var hash []byte
		if err := rows.Scan(&id, &payload, &hash); err != nil {
			return nil, nil, fmt.Errorf("GetMessagePayloads scan: %w", err)
		}
		payloadMap[id] = payload
		hashMap[id] = hash
	}
	payloads := make([][]byte, len(ids))
	hashes := make([][]byte, len(ids))
	for i, id := range ids {
		payloads[i] = payloadMap[id]
		hashes[i] = hashMap[id]
	}
	return payloads, hashes, nil
}

// InsertMerkleBatchTx вставляет запись merkle_batches в рамках tx и возвращает batch_id.
func InsertMerkleBatchTx(ctx context.Context, tx *sql.Tx, batch *MerkleBatch) (int64, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO merkle_batches (chat_id, root_hash, from_message_id, to_message_id)
         VALUES (?, ?, ?, ?)`,
		batch.ChatID, batch.RootHash, batch.FromMessageID, batch.ToMessageID,
	)
	if err != nil {
		return 0, fmt.Errorf("InsertMerkleBatchTx failed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("InsertMerkleBatchTx LastInsertId: %w", err)
	}
	return id, nil
}

// UpdateMessagesBatchIDTx присваивает batch_id для message_ids в рамках tx.
func UpdateMessagesBatchIDTx(ctx context.Context, tx *sql.Tx, messageIDs []int64, batchID int64) error {
	if len(messageIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs)+1)
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	args[len(messageIDs)] = batchID
	query := fmt.Sprintf(`UPDATE messages SET batch_id = ? WHERE message_id IN (%s)`, strings.Join(placeholders, ","))

	args2 := make([]interface{}, 0, len(messageIDs)+1)
	args2 = append(args2, batchID)
	for _, id := range messageIDs {
		args2 = append(args2, id)
	}
	_, err := tx.ExecContext(ctx, query, args2...)
	if err != nil {
		return fmt.Errorf("UpdateMessagesBatchIDTx failed: %w", err)
	}
	return nil
}