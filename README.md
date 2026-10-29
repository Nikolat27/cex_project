# CEX

A small centralized exchange written in Go. Orders match in memory, balances and history live in Postgres, and ETH deposits and withdrawals settle on Sepolia.

The HTTP API listens on port **8000**. On startup the process loads `.env`, connects to Postgres, creates the admin user if needed, reloads markets, balances, orders, and trades into memory, then starts three background jobs.

## Requirements

- Go 1.27.1
- Postgres
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

## Setup

```bash
cp .env.example .env
```

Fill in `.env`:

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `PASETO_SYMMETRIC_KEY` | Secret used to sign login cookies. Any non-empty string; it is hashed to 32 bytes |
| `ADMIN_USERNAME` | Admin created on startup if that username does not exist |
| `ADMIN_PASSWORD` | Password for that admin. Must match the stored hash on later startups |
| `ETH_XPUB` | Account extended public key used to derive Sepolia deposit addresses |
| `ETH_XPRV` | Account extended private key that matches `ETH_XPUB`. Withdrawals are signed with the child key `xprv / 0 / derivation_index` for that user's deposit address |
| `SEPOLIA_RPC_URL` | Sepolia JSON-RPC endpoint |

Apply the schema, then start the server:

```bash
make migrate-up
make run
```

`make test` runs `go test ./...`. `make migrate-down` rolls the schema back one step. The Makefile reads `.env`, so `DATABASE_URL` is available to the migrate commands.

Stop the server with Ctrl+C. The process shuts the HTTP server down within 5 seconds. The watchers use that same shutdown signal and keep running until then.

## How money and orders work

Assets in the catalog are **BTC** (8 decimals), **USDT** (6), and **ETH** (18). API amounts are human units (`1.5` ETH). The server converts them to minor units before they touch balances or the book.

The matching engine keeps one order book per market. Limit and market orders lock funds, match, and settle immediately. Each fill charges a **0.25%** fee, which is added to the exchange fee balance for that asset. A restart rebuilds users, markets, balances, open orders, trades, and fee balances from Postgres.

Two ways to move funds:

- `POST /api/users/{id}/deposit` and `POST /api/users/{id}/withdraw` change the ledger directly. They do not touch the chain.
- Chain deposits and withdrawals are ETH on Sepolia only. A user asks for a deposit address, sends ETH to it, and a watcher credits the balance after the transfer is in a finalized block. A withdrawal request locks the balance. The signer derives that user's deposit-address key from `ETH_XPRV` and broadcasts the transfer from that address. A second watcher marks it completed or failed once the receipt is finalized. Failed and cancelled withdrawals unlock the balance.

The watchers run once at startup and then every minute:

| Watcher | What it does |
|---|---|
| Sepolia | Scans new blocks for deposit-address transfers and credits finalized ones |
| Signer | Broadcasts the oldest `requested` withdrawal |
| Broadcast | Checks the oldest `broadcast` withdrawal and moves it to `completed` or `failed` |

## Authentication

`POST /api/users/register` and `POST /api/users/login` are public. Login sets an HttpOnly cookie named `auth_token` (PASETO, 12 hours, `Domain=localhost`). Every other route requires that cookie. Logout clears it.

A route with `{id}` also checks that the cookie belongs to that user. Creating a market, promoting or demoting an admin, and changing a withdrawal's status require the admin user.

## API

Register and log in:

```http
POST /api/users/register
{"username": "ada", "password": "at-least-8"}

POST /api/users/login
{"username": "ada", "password": "at-least-8"}

DELETE /api/users/logout
```

Balances. `deposit` and `withdraw` are ledger adjustments. `withdrawals` is the on-chain flow.

```http
POST /api/users/{id}/deposit
{"asset": "ETH", "amount": 1.5}

POST /api/users/{id}/withdraw
{"asset": "ETH", "amount": 0.25}

GET /api/users/{id}/balance/{asset}

GET /api/users/{id}/deposit_address?symbol=ETH&network=sepolia

POST /api/users/{id}/withdrawals
{"symbol": "ETH", "network": "sepolia", "destination": "0x...", "amount": 0.1}

GET  /api/users/{id}/withdrawals
POST /api/users/{id}/withdrawals/{withdrawalID}/cancel
POST /api/users/{id}/withdrawals/{withdrawalID}/status
{"status": "broadcast", "tx_hash": "0x..."}
```

`status` on that last route is admin-only and must be `broadcast`, `completed`, or `failed`. `broadcast` requires `tx_hash`. The signer and broadcast watchers call the same transition.

Markets and orders. `side` is `buy` or `sell`. `type` is `limit` or `market`. For a market order, `price` is the protection price.

```http
POST /api/markets
{"symbol": "ETH-USDT", "base_asset": "ETH", "quote_asset": "USDT"}

GET /api/markets/{symbol}/book
GET /api/markets/{symbol}/trades

POST /api/orders
{"user_id": 1, "symbol": "ETH-USDT", "side": "buy", "type": "limit", "price": 2500, "quantity": 0.5}

GET    /api/orders/{id}
DELETE /api/orders/{id}
{"user_id": 1, "symbol": "ETH-USDT"}

GET /api/users/{id}/orders
GET /api/users/{id}/trades
```

Admin user flags:

```http
POST   /api/users/{id}/make_super
DELETE /api/users/{id}/remove_super
```

## Layout

```
main.go                     startup, admin user, watchers
internal/core               matching engine, accounts, settlement
internal/services           orders, deposits, withdrawals, recovery
internal/database           Postgres models
internal/handlers           HTTP handlers
internal/HttpServer         Chi router
internal/wallet             deposit addresses and Sepolia transfers
internal/asset              BTC, USDT, ETH
internal/paseto             login tokens
migrations                  schema
```
