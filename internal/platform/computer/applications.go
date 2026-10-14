package computer

import (
	"context"
	"strings"
)

type nativeWindow struct {
	ID     int    `json:"window_id"`
	PID    int32  `json:"pid"`
	Title  string `json:"title"`
	Bounds Rect   `json:"bounds"`
}

func (s *Session) Apps(ctx context.Context) ([]App, error) {
	s.driver.desktop.Lock()
	defer s.driver.desktop.Unlock()
	apps, err := s.runningApps(ctx)
	if err != nil || len(apps) == 0 {
		return apps, err
	}
	windows, err := s.windows(ctx, 0)
	if err != nil {
		return nil, err
	}
	for i := range apps {
		attachWindows(&apps[i], windows)
	}
	return apps, nil
}

func (s *Session) runningApps(ctx context.Context) ([]App, error) {
	var response struct {
		Apps []struct {
			PID     int32  `json:"pid"`
			Bundle  string `json:"bundle_id"`
			Name    string `json:"name"`
			Active  bool   `json:"active"`
			Running bool   `json:"running"`
		} `json:"apps"`
	}
	if _, err := s.read(ctx, "list_apps", map[string]any{}, &response); err != nil {
		return nil, err
	}
	var apps []App
	for _, a := range response.Apps {
		if !a.Running || a.PID <= 0 {
			continue
		}
		bundle, exe, err := processIdentity(a.PID, a.Bundle)
		if err != nil {
			continue
		}
		apps = append(apps, App{PID: a.PID, Bundle: bundle, Exe: exe, Name: a.Name, Active: a.Active})
	}
	return apps, nil
}

func (s *Session) windows(ctx context.Context, pid int32) ([]nativeWindow, error) {
	args := map[string]any{}
	if pid > 0 {
		args["pid"] = pid
	}
	var response struct {
		Windows []nativeWindow `json:"windows"`
	}
	if _, err := s.read(ctx, "list_windows", args, &response); err != nil {
		return nil, err
	}
	return response.Windows, nil
}

func attachWindows(app *App, windows []nativeWindow) {
	for _, w := range windows {
		if w.PID == app.PID && w.ID > 0 {
			app.Windows = append(app.Windows, Window{ID: w.ID, Title: w.Title, Bounds: w.Bounds})
		}
	}
}

func (s *Session) app(ctx context.Context, bundle string) (App, error) {
	bundle = strings.TrimSpace(bundle)
	if why := Refused(bundle); why != "" {
		return App{}, fail(CodeAppRefused, "%s is %s", bundle, why)
	}
	apps, err := s.runningApps(ctx)
	if err != nil {
		return App{}, err
	}
	var found []App
	for _, app := range apps {
		if app.Bundle == bundle {
			if why := Refused(app.Exe); why != "" {
				return App{}, fail(CodeAppRefused, "%s is %s", bundle, why)
			}
			found = append(found, app)
		}
	}
	if len(found) == 0 {
		return App{}, fail(CodeNoApp, "no running application has identity %q; use the exact identity returned by computer_read what=apps", bundle)
	}
	if len(found) > 1 {
		return App{}, fail(CodeNoApp, "application identity %q resolves to multiple processes", bundle)
	}
	app := found[0]
	windows, err := s.windows(ctx, app.PID)
	if err != nil {
		return App{}, err
	}
	attachWindows(&app, windows)
	return app, nil
}
