// Package main — app.go contains all Wails-bound methods that the React frontend
// calls via the auto-generated TypeScript wrappers in frontend/wailsjs/.
//
// SECURITY: Passwords are received as strings from the frontend, immediately
// converted to []byte, used for key derivation, and then zeroed from memory.
// No password or key material is ever logged, stored, or returned to the frontend.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"personal-secure-encrypter/internal/encrypter"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the primary Wails-bound struct. Every exported method on App
// becomes callable from the frontend JavaScript/TypeScript code.
type App struct {
	ctx      context.Context
	engine   *encrypter.Engine
	settings *encrypter.Settings
}

// OperationResult is returned to the frontend after encrypt/decrypt operations
// so the UI can display a summary (successes, failures, timing).
type OperationResult struct {
	Success     int      `json:"success"`
	Failed      int      `json:"failed"`
	Total       int      `json:"total"`
	Errors      []string `json:"errors"`
	ElapsedTime string   `json:"elapsedTime"`
}

// ProgressEvent is emitted via Wails runtime events so the frontend can
// update a live progress bar during bulk operations.
type ProgressEvent struct {
	Current  int     `json:"current"`
	Total    int     `json:"total"`
	Filename string  `json:"filename"`
	Percent  float64 `json:"percent"`
	Speed    string  `json:"speed"`
	ETA      string  `json:"eta"`
}

// NewApp creates a new App instance, loading saved settings from disk.
func NewApp() *App {
	settings := encrypter.LoadSettings()
	return &App{
		settings: settings,
		engine:   encrypter.NewEngine(settings),
	}
}

// startup is called by Wails when the application starts.
// It stores the context used for runtime operations (dialogs, events).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// domReady is called when the frontend DOM is fully loaded.
func (a *App) domReady(ctx context.Context) {
	// Frontend is ready — no additional setup needed.
}

// ---------------------------------------------------------------------------
// File / Folder Selection Dialogs
// ---------------------------------------------------------------------------

// SelectFiles opens a native multi-file selection dialog.
func (a *App) SelectFiles() ([]string, error) {
	files, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Files to Process",
	})
	if err != nil {
		return nil, fmt.Errorf("dialog error")
	}
	return files, nil
}

// SelectFolder opens a native folder selection dialog.
func (a *App) SelectFolder() (string, error) {
	folder, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Folder",
	})
	if err != nil {
		return "", fmt.Errorf("dialog error")
	}
	return folder, nil
}

// SelectOutputFolder opens a native folder selection dialog for setting
// a custom output directory.
func (a *App) SelectOutputFolder() (string, error) {
	folder, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Output Folder",
	})
	if err != nil {
		return "", fmt.Errorf("dialog error")
	}
	return folder, nil
}

// ---------------------------------------------------------------------------
// Encryption
// ---------------------------------------------------------------------------

// EncryptFiles encrypts a list of files with the given password.
// Progress events are emitted to the frontend in real time.
// SECURITY: The password is zeroed from memory after key derivation.
func (a *App) EncryptFiles(paths []string, password string) *OperationResult {
	// Server-side password validation — never trust the frontend alone.
	if len(password) < 8 {
		return &OperationResult{
			Total:  0,
			Errors: []string{"password must be at least 8 characters"},
			Failed: 1,
		}
	}

	start := time.Now()
	result := &OperationResult{Total: len(paths), Errors: make([]string, 0)}

	// Convert password to bytes and defer zeroing.
	// NOTE: The original `password` string cannot be zeroed (Go strings are
	// immutable). This is a known Go limitation; the []byte copy is zeroed.
	pw := []byte(password)
	defer encrypter.ClearBytes(pw)

	for i, path := range paths {
		a.emitProgress(i, len(paths), filepath.Base(path), start)

		outputPath, err := a.engine.EncryptFile(path, pw)
		if err != nil {
			result.Failed++
			// Report only the filename — never include passwords or key material.
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s: encryption failed", filepath.Base(path)))
		} else {
			result.Success++
			// Optionally delete the original after successful encryption.
			// Verify the encrypted output exists before destroying the original.
			if a.settings.DeleteOriginals {
				if encrypter.IsEncryptedFile(outputPath) {
					os.Remove(path)
				}
			}
		}
	}

	a.emitProgress(len(paths), len(paths), "Complete", start)
	result.ElapsedTime = time.Since(start).Round(time.Millisecond).String()
	return result
}

