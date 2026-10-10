CREATE TABLE users (
    id BIGINT PRIMARY KEY
);

CREATE TABLE markets(
    symbol TEXT PRIMARY KEY,
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    CHECK (base_asset <> quote_asset)
);

CREATE TABLE balances (
    user_id BIGINT NOT NULL REFERENCES users(id),
    asset TEXT NOT NULL,
    available BIGINT NOT NULL CHECK (available >= 0 ),
    locked BIGINT NOT NULL CHECK (locked >= 0),
    PRIMARY KEY(user_id, asset)
);

CREATE TABLE orders (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    symbol TEXT NOT NULL REFERENCES markets(symbol),
    side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    type TEXT NOT NULL CHECK (type IN ('market', 'limit')),
    price BIGINT NOT NULL CHECK (price > 0),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    filled BIGINT NOT NULL DEFAULT 0 CHECK (filled >= 0 AND filled <= quantity),
    status TEXT NOT NULL CHECK (status IN ('pending', 'partially_filled', 'filled', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX orders_active_idx ON orders (symbol, side, price, id) WHERE status IN ('pending', 'partially_filled');
CREATE INDEX orders_user_idx ON orders (user_id, id);

CREATE TABLE trades(
    id BIGINT PRIMARY KEY,
    symbol TEXT NOT NULL REFERENCES markets(symbol),
    price BIGINT NOT NULL CHECK (price > 0),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    buy_order_id BIGINT NOT NULL REFERENCES orders(id),
    sell_order_id BIGINT NOT NULL REFERENCES orders(id),
    buy_user_id BIGINT NOT NULL REFERENCES users(id),
    sell_user_id BIGINT NOT NULL REFERENCES users(id),
    buy_limit_price BIGINT NOT NULL CHECK (buy_limit_price > 0),
    buyer_fee BIGINT NOT NULL CHECK (buyer_fee >= 0),
    seller_fee BIGINT NOT NULL CHECK (seller_fee >= 0),
    executed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX trades_symbol_idx ON trades(symbol, id);
CREATE INDEX trades_buy_user_idx ON trades(buy_user_id, id);
CREATE INDEX trades_sell_user_idx ON trades(sell_user_id, id);

CREATE TABLE exchange_fees(
    asset TEXT PRIMARY KEY,
    available BIGINT NOT NULL CHECK (available >= 0)
);