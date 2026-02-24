package updater

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	maxAssetSizeBytes    int64         = 512 * 1024 * 1024 // 512 MB
	downloadTimeout      time.Duration = 5 * time.Minute
	stagedExecutableName               = "concord.exe"
)

// Service applies desktop self-updates.
type Service struct {
	logger zerolog.Logger
	client *http.Client
}

// NewService creates a new updater service.
func NewService(logger zerolog.Logger) *Service {
	return &Service{
		logger: logger.With().Str("component", "updater").Logger(),
		client: &http.Client{Timeout: downloadTimeout},
	}
}

// ApplyDesktopUpdate downloads and stages a new desktop binary, then starts a
// detached update script that replaces the running executable after shutdown.
func (s *Service) ApplyDesktopUpdate(downloadURL, targetVersion, expectedSHA256 string) error {
	if runtime.GOOS != "windows" {
		return errors.New("automatic update is currently supported only on Windows desktop builds")
	}

	url := strings.TrimSpace(downloadURL)
	if url == "" {
		return errors.New("update download URL is required")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	if resolvedPath, resolveErr := filepath.EvalSymlinks(exePath); resolveErr == nil {
		exePath = resolvedPath
	}

	workDir, err := os.MkdirTemp("", "concord-update-*")
	if err != nil {
		return fmt.Errorf("create update workspace: %w", err)
	}

	archivePath := filepath.Join(workDir, "release.zip")
	downloadedSHA, downloadedBytes, err := s.downloadAsset(url, archivePath)
	if err != nil {
		return err
	}

	expected := normalizeSHA256(expectedSHA256)
	if expected != "" && !strings.EqualFold(expected, downloadedSHA) {
		return fmt.Errorf("update checksum mismatch: expected %s, got %s", expected, downloadedSHA)
	}

	stagedExePath, err := extractExecutableFromArchive(archivePath, workDir)
	if err != nil {
		return err
	}

	scriptPath := filepath.Join(workDir, "apply-update.ps1")
	if err := os.WriteFile(scriptPath, []byte(updateScriptContent), 0o600); err != nil {
		return fmt.Errorf("write update script: %w", err)
	}

	backupPath := exePath + ".bak"
	cmd := exec.Command(
		"powershell",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-ParentPID", strconv.Itoa(os.Getpid()),
		"-SourceExe", stagedExePath,
		"-TargetExe", exePath,
		"-BackupExe", backupPath,
		"-WorkDir", workDir,
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update process: %w", err)
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}

	s.logger.Info().
		Str("target_version", strings.TrimSpace(targetVersion)).
		Int64("downloaded_bytes", downloadedBytes).
		Str("asset_sha256", downloadedSHA).
		Msg("desktop update staged")

	return nil
}

func (s *Service) downloadAsset(downloadURL, destination string) (string, int64, error) {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("build update request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "concord-desktop-updater")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("download update asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("download update asset: unexpected HTTP %d", resp.StatusCode)
	}

	if resp.ContentLength > maxAssetSizeBytes {
		return "", 0, fmt.Errorf("update asset too large: %d bytes", resp.ContentLength)
	}

	out, err := os.Create(destination)
	if err != nil {
		return "", 0, fmt.Errorf("create update archive: %w", err)
	}
	defer out.Close()

	hasher := sha256.New()
	limitedBody := io.LimitReader(resp.Body, maxAssetSizeBytes+1)
	n, err := io.Copy(io.MultiWriter(out, hasher), limitedBody)
	if err != nil {
		return "", 0, fmt.Errorf("write update archive: %w", err)
	}
	if n > maxAssetSizeBytes {
		return "", 0, fmt.Errorf("update asset exceeds size limit of %d bytes", maxAssetSizeBytes)
	}

	digest := hex.EncodeToString(hasher.Sum(nil))
	return digest, n, nil
}

func extractExecutableFromArchive(archivePath, workDir string) (string, error) {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open update archive: %w", err)
	}
	defer archive.Close()

	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Base(file.Name), stagedExecutableName) {
			continue
		}

		src, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("open executable in archive: %w", err)
		}

		destination := filepath.Join(workDir, "concord.new.exe")
		dst, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			src.Close()
			return "", fmt.Errorf("create staged executable: %w", err)
		}

		_, copyErr := io.Copy(dst, src)
		closeErr := dst.Close()
		srcErr := src.Close()
		if copyErr != nil {
			return "", fmt.Errorf("extract executable: %w", copyErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("finalize staged executable: %w", closeErr)
		}
		if srcErr != nil {
			return "", fmt.Errorf("close executable stream: %w", srcErr)
		}

		info, err := os.Stat(destination)
		if err != nil {
			return "", fmt.Errorf("validate staged executable: %w", err)
		}
		if info.Size() == 0 {
			return "", errors.New("staged executable is empty")
		}

		return destination, nil
	}

	return "", errors.New("update archive does not contain concord.exe")
}

func normalizeSHA256(digest string) string {
	trimmed := strings.TrimSpace(strings.ToLower(digest))
	if strings.HasPrefix(trimmed, "sha256:") {
		trimmed = strings.TrimPrefix(trimmed, "sha256:")
	}
	return strings.TrimSpace(trimmed)
}

const updateScriptContent = `param(
  [int]$ParentPID,
  [string]$SourceExe,
  [string]$TargetExe,
  [string]$BackupExe,
  [string]$WorkDir
)

$ErrorActionPreference = "Stop"

for ($i = 0; $i -lt 400; $i++) {
  if (-not (Get-Process -Id $ParentPID -ErrorAction SilentlyContinue)) {
    break
  }
  Start-Sleep -Milliseconds 100
}

$updated = $false
for ($i = 0; $i -lt 80; $i++) {
  try {
    if (Test-Path -LiteralPath $BackupExe) {
      Remove-Item -LiteralPath $BackupExe -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path -LiteralPath $TargetExe) {
      Move-Item -LiteralPath $TargetExe -Destination $BackupExe -Force
    }

    Move-Item -LiteralPath $SourceExe -Destination $TargetExe -Force
    $updated = $true
    break
  } catch {
    Start-Sleep -Milliseconds 150
  }
}

if (-not $updated) {
  exit 1
}

Start-Process -FilePath $TargetExe
Start-Sleep -Milliseconds 500

if (Test-Path -LiteralPath $BackupExe) {
  Remove-Item -LiteralPath $BackupExe -Force -ErrorAction SilentlyContinue
}
if (Test-Path -LiteralPath $WorkDir) {
  Remove-Item -LiteralPath $WorkDir -Recurse -Force -ErrorAction SilentlyContinue
}
`
