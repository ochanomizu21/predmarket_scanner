mod clients;
mod strategies;
mod types;
mod fees;
mod scoring;
mod output;

use clap::{Parser, Subcommand};
use clients::PolymarketClient;
use output::print_opportunities;
use strategies::find_opportunities;

#[derive(Parser)]
#[command(name = "predmarket-scanner")]
#[command(about = "Prediction market arbitrage scanner", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    FetchMarkets {
        #[arg(short, long)]
        #[arg(default_value_t = 10)]
        limit: usize,
    },
    Scan {
        #[arg(short, long)]
        #[arg(default_value_t = 0.001)]
        min_profit: f64,

        #[arg(short, long)]
        #[arg(default_value_t = 100)]
        limit: usize,
    },
    Export {
        #[arg(short, long)]
        format: ExportFormat,

        #[arg(short, long)]
        #[arg(default_value = "opportunities")]
        output: String,
    },
}

#[derive(clap::ValueEnum, Clone)]
enum ExportFormat {
    Json,
    Csv,
}

#[tokio::main]
async fn main() -> Result<(), anyhow::Error> {
    env_logger::init();

    let cli = Cli::parse();

    match cli.command {
        Commands::FetchMarkets { limit } => {
            run_fetch_markets(limit).await
        },
        Commands::Scan { min_profit, limit } => {
            run_scan(min_profit, limit).await
        },
        Commands::Export { format, output } => {
            run_export(format, output).await
        },
    }
}

async fn run_fetch_markets(limit: usize) -> Result<(), anyhow::Error> {
    println!("Fetching markets from Polymarket...");

    let client = PolymarketClient::new();
    let markets = client.fetch_all_markets().await?;

    println!("\nFetched {} total markets", markets.len());

    let display_markets: Vec<_> = markets.into_iter().take(limit).collect();

    println!("\nFirst {} markets:", display_markets.len());
    println!("\n{:<40} {:<15} {:<15} {:<10}", "Question", "Liquidity", "Volume", "Outcomes");
    println!("{}", "-".repeat(85));

    for market in &display_markets {
        let question = market.question.chars().take(37).collect::<String>();
        let question = if market.question.len() > 37 {
            format!("{}...", question)
        } else {
            question
        };

        let outcomes_str: String = market.outcomes.iter()
            .take(2)
            .map(|o| o.name.clone())
            .collect::<Vec<_>>()
            .join(", ");

        println!("{:<40} {:<15} {:<15} {:<10}",
            question,
            format!("${:.2}", market.liquidity),
            format!("${:.2}", market.volume),
            format!("{} ({})", outcomes_str, market.outcomes.len())
        );
    }

    Ok(())
}

async fn run_scan(min_profit: f64, limit: usize) -> Result<(), anyhow::Error> {
    println!("Fetching markets from Polymarket...");

    let client = PolymarketClient::new();
    let markets = client.fetch_all_markets().await?;

    println!("Found {} markets", markets.len());

    println!("Scanning for arbitrage opportunities...");
    let opportunities = find_opportunities(&markets);

    let filtered: Vec<_> = opportunities
        .into_iter()
        .filter(|opp| opp.net_profit >= min_profit)
        .take(limit)
        .collect();

    println!("\nFound {} opportunities", filtered.len());
    print_opportunities(&filtered);

    Ok(())
}

async fn run_export(format: ExportFormat, output: String) -> Result<(), anyhow::Error> {
    println!("Fetching markets...");

    let client = PolymarketClient::new();
    let markets = client.fetch_all_markets().await?;

    println!("Scanning for opportunities...");
    let opportunities = find_opportunities(&markets);

    let filename = match format {
        ExportFormat::Json => format!("{}.json", output),
        ExportFormat::Csv => format!("{}.csv", output),
    };

    match format {
        ExportFormat::Json => {
            output::export_json(&opportunities, &filename)?;
            println!("Exported {} opportunities to {}", opportunities.len(), filename);
        },
        ExportFormat::Csv => {
            output::export_csv(&opportunities, &filename)?;
            println!("Exported {} opportunities to {}", opportunities.len(), filename);
        },
    }

    Ok(())
}
