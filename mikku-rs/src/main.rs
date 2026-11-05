//! Mikku Music - Rust port of the Python Telegram voice-chat music bot.

mod config;
mod db;
mod strings;

use config::CONFIG;
use tracing_subscriber::EnvFilter;

fn init_logging() {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")))
        .with_target(true)
        .init();
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let _ = dotenvy::dotenv();
    init_logging();

    if CONFIG.assistant_sessions().is_empty() {
        tracing::error!("⚙️ Assistant client variables not defined, exiting...");
        std::process::exit(1);
    }

    strings::load("./strings/langs")?;
    db::connect(&CONFIG.mongo_db_uri).await?;

    tracing::info!("mikku (rust) core initialised");
    Ok(())
}
