# financesite

A self-hosted personal finance dashboard: see where your money went by category, and track net worth across bank, investment, retirement and mortgage accounts. Go backend with no third-party dependencies, Vue 3 + TypeScript + Tailwind frontend, and [Plaid](https://plaid.com) for account connections.

It runs out of the box in **demo mode** on made-up data, so you can try it before creating a Plaid account.

## What it does

| Screen | What you get |
|---|---|
| **Spending** | Total spent, income and what's left for any month; spending ranked by category; a month-by-month trend; top merchants. Select a category to drill into its transactions. |
| **Transactions** | Filter by month, category or merchant. Change any transaction's category, and optionally save a rule so that merchant is always filed the same way. |
| **Net worth** | Assets, debts and net worth. Accounts grouped as cash, investments, retirement (401(k)/IRA), Thrift Savings Plan, credit cards and mortgages, with holdings. Home value, mortgage balance and equity per property. |
| **Connections** | Link institutions through Plaid, sync on demand, disconnect, or import a CSV statement. |

### Categories

Categories are Plaid's `personal_finance_category` primary taxonomy, used unchanged so linked-bank data needs no remapping: Rent & Utilities, Food & Drink, Shopping (general merchandise), Transportation, Travel, Entertainment, Medical, Personal Care, Home Improvement, Services, Loan Payments, Government & Non-profit, Bank Fees and Other. Income, Transfer In and Transfer Out are tracked but never counted as spending.

A transaction's category is decided in this order:

1. A category you set on that one transaction.
2. A rule you saved ("merchant contains *X* → category").
3. The category from the source (Plaid, or a keyword guess for CSV imports).

"Spent" is net of refunds and includes pending transactions.

## Quick start (demo mode)

Requires Go 1.25+ and Node 22+.

```bash
cd ui && npm ci && npm run build && cd ..
```

```bash
go run ./cmd/financesite
```

Open <http://localhost:8080>. Demo data lives in memory: edits work, but reset when the server restarts.

## Connecting real accounts

1. Create a free account at [dashboard.plaid.com](https://dashboard.plaid.com) and copy your **client ID** and **Sandbox secret**.
2. Copy `.env.example` to `.env` and fill in `PLAID_CLIENT_ID` and `PLAID_SECRET`.
3. Start the server with those variables loaded:

```bash
set -a; source .env; set +a; go run ./cmd/financesite
```

4. On the **Connections** tab choose what to connect. Plaid Link opens; in Sandbox, pick any institution and sign in with `user_good` / `pass_good`.

For real accounts, apply for Production access in the Plaid dashboard (enable the Transactions, Investments and Liabilities products), then set `PLAID_ENV=production` with your production secret.

You sign in to your bank inside Plaid's own window. This app never sees or stores your bank username or password; it receives an access token per institution.

### What each connection brings in

| Connection | Plaid product | Data |
|---|---|---|
| Bank accounts and credit cards | Transactions | Balances and categorised transactions, synced incrementally |
| Investments and 401(k) | Investments | Balances and holdings for brokerage, IRA and employer plans |
| Mortgage | Liabilities | Loan balance, interest rate, next payment and property address |

### What Plaid does not cover

- **Thrift Savings Plan.** TSP is not dependably available through Plaid. Enter your balance by fund (G, F, C, S, I, L) on the Net worth tab and update it from your statement. If Plaid does offer your TSP login, connecting it under *Investments* will file it in the TSP group automatically.
- **Home value.** A mortgage connection supplies the loan balance, not what the house is worth. Enter an estimated value per property and link it to the mortgage account; the app computes equity. Hooking up an automated valuation source is on the roadmap.
- **Institutions Plaid can't reach.** Import a CSV statement instead. The file needs date, description and amount (or debit/credit) columns; summary lines above the header are skipped.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `PLAID_CLIENT_ID`, `PLAID_SECRET` | unset | Plaid credentials. Leave unset for demo mode. |
| `PLAID_ENV` | `sandbox` | `sandbox` or `production` |
| `FINANCESITE_ADDR` | `localhost:8080` | Listen address |
| `FINANCESITE_DATA_DIR` | `./data/local` | Where state is saved in Plaid mode |
| `FINANCESITE_UI_DIR` | `./ui/dist` | Built UI to serve |

## Security notes

This is a single-user app meant to run on your own machine.

- There is **no login on the app itself**. It binds to `localhost` by default; do not expose it to a network without putting authentication in front of it.
- State, including Plaid access tokens, is stored unencrypted in `data/local/state.json` with `0600` permissions. That directory is git-ignored. Treat it like a password file.
- Access tokens are never sent to the browser.

## Development

Run the API and the UI dev server side by side; Vite proxies `/api` to the Go server.

```bash
go run ./cmd/financesite
```

```bash
cd ui && npm run dev
```

Tests cover categorisation precedence, spending and net-worth maths, CSV parsing, the HTTP API, and Plaid sync against a fake Plaid server:

```bash
go test ./...
```

### Docker

```bash
docker build -t financesite .
```

```bash
docker run --rm -p 127.0.0.1:8080:8080 --env-file .env -v financesite-data:/app/data/local financesite
```

## Layout

```
cmd/financesite/      server entry point
api/handler/          HTTP API (JSON)
internal/categories/  the category taxonomy and CSV keyword guesses
internal/plaid/       minimal Plaid API client (net/http only)
internal/service/     CSV import, spending and net-worth summaries, Plaid sync
internal/store/       in-memory state persisted to one JSON file
internal/demo/        sample data for demo mode
ui/                   Vue 3 + TypeScript + Tailwind dashboard
```

## API

| Method and path | Purpose |
|---|---|
| `GET /api/status` | Demo mode flag and Plaid environment |
| `GET /api/categories` | The category list |
| `GET /api/spending?month=YYYY-MM` | Monthly summary (latest month if omitted) |
| `GET /api/transactions?month=&category=` | Transactions, newest first |
| `PATCH /api/transactions/{id}` | Set `{ "category", "remember" }` |
| `POST /api/transactions/import` | Upload a CSV as form field `file` |
| `GET` / `POST /api/rules`, `DELETE /api/rules/{id}` | Category rules |
| `GET /api/accounts` | Accounts plus net-worth summary |
| `POST` / `PUT` / `DELETE /api/accounts/manual[/{id}]` | Hand-entered accounts such as TSP |
| `POST` / `PUT` / `DELETE /api/properties[/{id}]` | Properties and home values |
| `GET /api/connections`, `DELETE /api/connections/{id}` | Linked institutions |
| `POST /api/plaid/link-token` | Start Plaid Link for `{ "kind": "bank" \| "investment" \| "mortgage" }` |
| `POST /api/plaid/exchange` | Finish Plaid Link and run the first sync |
| `POST /api/plaid/sync` | Refresh every linked institution |

## Roadmap

- App-level login and encryption of stored tokens, so it can be hosted beyond localhost.
- Plaid webhooks and scheduled sync instead of the manual "Sync now".
- Re-authentication flow (Plaid Link update mode) when a bank login expires.
- Automated home valuation and net-worth history over time.
- Budgets per category.
