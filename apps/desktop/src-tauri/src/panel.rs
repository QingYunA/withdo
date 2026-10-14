//! The quick panel: a non-activating NSPanel that floats over full-screen apps on every space.
//!
//! "Non-activating" means showing it does not make this app the active one, so the app you were
//! using keeps its focus and its full-screen space.

// The panel_event! macro expands `-> ()` literally.
#![allow(clippy::unused_unit)]

use std::{sync::{Arc, Mutex}, time::{Duration, Instant}};

use tauri::{AppHandle, Emitter, LogicalSize, Manager, PhysicalPosition, Size, WebviewUrl};
use tauri_nspanel::{tauri_panel, CollectionBehavior, ManagerExt, PanelBuilder, PanelLevel, StyleMask};

pub const LABEL: &str = "panel";
pub const SHOWN_EVENT: &str = "panel-shown";
const WIDTH: f64 = 640.0;
const HEIGHT: f64 = 480.0;
/// Where the panel's top edge sits on the screen, as a fraction of its height (Spotlight-like).
const TOP_FRACTION: f64 = 0.2;
/// A tray click first blurs (and so hides) the panel; ignore the toggle that follows within this window.
const TRAY_BLUR_GRACE: Duration = Duration::from_millis(250);

tauri_panel! {
    panel!(QuickPanel {
        config: {
            can_become_key_window: true,
            can_become_main_window: false,
            is_floating_panel: true
        }
    })

    panel_event!(QuickPanelEvents {
        window_did_resign_key(notification: &NSNotification) -> ()
    })
}

/// When the panel last hid itself because it lost focus.
#[derive(Default)]
pub struct BlurClock(Mutex<Option<Instant>>);

impl BlurClock {
    fn mark(&self) {
        if let Ok(mut slot) = self.0.lock() {
            *slot = Some(Instant::now());
        }
    }

    fn just_blurred(&self) -> bool {
        self.0.lock().ok().and_then(|slot| *slot).is_some_and(|at| at.elapsed() < TRAY_BLUR_GRACE)
    }
}

/// Create the (hidden) panel. Must run on the main thread, which `setup` does.
pub fn create(app: &AppHandle) -> tauri::Result<()> {
    let panel = PanelBuilder::<_, QuickPanel>::new(app, LABEL)
        .url(WebviewUrl::App("index.html#/panel".into()))
        .title("ADHD Todo")
        .size(Size::Logical(LogicalSize::new(WIDTH, HEIGHT)))
        .level(PanelLevel::Status)
        .collection_behavior(CollectionBehavior::new().can_join_all_spaces().full_screen_auxiliary().stationary())
        .add_style_mask(StyleMask::empty().nonactivating_panel())
        .has_shadow(true)
        .corner_radius(12.0)
        .no_activate(true)
        .with_window(|window| window.decorations(false).resizable(false).skip_taskbar(true).visible(false))
        .build()?;

    let handler = QuickPanelEvents::new();
    let weak = Arc::downgrade(&panel);
    let handle = app.clone();
    handler.window_did_resign_key(move |_| {
        if let Some(panel) = weak.upgrade() {
            if panel.is_visible() {
                handle.state::<BlurClock>().mark();
                panel.hide();
            }
        }
    });
    // The panel keeps the handler alive.
    panel.set_event_handler(Some(handler.as_ref()));
    Ok(())
}

/// Centre the panel horizontally near the top of the monitor the mouse is on.
fn place(app: &AppHandle, window: &tauri::WebviewWindow) {
    let monitor = app
        .cursor_position()
        .ok()
        .and_then(|cursor| app.monitor_from_point(cursor.x, cursor.y).ok().flatten());
    let Some(monitor) = monitor else {
        let _ = window.center();
        return;
    };
    let scale = monitor.scale_factor();
    let (origin, size) = (monitor.position(), monitor.size());
    let x = origin.x as f64 + (size.width as f64 - WIDTH * scale) / 2.0;
    let y = origin.y as f64 + size.height as f64 * TOP_FRACTION - HEIGHT * scale * 0.1;
    let _ = window.set_position(PhysicalPosition::new(x.round() as i32, y.round() as i32));
}

/// AppKit objects may only be touched from the main thread, but callers (a shortcut handler, a
/// command) cannot always promise to be on it. Hop there, whatever thread we are on.
fn on_main(app: &AppHandle, work: impl FnOnce(&AppHandle) + Send + 'static) {
    let handle = app.clone();
    if let Err(error) = app.run_on_main_thread(move || work(&handle)) {
        eprintln!("[adhd-todo] cannot reach the main thread: {error}");
    }
}

fn show_now(app: &AppHandle) {
    let Ok(panel) = app.get_webview_panel(LABEL) else {
        eprintln!("[adhd-todo] panel is missing");
        return;
    };
    if let Some(window) = panel.to_window() {
        place(app, &window);
    }
    panel.show_and_make_key();
    let _ = app.emit_to(LABEL, SHOWN_EVENT, ());
}

fn hide_now(app: &AppHandle) {
    if let Ok(panel) = app.get_webview_panel(LABEL) {
        panel.hide();
    }
}

fn is_visible(app: &AppHandle) -> bool {
    app.get_webview_panel(LABEL).is_ok_and(|panel| panel.is_visible())
}

pub fn hide(app: &AppHandle) {
    on_main(app, hide_now);
}

/// Global shortcut: show if hidden, hide if shown.
pub fn toggle(app: &AppHandle) {
    on_main(app, |app| {
        if is_visible(app) {
            hide_now(app);
        } else {
            show_now(app);
        }
    });
}

/// Tray click: like `toggle`, but a click that merely dismissed the panel must not reopen it.
pub fn toggle_from_tray(app: &AppHandle) {
    if app.state::<BlurClock>().just_blurred() {
        return;
    }
    toggle(app);
}
