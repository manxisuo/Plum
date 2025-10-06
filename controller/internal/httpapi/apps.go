package httpapi

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"plum/controller/internal/store"
)

type AppInfo struct {
	ArtifactID string `json:"artifactId"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	URL        string `json:"url"` // /artifacts/<file>
	SHA256     string `json:"sha256"`
	SizeBytes  int64  `json:"sizeBytes"`
	CreatedAt  int64  `json:"createdAt"`
}

// POST /v1/apps/upload (multipart/form-data: file)
func handleAppUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil { // 64MB
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Persist to temp, compute sha256 and size
	dataDir := getDataDir()
	os.MkdirAll(filepath.Join(dataDir, "artifacts"), 0o755)
	tmpPath := filepath.Join(dataDir, "artifacts", fmt.Sprintf("upload_%d.tmp", time.Now().UnixNano()))
	f, err := os.Create(tmpPath)
	if err != nil {
		http.Error(w, "save error", http.StatusInternalServerError)
		return
	}
	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(f, h), file)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		http.Error(w, "save error", http.StatusInternalServerError)
		return
	}
	sum := hex.EncodeToString(h.Sum(nil))

	// Validate zip has start.sh and meta.ini; read name/version
	z, err := zip.OpenReader(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		http.Error(w, "not a zip", http.StatusBadRequest)
		return
	}
	var hasStart bool
	var name, version string
	for _, f := range z.File {
		base := filepath.Base(f.Name)
		if base == "start.sh" {
			hasStart = true
		}
		if strings.EqualFold(base, "meta.ini") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			b, _ := io.ReadAll(rc)
			rc.Close()
			n, v := parseMetaINI(string(b))
			name, version = n, v
		}
	}
	z.Close()
	if !hasStart || name == "" || version == "" {
		os.Remove(tmpPath)
		http.Error(w, "zip must contain start.sh and meta.ini(name,version)", http.StatusBadRequest)
		return
	}

	// Check if app with same name and version already exists
	artifacts, err := store.Current.ListArtifacts()
	if err != nil {
		os.Remove(tmpPath)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	for _, a := range artifacts {
		if a.AppName == name && a.Version == version {
			os.Remove(tmpPath)
			http.Error(w, fmt.Sprintf("应用 %s 版本 %s 已存在", name, version), http.StatusConflict)
			return
		}
	}

	// Finalize file name: <name>_<version>_<sha8>.zip
	safeName := sanitizeFilename(name)
	safeVer := sanitizeFilename(version)
	finalName := fmt.Sprintf("%s_%s_%s.zip", safeName, safeVer, sum[:8])
	finalPath := filepath.Join(dataDir, "artifacts", finalName)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}

	id, err := store.Current.SaveArtifact(store.Artifact{
		AppName:   name,
		Version:   version,
		Path:      "/artifacts/" + finalName,
		SHA256:    sum,
		SizeBytes: size,
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, AppInfo{
		ArtifactID: id,
		Name:       name,
		Version:    version,
		URL:        "/artifacts/" + finalName,
		SHA256:     sum,
		SizeBytes:  size,
		CreatedAt:  time.Now().Unix(),
	})
}

// GET /v1/apps
func handleListApps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	arts, err := store.Current.ListArtifacts()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	out := make([]AppInfo, 0, len(arts))
	for _, a := range arts {
		out = append(out, AppInfo{
			ArtifactID: a.ArtifactID,
			Name:       a.AppName,
			Version:    a.Version,
			URL:        a.Path,
			SHA256:     a.SHA256,
			SizeBytes:  a.SizeBytes,
			CreatedAt:  a.CreatedAt,
		})
	}
	writeJSON(w, out)
}

// DELETE /v1/apps/{id}
func handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/apps/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	// check references by assignments
	if a, ok, _ := store.Current.GetArtifact(id); ok {
		n, _ := store.Current.CountAssignmentsByArtifactPath(a.Path)
		if n > 0 {
			http.Error(w, fmt.Sprintf("应用包 %s 版本 %s 正在被 %d 个部署使用，无法删除", a.AppName, a.Version, n), http.StatusConflict)
			return
		}
		// also try remove file if exists
		if strings.HasPrefix(a.Path, "/artifacts/") {
			p := filepath.Join(getDataDir(), a.Path[1:])
			_ = os.Remove(p)
		}
		_ = store.Current.DeleteArtifact(id)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.NotFound(w, r)
}

// utils
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, s)
	if s == "" {
		return "app"
	}
	return s
}

func parseMetaINI(content string) (string, string) {
	var name, version string
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, ";") {
			continue
		}
		if i := strings.IndexAny(l, ":="); i > 0 {
			k := strings.TrimSpace(l[:i])
			v := strings.TrimSpace(l[i+1:])
			kLower := strings.ToLower(k)
			switch kLower {
			case "name":
				name = v
			case "version":
				version = v
			}
		}
	}
	return name, version
}

func getDataDir() string {
	if v := os.Getenv("CONTROLLER_DATA_DIR"); v != "" {
		return v
	}
	return "."
}
