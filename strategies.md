1. Same‑market arbitrage (single platform)
These are the simplest, most “textbook” arbitrages: all the logic sits within one market on one platform.

1.1 Single‑condition Dutch‑book arbitrage (binary YES/NO)
What it is
In a binary market, “YES” and “NO” should sum to 1 (or very close). When:

YES price + NO price < 1 → you can buy both sides and lock in a riskless profit.
The IMDEA “Unravelling the Probabilistic Forest” paper calls this single‑condition market rebalancing arbitrage and shows it was heavily exploited on Polymarket.

Example

YES @ 0.50, NO @ 0.47 → sum = 0.97
Buy 1 YES and 1 NO for 0.97; at resolution you receive 1.
Profit = 0.03 (≈3.1%).
This is exactly the “Dutch‑book arbitrage” described in programmatic Polymarket tutorials: buy a complete set of mutually exclusive outcomes for < $1.

Practical notes

Competition is extreme: on Polymarket, same‑market windows often close in ~200 ms and are dominated by HFT bots.
Fees matter: if the platform takes 2% of winnings, a 1% “Dutch book” edge can turn negative after costs.
1.2 Multi‑condition Dutch‑book arbitrage (multi‑outcome markets)
What it is
Many markets have more than two mutually exclusive outcomes (e.g., “Trump / Biden / Other wins election”). The prices of all YES contracts should sum to 1; for NO contracts they should sum to N−1 (in an N‑outcome market).

When the sum of all YES prices ≠ 1, you can:

Long arbitrage: if sum(YES) < 1, buy all YES contracts.
Short arbitrage: if sum(YES) > 1, sell exposure (e.g., via market making / shorting where supported).
The IMDEA paper calls this multiple‑condition arbitrage and quantifies large profits from both long and short sides on multi‑outcome markets.

Example (long)

YES‑Trump = 0.40
YES‑Biden = 0.35
YES‑Other = 0.20
Sum = 0.95 < 1
Buy one of each YES for 0.95; you are guaranteed 1 at resolution.
Profit = 0.05 (5.3%).
Example (short)

YES‑Trump = 0.45
YES‑Biden = 0.40
YES‑Other = 0.20
Sum = 1.05 > 1
Short the basket (or sell all YES via market making) to lock in 0.05 per unit.
Practical notes

Execution gets more complex as you must manage N legs.
Slippage and partial fills are more damaging when you need to trade in all outcomes.
1.3 Multi‑outcome NO‑basket “negative risk” arbitrage
This is a variant of multi‑condition arbitrage, highlighted in practitioner write‑ups as “Negative Risk Arbitrage”.

Idea

In an N‑outcome market, the NO contracts should sum to N−1.
If sum(NO) deviates from N−1, you can construct a basket of NO bets that hedges principal risk and yields deterministic profit in most scenarios.
Intuition

Each NO contract pays 1 if that outcome does not occur.
If the market misprices the total probability of “something else happening”, you can buy a set of NO contracts such that:
You’re over‑paid for the risk in expectation, or
You have a very high probability of small profit and a small probability of a larger loss.
This is not always strictly riskless, but in practice it’s often treated as “low‑risk arb” when mispricings are large.

2. Cross‑market and cross‑platform arbitrage
Here you exploit mispricings between markets or platforms.

2.1 Identical event, different platforms
What it is
The same event trades on multiple platforms (Polymarket, Kalshi, Opinion/Other) at different implied probabilities. You buy the underpriced side on one platform and the opposite side on the other, locking in a spread.

This is what many guides call cross‑platform arbitrage (Polymarket vs Kalshi etc.).

Example

Polymarket YES @ 0.45
Kalshi NO @ 0.52
Cost to control all outcomes: 0.45 + 0.52 = 0.97
Guaranteed payout = 1 → profit ≈ 3.1% before fees.
Key risks

