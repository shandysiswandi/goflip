# GO Flip

In-memory bank statement processor with streaming CSV ingestion, balance computation,
failed-transaction event handling, and queryable results.

## **Architecture Overview**
- HTTP layer in `internal/flip/inbound` accepts uploads and queries, validates params, and maps responses.
- Usecase layer in `internal/flip/usecase` streams CSV line-by-line, computes balances, collects issues, and updates metadata.
- Storage layer in `internal/flip/store` keeps uploads, balances, and issue transactions in a concurrency-safe in-memory store.
- Event layer in `internal/flip/event` publishes failed transactions to an in-memory bus and processes them with a worker pool.
- App wiring in `internal/app` builds dependencies, starts workers, and handles graceful shutdown.

## **Tradeoffs**
- In-memory only: data is lost on restart; suitable for the assignment but not durable.
- Issue transactions are stored fully in memory; large numbers of issues increase RAM usage.
- Idempotency uses an in-memory event ID map without eviction; long runs could grow it.
- Backoff is simple exponential; no jitter or per-event customization.

## **How To Run**
- Prerequisite: Go 1.25+
- `cp config/config.example.yaml config/config.yaml`
- `make run`
- Or: `LOCAL=true go run main.go`
- Server listens on `0.0.0.0:8080` by default, configured in `config/config.yaml`.

## **Testing**
- `go test ./...`
- `go test -race ./...`

## **Example CSV**
- `examples/statement.csv`

## **API Usage**

Upload a CSV (async processing):
```bash
curl -F "file=@examples/statement.csv" http://localhost:8080/statements
```
The response includes `upload_id`; poll `GET /balance` or `GET /transactions/issues` until status is `DONE`.

Get balance for an upload:
```bash
curl "http://localhost:8080/balance?upload_id=<UPLOAD_ID>"
```

List failed and pending transactions (pagination + filters):
```bash
curl "http://localhost:8080/transactions/issues?upload_id=<UPLOAD_ID>&page=1&page_size=10"
```

Filter by status/type:
```bash
curl "http://localhost:8080/transactions/issues?upload_id=<UPLOAD_ID>&status=FAILED,PENDING&type=DEBIT"
```

Health check:
```bash
curl http://localhost:8080/health
```
