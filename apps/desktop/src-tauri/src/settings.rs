//! `settings.json` in the app config dir. Today it holds one value: the global shortcut.

use std::{fs, io, path::{Path, PathBuf}};

use serde::{Deserialize, Serialize};

pub const DEFAULT_SHORTCUT: &str = "Alt+Space";
const FILE_NAME: &str = "settings.json";

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct Settings {
    #[serde(default = "default_shortcut")]
    pub shortcut: String,
}

fn default_shortcut() -> String {
    DEFAULT_SHORTCUT.to_string()
}

impl Default for Settings {
    fn default() -> Self {
        Self { shortcut: default_shortcut() }
    }
}

fn file_path(dir: &Path) -> PathBuf {
    dir.join(FILE_NAME)
}

/// Read the settings; a missing, unreadable or malformed file yields the defaults.
pub fn load(dir: &Path) -> Settings {
    fs::read(file_path(dir))
        .ok()
        .and_then(|bytes| serde_json::from_slice(&bytes).ok())
        .unwrap_or_default()
}

/// Write the settings, creating the directory if needed.
pub fn save(dir: &Path, settings: &Settings) -> io::Result<()> {
    fs::create_dir_all(dir)?;
    let json = serde_json::to_vec_pretty(settings).map_err(io::Error::other)?;
    // Write to a sibling file first so a crash never leaves a half-written settings.json.
    let tmp = dir.join(format!("{FILE_NAME}.tmp"));
    fs::write(&tmp, json)?;
    fs::rename(tmp, file_path(dir))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn temp_dir(name: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!("adhd-settings-{name}-{}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        dir
    }

    #[test]
    fn missing_file_gives_the_default_shortcut() {
        let dir = temp_dir("missing");
        assert_eq!(load(&dir).shortcut, "Alt+Space");
    }

    #[test]
    fn saved_shortcut_is_read_back() {
        let dir = temp_dir("roundtrip");
        save(&dir, &Settings { shortcut: "Command+Shift+K".into() }).unwrap();
        assert_eq!(load(&dir).shortcut, "Command+Shift+K");
        fs::remove_dir_all(dir).unwrap();
    }

    #[test]
    fn malformed_or_partial_files_fall_back_to_defaults() {
        let dir = temp_dir("malformed");
        fs::create_dir_all(&dir).unwrap();
        fs::write(file_path(&dir), "not json").unwrap();
        assert_eq!(load(&dir), Settings::default());
        fs::write(file_path(&dir), "{}").unwrap();
        assert_eq!(load(&dir), Settings::default());
        fs::remove_dir_all(dir).unwrap();
    }
}
