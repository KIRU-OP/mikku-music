//! Port of `config.py`. All values are read from the environment (`.env` is loaded first).

use once_cell::sync::Lazy;
use std::env;

fn get(name: &str) -> Option<String> {
    env::var(name).ok().filter(|v| !v.trim().is_empty())
}

fn get_or(name: &str, default: &str) -> String {
    get(name).unwrap_or_else(|| default.to_string())
}

fn get_int<T: std::str::FromStr>(name: &str, default: T) -> T {
    get(name).and_then(|v| v.trim().parse().ok()).unwrap_or(default)
}

/// "true", "1", "yes", "on" are truthy, everything else is false.
pub fn str_to_bool(value: Option<&str>, default: bool) -> bool {
    match value {
        None => default,
        Some(v) => matches!(v.trim().to_lowercase().as_str(), "true" | "1" | "yes" | "on"),
    }
}

/// "hh:mm:ss" / "mm:ss" / "ss" -> seconds
pub fn time_to_seconds(time: &str) -> i64 {
    time.split(':')
        .rev()
        .enumerate()
        .map(|(i, x)| x.trim().parse::<i64>().unwrap_or(0) * 60i64.pow(i as u32))
        .sum()
}

pub struct Config {
    pub api_id: i32,
    pub api_hash: String,
    pub bot_token: String,
    pub mongo_db_uri: String,
    pub owner_id: i64,
    pub logger_id: i64,

    pub support_channel: String,
    pub support_chat: String,
    pub update_channel: String,
    pub policy_link: String,

    pub ytproxy_url: String,
    pub yt_api_key: String,
    pub ytdl_cookies: String,
    pub cookies_url: Option<String>,
    pub downloads_dir: String,

    pub esse_api_url: String,
    pub esse_api_key: String,
    pub esse_timeout: u64,

    /// No hard-coded fallback: set these in `.env` (see README).
    pub spotify_client_id: String,
    pub spotify_client_secret: String,
    pub playlist_fetch_limit: usize,

    pub duration_limit_min: i64,
    pub duration_limit: i64,
    pub assistant_leave_time: u64,
    pub cache_duration: u64,
    pub cache_sleep: u64,
    pub tg_audio_filesize_limit: u64,
    pub tg_video_filesize_limit: u64,
    pub private_bot_mode_mem: i64,

    pub auto_leaving_assistant: bool,
    pub auto_end_stream: bool,

    /// Up to five assistant (userbot) session strings.
    pub strings: [Option<String>; 5],

    pub heroku_app_name: Option<String>,
    pub heroku_api_key: Option<String>,
    pub upstream_repo: String,
    /// The repository's real default branch is `main` (the Python default `Master` was wrong).
    pub upstream_branch: String,
    pub git_token: Option<String>,

    pub start_img_url: String,
    pub ping_img_url: String,
    pub playlist_img_url: String,
    pub stats_img_url: String,
    pub telegram_audio_url: String,
    pub telegram_video_url: String,
    pub stream_img_url: String,
    pub soundcloud_img_url: String,
    pub youtube_img_url: String,
    pub spotify_artist_img_url: String,
    pub spotify_album_img_url: String,
    pub spotify_playlist_img_url: String,

    /// Worker fallback API used by the YouTube platform.
    pub worker_fallback_api_url: String,
    pub worker_fallback_api_key: String,
}

fn validate_url(name: &str, value: &str) {
    if !value.is_empty() && !(value.starts_with("http://") || value.starts_with("https://")) {
        eprintln!("[ERROR] - {name} URL must start with http/https.");
        std::process::exit(1);
    }
}

fn required(name: &str) -> String {
    match get(name) {
        Some(v) => v,
        None => {
            eprintln!("[ERROR] - required variable {name} is missing (set it in .env).");
            std::process::exit(1);
        }
    }
}

fn required_int<T: std::str::FromStr>(name: &str) -> T {
    match required(name).trim().parse() {
        Ok(v) => v,
        Err(_) => {
            eprintln!("[ERROR] - variable {name} must be a number.");
            std::process::exit(1);
        }
    }
}

