//! Port of `mikku/utils/database.py` + `mikku/core/mongo.py`.
//! Same MongoDB database (`Anon`) and collection names, so existing data keeps working.

use anyhow::Result;
use bson::{doc, Bson, Document};
use dashmap::DashMap;
use futures::TryStreamExt;
use mongodb::{Client, Collection, Database};
use once_cell::sync::{Lazy, OnceCell};
use rand::seq::SliceRandom;
use std::sync::Mutex;

static DB: OnceCell<Database> = OnceCell::new();

pub async fn connect(uri: &str) -> Result<()> {
    tracing::info!("📜 Connecting to your Mongo Database...");
    let client = Client::with_uri_str(uri).await?;
    let db = client.database("Anon");
    db.run_command(doc! { "ping": 1 }).await?;
    let _ = DB.set(db);
    tracing::info!("🗃️ Connected to your Mongo Database.");
    Ok(())
}

fn col(name: &str) -> Collection<Document> {
    DB.get().expect("database not connected").collection::<Document>(name)
}

// ───────────── in-memory caches (same role as the module-level dicts in Python) ─────────────
static ACTIVE: Lazy<Mutex<Vec<i64>>> = Lazy::new(Default::default);
static ACTIVE_VIDEO: Lazy<Mutex<Vec<i64>>> = Lazy::new(Default::default);
static ASSISTANT: Lazy<DashMap<i64, u8>> = Lazy::new(Default::default);
static COUNT: Lazy<DashMap<i64, i64>> = Lazy::new(Default::default);
static CHANNEL_CONNECT: Lazy<DashMap<i64, i64>> = Lazy::new(Default::default);
static LANG: Lazy<DashMap<i64, String>> = Lazy::new(Default::default);
static LOOP: Lazy<DashMap<i64, i64>> = Lazy::new(Default::default);
static NONADMIN: Lazy<DashMap<i64, bool>> = Lazy::new(Default::default);
static PAUSE: Lazy<DashMap<i64, bool>> = Lazy::new(Default::default);
static PLAYMODE: Lazy<DashMap<i64, String>> = Lazy::new(Default::default);
static PLAYTYPE: Lazy<DashMap<i64, String>> = Lazy::new(Default::default);
static SKIPMODE: Lazy<DashMap<i64, bool>> = Lazy::new(Default::default);
static MAINTENANCE: Lazy<Mutex<Vec<u8>>> = Lazy::new(Default::default);

// ───────────── small helpers ─────────────
async fn exists(c: &str, filter: Document) -> Result<bool> {
    Ok(col(c).find_one(filter).await?.is_some())
}

async fn set_field(c: &str, key: &str, key_val: impl Into<Bson>, field: &str, val: impl Into<Bson>) -> Result<()> {
    col(c)
        .update_one(doc! { key: key_val.into() }, doc! { "$set": { field: val.into() } })
        .upsert(true)
        .await?;
    Ok(())
}

async fn collect_ids(c: &str, filter: Document, field: &str) -> Result<Vec<i64>> {
    let docs: Vec<Document> = col(c).find(filter).await?.try_collect().await?;
    Ok(docs.iter().filter_map(|d| d.get_i64(field).ok().or_else(|| d.get_i32(field).ok().map(i64::from))).collect())
}

fn doc_i64(d: &Document, key: &str) -> Option<i64> {
    d.get_i64(key).ok().or_else(|| d.get_i32(key).ok().map(i64::from))
}

// ───────────── AFK ─────────────
pub async fn is_afk(user_id: i64) -> Result<Option<String>> {
    Ok(col("afk")
        .find_one(doc! { "user_id": user_id })
        .await?
        .and_then(|d| d.get_str("reason").ok().map(String::from)))
}

pub async fn add_afk(user_id: i64, reason: &str) -> Result<()> {
    set_field("afk", "user_id", user_id, "reason", reason).await
}

pub async fn remove_afk(user_id: i64) -> Result<()> {
    col("afk").delete_one(doc! { "user_id": user_id }).await?;
    Ok(())
}

pub async fn get_afk_users() -> Result<Vec<Document>> {
    Ok(col("afk").find(doc! { "user_id": { "$gt": 0 } }).await?.try_collect().await?)
}

