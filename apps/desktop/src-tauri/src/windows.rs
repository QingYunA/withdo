//! The main window and the commands the frontend uses to move between windows.

use tauri::{AppHandle, Emitter, Manager};

use crate::panel;

pub const MAIN: &str = "main";
pub const NAVIGATE_EVENT: &str = "main-navigate";

/// Show and focus the main window. `hash` (such as `#/main/task/5`) tells the frontend what to display.
pub fn show_main(app: &AppHandle, hash: Option<String>) {
    let Some(window) = app.get_webview_window(MAIN) else {
        eprintln!("[adhd-todo] main window is missing");
        return;
    };
    let _ = window.show();
    let _ = window.unminimize();
    let _ = window.set_focus();
    if let Some(hash) = hash {
        let _ = app.emit_to(MAIN, NAVIGATE_EVENT, hash);
    }
}

#[tauri::command]
pub fn open_main(app: AppHandle, hash: Option<String>) {
    show_main(&app, hash);
}

#[tauri::command]
pub fn hide_panel(app: AppHandle) {
    panel::hide(&app);
}
