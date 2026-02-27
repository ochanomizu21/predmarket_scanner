use crate::types::Market;

pub fn calculate_polymarket_fee(profit: f64, _market: &Market) -> f64 {
    const TRADING_FEE_RATE: f64 = 0.02;
    const MAKER_REBATE: f64 = 0.0002;

    let fee = profit * TRADING_FEE_RATE;
    let rebate = profit * MAKER_REBATE;

    fee - rebate
}

pub fn calculate_net_roi(gross_profit: f64, cost: f64, fees: f64) -> f64 {
    let net_payout = cost + gross_profit - fees;
    (net_payout - cost) / cost
}