// ───────────── time formatting ─────────────
pub fn get_readable_time(mut seconds: i64) -> String {
    let suffix = ["s", "ᴍ", "ʜ", "ᴅᴀʏs"];
    let mut parts: Vec<String> = Vec::new();
    for count in 1..=4 {
        let (rem, res) = if count < 3 {
            (seconds / 60, seconds % 60)
        } else {
            (seconds / 24, seconds % 24)
        };
        if seconds == 0 && rem == 0 {
            break;
        }
        parts.push(format!("{}{}", res, suffix[parts.len()]));
        seconds = rem;
    }
    let mut out = String::new();
    if parts.len() == 4 {
        out.push_str(&parts.pop().unwrap());
        out.push_str(", ");
    }
    parts.reverse();
    out.push_str(&parts.join(":"));
    out
}

// ───────────── assistants ─────────────
/// Numbers (1..=5) of the assistants that were started successfully. Filled by `core::userbot`.
pub static ASSISTANTS: Lazy<Mutex<Vec<u8>>> = Lazy::new(Default::default);

pub fn assistants() -> Vec<u8> {
    ASSISTANTS.lock().unwrap().clone()
}

pub fn get_assistant_number(chat_id: i64) -> Option<u8> {
    ASSISTANT.get(&chat_id).map(|v| *v)
}

pub async fn set_assistant_new(chat_id: i64, number: u8) -> Result<()> {
    set_field("assistants", "chat_id", chat_id, "assistant", number as i32).await
}

/// Pick a random assistant for the chat, store it and return its number.
pub async fn set_calls_assistant(chat_id: i64) -> Result<u8> {
    let pick = *assistants()
        .choose(&mut rand::thread_rng())
        .ok_or_else(|| anyhow::anyhow!("no assistants available"))?;
    ASSISTANT.insert(chat_id, pick);
    set_assistant_new(chat_id, pick).await?;
    Ok(pick)
}

/// Same as Python's `get_assistant` / `group_assistant`: returns the assistant number for a chat.
pub async fn get_assistant(chat_id: i64) -> Result<u8> {
    let available = assistants();
    if let Some(n) = ASSISTANT.get(&chat_id).map(|v| *v) {
        if available.contains(&n) {
            return Ok(n);
        }
        return set_calls_assistant(chat_id).await;
    }
    match col("assistants").find_one(doc! { "chat_id": chat_id }).await? {
        None => set_calls_assistant(chat_id).await,
        Some(d) => {
            let n = doc_i64(&d, "assistant").unwrap_or(0) as u8;
            if available.contains(&n) {
                ASSISTANT.insert(chat_id, n);
                Ok(n)
            } else {
                set_calls_assistant(chat_id).await
            }
        }
    }
}

// ───────────── skip mode / upvotes ─────────────
pub async fn is_skipmode(chat_id: i64) -> Result<bool> {
    if let Some(v) = SKIPMODE.get(&chat_id).map(|v| *v) {
        if v {
            return Ok(true);
        }
    }
    let on = !exists("skipmode", doc! { "chat_id": chat_id }).await?;
    SKIPMODE.insert(chat_id, on);
    Ok(on)
}

pub async fn skip_on(chat_id: i64) -> Result<()> {
    SKIPMODE.insert(chat_id, true);
    col("skipmode").delete_one(doc! { "chat_id": chat_id }).await?;
    Ok(())
}

pub async fn skip_off(chat_id: i64) -> Result<()> {
    SKIPMODE.insert(chat_id, false);
    if !exists("skipmode", doc! { "chat_id": chat_id }).await? {
        col("skipmode").insert_one(doc! { "chat_id": chat_id }).await?;
    }
    Ok(())
}

pub async fn get_upvote_count(chat_id: i64) -> Result<i64> {
    if let Some(v) = COUNT.get(&chat_id).map(|v| *v) {
        return Ok(v);
    }
    match col("upcount").find_one(doc! { "chat_id": chat_id }).await? {
        None => Ok(5),
        Some(d) => {
            let m = doc_i64(&d, "mode").unwrap_or(5);
            COUNT.insert(chat_id, m);
            Ok(m)
        }
    }
}

pub async fn set_upvotes(chat_id: i64, mode: i64) -> Result<()> {
    COUNT.insert(chat_id, mode);
    set_field("upcount", "chat_id", chat_id, "mode", mode).await
}

// ───────────── autoend ─────────────
const AUTOEND_ID: i64 = 1234;

pub async fn is_autoend() -> Result<bool> {
    exists("autoend", doc! { "chat_id": AUTOEND_ID }).await
}

pub async fn autoend_on() -> Result<()> {
    col("autoend").insert_one(doc! { "chat_id": AUTOEND_ID }).await?;
    Ok(())
}

pub async fn autoend_off() -> Result<()> {
    col("autoend").delete_one(doc! { "chat_id": AUTOEND_ID }).await?;
    Ok(())
}

