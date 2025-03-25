package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleFileTransferWithMockFS(t *testing.T) {
	t.Parallel()

	// Create a new mock filesystem
	mockFS := NewMockFS()

	// Test data
	testDate := "2023-01-15"
	testTime := "13:45:30"
	testSciName := "Testus birdus"
	testComName := "Test Bird"
	testFileName := "test_audio.wav"
	sourceContent := []byte("This is test audio content")

	// Create test detection
	detection := &Detection{
		Date:       testDate,
		Time:       testTime,
		SciName:    testSciName,
		ComName:    testComName,
		Confidence: 0.85,
		FileName:   testFileName,
	}

	// Build expected paths
	sourceRoot := "/source"
	targetRoot := "/target"

	sourcePath := filepath.Join(sourceRoot, "Extracted", "By_Date", testDate, testComName, testFileName)
	expectedYear := "2023"
	expectedMonth := "01"
	expectedClipName := "testus_birdus_85p_20230115T134530Z.wav"
	targetPath := filepath.Join(targetRoot, expectedYear, expectedMonth, expectedClipName)

	// Setup source directory
	mockFS.MkdirAll(filepath.Dir(sourcePath), os.ModePerm)
	mockFS.WriteFile(sourcePath, sourceContent, 0o644)

	// Run tests for copy operation
	t.Run("Copy file operation", func(t *testing.T) {
		handleFileTransferWithFS(detection, sourceRoot, targetRoot, CopyFile, mockFS)

		// Verify target file exists and has correct content
		if !mockFS.FileExists(targetPath) {
			t.Errorf("Target file not created: %s", targetPath)
		}

		content, err := mockFS.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("Failed to read target file: %v", err)
		}

		if !bytes.Equal(content, sourceContent) {
			t.Errorf("Target file content mismatch. Got: %s, Want: %s", content, sourceContent)
		}

		// Verify source file still exists (copy, not move)
		if !mockFS.FileExists(sourcePath) {
			t.Errorf("Source file was removed during copy operation")
		}
	})

	// Test move operation
	t.Run("Move file operation", func(t *testing.T) {
		// Reset the target file
		if mockFS.FileExists(targetPath) {
			mockFS.Remove(targetPath)
		}

		handleFileTransferWithFS(detection, sourceRoot, targetRoot, MoveFile, mockFS)

		// Verify target file exists and has correct content
		if !mockFS.FileExists(targetPath) {
			t.Errorf("Target file not created during move: %s", targetPath)
		}

		content, err := mockFS.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("Failed to read target file after move: %v", err)
		}

		if !bytes.Equal(content, sourceContent) {
			t.Errorf("Target file content mismatch after move. Got: %s, Want: %s", content, sourceContent)
		}

		// Verify source file no longer exists (move, not copy)
		if mockFS.FileExists(sourcePath) {
			t.Errorf("Source file still exists after move operation")
		}
	})

	// Test handling of missing source file
	t.Run("Source file doesn't exist", func(t *testing.T) {
		nonExistentDetection := &Detection{
			Date:       testDate,
			Time:       testTime,
			SciName:    testSciName,
			ComName:    testComName,
			Confidence: 0.85,
			FileName:   "nonexistent.wav",
		}

		// Reset the target file
		if mockFS.FileExists(targetPath) {
			mockFS.Remove(targetPath)
		}

		// This should not panic or error, just skip silently
		handleFileTransferWithFS(nonExistentDetection, sourceRoot, targetRoot, CopyFile, mockFS)

		// Verify no target file was created
		if mockFS.FileExists(targetPath) {
			t.Errorf("Target file created despite missing source file")
		}
	})

	// Test error handling for mkdir failures
	t.Run("Error creating target directories", func(t *testing.T) {
		// Set failure mode for mkdir
		mockFS.SetFailMode("MkdirAll", true)
		defer mockFS.SetFailMode("MkdirAll", false)

		// Reset state
		if mockFS.FileExists(targetPath) {
			mockFS.Remove(targetPath)
		}

		// This should not panic but log an error
		handleFileTransferWithFS(detection, sourceRoot, targetRoot, CopyFile, mockFS)

		// Verify no target file was created due to mkdir failure
		if mockFS.FileExists(targetPath) {
			t.Errorf("Target file created despite mkdir failure")
		}
	})
}

