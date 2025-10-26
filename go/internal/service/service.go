package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"veriChat/go/internal/cgobridge"
	"veriChat/go/internal/db"

	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config для сервиса
type Config struct {
	BatchSize    int
	BatchTimeout time.Duration // время ожидания перед flush
	LockTTL      time.Duration // TTL для redis lock
	RedisClient  *redis.Client
}

// MessageService управляет поступлением сообщений и батчингом
type MessageService struct {
	cfg         Config
	activeChats map[int64]time.Time // chatID -> lastActivity
	mu          sync.Mutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
	redis       *redis.Client
}

// NewMessageService создает сервис и стартует background flusher
func NewMessageService(cfg Config) *MessageService {
	s := &MessageService{
		cfg:         cfg,
		activeChats: make(map[int64]time.Time),
		stopCh:      make(chan struct{}),
		redis:       cfg.RedisClient,
	}
	s.wg.Add(1)
	go s.flusher()
	return s
}

// Shutdown остановить сервис
func (s *MessageService) Shutdown(ctx context.Context) {
	close(s.stopCh)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// SubmitMessage сохраняет сообщение, пушит его в очередь для батчей и возвращает message_id.
// Алгоритм:
// 1. Проверка idempotency в Redis.
// 2. Insert в messages (MySQL).
// 3. RPUSH message_id в Redis list chat:{chat_id}:pending_batch
// 4. mark active 
// 5. len >= batchSize -> flush.
func (s *MessageService) SubmitMessage(ctx context.Context, chatID, userID int64, payload []byte, idempKey string) (int64, error) {
	// 1) Idempotency
	if idempKey != "" {
		val, err := db.RedisClient.Get(ctx, "idemp:"+idempKey).Result()
		if err == nil && val != "" {
			// уже есть
			var existingID int64
			_, err := fmt.Sscanf(val, "%d", &existingID)
			if err == nil {
				return existingID, nil
			}
		}
		if err != nil && err != redis.Nil {
			// TODO: continue but log
		}
	}

	// 2) Insert into MySQL
	h := sha256.Sum256(payload)
	msg := &db.Message{
		ChatID:      chatID,
		UserID:      userID,
		Payload:     payload,
		PayloadHash: h[:],
		BatchID:     nil,
	}
	id, err := db.InsertMessage(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("InsertMessage failed: %w", err)
	}

	// 3) Set idempotency -> message id
	if idempKey != "" {
		_ = db.RedisClient.Set(ctx, "idemp:"+idempKey, fmt.Sprintf("%d", id), 24*time.Hour).Err()
	}

	// 4) Push to pending batch list
	if err := db.RedisClient.RPush(ctx, fmt.Sprintf("chat:%d:pending_batch", chatID), id).Err(); err != nil {
		// TODO: log error
		return id, fmt.Errorf("RPush failed: %w", err)
	}

	// 5) mark chat active
	s.mu.Lock()
	s.activeChats[chatID] = time.Now()
	s.mu.Unlock()

	// 6) quick check length and flush if threshold reached
	if l, _ := db.RedisClient.LLen(ctx, fmt.Sprintf("chat:%d:pending_batch", chatID)).Result(); l >= int64(s.cfg.BatchSize) {
		go func() {
			// TODO: process error
			_ = s.flushChat(context.Background(), chatID)
		}()
	}

	return id, nil
}

// GetLatestRoot получает root из Redis или из MySQL
func (s *MessageService) GetLatestRoot(ctx context.Context, chatID int64) ([]byte, error) {
	key := fmt.Sprintf("chat:%d:latest_root", chatID)
	b, err := db.RedisClient.Get(ctx, key).Bytes()
	if err == nil {
		return b, nil
	}
	if err != redis.Nil {
		// Redis error -> try fallback
	}

	// Fallback
	row := db.DB.QueryRowContext(ctx, `SELECT root_hash FROM merkle_batches WHERE chat_id = ? ORDER BY created_at DESC LIMIT 1`, chatID)
	var root []byte
	if err := row.Scan(&root); err != nil {
		return nil, fmt.Errorf("failed to get latest root: %w", err)
	}

	_ = db.RedisClient.Set(ctx, key, root, 0).Err()
	return root, nil
}

func (s *MessageService) flusher() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.BatchTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			var chatsToFlush []int64
			s.mu.Lock()
			for chatID, last := range s.activeChats {
				if now.Sub(last) >= s.cfg.BatchTimeout {
					chatsToFlush = append(chatsToFlush, chatID)
					delete(s.activeChats, chatID)
				}
			}
			s.mu.Unlock()

			for _, chatID := range chatsToFlush {
				_ = s.flushChat(context.Background(), chatID)
			}
		}
	}
}


