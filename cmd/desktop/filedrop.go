package main

import (
	"log/slog"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/library/workspace"
)

func bindWorkspaceFileDrop(win *application.WebviewWindow, cfg config.Config) {
	win.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		dir := "."
		if details := event.Context().DropTargetDetails(); details != nil {
			if p := strings.TrimSpace(details.Attributes["data-upload-path"]); p != "" {
				dir = p
			}
		}

		uploaded, err := workspace.ImportFiles(cfg.WorkspaceDir, dir, files)
		payload := map[string]any{
			"path":     dir,
			"uploaded": uploaded,
		}
		if err != nil {
			payload["error"] = err.Error()
			slog.Warn("workspace file drop", "error", err, "files", len(files))
		} else {
			slog.Info("workspace file drop", "path", dir, "count", len(uploaded))
		}
		application.Get().Event.Emit("workspace-uploaded", payload)
	})
}