func TestCopyFileWithMockFS(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFS()

	sourcePath := "/source/test.txt"
	targetPath := "/target/test.txt"
	content := []byte("Test content for copy operation")

	// Set up source file
	mockFS.MkdirAll(filepath.Dir(sourcePath), os.ModePerm)
	mockFS.WriteFile(sourcePath, content, 0o644)

	// Set up target directory
	mockFS.MkdirAll(filepath.Dir(targetPath), os.ModePerm)

	// Test normal copy
	t.Run("Successful copy", func(t *testing.T) {
		err := copyFileWithFS(sourcePath, targetPath, mockFS)
		if err != nil {
			t.Fatalf("copyFileWithFS() error = %v", err)
		}

		// Verify file was copied correctly
		targetContent, err := mockFS.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("Failed to read target file: %v", err)
		}

		if !bytes.Equal(targetContent, content) {
			t.Errorf("Target content = %s, want %s", targetContent, content)
		}
	})

	// Test with non-existent source
	t.Run("Non-existent source", func(t *testing.T) {
		nonExistentPath := "/source/nonexistent.txt"
		err := copyFileWithFS(nonExistentPath, targetPath, mockFS)
		if err == nil {
			t.Errorf("copyFileWithFS() with non-existent source should error")
		}
	})

	// Test with read failure
	t.Run("Read failure", func(t *testing.T) {
		mockFS.SetFailMode("Open", true)
		defer mockFS.SetFailMode("Open", false)

		err := copyFileWithFS(sourcePath, targetPath, mockFS)
		if err == nil {
			t.Errorf("copyFileWithFS() with read failure should error")
		}
	})

	// Test with write failure
	t.Run("Write failure", func(t *testing.T) {
		mockFS.SetFailMode("Create", true)
		defer mockFS.SetFailMode("Create", false)

		err := copyFileWithFS(sourcePath, targetPath, mockFS)
		if err == nil {
			t.Errorf("copyFileWithFS() with write failure should error")
		}
	})
}

func TestMoveFileWithMockFS(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFS()

	sourcePath := "/source/move-test.txt"
	targetPath := "/target/move-test.txt"
	content := []byte("Test content for move operation")

	// Test normal move
	t.Run("Successful move", func(t *testing.T) {
		// Setup source file for each test
		mockFS.MkdirAll(filepath.Dir(sourcePath), os.ModePerm)
		mockFS.WriteFile(sourcePath, content, 0o644)
		mockFS.MkdirAll(filepath.Dir(targetPath), os.ModePerm)

		err := moveFileWithFS(sourcePath, targetPath, mockFS)
		if err != nil {
			t.Fatalf("moveFileWithFS() error = %v", err)
		}

		// Verify file was moved correctly
		if mockFS.FileExists(sourcePath) {
			t.Errorf("Source file still exists after move")
		}

		targetContent, err := mockFS.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("Failed to read target file: %v", err)
		}

		if !bytes.Equal(targetContent, content) {
			t.Errorf("Target content = %s, want %s", targetContent, content)
		}
	})

	// Test with copy failure
	t.Run("Copy failure", func(t *testing.T) {
		// Setup source file for each test
		mockFS.MkdirAll(filepath.Dir(sourcePath), os.ModePerm)
		mockFS.WriteFile(sourcePath, content, 0o644)

		// Set create to fail
		mockFS.SetFailMode("Create", true)
		defer mockFS.SetFailMode("Create", false)

		err := moveFileWithFS(sourcePath, targetPath, mockFS)
		if err == nil {
			t.Errorf("moveFileWithFS() with copy failure should error")
		}

		// Verify source file still exists (move failed)
		if !mockFS.FileExists(sourcePath) {
			t.Errorf("Source file was removed despite copy failure")
		}
	})

	// Test with remove failure
	t.Run("Remove failure", func(t *testing.T) {
		// Setup source file for each test
		mockFS.MkdirAll(filepath.Dir(sourcePath), os.ModePerm)
		mockFS.WriteFile(sourcePath, content, 0o644)
		mockFS.MkdirAll(filepath.Dir(targetPath), os.ModePerm)

		// Set remove to fail
		mockFS.SetFailMode("Remove", true)
		defer mockFS.SetFailMode("Remove", false)

		err := moveFileWithFS(sourcePath, targetPath, mockFS)
		if err == nil {
			t.Errorf("moveFileWithFS() with remove failure should error")
		}

		// Verify target file was created
		if !mockFS.FileExists(targetPath) {
			t.Errorf("Target file not created despite successful copy")
		}

		// Source should still exist due to remove failure
		if !mockFS.FileExists(sourcePath) {
			t.Errorf("Source file was removed despite remove failure mode")
		}
	})
}