// ───────────── loop (memory only, like Python) ─────────────
pub fn get_loop(chat_id: i64) -> i64 {
    LOOP.get(&chat_id).map(|v| *v).unwrap_or(0)
}

pub fn set_loop(chat_id: i64, mode: i64) {
    LOOP.insert(chat_id, mode);
}

// ───────────── channel play mode ─────────────
pub async fn get_cmode(chat_id: i64) -> Result<Option<i64>> {
    if let Some(v) = CHANNEL_CONNECT.get(&chat_id).map(|v| *v) {
        return Ok(Some(v));
    }
    match col("cplaymode").find_one(doc! { "chat_id": chat_id }).await? {
        None => Ok(None),
        Some(d) => {
            let m = doc_i64(&d, "mode");
            if let Some(m) = m {
                CHANNEL_CONNECT.insert(chat_id, m);
            }
            Ok(m)
        }
    }
}

pub async fn set_cmode(chat_id: i64, mode: i64) -> Result<()> {
    CHANNEL_CONNECT.insert(chat_id, mode);
    set_field("cplaymode", "chat_id", chat_id, "mode", mode).await
}

// ───────────── string-valued per-chat settings ─────────────
async fn get_str_setting(
    cache: &DashMap<i64, String>,
    collection: &str,
    field: &str,
    chat_id: i64,
    default: &str,
) -> Result<String> {
    if let Some(v) = cache.get(&chat_id) {
        return Ok(v.clone());
    }
    let v = match col(collection).find_one(doc! { "chat_id": chat_id }).await? {
        Some(d) => d.get_str(field).map(String::from).unwrap_or_else(|_| default.to_string()),
        None => default.to_string(),
    };
    cache.insert(chat_id, v.clone());
    Ok(v)
}

pub async fn get_playtype(chat_id: i64) -> Result<String> {
    get_str_setting(&PLAYTYPE, "playtypedb", "mode", chat_id, "Everyone").await
}

pub async fn set_playtype(chat_id: i64, mode: &str) -> Result<()> {
    PLAYTYPE.insert(chat_id, mode.to_string());
    set_field("playtypedb", "chat_id", chat_id, "mode", mode).await
}

pub async fn get_playmode(chat_id: i64) -> Result<String> {
    get_str_setting(&PLAYMODE, "playmode", "mode", chat_id, "Direct").await
}

pub async fn set_playmode(chat_id: i64, mode: &str) -> Result<()> {
    PLAYMODE.insert(chat_id, mode.to_string());
    set_field("playmode", "chat_id", chat_id, "mode", mode).await
}

pub async fn get_lang(chat_id: i64) -> Result<String> {
    get_str_setting(&LANG, "language", "lang", chat_id, "en").await
}

pub async fn set_lang(chat_id: i64, lang: &str) -> Result<()> {
    LANG.insert(chat_id, lang.to_string());
    set_field("language", "chat_id", chat_id, "lang", lang).await
}

// ───────────── pause state (memory only) ─────────────
pub fn is_music_playing(chat_id: i64) -> bool {
    PAUSE.get(&chat_id).map(|v| *v).unwrap_or(false)
}

pub fn music_on(chat_id: i64) {
    PAUSE.insert(chat_id, true);
}

pub fn music_off(chat_id: i64) {
    PAUSE.insert(chat_id, false);
}

// ───────────── active voice/video chats (memory only) ─────────────
pub fn get_active_chats() -> Vec<i64> {
    ACTIVE.lock().unwrap().clone()
}

pub fn is_active_chat(chat_id: i64) -> bool {
    ACTIVE.lock().unwrap().contains(&chat_id)
}

pub fn add_active_chat(chat_id: i64) {
    let mut a = ACTIVE.lock().unwrap();
    if !a.contains(&chat_id) {
        a.push(chat_id);
    }
}

pub fn remove_active_chat(chat_id: i64) {
    ACTIVE.lock().unwrap().retain(|c| *c != chat_id);
}

pub fn get_active_video_chats() -> Vec<i64> {
    ACTIVE_VIDEO.lock().unwrap().clone()
}

pub fn is_active_video_chat(chat_id: i64) -> bool {
    ACTIVE_VIDEO.lock().unwrap().contains(&chat_id)
}

pub fn add_active_video_chat(chat_id: i64) {
    let mut a = ACTIVE_VIDEO.lock().unwrap();
    if !a.contains(&chat_id) {
        a.push(chat_id);
    }
}

pub fn remove_active_video_chat(chat_id: i64) {
    ACTIVE_VIDEO.lock().unwrap().retain(|c| *c != chat_id);
}

