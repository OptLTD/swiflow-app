# build && dev

## move go.mod -> app/go.mod

1. config.yml
```yaml
dev_mode:
-  root_path: .
+  root_path: app
```

2. build/Taskfile.yml
```yaml
 tasks:
   go:mod:tidy:
+    dir: app
   generate:bindings:
+    dir: app
```

3. build/darwin/Taskfile.yml
```yaml
 tasks:
   build:
+    dir: app
   vars:
-      DEFAULT_OUTPUT: '{{.BIN_DIR}}/{{.APP_NAME}}'
+      DEFAULT_OUTPUT: '../{{.BIN_DIR}}/{{.APP_NAME}}'
```