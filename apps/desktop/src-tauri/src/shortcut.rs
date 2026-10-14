//! The global shortcut that toggles the panel: registered at start, changeable at runtime.

use std::{path::PathBuf, str::FromStr, sync::Mutex};

use tauri::{AppHandle, Manager, State};
use tauri_plugin_global_shortcut::{GlobalShortcutExt, Shortcut};

use crate::settings::{self, Settings};

/// The shortcut currently registered, as the string the user typed (so it can be shown and saved).
pub struct ShortcutState {
    current: Mutex<String>,
}

impl ShortcutState {
    pub fn new(initial: String) -> Self {
        Self { current: Mutex::new(initial) }
    }

    fn get(&self) -> String {
        self.current.lock().map(|s| s.clone()).unwrap_or_default()
    }

    fn set(&self, value: String) {
        if let Ok(mut slot) = self.current.lock() {
            *slot = value;
        }
    }
}

fn config_dir(app: &AppHandle) -> Result<PathBuf, String> {
    app.path().app_config_dir().map_err(|e| e.to_string())
}

pub fn parse(accelerator: &str) -> Result<Shortcut, String> {
    Shortcut::from_str(accelerator.trim()).map_err(|e| format!("{accelerator} is not a valid shortcut: {e}"))
}

/// Load the saved shortcut (or the default) and register it. A failure is logged, not fatal:
/// the tray icon still opens the panel, and Settings can pick another shortcut.
pub fn init(app: &AppHandle) -> ShortcutState {
    let settings = config_dir(app).map(|dir| settings::load(&dir)).unwrap_or_default();
    let state = ShortcutState::new(settings.shortcut.clone());
    if let Err(error) = parse(&settings.shortcut).and_then(|s| app.global_shortcut().register(s).map_err(|e| e.to_string())) {
        eprintln!("[adhd-todo] cannot register global shortcut {}: {error}", settings.shortcut);
    }
    state
}

#[tauri::command]
pub fn get_shortcut(state: State<'_, ShortcutState>) -> String {
    state.get()
}

/// Register `shortcut`, drop the old one and persist the choice. On failure the old shortcut stays.
#[tauri::command]
pub fn set_shortcut(app: AppHandle, state: State<'_, ShortcutState>, shortcut: String) -> Result<String, String> {
    let wanted = parse(&shortcut)?;
    let previous = state.get();
    if previous != shortcut {
        let registry = app.global_shortcut();
        registry.register(wanted).map_err(|e| format!("cannot use {shortcut}: {e}"))?;
        if let Ok(old) = parse(&previous) {
            let _ = registry.unregister(old);
        }
    }
    settings::save(&config_dir(&app)?, &Settings { shortcut: shortcut.clone() }).map_err(|e| format!("cannot save settings: {e}"))?;
    state.set(shortcut.clone());
    Ok(shortcut)
}

#[cfg(test)]
mod tests {
    use super::parse;

    #[test]
    fn accepts_every_accelerator_shape_the_settings_view_produces() {
        for accelerator in ["Alt+Space", "Command+Shift+K", "Control+Alt+3", "Alt+Up", "Command+F5", "Control+Enter", "Alt+Tab"] {
            assert!(parse(accelerator).is_ok(), "{accelerator} should parse");
        }
    }

    #[test]
    fn rejects_nonsense() {
        assert!(parse("").is_err());
        assert!(parse("Alt+").is_err());
        assert!(parse("Banana+K").is_err());
    }
}