// ───────────── non-admin play permission ─────────────
pub async fn check_nonadmin_chat(chat_id: i64) -> Result<bool> {
    exists("adminauth", doc! { "chat_id": chat_id }).await
}

pub async fn is_nonadmin_chat(chat_id: i64) -> Result<bool> {
    if let Some(true) = NONADMIN.get(&chat_id).map(|v| *v) {
        return Ok(true);
    }
    let v = check_nonadmin_chat(chat_id).await?;
    NONADMIN.insert(chat_id, v);
    Ok(v)
}

pub async fn add_nonadmin_chat(chat_id: i64) -> Result<()> {
    NONADMIN.insert(chat_id, true);
    if !check_nonadmin_chat(chat_id).await? {
        col("adminauth").insert_one(doc! { "chat_id": chat_id }).await?;
    }
    Ok(())
}

pub async fn remove_nonadmin_chat(chat_id: i64) -> Result<()> {
    NONADMIN.insert(chat_id, false);
    if check_nonadmin_chat(chat_id).await? {
        col("adminauth").delete_one(doc! { "chat_id": chat_id }).await?;
    }
    Ok(())
}

// ───────────── on/off flags + maintenance ─────────────
pub async fn is_on_off(flag: i64) -> Result<bool> {
    exists("onoffper", doc! { "on_off": flag }).await
}

pub async fn add_on(flag: i64) -> Result<()> {
    if !is_on_off(flag).await? {
        col("onoffper").insert_one(doc! { "on_off": flag }).await?;
    }
    Ok(())
}

pub async fn add_off(flag: i64) -> Result<()> {
    if is_on_off(flag).await? {
        col("onoffper").delete_one(doc! { "on_off": flag }).await?;
    }
    Ok(())
}

/// `true` means maintenance mode is OFF-limits for users (same inverted semantics as the Python code).
pub async fn is_maintenance() -> Result<bool> {
    let empty = MAINTENANCE.lock().unwrap().is_empty();
    if empty {
        let got = exists("onoffper", doc! { "on_off": 1 }).await?;
        let mut m = MAINTENANCE.lock().unwrap();
        m.clear();
        if !got {
            m.push(2);
            Ok(true)
        } else {
            m.push(1);
            Ok(false)
        }
    } else {
        Ok(!MAINTENANCE.lock().unwrap().contains(&1))
    }
}

pub async fn maintenance_off() -> Result<()> {
    {
        let mut m = MAINTENANCE.lock().unwrap();
        m.clear();
        m.push(2);
    }
    add_off(1).await
}

pub async fn maintenance_on() -> Result<()> {
    {
        let mut m = MAINTENANCE.lock().unwrap();
        m.clear();
        m.push(1);
    }
    add_on(1).await
}

// ───────────── served users / chats ─────────────
pub async fn is_served_user(user_id: i64) -> Result<bool> {
    exists("tgusersdb", doc! { "user_id": user_id }).await
}

pub async fn get_served_users() -> Result<Vec<Document>> {
    Ok(col("tgusersdb").find(doc! { "user_id": { "$gt": 0 } }).await?.try_collect().await?)
}

pub async fn add_served_user(user_id: i64) -> Result<()> {
    if !is_served_user(user_id).await? {
        col("tgusersdb").insert_one(doc! { "user_id": user_id }).await?;
    }
    Ok(())
}

pub async fn get_served_chats() -> Result<Vec<Document>> {
    Ok(col("chats").find(doc! { "chat_id": { "$lt": 0 } }).await?.try_collect().await?)
}

pub async fn is_served_chat(chat_id: i64) -> Result<bool> {
    exists("chats", doc! { "chat_id": chat_id }).await
}

pub async fn add_served_chat(chat_id: i64) -> Result<()> {
    if !is_served_chat(chat_id).await? {
        col("chats").insert_one(doc! { "chat_id": chat_id }).await?;
    }
    Ok(())
}

// ───────────── chat blacklist ─────────────
pub async fn blacklisted_chats() -> Result<Vec<i64>> {
    collect_ids("blacklistChat", doc! { "chat_id": { "$lt": 0 } }, "chat_id").await
}

pub async fn blacklist_chat(chat_id: i64) -> Result<bool> {
    if !exists("blacklistChat", doc! { "chat_id": chat_id }).await? {
        col("blacklistChat").insert_one(doc! { "chat_id": chat_id }).await?;
        return Ok(true);
    }
    Ok(false)
}

pub async fn whitelist_chat(chat_id: i64) -> Result<bool> {
    if exists("blacklistChat", doc! { "chat_id": chat_id }).await? {
        col("blacklistChat").delete_one(doc! { "chat_id": chat_id }).await?;
        return Ok(true);
    }
    Ok(false)
}

