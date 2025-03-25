package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MockFileInfo implements fs.FileInfo for testing
type MockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return m.size }
func (m MockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m MockFileInfo) ModTime() time.Time { return m.modTime }
func (m MockFileInfo) IsDir() bool        { return m.isDir }
func (m MockFileInfo) Sys() any           { return nil }

// MockFile represents an in-memory file for testing
type MockFile struct {
	content *bytes.Buffer
	closed  bool
	path    string
}

func (f *MockFile) Read(p []byte) (n int, err error) {
	if f.closed {
		return 0, errors.New("file closed")
	}
	return f.content.Read(p)
}

func (f *MockFile) Write(p []byte) (n int, err error) {
	if f.closed {
		return 0, errors.New("file closed")
	}
	return f.content.Write(p)
}

func (f *MockFile) Close() error {
	f.closed = true
	return nil
}

// MockFS implements a simple in-memory filesystem for testing
type MockFS struct {
	files    map[string][]byte
	dirs     map[string]bool
	mu       sync.RWMutex
	failMode map[string]bool // Used to simulate failures
}

// NewMockFS creates a new mock filesystem
func NewMockFS() *MockFS {
	return &MockFS{
		files:    make(map[string][]byte),
		dirs:     make(map[string]bool),
		failMode: make(map[string]bool),
	}
}

// SetFailMode enables or disables failure simulation for specific operations
func (m *MockFS) SetFailMode(operation string, shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failMode[operation] = shouldFail
}

// MkdirAll creates a directory and all parent directories
func (m *MockFS) MkdirAll(path string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failMode["MkdirAll"] {
		return errors.New("simulated mkdir failure")
	}

	// Clean and normalize path
	path = filepath.Clean(path)

	// Mark this dir and all parent dirs as existing
	m.dirs[path] = true
	dir := path
	for dir != "." && dir != "/" {
		dir = filepath.Dir(dir)
		m.dirs[dir] = true
	}

	return nil
}

// Stat returns file info for a path
func (m *MockFS) Stat(name string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.failMode["Stat"] {
		return nil, errors.New("simulated stat failure")
	}

	name = filepath.Clean(name)

	// Check if it's a directory
	if _, ok := m.dirs[name]; ok {
		return MockFileInfo{
			name:    filepath.Base(name),
			size:    0,
			mode:    0o755 | fs.ModeDir,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	// Check if it's a file
	content, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	return MockFileInfo{
		name:    filepath.Base(name),
		size:    int64(len(content)),
		mode:    0o644,
		modTime: time.Now(),
		isDir:   false,
	}, nil
}

// Remove deletes a file
func (m *MockFS) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failMode["Remove"] {
		return errors.New("simulated remove failure")
	}

	name = filepath.Clean(name)

	// Check if file exists
	if _, ok := m.files[name]; !ok {
		return os.ErrNotExist
	}

	delete(m.files, name)
	return nil
}

// Create creates or truncates a file
func (m *MockFS) Create(name string) (io.WriteCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failMode["Create"] {
		return nil, errors.New("simulated create failure")
	}

	name = filepath.Clean(name)

	// Ensure parent directory exists
	dir := filepath.Dir(name)
	if _, ok := m.dirs[dir]; !ok && dir != "." {
		return nil, fmt.Errorf("parent directory does not exist: %s", dir)
	}

	// Create empty file
	m.files[name] = []byte{}

	// Return a mockFile that writes to our map
	mf := &MockFile{
		content: bytes.NewBuffer(nil),
		path:    name,
	}

	// Update the file content when closed
	return &mockWriteCloser{
		MockFile: mf,
		onClose: func() error {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.files[name] = mf.content.Bytes()
			return nil
		},
	}, nil
}

// Open opens a file for reading
func (m *MockFS) Open(name string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.failMode["Open"] {
		return nil, errors.New("simulated open failure")
	}

	name = filepath.Clean(name)

	content, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	return &MockFile{
		content: bytes.NewBuffer(content),
		path:    name,
	}, nil
}

// WriteFile is a convenience method to write data to a file in one call
func (m *MockFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	file, err := m.Create(name)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		file.Close()
		return err
	}

	return file.Close()
}

// ReadFile is a convenience method to read all data from a file
func (m *MockFS) ReadFile(name string) ([]byte, error) {
	file, err := m.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// FileExists checks if a file exists
func (m *MockFS) FileExists(path string) bool {
	info, err := m.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func (m *MockFS) DirExists(path string) bool {
	info, err := m.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ListDir lists contents of a directory
func (m *MockFS) ListDir(path string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path = filepath.Clean(path)
	if path != "/" && !strings.HasSuffix(path, "/") {
		path += "/"
	}

	var results []string

	// Find all files and dirs that have this prefix
	for file := range m.files {
		if strings.HasPrefix(file, path) {
			relativePath := strings.TrimPrefix(file, path)
			if !strings.Contains(relativePath, "/") { // Only direct children
				results = append(results, relativePath)
			}
		}
	}

	for dir := range m.dirs {
		if dir != path && strings.HasPrefix(dir, path) {
			relativePath := strings.TrimPrefix(dir, path)
			if !strings.Contains(relativePath, "/") { // Only direct children
				results = append(results, relativePath)
			}
		}
	}

	return results, nil
}

// Helper wrapper to handle close operations
type mockWriteCloser struct {
	*MockFile
	onClose func() error
}

func (m *mockWriteCloser) Close() error {
	err := m.MockFile.Close()
	if err != nil {
		return err
	}
	return m.onClose()
}
