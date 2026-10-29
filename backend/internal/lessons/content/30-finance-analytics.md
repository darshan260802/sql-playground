title: Market and banking analytics
module: Analytics case studies
level: Advanced
minutes: 22
summary: Moving averages, drawdowns, as-of currency conversion and portfolio positions.
---
Financial data is a great window-function playground: prices are ordered in time, and most metrics compare a value with its history.

### Window first, filter second

A 20-day moving average for early December needs November's prices. If you filter to December **before** computing the window, the first rows only see a few days. Compute in a CTE over the full history, then filter outside:

```sql
WITH ma AS (
  SELECT trade_date, close,
         AVG(close) OVER (ORDER BY trade_date ROWS BETWEEN 19 PRECEDING AND CURRENT ROW) AS ma_20
  FROM stock_prices WHERE stock_id = 1
)
SELECT * FROM ma WHERE trade_date >= '2025-12-01';
```

### Drawdown

The **drawdown** is how far a price sits below its highest point so far, a key measure of risk:

```sql
close / MAX(close) OVER (PARTITION BY stock_id ORDER BY trade_date
                         ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) - 1
```

### As-of joins

Exchange rates exist only on business days. To convert a balance "as of" a Sunday, use the **latest rate on or before** that date. First find that date per currency, then join back to fetch the rate:

```sql
SELECT er.currency, er.rate_to_usd
FROM exchange_rates er
JOIN (SELECT currency, MAX(rate_date) AS d
      FROM exchange_rates
      WHERE rate_date <= '2025-12-28'
      GROUP BY currency) latest
  ON latest.currency = er.currency AND latest.d = er.rate_date;
```

A correlated subquery (`WHERE rate_date = (SELECT MAX(...) ... WHERE e2.currency = er.currency ...)`) gives the same answer, but it runs once per row, which takes seconds on 18,000 rates. Note that `exchange_rates` has no `USD` rows: `COALESCE(rate, 1)` handles USD accounts.

### Positions from a trade log

A position is a running balance of shares: `SUM(CASE WHEN side = 'buy' THEN quantity ELSE -quantity END)`. Value it with the **latest** close, using `ROW_NUMBER() ... ORDER BY trade_date DESC` and keeping row 1.

=== exercise
id: fin-1
title: 20-day moving average
ordered: true
--- prompt
For stock **1**, return each trading day of **December 2025** with `trade_date`, `close` and **ma_20**: the average close over that day and the **19 previous trading days** (including November's), rounded to 4 decimals. Sort by date.
--- solution
WITH ma AS (
  SELECT trade_date, close,
         AVG(close) OVER (ORDER BY trade_date ROWS BETWEEN 19 PRECEDING AND CURRENT ROW) AS ma_20
  FROM stock_prices
  WHERE stock_id = 1
)
SELECT trade_date, close, ROUND(ma_20, 4) AS ma_20
FROM ma
WHERE trade_date >= '2025-12-01' AND trade_date < '2026-01-01'
ORDER BY trade_date;
--- hint
Compute the window over the whole history in a CTE, then filter to December outside it.
===

=== exercise
id: fin-2
title: Worst drawdown of 2025
ordered: true
--- prompt
Using only **2025** prices, compute each stock's **maximum drawdown**: the lowest value of (close / highest close so far in 2025 − 1), as a percentage rounded to 2 decimals (it will be ≤ 0).

Return `ticker` and **max_drawdown_pct**, worst (most negative) first, ties by ticker.
--- solution
WITH p AS (
  SELECT stock_id, close,
         MAX(close) OVER (PARTITION BY stock_id ORDER BY trade_date
                          ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS peak
  FROM stock_prices
  WHERE trade_date >= '2025-01-01'
)
SELECT s.ticker, ROUND(MIN(p.close / p.peak - 1) * 100, 2) AS max_drawdown_pct
FROM p
JOIN stocks s ON s.stock_id = p.stock_id
GROUP BY s.ticker
ORDER BY max_drawdown_pct, s.ticker;
--- hint
Running MAX(close) per stock gives the peak so far; the drawdown's minimum is the worst one.
===

=== exercise
id: fin-3
title: Cash in dollars, as of a Sunday
ordered: true
--- prompt
Total the balances (sum of all transaction amounts) of **checking and savings** accounts per `currency`, then convert them to USD using the **latest exchange rate on or before 2025-12-28** (a Sunday). USD balances use a rate of 1.

Return `currency`, **balance** (rounded to 2), **rate_to_usd** and **balance_usd** (rounded to 2), biggest USD balance first.
--- solution
WITH balances AS (
  SELECT a.currency, SUM(t.amount) AS balance
  FROM transactions t
  JOIN accounts a ON a.account_id = t.account_id
  WHERE a.account_type IN ('checking', 'savings')
  GROUP BY a.currency
),
rates AS (
  SELECT er.currency, er.rate_to_usd
  FROM exchange_rates er
  JOIN (SELECT currency, MAX(rate_date) AS d
        FROM exchange_rates
        WHERE rate_date <= '2025-12-28'
        GROUP BY currency) latest
    ON latest.currency = er.currency AND latest.d = er.rate_date
)
SELECT b.currency, ROUND(b.balance, 2) AS balance,
       COALESCE(r.rate_to_usd, 1) AS rate_to_usd,
       ROUND(b.balance * COALESCE(r.rate_to_usd, 1), 2) AS balance_usd
FROM balances b
LEFT JOIN rates r ON r.currency = b.currency
ORDER BY balance_usd DESC;
--- hint
Find MAX(rate_date) <= '2025-12-28' per currency, join back to exchange_rates for the rate, LEFT JOIN that to the balances and COALESCE USD to 1.
===

=== exercise
id: fin-4
title: A portfolio snapshot
ordered: true
--- prompt
For brokerage account **67**, return its current open positions: `ticker`, **shares** (bought minus sold, only positions above 0), **last_close** (the stock's most recent close in the data) and **market_value** = shares × last_close, rounded to 2 decimals. Largest market value first.
--- solution
WITH pos AS (
  SELECT stock_id, SUM(CASE WHEN side = 'buy' THEN quantity ELSE -quantity END) AS shares
  FROM trades
  WHERE account_id = 67
  GROUP BY stock_id
  HAVING shares > 0
),
last_px AS (
  SELECT stock_id, close
  FROM (SELECT stock_id, close,
               ROW_NUMBER() OVER (PARTITION BY stock_id ORDER BY trade_date DESC) AS rn
        FROM stock_prices) x
  WHERE rn = 1
)
SELECT s.ticker, p.shares, l.close AS last_close, ROUND(p.shares * l.close, 2) AS market_value
FROM pos p
JOIN last_px l ON l.stock_id = p.stock_id
JOIN stocks s ON s.stock_id = p.stock_id
ORDER BY market_value DESC;
--- hint
Net the trades per stock with a signed SUM, find each stock's latest close with ROW_NUMBER, then join.
===
