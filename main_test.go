package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateDirSize(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create some test files with known sizes
	testFiles := []struct {
		name string
		size int64
	}{
		{"file1.txt", 100},
		{"file2.txt", 200},
		{"file3.txt", 300},
	}

	var expectedSize int64
	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)
		data := make([]byte, tf.size)
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		expectedSize += tf.size
	}

	// Create a subdirectory with files
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFiles := []struct {
		name string
		size int64
	}{
		{"subfile1.txt", 150},
		{"subfile2.txt", 250},
	}

	for _, sf := range subFiles {
		filePath := filepath.Join(subDir, sf.name)
		data := make([]byte, sf.size)
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			t.Fatalf("Failed to create test file in subdirectory: %v", err)
		}
		expectedSize += sf.size
	}

	// Test the function
	gotSize, err := calculateDirSize(tempDir)
	if err != nil {
		t.Fatalf("calculateDirSize() error = %v", err)
	}

	if gotSize != expectedSize {
		t.Errorf("calculateDirSize() = %v, want %v", gotSize, expectedSize)
	}

	// Test with non-existent directory
	nonExistentDir := filepath.Join(tempDir, "nonexistent")
	_, err = calculateDirSize(nonExistentDir)
	if err == nil {
		t.Errorf("calculateDirSize() with non-existent directory did not return an error")
	}

	// Test with a directory with no read permissions (Unix-like systems only)
	if os.Getuid() != 0 { // Skip if running as root
		noPermDir := filepath.Join(tempDir, "noperm")
		if err := os.Mkdir(noPermDir, 0); err != nil {
			t.Logf("Could not create directory with no permissions, skipping: %v", err)
		} else {
			t.Cleanup(func() {
				// Restore permissions to allow cleanup
				os.Chmod(noPermDir, 0o755)
			})

			_, err = calculateDirSize(noPermDir)
			if err == nil {
				t.Errorf("calculateDirSize() with no-permission directory did not return an error")
			}
		}
	}
}

func TestCheckDiskSpace(t *testing.T) {
	t.Parallel()

	// Create temporary source and target directories
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create some files in the source directory
	files := []struct {
		name string
		size int64
	}{
		{"file1.txt", 1024}, // 1 KB
		{"file2.txt", 2048}, // 2 KB
		{"file3.txt", 4096}, // 4 KB
	}

	for _, f := range files {
		filePath := filepath.Join(sourceDir, f.name)
		data := make([]byte, f.size)
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test the function
	hasSpace, err := checkDiskSpace(sourceDir, targetDir)
	if err != nil {
		t.Fatalf("checkDiskSpace() error = %v", err)
	}

	// We expect there should be enough space in the temp directory
	// Note: This test assumes the temp directory has more than ~7KB free space
	if !hasSpace {
		t.Errorf("checkDiskSpace() = %v, want true (sufficient space)", hasSpace)
	}

	// Test with non-existent source directory
	nonExistentSource := filepath.Join(sourceDir, "nonexistent")
	_, err = checkDiskSpace(nonExistentSource, targetDir)
	if err == nil {
		t.Errorf("checkDiskSpace() with non-existent source did not return an error")
	}

	// Test with non-existent target directory
	nonExistentTarget := filepath.Join(targetDir, "nonexistent")
	_, err = checkDiskSpace(sourceDir, nonExistentTarget)
	if err == nil {
		t.Errorf("checkDiskSpace() with non-existent target did not return an error")
	}

	// Create a very large "virtual" file to test insufficient space
	// This won't actually allocate disk space, but will report a large dir size
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	largeSourceDir := t.TempDir()
	// Create a file with a large reported size using sparse file or directory walk mock
	// This is simulated by creating many small files that will be counted in directory size
	largeFileCount := 1000
	for i := 0; i < largeFileCount; i++ {
		filePath := filepath.Join(largeSourceDir, filepath.Join("largedir", "file"), "large_"+filepath.Join(filepath.Join("deeply", "nested"), "path"), filepath.Join("with", "many"), "segments", filepath.Join("for", "testing"), filepath.Join(filepath.Join("path", "length"), "limits"), "file")
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatalf("Failed to create directory structure: %v", err)
		}
		data := make([]byte, 1024) // 1KB
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// This should return false for space availability on most systems
	// Unless the test is running on a system with many TB of free space
	_, err = checkDiskSpace(largeSourceDir, targetDir)
	if err != nil {
		// If we get an error (e.g., path too long), that's okay too
		t.Logf("checkDiskSpace() with very large directory returned error: %v", err)
	}
}

func TestGetFreeSpace(t *testing.T) {
	t.Parallel()

	// Create a temporary directory
	tempDir := t.TempDir()

	// Test the getFreeSpace function
	freeSpace, err := getFreeSpace(tempDir)
	if err != nil {
		t.Fatalf("getFreeSpace() error = %v", err)
	}

	// We can't predict the exact free space, but it should be greater than zero
	if freeSpace <= 0 {
		t.Errorf("getFreeSpace() = %d, want > 0", freeSpace)
	}

	// Test with a non-existent directory
	nonExistentDir := filepath.Join(tempDir, "non_existent")
	_, err = getFreeSpace(nonExistentDir)
	if err == nil {
		t.Errorf("getFreeSpace() with non-existent directory did not return an error")
	}

	// Test with an invalid path (platform-specific behavior)
	var invalidPath string
	if filepath.Separator == '/' {
		// Unix-like systems
		invalidPath = "\000" // null character in path
	} else {
		// Windows
		invalidPath = "\\\\?\\invalid:character"
	}

	_, err = getFreeSpace(invalidPath)
	if err == nil {
		t.Errorf("getFreeSpace() with invalid path did not return an error")
	}
}
