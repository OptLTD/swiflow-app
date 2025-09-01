use tauri::{AppHandle, Manager, async_runtime::spawn};
use tauri_plugin_deep_link::DeepLinkExt;

/// Setup deep link handling 
pub fn setup_deep_links(app: &tauri::App) -> Result<(), Box<dyn std::error::Error>> {
    #[cfg(any(target_os = "linux", all(debug_assertions, windows)))]
    {
        match app.deep_link().register_all() {
            Ok(_) => log::info!("[DeepLink] 深层链接协议注册成功"),
            Err(e) => {
                log::error!("[DeepLink] 深层链接协议注册失败: {}", e);
                return Err(e);
            }
        }
    }

    let app_handle = app.handle().clone();
    app.deep_link().on_open_url(move |event| {
        let urls = event.urls().to_vec();
        let handle = app_handle.clone();
        spawn(async move {
            if let Some(url) = urls.first() {
                log::info!("[DeepLink] 开始处理第一个 URL: {}", url);
                handle_deep_link_url(&handle, &url.to_string()).await;
            }
        });
    });
    Ok(())
}

/// 处理deep-link URL
async fn handle_deep_link_url(app: &AppHandle, url: &str) {
    log::debug!("[DeepLink] 开始处理 deep-link URL: {}", url);
    
    // 检查主窗口
    if let Some(window) = app.get_webview_window("main") {
        // 显示窗口
        match window.show() {
            Ok(_) => log::debug!("[DeepLink] 窗口显示成功"),
            Err(e) => log::warn!("[DeepLink] 窗口显示失败: {}", e),
        }
        
        // 聚焦窗口
        match window.set_focus() {
            Ok(_) => log::debug!("[DeepLink] 窗口聚焦成功"),
            Err(e) => log::warn!("[DeepLink] 窗口聚焦失败: {}", e),
        }

        let script = format!("window.dispatchEvent(
            new CustomEvent('dispatch', {{detail: {}}})
        );", serde_json::to_string(url).unwrap());
        match window.eval(&script) {
            Ok(_) => log::info!("[DeepLink] 执行脚本成功: {}", script),
            Err(e) => log::error!("[DeepLink] 执行脚本失败: {}", e),
        }
    }
    
    log::debug!("[DeepLink] Deep-link URL 处理完成: {}", url);
}
