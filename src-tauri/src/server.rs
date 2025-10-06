use anyhow::Result;
use command_group::{CommandGroup, GroupChild};
use lazy_static::lazy_static;
use std::cmp::PartialEq;
use std::process::Command;
use std::sync::Mutex;
use tauri_plugin_shell::process::Command as TauriCommand;
use tauri_plugin_shell::process::CommandChild as TauriCommandChild;
use tauri_plugin_shell::ShellExt;

#[derive(Debug, PartialEq)]
pub enum ServerMode {
    #[allow(dead_code)]
    OneFile,
    MultiFile,
}

#[derive(Debug)]
struct WebServer {
    processes: Option<ProcessHandle>,
    mode: ServerMode,
}

#[derive(Debug)]
enum ProcessHandle {
    Group(GroupChild),
    Single(TauriCommandChild),
}

impl WebServer {
    fn new(mode: ServerMode) -> Self {
        Self {
            processes: None,
            mode,
        }
    }

    fn start(&mut self, app: &tauri::App) -> Result<()> {
        let handle = match self.mode {
            ServerMode::OneFile => {
                let group = Command::new("main")
                    .group_spawn()
                    .map_err(|e| anyhow::anyhow!("Failed to spawn 'main' group: {}", e))?;
                ProcessHandle::Group(group)
            }
            ServerMode::MultiFile => {
                let sidecar: TauriCommand = app
                    .shell()
                    .sidecar("main")
                    .map_err(|e| anyhow::anyhow!("Failed to get sidecar 'main': {}", e))?
                    .args(["-m", "serve", "-d", "com.option.swiflow"]);

                match sidecar.spawn() {
                    Ok((_, child)) => ProcessHandle::Single(child),
                    Err(e) => return Err(anyhow::anyhow!("Failed to spawn sidecar process: {}", e)),
                }
            }
        };

        self.processes = Some(handle);
        Ok(())
    }

    fn start_with_retry(&mut self, app: &tauri::App, retries: usize, delay_ms: u64) -> Result<()> {
        let mut last_err: Option<anyhow::Error> = None;
        for attempt in 1..=retries {
            match self.start(app) {
                Ok(()) => {
                    if attempt > 1 {
                        log::info!("[Server] Start succeeded on attempt {}", attempt);
                    }
                    return Ok(())
                }
                Err(e) => {
                    log::warn!("[Server] Start attempt {} failed: {}", attempt, e);
                    last_err = Some(e);
                    if attempt < retries {
                        std::thread::sleep(std::time::Duration::from_millis(delay_ms));
                    }
                }
            }
        }

        Err(anyhow::anyhow!(
            "Server failed to start after {} attempts: {}",
            retries,
            last_err
                .map(|e| e.to_string())
                .unwrap_or_else(|| "unknown error".to_string())
        ))
    }

    fn shutdown(&mut self) -> Result<()> {
        log::info!("[Server] Starting server shutdown process...");
        if let Some(processes) = self.processes.take() {
            log::info!("[Server] Found active processes, attempting to terminate...");
            match processes {
                ProcessHandle::Group(mut group) => {
                    log::info!("[Server] Killing process group...");
                    group.kill()?;
                    log::info!("[Server] Process group terminated successfully");
                }
                ProcessHandle::Single(single) => {
                    log::info!("[Server] Killing single process...");
                    single.kill()?;
                    log::info!("[Server] Single process terminated successfully");
                }
            }
        } else {
            log::info!("[Server] No active processes found to shutdown");
        }
        log::info!("[Server] Server shutdown completed");
        Ok(())
    }
}

// Create a global static instance wrapped in a Mutex
lazy_static! {
    static ref SERVER: Mutex<Option<WebServer>> = Mutex::new(None);
}

/// Initialize and run the server with the given mode and proper error handling
pub async fn run(app: &tauri::App, mode: ServerMode) -> Result<()> {
    // Lock the mutex and initialize the server if not already initialized
    let mut server_guard = SERVER
        .lock()
        .map_err(|e| anyhow::anyhow!("Failed to lock server: {}", e))?;

    if server_guard.is_none() {
        *server_guard = Some(WebServer::new(mode));
    }

    if let Some(server) = server_guard.as_mut() {
        server.start_with_retry(app, 3, 800)?;
    }

    Ok(())
}

/// Shutdown the server with proper error handling
pub async fn shutdown() -> Result<()> {
    log::info!("[Server] Global shutdown function called");
    // Lock the mutex and shutdown the server
    let mut server_guard = SERVER
        .lock()
        .map_err(|e| {
            log::error!("[Server] Failed to lock server mutex: {}", e);
            anyhow::anyhow!("Failed to lock server: {}", e)
        })?;

    if let Some(server) = server_guard.as_mut() {
        log::info!("[Server] Server instance found, calling shutdown...");
        server.shutdown()?
    } else {
        log::warn!("[Server] No server instance found to shutdown");
    }

    log::info!("[Server] Global shutdown function completed");
    Ok(())
}
