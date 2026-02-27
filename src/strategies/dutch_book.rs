use crate::fees;
use crate::scoring;
use crate::types::*;

pub fn find_opportunities(markets: &[Market]) -> Vec<ArbitrageOpportunity> {
    markets
        .iter()
        .filter(|m| is_binary_market(m))
        .filter_map(|m| check_dutch_book(m))
        .collect()
}

fn is_binary_market(market: &Market) -> bool {
    market.outcomes.len() == 2
        && market.outcomes.iter().any(|o| o.name == "YES")
        && market.outcomes.iter().any(|o| o.name == "NO")
}

fn check_dutch_book(market: &Market) -> Option<ArbitrageOpportunity> {
    let yes_price = market.outcomes.iter().find(|o| o.name == "YES")?.price;
    let no_price = market.outcomes.iter().find(|o| o.name == "NO")?.price;

    let sum = yes_price + no_price;

    if sum >= 1.0 {
        return None;
    }

    let gross_profit = 1.0 - sum;
    let fee_cost = fees::calculate_polymarket_fee(gross_profit, market);
    let net_profit = gross_profit - fee_cost;

    if net_profit < 0.001 {
        return None;
    }

    Some(ArbitrageOpportunity {
        market: market.clone(),
        strategy: StrategyType::DutchBook,
        gross_profit,
        net_profit,
        fee_cost,
        score: scoring::calculate_score(market, net_profit),
        execution_plan: build_execution_plan(market, yes_price, no_price),
    })
}

fn build_execution_plan(_market: &Market, yes_price: f64, no_price: f64) -> ExecutionPlan {
    ExecutionPlan {
        legs: vec![
            TradeLeg {
                outcome: "YES".to_string(),
                side: Side::Bid,
                price: yes_price,
                size: 1.0,
            },
            TradeLeg {
                outcome: "NO".to_string(),
                side: Side::Bid,
                price: no_price,
                size: 1.0,
            },
        ],
        total_cost: yes_price + no_price,
        guaranteed_payout: 1.0,
    }
}