Settlement mismatch: platforms may resolve the same event differently due to different oracles or rules.
Fee drag: combined fees of 5%+ can easily wipe out 3–4% spreads.
Capital and latency: you need capital on both venues and fast execution; cross‑platform arb is heavily automated in practice.
2.2 Identical contract, different exchanges
The academic literature on “inter‑market arbitrage in betting” and prediction markets shows persistent arbitrage opportunities for identical contracts listed on different exchanges.
researchgate

What it is

The same contract (same question, same resolution rules) trades at different prices on different platforms.
You buy low on one venue, sell high (or buy the opposing side) on the other.
This is basically “prediction market ETF arbitrage”, but with event contracts instead of ETFs.

Why it persists

Liquidity fragmentation and risk limits: some markets are cheaper to trade on one venue, some on another.
Regulatory or access constraints: some platforms restrict US users, others don’t; this keeps price gaps alive.
2.3 Logically related markets (same platform)
Here you don’t need identical contracts; you only need logical relationships between markets.

Research on “the extent of price misalignment in prediction markets” documents mispricings among logically related contracts on the same exchange, especially around high‑information events.
researchgate

Examples

“Candidate X wins presidency” vs “Party Y wins presidency” (X is a member of Y).
“Team A wins match” vs “Team A wins by >10 points” vs “Team A wins by 1–10 points”.
If the joint probabilities don’t line up, you can construct portfolios that lock in profit via combinatorial logic.

2.4 Combinatorial arbitrage (strict dependencies)
The IMDEA work and Flashbots summary define combinatorial arbitrage as exploiting strict dependencies between markets (if A happens, B must happen; or if A happens, B cannot happen).

Example from the paper

Market M1: “Democrats win presidency”
Market M2: “Winning margin buckets” (D by 0–5, 5–10, 10+; R by 0–5, etc.)
Strategy:

Buy YES–Democrats in M1 @ 0.48
Buy all four Republican‑margin YES contracts in M2 @ total 0.40
Total cost = 0.88, guaranteed payout = 1 (if Democrats win, M1 pays; if Republicans win, one of the M2 R‑margin contracts pays).
Profit = 0.12 (13.6%).
This is true combinatorial arb: you construct a portfolio that covers every possible world with overlapping bets.

Empirical note

The Flashbots/IMDEA analysis finds only a small share of profits comes from combinatorial arb vs simple market‑rebalancing arb, despite its theoretical attractiveness.
medium
3. Time‑dynamic and microstructure strategies
These are not always “pure” arbitrage in the academic sense, but in practice they’re treated as low‑risk strategies that exploit dynamics over time.

3.1 Information‑lag arbitrage
What it is

One information source (e.g., live TV feed, exchange data, official API) updates faster than the prediction market’s order book.
You trade on the faster source before the market fully adjusts.
The “five major schools of arbitrage” article calls this information‑based front‑running arbitrage and gives examples from Fed speeches and sports broadcasts.

Example

During a Fed press conference, NLP models parse dovish/hawkish language in real time.
If the tone is more dovish than expected, you immediately buy “rate‑cut” contracts or related macro markets before the probability fully moves.
Risk

Not riskless: you’re betting that the market will eventually reprice; you can be wrong-footed in fast-moving situations.
3.2 Volatility / panic‑driven rebalancing
Practitioner guides highlight that volatility spikes and panic create many of the YES+NO≠1 opportunities.
collective.flashbots

What it is

During big news (elections, macro announcements, crypto crashes), participants trade emotionally and asymmetrically.
That can push YES+NO away from 1 for short periods.
Automated strategies systematically buy both sides when the spread opens, often scaling with volatility models.
Empirical evidence

IMDEA data shows arbitrage opportunities cluster around high‑uncertainty periods and that realized profits from rebalancing strategies are substantial.
3.3 End‑of‑day / end‑of‑event “sweep” strategies
Some guides describe end‑of‑day sweep arbitrage:

Near market close, the outcome is almost certain but the price hasn’t fully converged to 0 or 1.
You buy the heavily favored outcome when its probability is very high but not 1, expecting that the remaining time window is too short for a reversal.
arxiv
Characteristics

Not strictly riskless, but statistically very low risk if:
Resolution is mechanical (e.g., price of an asset at a known time),
There’s little time left for a swing.
3.4 Market‑making and spread capture
Some “arbitrage‑style” strategies are essentially market‑making:

