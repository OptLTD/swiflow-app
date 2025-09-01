use tauri::plugin::{Builder, TauriPlugin};
use tauri::Runtime;

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("inject_js_plugin")
        .on_page_load(|window, _payload: &tauri::webview::PageLoadPayload| {
            if window.label() != "main" {
                return;
            }
            #[cfg(target_os = "macos")]
            {
                // 使用 eval 方法注入 JavaScript
                let script = include_str!("inject_js_mixed.js");
                window.eval(script).unwrap_or_else(|e| {
                    log::info!("inject-js-mixed error: {:?}", e);
                });
            }
        })
        // .on_webview_ready(|window| {
        // })
        .build()
}
