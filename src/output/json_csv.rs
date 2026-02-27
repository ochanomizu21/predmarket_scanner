use crate::types::ArbitrageOpportunity;
use std::fs::File;
use std::io::Write;

pub fn export_json(
    opportunities: &[ArbitrageOpportunity],
    path: &str,
) -> Result<(), anyhow::Error> {
    let json = serde_json::to_string_pretty(opportunities)?;
    let mut file = File::create(path)?;
    file.write_all(json.as_bytes())?;
    Ok(())
}

pub fn export_csv(opportunities: &[ArbitrageOpportunity], path: &str) -> Result<(), anyhow::Error> {
    let mut wtr = csv::Writer::from_path(path)?;

    wtr.write_record(&[
        "market_id",
        "question",
        "strategy",
        "gross_profit",
        "net_profit",
        "fee_cost",
        "score",
        "platform",
    ])?;

    for opp in opportunities {
        wtr.write_record(&[
            &opp.market.id,
            &opp.market.question,
            &format!("{:?}", opp.strategy),
            &opp.gross_profit.to_string(),
            &opp.net_profit.to_string(),
            &opp.fee_cost.to_string(),
            &opp.score.to_string(),
            &format!("{:?}", opp.market.platform),
        ])?;
    }

    wtr.flush()?;
    Ok(())
}
