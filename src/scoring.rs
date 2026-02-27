use crate::types::Market;
use chrono::Utc;

pub fn calculate_score(market: &Market, net_profit: f64) -> f64 {
    let factors = ScoreFactors {
        profit_score: normalize_profit(net_profit),
        liquidity_score: normalize_liquidity(market.liquidity),
        volume_score: normalize_volume(market.volume),
        execution_risk: calculate_execution_risk(market),
        time_decay: calculate_time_decay(market.end_time),
    };

    (factors.profit_score * 0.4)
        + (factors.liquidity_score * 0.25)
        + (factors.volume_score * 0.15)
        + (factors.execution_risk * 0.15)
        + (factors.time_decay * 0.05)
}

struct ScoreFactors {
    profit_score: f64,
    liquidity_score: f64,
    volume_score: f64,
    execution_risk: f64,
    time_decay: f64,
}

fn normalize_profit(profit: f64) -> f64 {
    1.0 / (1.0 + std::f64::consts::E.powf(-50.0 * (profit - 0.025)))
}

fn normalize_liquidity(liquidity: f64) -> f64 {
    (liquidity / 100_000.0).min(1.0).ln_1p()
}

fn normalize_volume(volume: f64) -> f64 {
    (volume / 1_000_000.0).min(1.0).ln_1p()
}

fn calculate_execution_risk(_market: &Market) -> f64 {
    1.0
}

fn calculate_time_decay(end_time: Option<chrono::DateTime<Utc>>) -> f64 {
    match end_time {
        None => 0.5,
        Some(end) => {
            let remaining = (end - Utc::now()).num_hours() as f64;
            if remaining < 0.0 {
                0.0
            } else {
                (remaining / 168.0).min(1.0)
            }
        }
    }
}
