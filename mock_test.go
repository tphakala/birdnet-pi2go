package main

import (
	"path/filepath"
	"testing"
	"time"
)

// TestHandleFileTransferWithMocks tests the handleFileTransfer function with mocked filesystem
func TestHandleFileTransferWithMocks(t *testing.T) {
	t.Parallel()

	// Test case 1: Successful copy operation
	t.Run("Successful copy", func(t *testing.T) {
		t.Parallel()

		// Setup a mock filesystem
		mockFS := NewMockFS()

		// Test data
		detection := &Detection{
			Date:       "2023-01-15",
			Time:       "13:45:30",
			SciName:    "Testus birdus",
			ComName:    "Test Bird",
			Confidence: 0.85,
			FileName:   "test_audio.wav",
		}

		sourceDir := "/source"
		targetDir := "/target"

		// Source file path
		sourceDirPath := filepath.Join(sourceDir, "Extracted", "By_Date", detection.Date, detection.ComName)
		sourceFilePath := filepath.Join(sourceDirPath, detection.FileName)

		// Create source file in the mock filesystem
		mockFS.MkdirAll(filepath.Dir(sourceFilePath), 0o755)
		mockFS.WriteFile(sourceFilePath, []byte("test content"), 0o644)

		// Execute
		handleFileTransferWithFS(detection, sourceDir, targetDir, CopyFile, mockFS)

		// Expected target paths
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", detection.Date+"T"+detection.Time)
		expectedYear, expectedMonth := parsedDate.Format("2006"), parsedDate.Format("01")
		expectedFileName := "testus_birdus_85p_20230115T134530Z.wav"
		targetFilePath := filepath.Join(targetDir, expectedYear, expectedMonth, expectedFileName)

		// Verify the file was copied to the expected location
		if !mockFS.FileExists(targetFilePath) {
			t.Errorf("handleFileTransfer() did not copy file to expected location: %s", targetFilePath)
		}

		// Verify the source file still exists (copy, not move)
		if !mockFS.FileExists(sourceFilePath) {
			t.Errorf("handleFileTransfer() with CopyFile removed the source file")
		}

		// Verify the content of the copied file
		targetContent, err := mockFS.ReadFile(targetFilePath)
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}

		if string(targetContent) != "test content" {
			t.Errorf("handleFileTransfer() copied content does not match source content")
		}
	})

	// Test case 2: Source file doesn't exist
	t.Run("Source file doesn't exist", func(t *testing.T) {
		t.Parallel()

		// Setup a mock filesystem
		mockFS := NewMockFS()

		// Test data
		detection := &Detection{
			Date:       "2023-01-15",
			Time:       "13:45:30",
			SciName:    "Testus birdus",
			ComName:    "Test Bird",
			Confidence: 0.85,
			FileName:   "missing_file.wav",
		}

		sourceDir := "/source"
		targetDir := "/target"

		// No need to create the missing file

		// Execute - this should not panic and simply return without performing any action
		handleFileTransferWithFS(detection, sourceDir, targetDir, CopyFile, mockFS)

		// Expected target paths
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", detection.Date+"T"+detection.Time)
		expectedYear, expectedMonth := parsedDate.Format("2006"), parsedDate.Format("01")
		expectedFileName := "testus_birdus_85p_20230115T134530Z.wav"
		targetFilePath := filepath.Join(targetDir, expectedYear, expectedMonth, expectedFileName)

		// Verify that no file was created
		if mockFS.FileExists(targetFilePath) {
			t.Errorf("handleFileTransfer() created a target file when source doesn't exist")
		}
	})

	// Test case 3: Error creating target directories
	t.Run("Error creating target directories", func(t *testing.T) {
		t.Parallel()

		// Setup a mock filesystem
		mockFS := NewMockFS()

		// Setup the failure mode
		mockFS.SetFailMode("MkdirAll", true)

		// Test data
		detection := &Detection{
			Date:       "2023-01-15",
			Time:       "13:45:30",
			SciName:    "Testus birdus",
			ComName:    "Test Bird",
			Confidence: 0.85,
			FileName:   "test_audio.wav",
		}

		sourceDir := "/source"
		targetDir := "/target"

		// Source file path
		sourceDirPath := filepath.Join(sourceDir, "Extracted", "By_Date", detection.Date, detection.ComName)
		sourceFilePath := filepath.Join(sourceDirPath, detection.FileName)

		// Create source file in the mock filesystem
		mockFS.MkdirAll(filepath.Dir(sourceFilePath), 0o755)
		mockFS.WriteFile(sourceFilePath, []byte("test content"), 0o644)

		// Expected target paths
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", detection.Date+"T"+detection.Time)
		expectedYear, expectedMonth := parsedDate.Format("2006"), parsedDate.Format("01")
		expectedFileName := "testus_birdus_85p_20230115T134530Z.wav"
		targetFilePath := filepath.Join(targetDir, expectedYear, expectedMonth, expectedFileName)

		// Execute - this should not panic and handle the error gracefully
		handleFileTransferWithFS(detection, sourceDir, targetDir, CopyFile, mockFS)

		// Verify that no file was created due to directory creation failure
		if mockFS.FileExists(targetFilePath) {
			t.Errorf("handleFileTransfer() created a target file despite directory creation failure")
		}
	})
}
