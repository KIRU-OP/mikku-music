//! Mikku Music - pure Rust Telegram voice-chat music bot.

mod app;
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

    if std::env::args().nth(1).as_deref() == Some("gen-session") {
        return app::gen_session().await;
    }

    if CONFIG.assistant_sessions().is_empty() {
        tracing::error!("⚙️ Assistant client variables not defined, exiting...");
        std::process::exit(1);
    }

    strings::load("./strings/langs")?;
    db::connect(&CONFIG.mongo_db_uri).await?;
    app::load_sudoers().await;
    app::load_banned().await;
    let app = app::init().await?;
    tracing::info!("mikku started as @{} with {} assistant(s)", app.bot_username, app.assistants.len());

    tokio::signal::ctrl_c().await?;
    tracing::info!("🚫 Stopping Mikku Music Bot...");
    Ok(())
}
