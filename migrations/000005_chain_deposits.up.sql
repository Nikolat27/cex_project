CREATE TABLE chain_deposits (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    symbol TEXT NOT NULL,
    network TEXT NOT NULL,
    address TEXT NOT NULL,
    tx_hash TEXT NOT NULL,
    amount TEXT NOT NULL,
    confirmations INT NOT NULL DEFAULT 0,
    credited BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (network, tx_hash)
);