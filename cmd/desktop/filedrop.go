package main

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/library/workspace"
)

func bindWorkspaceFileDrop(win *application.WebviewWindow, cfg config.Config) {
	win.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		uploaded, err := workspace.ImportFiles(cfg.WorkspaceDir, files)
		payload := map[string]any{
			"path":     workspace.UploadsRoot,
			"uploaded": uploaded,
		}
		if err != nil {
			payload["error"] = err.Error()
			slog.Warn("workspace file drop", "error", err, "files", len(files))
		} else {
			slog.Info("workspace file drop", "path", workspace.UploadsRoot, "count", len(uploaded))
		}
		application.Get().Event.Emit("workspace-uploaded", payload)
	})
}