In low‑liquidity markets, the bid–ask spread can be very wide.
You place limit orders on both sides, capturing the spread repeatedly.
Example from Polymarket strategies

Best buy = 0.30, best sell = 0.70 → spread = 0.40
You place buy at 0.31 and sell at 0.69, capturing 0.38 if both are filled.
This is not riskless: you can be left with a one‑sided position if the market moves, but in liquid, mean‑reverting markets it behaves like a high‑Sharpe, quasi‑arbitrage strategy.

4. Advanced / hybrid strategies
These are more sophisticated, often combining prediction markets with other assets or exploiting structural features.

4.1 Cross‑asset arbitrage vs options / betting / crypto derivatives
A recent Coindesk piece describes AI systems arbitraging prediction markets against options and derivatives pricing.
researchgate

Idea

Prediction market probability → convert to implied odds.
Compare with:
Options markets (e.g., probability of large moves),
Sports betting odds,
Crypto perpetual/futures pricing.
When they diverge, construct a portfolio that isolates the mispricing.
Example

A prediction market says “BTC above $100k by Dec 31” is 30%.
A BTC options‑based estimate is 25%.
You sell the expensive side (e.g., sell YES in the prediction market, or hedge via options) and buy the cheaper side.
Risk

Model risk: converting between probabilities, odds, and option prices is non‑trivial.
Basis risk: the events may not be perfectly identical.
4.2 Structural / rule‑based arbitrage
Different platforms use different rules, oracles, and settlement mechanisms. This creates structural arb opportunities:

Examples

One platform uses a single exchange’s price; another uses a composite index. You can arb the expected basis or oracle differences.
navnoorbawa.substack
Different handling of edge cases (e.g., delayed elections, cancelled games, disputed calls) can create one‑sided bets.
The “cross‑platform arbitrage” discussion explicitly warns that settlement mismatch is the biggest risk when arbing across platforms.

4.3 Manipulation‑adjacent strategies
The “five major arbitrage schools” article documents an aggressive case:

A trader exploited low liquidity in 15‑minute XRP price markets:
They bought “UP” positions cheaply.
Then used $1M on Binance to push the spot XRP price up right before the 15‑minute candle close.
This forced the prediction market to resolve in their favor, yielding ≈$280k profit.
This is not pure arbitrage and sits in a gray/illegal zone, but it’s a real structural vulnerability in some markets.

5. Putting it together: how strategies relate
Here’s a concise mapping:

Layer
Strategy type
Risk profile in practice
Same‑market, single condition	Dutch‑book YES/NO arbitrage	Very low risk, heavily competed, very short-lived
Same‑market, multi condition	Multi‑outcome YES/NO basket arbitrage	Low risk, more execution complexity
Cross‑market, same contract	Identical contract across platforms	Low risk, settlement & fee risk
Cross‑market, logical relations	Combinatorial arbitrage, logical dependencies	Medium risk, model & liquidity risk
Time‑dynamic	Information‑lag, volatility rebalancing, end‑of‑day	Low–medium risk, depends on speed & models
Market‑making	Spread capture in low‑liquidity markets	Medium risk, inventory risk
Cross‑asset	Arb vs options/betting/perps	Medium risk, model & basis risk
Structural / manipulation	Rule differences, oracle gaming, manipulative tactics	High risk, ethical/legal issues

6. Practical takeaways
Simple is powerful
Single‑condition Dutch‑book and multi‑outcome basket arbitrage alone accounted for tens of millions in profits on Polymarket in one year.
Cross‑platform is retail‑friendly but fee‑sensitive
Spreads under ~5% are often unprofitable after fees and settlement risk.
Combinatorial is theoretically rich but empirically modest
Despite the appealing math, combinatorial arb is a small share of total profits due to execution complexity and thin opportunities.
medium
Speed and automation are decisive
Same‑market and cross‑platform opportunities often live <1 second; without low‑latency infrastructure and bots, your effective edge is tiny.

