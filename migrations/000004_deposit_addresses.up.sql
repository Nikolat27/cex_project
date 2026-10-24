CREATE SEQUENCE deposit_addresses_index_seq;

CREATE TABLE deposit_addresses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    symbol TEXT NOT NULL,
    network TEXT NOT NULL,
    address TEXT NOT NULL UNIQUE,
    derivation_index BIGINT NOT NULL DEFAULT nextval('deposit_addresses_index_seq'),
    UNIQUE (user_id, symbol, network),
    UNIQUE (symbol, network, derivation_index)
);