// ───────────── auth users (notes map: name -> {auth_user_id, auth_name, admin_id, admin_name}) ─────────────
async fn get_authusers(chat_id: i64) -> Result<Document> {
    Ok(col("authuser")
        .find_one(doc! { "chat_id": chat_id })
        .await?
        .and_then(|d| d.get_document("notes").ok().cloned())
        .unwrap_or_default())
}

pub async fn get_authuser_names(chat_id: i64) -> Result<Vec<String>> {
    Ok(get_authusers(chat_id).await?.keys().cloned().collect())
}

pub async fn get_authuser(chat_id: i64, name: &str) -> Result<Option<Document>> {
    Ok(get_authusers(chat_id).await?.get_document(name).ok().cloned())
}

pub async fn save_authuser(chat_id: i64, name: &str, note: Document) -> Result<()> {
    let mut notes = get_authusers(chat_id).await?;
    notes.insert(name, note);
    set_field("authuser", "chat_id", chat_id, "notes", notes).await
}

pub async fn delete_authuser(chat_id: i64, name: &str) -> Result<bool> {
    let mut notes = get_authusers(chat_id).await?;
    if notes.remove(name).is_some() {
        set_field("authuser", "chat_id", chat_id, "notes", notes).await?;
        return Ok(true);
    }
    Ok(false)
}

// ───────────── gban ─────────────
pub async fn get_gbanned() -> Result<Vec<i64>> {
    collect_ids("gban", doc! { "user_id": { "$gt": 0 } }, "user_id").await
}

pub async fn is_gbanned_user(user_id: i64) -> Result<bool> {
    exists("gban", doc! { "user_id": user_id }).await
}

pub async fn add_gban_user(user_id: i64) -> Result<()> {
    if !is_gbanned_user(user_id).await? {
        col("gban").insert_one(doc! { "user_id": user_id }).await?;
    }
    Ok(())
}

pub async fn remove_gban_user(user_id: i64) -> Result<()> {
    if is_gbanned_user(user_id).await? {
        col("gban").delete_one(doc! { "user_id": user_id }).await?;
    }
    Ok(())
}

// ───────────── sudoers ─────────────
pub async fn get_sudoers() -> Result<Vec<i64>> {
    Ok(match col("sudoers").find_one(doc! { "sudo": "sudo" }).await? {
        None => vec![],
        Some(d) => d
            .get_array("sudoers")
            .map(|a| {
                a.iter()
                    .filter_map(|b| b.as_i64().or_else(|| b.as_i32().map(i64::from)))
                    .collect()
            })
            .unwrap_or_default(),
    })
}

async fn save_sudoers(list: Vec<i64>) -> Result<()> {
    col("sudoers")
        .update_one(doc! { "sudo": "sudo" }, doc! { "$set": { "sudoers": list } })
        .upsert(true)
        .await?;
    Ok(())
}

pub async fn add_sudo(user_id: i64) -> Result<bool> {
    let mut s = get_sudoers().await?;
    s.push(user_id);
    save_sudoers(s).await?;
    Ok(true)
}

pub async fn remove_sudo(user_id: i64) -> Result<bool> {
    let mut s = get_sudoers().await?;
    if let Some(pos) = s.iter().position(|u| *u == user_id) {
        s.remove(pos);
    }
    save_sudoers(s).await?;
    Ok(true)
}

// ───────────── blocked users ─────────────
pub async fn get_banned_users() -> Result<Vec<i64>> {
    collect_ids("blockedusers", doc! { "user_id": { "$gt": 0 } }, "user_id").await
}

pub async fn get_banned_count() -> Result<usize> {
    Ok(get_banned_users().await?.len())
}

pub async fn is_banned_user(user_id: i64) -> Result<bool> {
    exists("blockedusers", doc! { "user_id": user_id }).await
}

pub async fn add_banned_user(user_id: i64) -> Result<()> {
    if !is_banned_user(user_id).await? {
        col("blockedusers").insert_one(doc! { "user_id": user_id }).await?;
    }
    Ok(())
}

pub async fn remove_banned_user(user_id: i64) -> Result<()> {
    if is_banned_user(user_id).await? {
        col("blockedusers").delete_one(doc! { "user_id": user_id }).await?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn readable_time() {
        assert_eq!(get_readable_time(65), "1ᴍ:5s");
        assert_eq!(get_readable_time(3725), "1ʜ:2ᴍ:5s");
        assert_eq!(get_readable_time(90061), "1ᴅᴀʏs, 1ʜ:1ᴍ:1s");
    }
}
