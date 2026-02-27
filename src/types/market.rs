use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Market {
    pub id: String,
    pub question: String,
    pub platform: Platform,
    pub outcomes: Vec<Outcome>,
    pub liquidity: f64,
    pub volume: f64,
    pub end_time: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Outcome {
    pub name: String,
    pub price: f64,
    pub side: Side,
    pub order_book_depth: usize,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum Platform {
    Polymarket,
    Limitless,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum Side {
    Bid,
    Ask,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ArbitrageOpportunity {
    pub market: Market,
    pub strategy: StrategyType,
    pub gross_profit: f64,
    pub net_profit: f64,
    pub fee_cost: f64,
    pub score: f64,
    pub execution_plan: ExecutionPlan,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum StrategyType {
    DutchBook,
    MultiOutcome,
    NoBasket,
    CrossPlatform,
    Combinatorial,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionPlan {
    pub legs: Vec<TradeLeg>,
    pub total_cost: f64,
    pub guaranteed_payout: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TradeLeg {
    pub outcome: String,
    pub side: Side,
    pub price: f64,
    pub size: f64,
}
