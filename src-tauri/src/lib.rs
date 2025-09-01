use server::ServerMode;
use tauri::async_runtime::block_on;
use tauri::{Manager, RunEvent};
use tauri_plugin_decorum::WebviewWindowExt;

mod notify;
mod plugins;
mod server;
mod tray;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    #[allow(unused_mut)]
    let builder = tauri::Builder::default()
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_http::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_decorum::init())
        .plugin(tauri_plugin_deep_link::init())
        .plugin(plugins::inject_js_plugin::init()) // 注册插件
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .setup(|app| {
            if cfg!(debug_assertions) {
                app.handle().plugin(
                    tauri_plugin_log::Builder::default()
                        .level(log::LevelFilter::Info)
                        .build(),
                )?;
            }

            #[cfg(all(desktop, not(test)))]
            {
                let handle = app.handle();
                tray::create_tray(handle)?;
            }

            let main_window = app.get_webview_window("main").unwrap();
            #[cfg(target_os = "macos")]
            let _ = main_window.set_traffic_lights_inset(18.0, 22.0).unwrap();
            // Linux & windows avaliable
            let _ = main_window.create_overlay_titlebar().unwrap();

            // 初始化deep-link功能
            if let Err(e) = plugins::deep_link::setup_deep_links(&app) {
                log::info!("Failed to initialize deep link: {}", e);
            }

            // fix env to start server
            let _ = plugins::fix_path_env::fix();
            block_on(server::run(app, ServerMode::MultiFile))?;

            // 监控 notify.lock
            notify::monitor_and_notify();
            Ok(())
        });

    #[allow(unused_mut)]
    let mut app = builder
        .invoke_handler(tauri::generate_handler![])
        .build(tauri::generate_context!())
        .expect("error while building tauri application");

    #[cfg(target_os = "macos")]
    app.set_activation_policy(tauri::ActivationPolicy::Regular);

    app.run(move |_app_handle, _event| {
        #[cfg(all(desktop, not(test)))]
        match &_event {
            RunEvent::ExitRequested { api, code, .. } => {
                // Keep the event loop running even if all windows are closed
                // This allow us to catch tray icon events when there is no window
                // if we manually requested an exit (code is Some(_)) we will let it go through
                if code.is_none() {
                    api.prevent_exit();
                }
                let _ = block_on(server::shutdown());
                log::info!("ExitRequested...");
            }
            RunEvent::WindowEvent {
                event: tauri::WindowEvent::CloseRequested { api, .. },
                label,
                ..
            } => {
                log::info!("CloseRequested...");
                // run the window destroy manually just for fun :)
                // usually you'd show a dialog here to ask for confirmation or whatever
                api.prevent_close();
                let _ = _app_handle
                    .get_webview_window(label)
                    .unwrap()
                    .hide()
                    .unwrap();
                #[cfg(target_os = "macos")]
                let _ = _app_handle.set_dock_visibility(false);
            }
            _ => (),
        }
    })
}
