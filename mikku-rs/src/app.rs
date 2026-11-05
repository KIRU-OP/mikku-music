//! Port of `mikku/core/bot.py`, `mikku/core/userbot.py` and the client part of `core/call.py`:
//! the bot client, the (up to five) assistant accounts and one `tgcalls::Calls` per assistant.

use anyhow::{Context, Result};
use ferogram::{Client, ShutdownToken};
use once_cell::sync::OnceCell;
use std::collections::HashSet;
use std::sync::RwLock;
use tgcalls::Calls;

use crate::config::CONFIG;
use crate::db;

/// One assistant (userbot) account that joins voice chats.
#[derive(Clone)]
pub struct Assistant {
    /// 1..=5, same numbering as `STRING_SESSION`, `STRING_SESSION2`, ...
    pub number: u8,
    pub client: Client,
    pub calls: Calls,
    pub id: i64,
    pub username: Option<String>,
    pub name: String,
}

pub struct App {
    pub bot: Client,
    pub bot_id: i64,
    pub bot_username: String,
    pub bot_name: String,
    pub assistants: Vec<Assistant>,
    /// Keeps the connections alive; dropping a token shuts that client down.
    _shutdowns: Vec<ShutdownToken>,
}

impl App {
    pub fn assistant(&self, number: u8) -> Option<&Assistant> {
        self.assistants.iter().find(|a| a.number == number)
    }

    pub fn mention(&self) -> String {
        format!("<a href=\"https://t.me/{}\">{}</a>", self.bot_username, self.bot_name)
    }
}

static APP: OnceCell<App> = OnceCell::new();

/// Global application handle (set once by [`init`]).
pub fn app() -> &'static App {
    APP.get().expect("app not initialised")
}

// ───────────── global user sets (config.BANNED_USERS and misc.SUDOERS) ─────────────
static SUDOERS: OnceCell<RwLock<HashSet<i64>>> = OnceCell::new();
static BANNED: OnceCell<RwLock<HashSet<i64>>> = OnceCell::new();

fn sudoers_set() -> &'static RwLock<HashSet<i64>> {
    SUDOERS.get_or_init(Default::default)
}

fn banned_set() -> &'static RwLock<HashSet<i64>> {
    BANNED.get_or_init(Default::default)
}

pub fn is_sudo(user_id: i64) -> bool {
    user_id == CONFIG.owner_id || sudoers_set().read().unwrap().contains(&user_id)
}

pub fn add_sudo_user(user_id: i64) {
    sudoers_set().write().unwrap().insert(user_id);
}

pub fn remove_sudo_user(user_id: i64) {
    sudoers_set().write().unwrap().remove(&user_id);
}

pub fn sudo_users() -> Vec<i64> {
    sudoers_set().read().unwrap().iter().copied().collect()
}

pub fn is_banned(user_id: i64) -> bool {
    banned_set().read().unwrap().contains(&user_id)
}

pub fn ban_user(user_id: i64) {
    banned_set().write().unwrap().insert(user_id);
}

pub fn unban_user(user_id: i64) {
    banned_set().write().unwrap().remove(&user_id);
}

/// Port of `misc.sudo()`: owner is always sudo, plus everyone stored in MongoDB.
pub async fn load_sudoers() {
    add_sudo_user(CONFIG.owner_id);
    match db::get_sudoers().await {
        Ok(list) => {
            if !list.contains(&CONFIG.owner_id) {
                let _ = db::add_sudo(CONFIG.owner_id).await;
            }
            for id in &list {
                add_sudo_user(*id);
            }
            tracing::info!("✅ Loaded {} sudo users.", list.len());
        }
        Err(e) => tracing::error!("⚠️ Failed to load sudoers: {e}"),
    }
}

/// Load gbanned + blocked users into the in-memory ban list (same as `__main__.init`).
pub async fn load_banned() {
    for list in [db::get_gbanned().await, db::get_banned_users().await] {
        if let Ok(ids) = list {
            for id in ids {
                ban_user(id);
            }
        }
    }
}

fn user_display(u: &ferogram::tl::types::User) -> String {
    let first = u.first_name.clone().unwrap_or_default();
    match u.last_name.as_deref() {
        Some(l) if !l.is_empty() => format!("{first} {l}"),
        _ => first,
    }
}

async fn start_bot() -> Result<(Client, ShutdownToken)> {
    let (client, shutdown) = Client::builder()
        .api_id(CONFIG.api_id)
        .api_hash(CONFIG.api_hash.clone())
        .session("mikku_bot.session")
        .catch_up(true)
        .connect()
        .await
        .context("connecting bot client")?;
    if !client.is_authorized().await.unwrap_or(false) {
        client
            .bot_sign_in(&CONFIG.bot_token)
            .await
            .context("bot sign-in (check BOT_TOKEN)")?;
        let _ = client.save_session().await;
    }
    Ok((client, shutdown))
}

async fn start_assistant(number: u8, session: &str) -> Result<(Assistant, ShutdownToken)> {
    let (client, shutdown) = Client::builder()
        .api_id(CONFIG.api_id)
        .api_hash(CONFIG.api_hash.clone())
        .session_string(session)
        .catch_up(false)
        .connect()
        .await
        .with_context(|| format!("connecting assistant {number}"))?;
    if !client.is_authorized().await.unwrap_or(false) {
        anyhow::bail!(
            "assistant {number}: session is not authorised. Old Pyrogram session strings do not work \
             here - generate a new one with `mikku gen-session`."
        );
    }
    let me = client.get_me().await.context("assistant get_me")?;
    let calls = Calls::new(client.clone());
    Ok((
        Assistant {
            number,
            client,
            calls,
            id: me.id,
            username: me.username.clone(),
            name: user_display(&me),
        },
        shutdown,
    ))
}

/// Start the bot and every configured assistant, then publish the global [`App`].
pub async fn init() -> Result<&'static App> {
    tracing::info!("Starting Bot...");
    let (bot, bot_shutdown) = start_bot().await?;
    let me = bot.get_me().await?;
    let mut shutdowns = vec![bot_shutdown];

    let mut assistants = Vec::new();
    for (number, session) in CONFIG.assistant_sessions() {
        match start_assistant(number, session).await {
            Ok((a, sd)) => {
                tracing::info!("Assistant {number} started as {}", a.name);
                db::ASSISTANTS.lock().unwrap().push(number);
                shutdowns.push(sd);
                assistants.push(a);
            }
            Err(e) => tracing::error!("Assistant {number} failed to start: {e:#}"),
        }
    }
    if assistants.is_empty() {
        anyhow::bail!("no assistant could be started");
    }

    let app = App {
        bot,
        bot_id: me.id,
        bot_username: me.username.clone().unwrap_or_default(),
        bot_name: user_display(&me),
        assistants,
        _shutdowns: shutdowns,
    };
    let _ = APP.set(app);
    Ok(self::app())
}

/// `mikku gen-session`: interactive login that prints a ferogram session string for an assistant.
pub async fn gen_session() -> Result<()> {
    let (client, _shutdown) = Client::builder()
        .api_id(CONFIG.api_id)
        .api_hash(CONFIG.api_hash.clone())
        .session_string("")
        .connect_and_login()
        .await
        .map_err(|e| anyhow::anyhow!("login failed: {e}"))?;
    let s = client.export_session_string().await?;
    println!("\nYour session string (put it in STRING_SESSION):\n\n{s}\n\nKeep it private - it gives full access to the account.");
    Ok(())
}
