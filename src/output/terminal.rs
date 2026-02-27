use crate::types::ArbitrageOpportunity;
use tabled::{Table, Tabled};

#[derive(Tabled)]
struct OpportunityRow {
    #[tabled(rename = "Market")]
    question: String,
    #[tabled(rename = "Strategy")]
    strategy: String,
    #[tabled(rename = "Gross %")]
    gross_profit: String,
    #[tabled(rename = "Net %")]
    net_profit: String,
    #[tabled(rename = "Fee %")]
    fee_cost: String,
    #[tabled(rename = "Score")]
    score: String,
}

pub fn print_opportunities(opportunities: &[ArbitrageOpportunity]) {
    if opportunities.is_empty() {
        println!("No arbitrage opportunities found.");
        return;
    }

    let rows: Vec<OpportunityRow> = opportunities
        .iter()
        .take(20)
        .map(|opp| OpportunityRow {
            question: truncate(&opp.market.question, 50),
            strategy: format!("{:?}", opp.strategy),
            gross_profit: format!("{:.3}%", opp.gross_profit * 100.0),
            net_profit: format!("{:.3}%", opp.net_profit * 100.0),
            fee_cost: format!("{:.3}%", opp.fee_cost * 100.0),
            score: format!("{:.3}", opp.score),
        })
        .collect();

    let table = Table::new(rows).to_string();
    println!("\n=== Arbitrage Opportunities ===\n");
    println!("{}", table);

    if opportunities.len() > 20 {
        println!("\n... and {} more opportunities", opportunities.len() - 20);
    }
}

fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len - 3])
    }
}