// EncryptFolder recursively collects all non-encrypted files in a folder
// and encrypts them.
func (a *App) EncryptFolder(folderPath string, password string) *OperationResult {
	var files []string
	_ = filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		// Skip symlinks to prevent symlink-following attacks.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() && !encrypter.IsEncryptedFile(path) {
			files = append(files, path)
		}
		return nil
	})

	if len(files) == 0 {
		return &OperationResult{Total: 0, Errors: make([]string, 0)}
	}
	return a.EncryptFiles(files, password)
}

// ---------------------------------------------------------------------------
// Decryption
// ---------------------------------------------------------------------------

// DecryptFiles decrypts a list of PSE-encrypted files with the given password.
func (a *App) DecryptFiles(paths []string, password string) *OperationResult {
	// Server-side password validation — never trust the frontend alone.
	if len(password) < 8 {
		return &OperationResult{
			Total:  0,
			Errors: []string{"password must be at least 8 characters"},
			Failed: 1,
		}
	}

	start := time.Now()
	result := &OperationResult{Total: len(paths), Errors: make([]string, 0)}

	// NOTE: The original `password` string cannot be zeroed (Go limitation).
	pw := []byte(password)
	defer encrypter.ClearBytes(pw)

	for i, path := range paths {
		a.emitProgress(i, len(paths), filepath.Base(path), start)

		_, err := a.engine.DecryptFile(path, pw)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s: decryption failed", filepath.Base(path)))
		} else {
			result.Success++
		}
	}

	a.emitProgress(len(paths), len(paths), "Complete", start)
	result.ElapsedTime = time.Since(start).Round(time.Millisecond).String()
	return result
}

// DecryptFolder recursively finds all encrypted files (by extension) in a
// folder and decrypts them.
func (a *App) DecryptFolder(folderPath string, password string) *OperationResult {
	ext := a.settings.Extension
	if ext == "" {
		ext = ".pse"
	}

	var files []string
	_ = filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Skip symlinks to prevent symlink-following attacks.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ext) {
			files = append(files, path)
		}
		return nil
	})

	if len(files) == 0 {
		return &OperationResult{Total: 0, Errors: make([]string, 0)}
	}
	return a.DecryptFiles(files, password)
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

// GetSettings returns the current application settings to the frontend.
func (a *App) GetSettings() *encrypter.Settings {
	return a.settings
}

// SaveSettings persists updated settings and re-creates the crypto engine
// so that subsequent operations use the new configuration (e.g. new extension).
func (a *App) SaveSettings(s encrypter.Settings) error {
	a.settings = &s
	a.engine = encrypter.NewEngine(&s)
	return s.Save()
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

// GetAppVersion returns the semantic version string shown in the About page.
func (a *App) GetAppVersion() string {
	return "1.0.0"
}

// CheckFile reports whether a file has a valid PSE magic header.
func (a *App) CheckFile(path string) bool {
	return encrypter.IsEncryptedFile(path)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// emitProgress sends a ProgressEvent to the frontend via Wails runtime events.
func (a *App) emitProgress(current, total int, filename string, startTime time.Time) {
	elapsed := time.Since(startTime)
	var percent float64
	var speed string
	var eta string

	if total > 0 {
		percent = float64(current) / float64(total) * 100
	}

	if current > 0 && elapsed.Seconds() > 0 {
		filesPerSec := float64(current) / elapsed.Seconds()
		remaining := time.Duration(float64(total-current)/filesPerSec) * time.Second
		eta = remaining.Round(time.Second).String()
		speed = fmt.Sprintf("%.1f files/sec", filesPerSec)
	}

	wailsRuntime.EventsEmit(a.ctx, "progress", ProgressEvent{
		Current:  current,
		Total:    total,
		Filename: filename,
		Percent:  percent,
		Speed:    speed,
		ETA:      eta,
	})
}
