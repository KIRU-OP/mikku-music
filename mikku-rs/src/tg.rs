//! Thin shared helpers over `ferogram` used by every plugin.
//! (Keyboards: use `ferogram::keyboard::{InlineKeyboard, Button}`; this module wraps the common cases.)

use ferogram::keyboard::{Button, InlineKeyboard};
use ferogram::update::{CallbackQuery, IncomingMessage};
use ferogram::{tl, Client, InputMessage};

pub type Markup = tl::enums::ReplyMarkup;

/// Callback button.
pub fn cb(text: impl Into<String>, data: impl Into<String>) -> Button {
    Button::callback(text, data.into().into_bytes())
}

/// URL button.
pub fn url(text: impl Into<String>, url: impl Into<String>) -> Button {
    Button::url(text, url)
}

/// Build an inline keyboard from rows of buttons.
pub fn kb(rows: Vec<Vec<Button>>) -> Markup {
    let mut k = InlineKeyboard::new();
    for r in rows {
        k = k.row(r);
    }
    k.into_markup()
}

/// Telegram user id of the sender (None for anonymous admins / channels).
pub fn sender_id(msg: &IncomingMessage) -> Option<i64> {
    msg.sender_user_id()
}

/// Reply with HTML text.
pub async fn reply_html(msg: &IncomingMessage, text: impl AsRef<str>) -> Option<IncomingMessage> {
    msg.reply(InputMessage::html(text)).await.ok()
}

/// Reply with HTML text and an inline keyboard.
pub async fn reply_kb(msg: &IncomingMessage, text: impl AsRef<str>, markup: Markup) -> Option<IncomingMessage> {
    msg.reply(InputMessage::html(text).reply_markup(markup)).await.ok()
}

/// Send HTML text (optionally with keyboard) to a chat id.
pub async fn send_html(
    client: &Client,
    chat_id: i64,
    text: impl AsRef<str>,
    markup: Option<Markup>,
) -> Option<IncomingMessage> {
    let mut m = InputMessage::html(text);
    if let Some(k) = markup {
        m = m.reply_markup(k);
    }
    client.send_message(chat_id, m).await.ok()
}

/// Toast answer for a callback query.
pub async fn toast(client: &Client, q: &CallbackQuery, text: impl Into<String>) {
    let _ = q.answer().text(text).send(client).await;
}

/// Modal alert answer for a callback query.
pub async fn alert(client: &Client, q: &CallbackQuery, text: impl Into<String>) {
    let _ = q.answer().alert(text).send(client).await;
}

/// Escape text for Telegram HTML parse mode.
pub fn esc(s: &str) -> String {
    s.replace('&', "&amp;").replace('<', "&lt;").replace('>', "&gt;")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn escapes_html() {
        assert_eq!(esc("<a & b>"), "&lt;a &amp; b&gt;");
    }
}