// простой мьютекс 
func (s *MessageService) acquireLock(ctx context.Context, chatID int64) (bool, error) {
	key := fmt.Sprintf("lock:chat:%d", chatID)
	ok, err := db.RedisClient.SetNX(ctx, key, "1", s.cfg.LockTTL).Result()
	return ok && err == nil, err
}

func (s *MessageService) releaseLock(ctx context.Context, chatID int64) {
	key := fmt.Sprintf("lock:chat:%d", chatID)
	_ = db.RedisClient.Del(ctx, key).Err()
}



// Вызывается, когда batch заполнился. 
// 1. По ключу pending_batch`а берет последние BatchSize сообщений
// 2. Отправляет их payloads в c++ engine, который строит merkle tree и возвращает root
// 3. Сохраняет root в БД и проставляет batch_id для сообщений
// 4. И устанавливает latest root для чата
func (s *MessageService) flushChat(ctx context.Context, chatID int64) error {
	ok, err := s.acquireLock(ctx, chatID)
	if err != nil {
		return fmt.Errorf("acquire lock error: %w", err)
	}
	if !ok {
		return nil
	}
	defer s.releaseLock(ctx, chatID)

	key := fmt.Sprintf("chat:%d:pending_batch", chatID)
	ids := make([]int64, 0, s.cfg.BatchSize)
	for i := 0; i < s.cfg.BatchSize; i++ {
		val, err := db.RedisClient.LPop(ctx, key).Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			// TODO: process error
			return fmt.Errorf("LPop error: %w", err)
		}
		var id int64
		fmt.Sscanf(val, "%d", &id)
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}

	payloads, _, err := db.GetMessagePayloads(ctx, ids)
	if err != nil {
		return fmt.Errorf("GetMessagePayloads failed: %w", err)
	}

	// Prepare [][]byte for bridge
	msgs := make([][]byte, len(payloads))
	for i := range payloads {
		msgs[i] = payloads[i]
	}

	root, err := cgobridge.MerkleRoot(msgs)
	if err != nil {
		// TODO: process error
		for _, id := range ids {
			_ = db.RedisClient.LPush(ctx, key, id).Err()
		}
		return fmt.Errorf("MerkleRoot failed: %w", err)
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		// push back to redis
		for _, id := range ids {
			_ = db.RedisClient.LPush(ctx, key, id).Err()
		}
		return fmt.Errorf("BeginTx failed: %w", err)
	}

	batch := &db.MerkleBatch{
		ChatID:        chatID,
		RootHash:      root,
		FromMessageID: ids[0],
		ToMessageID:   ids[len(ids)-1],
	}
	batchID, err := db.InsertMerkleBatchTx(ctx, tx, batch)
	if err != nil {
		_ = tx.Rollback()
		for _, id := range ids {
			_ = db.RedisClient.LPush(ctx, key, id).Err()
		}
		return fmt.Errorf("InsertMerkleBatchTx failed: %w", err)
	}

	if err := db.UpdateMessagesBatchIDTx(ctx, tx, ids, batchID); err != nil {
		_ = tx.Rollback()
		for _, id := range ids {
			_ = db.RedisClient.LPush(ctx, key, id).Err()
		}
		return fmt.Errorf("UpdateMessagesBatchIDTx failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		for _, id := range ids {
			_ = db.RedisClient.LPush(ctx, key, id).Err()
		}
		return fmt.Errorf("tx commit failed: %w", err)
	}

	if err := db.RedisClient.Set(ctx, fmt.Sprintf("chat:%d:latest_root", chatID), root, 0).Err(); err != nil {
		// TODO: process error
	}

	return nil
}
