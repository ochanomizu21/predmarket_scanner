use crate::types::{Market, Platform, Outcome, Side};
use polymarket_client_sdk::clob::types::Side as ClobSide;
use polymarket_client_sdk::gamma::types::response as GammaTypes;
use rust_decimal::prelude::ToPrimitive;
use std::collections::HashMap;

pub struct PolymarketClient {
    gamma_client: polymarket_client_sdk::gamma::Client,
    clob_client: polymarket_client_sdk::clob::Client,
}

impl PolymarketClient {
    pub fn new() -> Self {
        Self {
            gamma_client: polymarket_client_sdk::gamma::Client::default(),
            clob_client: polymarket_client_sdk::clob::Client::default(),
        }
    }

    pub async fn fetch_all_markets(&self) -> Result<Vec<Market>, anyhow::Error> {
        log::info!("Fetching markets from Polymarket Gamma API");

        let request = polymarket_client_sdk::gamma::types::request::MarketsRequest::builder()
            .maybe_closed(Some(false))
            .limit(1000)
            .build();

        let gamma_markets = self.gamma_client.markets(&request).await?;

        log::info!("Fetched {} markets from Gamma API", gamma_markets.len());

        self.fetch_markets_with_prices(gamma_markets).await
    }

    pub async fn fetch_markets_with_prices(
        &self,
        gamma_markets: Vec<GammaTypes::Market>,
    ) -> Result<Vec<Market>, anyhow::Error> {
        log::info!("Fetching prices from CLOB API");

        let all_prices = self.clob_client.all_prices().await?;
        let prices_map = all_prices.prices.unwrap_or_default();

        let mut markets = Vec::new();

        for gamma_market in gamma_markets {
            if let Some(converted) = self.convert_market(gamma_market, &prices_map) {
                markets.push(converted);
            }
        }

        log::info!("Converted {} markets with prices", markets.len());
        Ok(markets)
    }

    fn convert_market(
        &self,
        gamma_market: GammaTypes::Market,
        prices_map: &HashMap<alloy::primitives::U256, HashMap<ClobSide, rust_decimal::Decimal>>,
    ) -> Option<Market> {
        let question = gamma_market.question.clone()?;

        if gamma_market.clob_token_ids.is_none() || gamma_market.clob_token_ids.as_ref().unwrap().is_empty() {
            return None;
        }

        let outcomes = self.extract_outcomes(&gamma_market, prices_map)?;

        Some(Market {
            id: gamma_market.id.clone(),
            question,
            platform: Platform::Polymarket,
            outcomes,
            liquidity: gamma_market.liquidity
                .and_then(|d| d.to_f64())
                .unwrap_or(0.0),
            volume: gamma_market.volume
                .and_then(|d| d.to_f64())
                .unwrap_or(0.0),
            end_time: gamma_market.end_date,
        })
    }

    fn extract_outcomes(
        &self,
        gamma_market: &GammaTypes::Market,
        prices_map: &HashMap<alloy::primitives::U256, HashMap<ClobSide, rust_decimal::Decimal>>,
    ) -> Option<Vec<Outcome>> {
        let outcome_names = gamma_market.outcomes.as_ref()?;
        let token_ids = gamma_market.clob_token_ids.as_ref()?;

        let mut outcomes = Vec::new();

        for (idx, outcome_name) in outcome_names.iter().enumerate() {
            if idx >= token_ids.len() {
                break;
            }

            let token_id = &token_ids[idx];
            let prices = prices_map.get(token_id)?;

            let buy_price = prices.get(&ClobSide::Buy)
                .and_then(|d| d.to_f64())
                .unwrap_or(0.0);

            let sell_price = prices.get(&ClobSide::Sell)
                .and_then(|d| d.to_f64())
                .unwrap_or(0.0);

            outcomes.push(Outcome {
                name: outcome_name.clone(),
                price: if buy_price > 0.0 { buy_price } else { sell_price },
                side: Side::Bid,
                order_book_depth: 0,
            });
        }

        if outcomes.is_empty() {
            None
        } else {
            Some(outcomes)
        }
    }

    pub async fn fetch_order_book(&self, token_id: &str) -> Result<(), anyhow::Error> {
        let token_u256: alloy::primitives::U256 = token_id.trim_start_matches("0x").parse().unwrap_or_default();

        let request = polymarket_client_sdk::clob::types::request::OrderBookSummaryRequest::builder()
            .token_id(token_u256)
            .build();

        let _order_book = self.clob_client.order_book(&request).await?;

        Ok(())
    }
}

impl Default for PolymarketClient {
    fn default() -> Self {
        Self::new()
    }
}
