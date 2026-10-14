//! Menu bar icon. Left click toggles the panel; right click opens the menu.

use tauri::{
    image::Image,
    menu::{Menu, MenuItem, PredefinedMenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    AppHandle,
};

use crate::{panel, windows};

const OPEN: &str = "open";
const SETTINGS: &str = "settings";
const QUIT: &str = "quit";
const SETTINGS_HASH: &str = "#/main/settings";

pub fn create(app: &AppHandle) -> tauri::Result<()> {
    let menu = Menu::with_items(
        app,
        &[
            &MenuItem::with_id(app, OPEN, "Open main window", true, None::<&str>)?,
            &MenuItem::with_id(app, SETTINGS, "Settings...", true, None::<&str>)?,
            &PredefinedMenuItem::separator(app)?,
            &MenuItem::with_id(app, QUIT, "Quit ADHD Todo", true, None::<&str>)?,
        ],
    )?;
    TrayIconBuilder::with_id("tray")
        .icon(Image::from_bytes(include_bytes!("../icons/tray.png"))?)
        .icon_as_template(true)
        .tooltip("ADHD Todo")
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| match event.id().as_ref() {
            OPEN => windows::show_main(app, Some("#/main".into())),
            SETTINGS => windows::show_main(app, Some(SETTINGS_HASH.into())),
            QUIT => app.exit(0),
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click { button: MouseButton::Left, button_state: MouseButtonState::Up, .. } = event {
                panel::toggle_from_tray(tray.app_handle());
            }
        })
        .build(app)?;
    Ok(())
}
