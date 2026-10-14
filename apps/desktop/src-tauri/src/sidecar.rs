//! Runs `adhd-core serve` as a sidecar and publishes where it listens.
//!
//! The Rust side holds no business logic: it starts the process, reads the one stdout line
//! `{"port":n,"token":"..."}`, and hands that to the frontend through `core_endpoint`.

use std::{collections::HashMap, sync::atomic::{AtomicBool, Ordering}, sync::Mutex, time::Duration};

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, Manager, Runtime, State};
use tauri_plugin_shell::{
    process::{CommandChild, CommandEvent},
    ShellExt,
};
use tokio::sync::watch;

/// Name of the sidecar as declared in `bundle.externalBin`.
const SIDECAR: &str = "adhd-core";
/// The sidecar is started once and restarted once; after that the UI shows an error.
const MAX_STARTS: usize = 2;
/// A cold start of the compiled binary can take over a second.
const READY_TIMEOUT: Duration = Duration::from_secs(30);

pub const CORE_ERROR_EVENT: &str = "core-error";

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct Endpoint {
    pub port: u16,
    pub token: String,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum CoreState {
    Starting,
    Ready(Endpoint),
    Failed(String),
}

/// Managed state: the current sidecar state plus its process handle.
pub struct Core {
    state: watch::Sender<CoreState>,
    child: Mutex<Option<CommandChild>>,
    shutting_down: AtomicBool,
}

impl Core {
    pub fn new() -> Self {
        Self {
            state: watch::channel(CoreState::Starting).0,
            child: Mutex::new(None),
            shutting_down: AtomicBool::new(false),
        }
    }

    /// Stop the sidecar and tell the supervisor not to restart it. Called when the app exits.
    pub fn shutdown(&self) {
        self.shutting_down.store(true, Ordering::SeqCst);
        if let Some(child) = self.child.lock().ok().and_then(|mut slot| slot.take()) {
            let _ = child.kill();
        }
    }

    fn is_shutting_down(&self) -> bool {
        self.shutting_down.load(Ordering::SeqCst)
    }

    fn publish(&self, state: CoreState) {
        self.state.send_replace(state);
    }

    fn set_child(&self, child: Option<CommandChild>) {
        if let Ok(mut slot) = self.child.lock() {
            *slot = child;
        }
    }
}

impl Default for Core {
    fn default() -> Self {
        Self::new()
    }
}

/// Parse the single ready line printed by `adhd-core serve`.
pub fn parse_ready_line(line: &[u8]) -> Option<Endpoint> {
    serde_json::from_slice(line.trim_ascii()).ok()
}

/// A restart reuses the previous port and token, so the frontend's cached endpoint stays valid
/// and its `EventSource` simply reconnects.
fn restart_env(previous: Option<&Endpoint>) -> HashMap<String, String> {
    previous
        .map(|e| {
            HashMap::from([
                ("ADHD_CORE_PORT".to_string(), e.port.to_string()),
                ("ADHD_CORE_TOKEN".to_string(), e.token.clone()),
            ])
        })
        .unwrap_or_default()
}

/// Wait until the sidecar is ready, or fail with its error. Used by the `core_endpoint` command.
pub async fn wait_ready(core: &Core) -> Result<Endpoint, String> {
    let mut rx = core.state.subscribe();
    loop {
        let settled = match &*rx.borrow_and_update() {
            CoreState::Ready(endpoint) => Some(Ok(endpoint.clone())),
            CoreState::Failed(message) => Some(Err(message.clone())),
            CoreState::Starting => None,
        };
        if let Some(result) = settled {
            return result;
        }
        rx.changed().await.map_err(|e| e.to_string())?;
    }
}

#[tauri::command]
pub async fn core_endpoint(core: State<'_, Core>) -> Result<Endpoint, String> {
    wait_ready(&core).await
}

/// Begin supervising the sidecar in the background.
pub fn start<R: Runtime>(app: &AppHandle<R>) {
    let handle = app.clone();
    tauri::async_runtime::spawn(async move { supervise(handle).await });
}

async fn supervise<R: Runtime>(app: AppHandle<R>) {
    let core = app.state::<Core>();
    let mut endpoint: Option<Endpoint> = None;
    let mut reason = String::from("adhd-core did not start");
    for attempt in 0..MAX_STARTS {
        if core.is_shutting_down() {
            return;
        }
        core.publish(CoreState::Starting);
        let outcome = run_once(&app, &core, endpoint.as_ref()).await;
        if core.is_shutting_down() {
            return;
        }
        if outcome.endpoint.is_some() {
            endpoint = outcome.endpoint;
        }
        reason = outcome.reason;
        eprintln!("[adhd-todo] adhd-core stopped (start {} of {MAX_STARTS}): {reason}", attempt + 1);
    }
    core.publish(CoreState::Failed(reason.clone()));
    let _ = app.emit(CORE_ERROR_EVENT, reason);
}

struct Outcome {
    /// Set when the process reached the ready line.
    endpoint: Option<Endpoint>,
    reason: String,
}

async fn run_once<R: Runtime>(app: &AppHandle<R>, core: &Core, previous: Option<&Endpoint>) -> Outcome {
    let spawned = app
        .shell()
        .sidecar(SIDECAR)
        .map(|command| command.args(["serve"]).envs(restart_env(previous)))
        .and_then(|command| command.spawn());
    let (mut events, child) = match spawned {
        Ok(pair) => pair,
        Err(error) => return Outcome { endpoint: None, reason: format!("cannot start {SIDECAR}: {error}") },
    };
    core.set_child(Some(child));

    let mut ready: Option<Endpoint> = None;
    let mut reason = String::from("adhd-core exited");
    let deadline = tokio::time::Instant::now() + READY_TIMEOUT;
    loop {
        let next = if ready.is_none() {
            match tokio::time::timeout_at(deadline, events.recv()).await {
                Ok(event) => event,
                Err(_) => {
                    reason = "adhd-core did not report a port in time".into();
                    break;
                }
            }
        } else {
            events.recv().await
        };
        match next {
            Some(CommandEvent::Stdout(line)) if ready.is_none() => {
                if let Some(endpoint) = parse_ready_line(&line) {
                    core.publish(CoreState::Ready(endpoint.clone()));
                    ready = Some(endpoint);
                }
            }
            Some(CommandEvent::Stderr(line)) => eprint!("[adhd-core] {}", String::from_utf8_lossy(&line)),
            Some(CommandEvent::Error(message)) => reason = message,
            Some(CommandEvent::Terminated(payload)) => {
                reason = format!("adhd-core exited (code {:?})", payload.code);
                break;
            }
            Some(_) => {}
            None => break,
        }
    }
    if let Some(child) = core.child.lock().ok().and_then(|mut slot| slot.take()) {
        let _ = child.kill();
    }
    Outcome { endpoint: ready, reason }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_the_ready_line() {
        let endpoint = parse_ready_line(b"{\"port\":52341,\"token\":\"abc\"}\n").unwrap();
        assert_eq!(endpoint, Endpoint { port: 52341, token: "abc".into() });
    }

    #[test]
    fn ignores_lines_that_are_not_the_ready_line() {
        assert_eq!(parse_ready_line(b"listening on 127.0.0.1\n"), None);
        assert_eq!(parse_ready_line(b""), None);
    }

    #[test]
    fn restart_reuses_port_and_token() {
        let env = restart_env(Some(&Endpoint { port: 4000, token: "t".into() }));
        assert_eq!(env.get("ADHD_CORE_PORT").map(String::as_str), Some("4000"));
        assert_eq!(env.get("ADHD_CORE_TOKEN").map(String::as_str), Some("t"));
        assert!(restart_env(None).is_empty());
    }
}
