//! Port of `strings/__init__.py`: loads every `strings/langs/*.yml`, with English as the fallback
//! for any key a translation is missing.

use anyhow::{Context, Result};
use once_cell::sync::OnceCell;
use serde_yaml::Value;
use std::collections::{BTreeMap, HashMap};
use std::path::Path;

pub type Lang = HashMap<String, String>;

static LANGS: OnceCell<BTreeMap<String, Lang>> = OnceCell::new();

fn load_file(path: &Path) -> Result<Lang> {
    let text = std::fs::read_to_string(path).with_context(|| format!("reading {}", path.display()))?;
    let value: Value = serde_yaml::from_str(&text).with_context(|| format!("parsing {}", path.display()))?;
    let mut out = Lang::new();
    if let Value::Mapping(map) = value {
        for (k, v) in map {
            let key = match k {
                Value::String(s) => s,
                other => format!("{other:?}"),
            };
            let val = match v {
                Value::String(s) => s,
                Value::Number(n) => n.to_string(),
                Value::Bool(b) => b.to_string(),
                Value::Null => String::new(),
                other => serde_yaml::to_string(&other)?.trim().to_string(),
            };
            out.insert(key, val);
        }
    }
    Ok(out)
}

/// Load all language files from `dir` (normally `./strings/langs`).
pub fn load(dir: impl AsRef<Path>) -> Result<()> {
    let dir = dir.as_ref();
    let en = load_file(&dir.join("en.yml"))?;
    let mut all: BTreeMap<String, Lang> = BTreeMap::new();

    for entry in std::fs::read_dir(dir).with_context(|| format!("reading {}", dir.display()))? {
        let path = entry?.path();
        if path.extension().and_then(|e| e.to_str()) != Some("yml") {
            continue;
        }
        let name = path.file_stem().unwrap().to_string_lossy().to_string();
        if name == "en" {
            continue;
        }
        let mut lang = load_file(&path)?;
        for (k, v) in &en {
            lang.entry(k.clone()).or_insert_with(|| v.clone());
        }
        if !lang.contains_key("name") {
            anyhow::bail!("There is some issue with the language file {name}.yml");
        }
        all.insert(name, lang);
    }
    all.insert("en".to_string(), en);
    let _ = LANGS.set(all);
    Ok(())
}

/// Strings for a language code, falling back to English for unknown codes.
pub fn get_string(lang: &str) -> &'static Lang {
    let langs = LANGS.get().expect("strings not loaded");
    langs.get(lang).unwrap_or_else(|| &langs["en"])
}

/// `code -> display name` for the language picker.
pub fn languages_present() -> BTreeMap<String, String> {
    LANGS
        .get()
        .expect("strings not loaded")
        .iter()
        .map(|(code, l)| (code.clone(), l.get("name").cloned().unwrap_or_default()))
        .collect()
}

/// Look up one key (`_["start_2"]` in Python) and fill positional `{}` placeholders in order.
pub fn t(lang: &str, key: &str, args: &[&str]) -> String {
    let mut text = get_string(lang).get(key).cloned().unwrap_or_else(|| key.to_string());
    for a in args {
        text = text.replacen("{}", a, 1);
    }
    text
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn loads_repo_languages_with_english_fallback() {
        let dir = concat!(env!("CARGO_MANIFEST_DIR"), "/../strings/langs");
        load(dir).expect("languages load");
        let langs = languages_present();
        assert!(langs.contains_key("en") && langs.contains_key("hi"));
        let en = get_string("en");
        for code in langs.keys() {
            let l = get_string(code);
            assert!(en.keys().all(|k| l.contains_key(k)), "{code} missing keys after fallback");
        }
    }
}
