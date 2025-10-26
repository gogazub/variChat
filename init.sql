
CREATE TABLE messages (
    message_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    payload BLOB NOT NULL,
    payload_hash BINARY(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    batch_id BIGINT NULL,
    INDEX idx_chat_time(chat_id, created_at)
);

CREATE TABLE merkle_batches (
    batch_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    root_hash BINARY(32) NOT NULL,
    from_message_id BIGINT NOT NULL,
    to_message_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_chat_range(chat_id, from_message_id, to_message_id),
    INDEX idx_chat_created(chat_id, created_at)
);