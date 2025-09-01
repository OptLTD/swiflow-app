
# 打包脚本

https://www.topgoer.com/%E5%85%B6%E4%BB%96/%E8%B7%A8%E5%B9%B3%E5%8F%B0%E4%BA%A4%E5%8F%89%E7%BC%96%E8%AF%91.html


## docker 打包
```sh
# 打包
docker buildx build -f ./scripts/Dockerfile -t swiflow:latest ./
# push
docker push swiflow:latest
# pull
docker pull swiflow:latest
# run
## network mode
docker run -d --name swiflow --network host swiflow:latest
## port mode
docker run -d --name swiflow --port 112358:11235 swiflow:latest
```

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o swiflow -ldflags '-w -s' ./main.go && upx -9 swiflow
```

### 签名

### 公证

1. 构建应用：
```sh
yarn tauri build
```

2. 压缩应用：
```sh
# ditto -c -k --keepParent "target/release/bundle/macOS/YourApp.app" "YourApp.zip"
ditto -c -k --keepParent "target/aarch64-apple-darwin/release/bundle/macos/Swiflow.app" "Swiflow.zip"
```

3. 提交公证：
```sh
xcrun notarytool submit Swiflow.zip \
  --apple-id "$APPLE_ID" \
  --team-id "$APPLE_TEAM_ID" \
  --key-id "$APPLE_API_KEY" \
  --issuer "$APPLE_API_ISSUER" \
  --key "$APPLE_API_KEY_PATH" \
  --wait
```

```text
Conducting pre-submission checks for Swiflow.zip and initiating connection to the Apple notary service...
Submission ID received
  id: 5d6b0a92-48a6-4630-9dc2-5ecc47dbf08e
Upload progress: 100.00% (9.48 MB of 9.48 MB)
Successfully uploaded file
  id: 5d6b0a92-48a6-4630-9dc2-5ecc47dbf08e
  path: /Users/chenwen/XCode/swiflow-app/src-tauri/Swiflow.zip
Waiting for processing to complete.
Current status: Accepted........
Processing complete
  id: 5d6b0a92-48a6-4630-9dc2-5ecc47dbf08e
  status: Accepted
```

4. 
钉书机(Staple)公证票证：
```sh
xcrun stapler staple "target/aarch64-apple-darwin/release/bundle/macos/Swiflow.app"
```

-----

# windows 打包

How to use this image
Go to the root folder of your Tauri project en run the following command to enter the container:

```sh
# 推出后销毁
docker run --rm -it -v $(pwd):/io -w /io websmurf/tauri-builder:1.1.0 bash
# 退出后不销毁
docker run -it -v $(pwd):/io -w /io --name tauri-builder websmurf/tauri-builder:1.1.0 bash
# 重新进入docker 1
docker start -ai tauri-builder
# 重新进入docker 2
docker exec -it tauri-builder bash
```
Then run the following commands to build the windows version of your Tauri application:
```sh
rm -Rf node_modules # 可以不要
yarn install
# 正常打包，记得关代理
yarn tauri build --runner cargo-xwin --target x86_64-pc-windows-msvc
# 跳过前端编译
yarn tauri build --runner cargo-xwin --target x86_64-pc-windows-msvc --config '{"build": {"beforeBuildCommand": "echo skip"}}'
# debug包，跳过前端编译，会有黑窗
yarn tauri build --runner cargo-xwin --target x86_64-pc-windows-msvc --debug --config '{"build": {"beforeBuildCommand": "echo skip"}}'
```
