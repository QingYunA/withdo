mod panel;
mod settings;
mod shortcut;
mod sidecar;
mod tray;
mod windows;

use tauri::{ActivationPolicy, Manager, RunEvent, WindowEvent};
use tauri_plugin_global_shortcut::ShortcutState;

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_nspanel::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(
            tauri_plugin_global_shortcut::Builder::new()
                .with_handler(|app, _shortcut, event| {
                    if event.state == ShortcutState::Pressed {
                        panel::toggle(app);
                    }
                })
                .build(),
        )
        .manage(sidecar::Core::new())
        .manage(panel::BlurClock::default())
        .invoke_handler(tauri::generate_handler![
            sidecar::core_endpoint,
            shortcut::get_shortcut,
            shortcut::set_shortcut,
            windows::open_main,
            windows::hide_panel,
        ])
        .on_window_event(|window, event| {
            // Closing the main window hides it; the app lives in the menu bar until Quit.
            if let WindowEvent::CloseRequested { api, .. } = event {
                if window.label() == windows::MAIN {
                    api.prevent_close();
                    let _ = window.hide();
                }
            }
        })
        .setup(|app| {
            // No Dock icon, no app menu: this is a menu bar app.
            app.set_activation_policy(ActivationPolicy::Accessory);
            let handle = app.handle();
            handle.manage(shortcut::init(handle));
            panel::create(handle)?;
            tray::create(handle)?;
            sidecar::start(handle);
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|app, event| {
            if let RunEvent::Exit = event {
                app.state::<sidecar::Core>().shutdown();
            }
        });
}
