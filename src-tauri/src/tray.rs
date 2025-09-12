// Copyright 2019-2024 Tauri Programme within The Commons Conservancy
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: MIT

#![cfg(all(desktop, not(test)))]

use tauri::{
    menu::{Menu, MenuItem}, image::Image,
    tray::{MouseButton, TrayIconBuilder, TrayIconEvent},
    Manager, Runtime, WebviewUrl,
};

pub fn create_tray<R: Runtime>(app: &tauri::AppHandle<R>) -> tauri::Result<()> {
    let show_i = MenuItem::with_id(app, "show", "Show", true, None::<&str>)?;
    let quit_i = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&show_i, &quit_i])?;

    let bytes = include_bytes!("../icons/icon-tray.png");
    let _ = TrayIconBuilder::new().tooltip("Swiflow")
        .icon(Image::from_bytes(bytes)?)
        .menu(&menu).show_menu_on_left_click(false)
        .on_menu_event(move |app, event| match event.id.as_ref() {
            "quit" => {
                log::info!("[Tray] Exiting application...");
                app.exit(0);
            }
            "restart" => {
                use std::process::Command;
                let curr = std::env::current_exe().unwrap();
                log::info!("[Tray] Starting new instance: {:?}", curr);
                let _ = Command::new(curr).spawn().unwrap();
                log::info!("[Tray] Exiting current instance for restart...");
                app.exit(0);
            }
            "show" => {
                if let Some(window) = app.get_webview_window("main") {
                    log::info!("show window menu");
                    let _ = window.show().unwrap();
                    let _ = window.set_focus().unwrap();
                    #[cfg(target_os = "macos")]
                    let _ = app.set_dock_visibility(true);
                }
            }
            "new-window" => {
                let _webview = tauri::WebviewWindowBuilder::new(
                    app,
                    "new",
                    WebviewUrl::App("index.html".into()),
                )
                .title("Swiflow")
                .build()
                .unwrap();
            }
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                ..
            } = event
            {
                let app = tray.app_handle();
                #[cfg(target_os = "macos")]
                let _ = app.set_dock_visibility(true);
                if let Some(window) = app.get_webview_window("main") {
                    log::info!("show window click");
                    let _ = window.show().unwrap();
                    let _ = window.set_focus().unwrap();
                }
            }
        })
        .build(app);

    Ok(())
}
