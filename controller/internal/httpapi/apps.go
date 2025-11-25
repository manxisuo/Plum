package httpapi

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/dustin/go-humanize"
	"github.com/manxisuo/plum/controller/internal/store"
)

type AppInfo struct {
	ArtifactID      string `json:"artifactId"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	URL             string `json:"url"` // /artifacts/<file> (for zip) or empty (for image)
	SHA256          string `json:"sha256"`
	SizeBytes       int64  `json:"sizeBytes"`
	CreatedAt       int64  `json:"createdAt"`
	Type            string `json:"type"` // "zip" or "image"
	ImageRepository string `json:"imageRepository,omitempty"`
	ImageTag        string `json:"imageTag,omitempty"`
	PortMappings    string `json:"portMappings,omitempty"` // JSON string
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
		Type:       "zip",
	})
}

// POST /v1/apps/create-image (JSON body)
func handleCreateImageApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name            string                   `json:"name"`
		Version         string                   `json:"version"`
		ImageRepository string                   `json:"imageRepository"`
		ImageTag        string                   `json:"imageTag"`
		PortMappings    []map[string]interface{} `json:"portMappings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Version == "" || req.ImageRepository == "" || req.ImageTag == "" {
		http.Error(w, "name, version, imageRepository, and imageTag are required", http.StatusBadRequest)
		return
	}

	// Check if app with same name and version already exists
	artifacts, err := store.Current.ListArtifacts()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	for _, a := range artifacts {
		if a.AppName == req.Name && a.Version == req.Version {
			http.Error(w, fmt.Sprintf("应用 %s 版本 %s 已存在", req.Name, req.Version), http.StatusConflict)
			return
		}
	}

	// Serialize port mappings to JSON
	portMappingsJSON := ""
	if len(req.PortMappings) > 0 {
		bytes, err := json.Marshal(req.PortMappings)
		if err != nil {
			http.Error(w, "invalid port mappings: "+err.Error(), http.StatusBadRequest)
			return
		}
		portMappingsJSON = string(bytes)
	}

	// Generate artifact ID and save
	id, err := store.Current.SaveArtifact(store.Artifact{
		AppName:         req.Name,
		Version:         req.Version,
		Path:            "", // No file path for image-based apps
		SHA256:          "", // No SHA256 for image-based apps
		SizeBytes:       0,  // No size for image-based apps
		CreatedAt:       time.Now().Unix(),
		Type:            "image",
		ImageRepository: req.ImageRepository,
		ImageTag:        req.ImageTag,
		PortMappings:    portMappingsJSON,
	})
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, AppInfo{
		ArtifactID:      id,
		Name:            req.Name,
		Version:         req.Version,
		URL:             "",
		SHA256:          "",
		SizeBytes:       0,
		CreatedAt:       time.Now().Unix(),
		Type:            "image",
		ImageRepository: req.ImageRepository,
		ImageTag:        req.ImageTag,
		PortMappings:    portMappingsJSON,
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
			ArtifactID:      a.ArtifactID,
			Name:            a.AppName,
			Version:         a.Version,
			URL:             a.Path,
			SHA256:          a.SHA256,
			SizeBytes:       a.SizeBytes,
			CreatedAt:       a.CreatedAt,
			Type:            a.Type,
			ImageRepository: a.ImageRepository,
			ImageTag:        a.ImageTag,
			PortMappings:    a.PortMappings,
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

// DockerImageInfo Docker 镜像信息
type DockerImageInfo struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	ImageID    string `json:"imageId"`
	Created    string `json:"created"`
	Size       string `json:"size"`
}

// GET /v1/apps/docker-images
// 查询本地 Docker 镜像列表
func handleListDockerImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	images, err := fetchDockerImages()
	if err != nil {
		log.Printf("Failed to list docker images via SDK: %v", err)
		writeJSON(w, []DockerImageInfo{})
		return
	}

	writeJSON(w, images)
}

func fetchDockerImages() ([]DockerImageInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	imageSummaries, err := cli.ImageList(ctx, imagetypes.ListOptions{})
	if err != nil {
		return nil, err
	}

	images := make([]DockerImageInfo, 0, len(imageSummaries))
	for _, summary := range imageSummaries {
		repoTags := summary.RepoTags
		if len(repoTags) == 0 {
			repoTags = []string{"<none>:<none>"}
		}

		created := time.Unix(summary.Created, 0).Format(time.RFC3339)
		size := humanize.Bytes(uint64(summary.Size))

		for _, repoTag := range repoTags {
			repo := repoTag
			tag := ""
			if idx := strings.LastIndex(repoTag, ":"); idx != -1 {
				repo = repoTag[:idx]
				tag = repoTag[idx+1:]
			}
			images = append(images, DockerImageInfo{
				Repository: repo,
				Tag:        tag,
				ImageID:    summary.ID,
				Created:    created,
				Size:       size,
			})
		}
	}

	return images, nil
}
