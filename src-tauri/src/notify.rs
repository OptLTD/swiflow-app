use dirs;
use notify_rust::Notification;
use std::fs;
use std::fs::OpenOptions;
use std::io::Read;
use std::thread;
use std::time::Duration;

pub fn monitor_and_notify() {
    let base = dirs::config_dir().expect("无法获取用户配置目录");
    let home = base.join("App.Swiflow");
    if !home.exists() {
        fs::create_dir_all(&home).expect("无法创建 App.Swiflow 目录");
    }
    let lock_path = home.join("notify.lock");
    if !lock_path.exists() {
        let _ = std::fs::File::create(&lock_path);
    }
    thread::spawn(move || loop {
        if lock_path.exists() {
            if let Ok(mut file) = OpenOptions::new().read(true).write(true).open(&lock_path) {
                let mut content = String::new();
                if file.read_to_string(&mut content).is_ok() && !content.trim().is_empty() {
                    let _ = Notification::new().summary("通知").body(&content).show();
                    let _ = file.set_len(0);
                }
            }
        }
        thread::sleep(Duration::from_secs(2));
    });
}
