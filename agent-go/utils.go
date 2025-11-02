package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTTPClient HTTP客户端
type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPClient) PostJSON(url string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *HTTPClient) Get(url string) ([]byte, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *HTTPClient) Delete(url string) error {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// EnsureDir 确保目录存在
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// UnzipFile 解压ZIP文件
func UnzipFile(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ensureExecutablePermissions 确保应用目录中的可执行文件有执行权限
// 检查ELF文件（Linux可执行文件）和没有扩展名的文件
func ensureExecutablePermissions(appDir string) error {
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue // 跳过目录
		}
		
		fileName := entry.Name()
		filePath := filepath.Join(appDir, fileName)
		
		// 跳过已知的非可执行文件
		if strings.HasSuffix(fileName, ".ini") ||
			strings.HasSuffix(fileName, ".json") ||
			strings.HasSuffix(fileName, ".txt") ||
			strings.HasSuffix(fileName, ".log") ||
			strings.HasSuffix(fileName, ".zip") ||
			fileName == "start.sh" || // start.sh已经在上面处理了
			fileName == "log" { // log可能是运行产生的日志文件
			continue
		}
		
		// 检查文件信息
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		// 读取文件前几个字节检查是否是ELF文件
		isELF := false
		if f, err := os.Open(filePath); err == nil {
			var header [4]byte
			if n, _ := f.Read(header[:]); n >= 4 {
				// ELF文件魔数: 0x7F 'E' 'L' 'F'
				if header[0] == 0x7F && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
					isELF = true
				}
			}
			f.Close()
		}
		
		// 如果是ELF文件，或者文件名没有扩展名（很可能是可执行文件）
		hasExt := strings.Contains(fileName, ".")
		if isELF || !hasExt {
			// 检查当前权限
			mode := info.Mode()
			if mode&0111 == 0 { // 没有执行权限
				if err := os.Chmod(filePath, mode|0111); err != nil {
					log.Printf("Warning: failed to chmod %s: %v", fileName, err)
				} else {
					log.Printf("Set executable permission for %s", fileName)
				}
			}
		}
	}
	
	return nil
}

// ParseMetaINI 解析meta.ini文件
func ParseMetaINI(path string) ([]ServiceEndpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var endpoints []ServiceEndpoint
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "service=") {
			continue
		}
		val := strings.TrimPrefix(line, "service=")
		parts := strings.Split(val, ":")
		if len(parts) != 3 {
			continue
		}
		var port int
		fmt.Sscanf(parts[2], "%d", &port)
		if port > 0 {
			endpoints = append(endpoints, ServiceEndpoint{
				ServiceName: parts[0],
				Protocol:    parts[1],
				Port:        port,
			})
		}
	}
	return endpoints, nil
}