impl Config {
    fn load() -> Self {
        let _ = dotenvy::dotenv();

        let duration_limit_min: i64 = get_int("DURATION_LIMIT", 300);
        let img = |name: &str, default: &str| get_or(name, default);
        let default_img = "https://litter.catbox.moe/0pizzr.jpg";

        let cfg = Config {
            api_id: required_int("API_ID"),
            api_hash: required("API_HASH"),
            bot_token: required("BOT_TOKEN"),
            mongo_db_uri: required("MONGO_DB_URI"),
            owner_id: required_int("OWNER_ID"),
            logger_id: required_int("LOGGER_ID"),

            support_channel: get_or("SUPPORT_CHANNEL", "https://t.me/about_deadly_venom"),
            support_chat: get_or("SUPPORT_CHAT", "https://t.me/group_voice_chat"),
            update_channel: get_or("UPDATE_CHANNEL", "https://t.me/NEXO_BOT_SUPPORT"),
            policy_link: get_or(
                "POLICY_LINK",
                "https://venom-profile.shivampatel425685.workers.dev/",
            ),

            // https default (the Python default was a plain-http IP address).
            ytproxy_url: get_or("YTPROXY_URL", "https://x.spotifytech.shop"),
            yt_api_key: get_or("YT_API_KEY", ""),
            ytdl_cookies: get_or("YTDL_COOKIES", "cookies.txt"),
            cookies_url: get("COOKIES_URL"),
            downloads_dir: get_or("DOWNLOADS_DIR", "downloads"),

            esse_api_url: get_or("ESSE_API_URL", "https://esse.skybotsdeveloper.workers.dev"),
            esse_api_key: get_or("ESSE_API_KEY", ""),
            esse_timeout: get_int("ESSE_TIMEOUT", 20),

            spotify_client_id: get_or("SPOTIFY_CLIENT_ID", ""),
            spotify_client_secret: get_or("SPOTIFY_CLIENT_SECRET", ""),
            playlist_fetch_limit: get_int("PLAYLIST_FETCH_LIMIT", 25),

            duration_limit_min,
            duration_limit: time_to_seconds(&format!("{duration_limit_min}:00")),
            assistant_leave_time: get_int("ASSISTANT_LEAVE_TIME", 5400),
            cache_duration: get_int("CACHE_DURATION", 86400),
            cache_sleep: get_int("CACHE_SLEEP", 3600),
            tg_audio_filesize_limit: get_int("TG_AUDIO_FILESIZE_LIMIT", 204_857_600),
            tg_video_filesize_limit: get_int("TG_VIDEO_FILESIZE_LIMIT", 2_073_741_824),
            private_bot_mode_mem: get_int("PRIVATE_BOT_MODE_MEM", 1),

            auto_leaving_assistant: str_to_bool(get("AUTO_LEAVING_ASSISTANT").as_deref(), false),
            auto_end_stream: str_to_bool(get("AUTO_END_STREAM").as_deref(), false),

            strings: [
                get("STRING_SESSION"),
                get("STRING_SESSION2"),
                get("STRING_SESSION3"),
                get("STRING_SESSION4"),
                get("STRING_SESSION5"),
            ],

            heroku_app_name: get("HEROKU_APP_NAME"),
            heroku_api_key: get("HEROKU_API_KEY"),
            upstream_repo: get_or("UPSTREAM_REPO", "https://github.com/KIRU-OP/mikku-music"),
            upstream_branch: get_or("UPSTREAM_BRANCH", "main"),
            git_token: get("GIT_TOKEN"),

            start_img_url: img("START_IMG_URL", "https://litter.catbox.moe/ja82tm.mp4"),
            ping_img_url: img("PING_IMG_URL", "https://litter.catbox.moe/sww1by.jpg"),
            playlist_img_url: img("PLAYLIST_IMG_URL", "https://litter.catbox.moe/0pizzr.jpg"),
            stats_img_url: img("STATS_IMG_URL", "https://litter.catbox.moe/0pizzr.jpg"),
            telegram_audio_url: img("TELEGRAM_AUDIO_URL", "https://files.catbox.moe/fahcob.jpg"),
            telegram_video_url: img("TELEGRAM_VIDEO_URL", default_img),
            stream_img_url: img("STREAM_IMG_URL", default_img),
            soundcloud_img_url: img("SOUNCLOUD_IMG_URL", default_img),
            youtube_img_url: img("YOUTUBE_IMG_URL", default_img),
            spotify_artist_img_url: img("SPOTIFY_ARTIST_IMG_URL", default_img),
            spotify_album_img_url: img("SPOTIFY_ALBUM_IMG_URL", default_img),
            spotify_playlist_img_url: img("SPOTIFY_PLAYLIST_IMG_URL", default_img),

            worker_fallback_api_url: get_or(
                "WORKER_FALLBACK_API_URL",
                "https://yt-api-hub-repo.shivampatel425685.workers.dev",
            ),
            // No hard-coded default key: it must come from the environment.
            worker_fallback_api_key: get_or("WORKER_FALLBACK_API_KEY", ""),
        };

        validate_url("SUPPORT_CHANNEL", &cfg.support_channel);
        validate_url("SUPPORT_CHAT", &cfg.support_chat);
        validate_url("UPDATE_CHANNEL", &cfg.update_channel);
        validate_url("POLICY_LINK", &cfg.policy_link);
        cfg
    }

    /// Session strings that are actually set, with their 1-based assistant number.
    pub fn assistant_sessions(&self) -> Vec<(u8, &str)> {
        self.strings
            .iter()
            .enumerate()
            .filter_map(|(i, s)| s.as_deref().map(|s| ((i + 1) as u8, s)))
            .collect()
    }
}

pub static CONFIG: Lazy<Config> = Lazy::new(Config::load);

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_time() {
        assert_eq!(time_to_seconds("300:00"), 18000);
        assert_eq!(time_to_seconds("1:02:03"), 3723);
        assert_eq!(time_to_seconds("45"), 45);
    }

    #[test]
    fn parses_bool() {
        assert!(str_to_bool(Some("Yes"), false));
        assert!(!str_to_bool(Some("nope"), true));
        assert!(str_to_bool(None, true));
    }
}
