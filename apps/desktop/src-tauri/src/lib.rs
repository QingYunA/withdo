mod sidecar;

use tauri::{Manager, RunEvent};

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_opener::init())
        .manage(sidecar::Core::new())
        .invoke_handler(tauri::generate_handler![sidecar::core_endpoint])
        .setup(|app| {
            sidecar::start(app.handle());
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
