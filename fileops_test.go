package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateClipName(t *testing.T) {
	t.Parallel()

	// Table-driven test cases
	tests := []struct {
		name      string
		detection Detection
		want      string
	}{
		{
			name: "Basic detection",
			detection: Detection{
				Date:       "2023-01-01",
				Time:       "12:34:56",
				SciName:    "Corvus corax",
				Confidence: 0.95,
			},
			want: "corvus_corax_95p_20230101T123456Z.wav",
		},
		{
			name: "Detection with spaces in name",
			detection: Detection{
				Date:       "2023-02-15",
				Time:       "08:15:30",
				SciName:    "Parus major",
				Confidence: 0.75,
			},
			want: "parus_major_75p_20230215T081530Z.wav",
		},
		{
			name: "Name with hyphens and special chars",
			detection: Detection{
				Date:       "2023-03-20",
				Time:       "10:00:00",
				SciName:    "Sitta-europaea:complex",
				Confidence: 0.88,
			},
			want: "sittaeuropaeacomplex_88p_20230320T100000Z.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GenerateClipName(&tt.detection)
			if got != tt.want {
				t.Errorf("GenerateClipName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a source file with some content
	sourceContent := []byte("This is a test file content")
	sourceFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourceFile, sourceContent, 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Define destination file
	destFile := filepath.Join(tempDir, "destination.txt")

	// Test copying the file
	err := copyFile(sourceFile, destFile)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify the destination file exists and has the correct content
	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if !bytes.Equal(sourceContent, destContent) {
		t.Errorf("copyFile() copied content = %s, want %s", destContent, sourceContent)
	}

	// Test error cases
	nonExistentSrc := filepath.Join(tempDir, "nonexistent.txt")
	err = copyFile(nonExistentSrc, destFile)
	if err == nil {
		t.Error("copyFile() with non-existent source file did not return an error")
	}

	invalidDest := filepath.Join(tempDir, "invalid", "path.txt")
	err = copyFile(sourceFile, invalidDest)
	if err == nil {
		t.Error("copyFile() with invalid destination path did not return an error")
	}
}

func TestMoveFile(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a source file with some content
	sourceContent := []byte("This is a test file content for move operation")
	sourceFile := filepath.Join(tempDir, "source_move.txt")
	if err := os.WriteFile(sourceFile, sourceContent, 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Define destination file
	destFile := filepath.Join(tempDir, "destination_move.txt")

	// Test moving the file
	err := moveFile(sourceFile, destFile)
	if err != nil {
		t.Fatalf("moveFile() error = %v", err)
	}

	// Verify the source file no longer exists
	if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
		t.Errorf("moveFile() source file still exists after move")
	}

	// Verify the destination file exists and has the correct content
	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if !bytes.Equal(sourceContent, destContent) {
		t.Errorf("moveFile() moved content = %s, want %s", destContent, sourceContent)
	}

	// Test error cases
	nonExistentSrc := filepath.Join(tempDir, "nonexistent.txt")
	err = moveFile(nonExistentSrc, destFile)
	if err == nil {
		t.Error("moveFile() with non-existent source file did not return an error")
	}
}

func TestHandleFileTransfer(t *testing.T) {
	t.Parallel()

	// Use our mock filesystem instead of the real filesystem
	mockFS := NewMockFS()

	// Create test data
	testDate := "2023-01-15"
	testComName := "Test Bird"
	testFileName := "test_audio.wav"
	sourceContent := []byte("This is test audio content")

	// Create the expected source directory structure in the mock filesystem
	sourceRoot := "/source"
	targetRoot := "/target"
	extractedDir := filepath.Join(sourceRoot, "Extracted", "By_Date", testDate, testComName)
	sourceFilePath := filepath.Join(extractedDir, testFileName)

	// Set up directories and file in the mock filesystem
	mockFS.MkdirAll(extractedDir, 0o755)
	mockFS.WriteFile(sourceFilePath, sourceContent, 0o644)

	// Create a test detection
	detection := &Detection{
		Date:       testDate,
		Time:       "13:45:30",
		SciName:    "Testus birdus",
		ComName:    testComName,
		Confidence: 0.85,
		FileName:   testFileName,
	}

	// Test handleFileTransfer with copy operation
	t.Run("Copy operation", func(t *testing.T) {
		// Use the FS-parameterized version of handleFileTransfer
		handleFileTransferWithFS(detection, sourceRoot, targetRoot, CopyFile, mockFS)

		// The function will create a directory structure like YYYY/MM with the new filename
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", testDate+"T"+"13:45:30")
		expectedYear := parsedDate.Format("2006")
		expectedMonth := parsedDate.Format("01")
		expectedFilename := "testus_birdus_85p_20230115T134530Z.wav"

		expectedTargetPath := filepath.Join(targetRoot, expectedYear, expectedMonth, expectedFilename)

		// Verify the file was copied to the expected location
		if !mockFS.FileExists(expectedTargetPath) {
			t.Errorf("handleFileTransfer() did not copy file to expected location: %s", expectedTargetPath)
		} else {
			// Verify the source file still exists (copy, not move)
			if !mockFS.FileExists(sourceFilePath) {
				t.Errorf("handleFileTransfer() with CopyFile removed the source file")
			}

			// Verify the content of the copied file
			targetContent, err := mockFS.ReadFile(expectedTargetPath)
			if err != nil {
				t.Fatalf("Failed to read copied file: %v", err)
			}

			if string(targetContent) != string(sourceContent) {
				t.Errorf("handleFileTransfer() copied content does not match source content")
			}
		}
	})

	// Test handling of non-existent source file
	t.Run("Non-existent source file", func(t *testing.T) {
		nonExistentDetection := &Detection{
			Date:       testDate,
			Time:       "14:00:00",
			SciName:    "Nonexistus birdus",
			ComName:    "Non-existent Bird",
			Confidence: 0.90,
			FileName:   "nonexistent.wav",
		}

		// This should not panic or error, but should silently skip
		handleFileTransferWithFS(nonExistentDetection, sourceRoot, targetRoot, CopyFile, mockFS)
	})

	// Now test with move operation
	t.Run("Move operation", func(t *testing.T) {
		// Reset by creating a new file
		moveSourceDir := "/move_source"
		moveExtractedDir := filepath.Join(moveSourceDir, "Extracted", "By_Date", testDate, testComName)
		mockFS.MkdirAll(moveExtractedDir, 0o755)

		moveSourceFilePath := filepath.Join(moveExtractedDir, testFileName)
		mockFS.WriteFile(moveSourceFilePath, sourceContent, 0o644)

		// Test handleFileTransfer with move operation
		handleFileTransferWithFS(detection, moveSourceDir, targetRoot, MoveFile, mockFS)

		parsedDate, _ := time.Parse("2006-01-02T15:04:05", testDate+"T"+"13:45:30")
		expectedYear := parsedDate.Format("2006")
		expectedMonth := parsedDate.Format("01")
		expectedFilename := "testus_birdus_85p_20230115T134530Z.wav"

		expectedTargetPath := filepath.Join(targetRoot, expectedYear, expectedMonth, expectedFilename)

		// Verify the file was moved to the expected location
		if !mockFS.FileExists(expectedTargetPath) {
			t.Errorf("handleFileTransfer() did not move file to expected location: %s", expectedTargetPath)
		} else {
			// Verify the source file no longer exists (move, not copy)
			if mockFS.FileExists(moveSourceFilePath) {
				t.Errorf("handleFileTransfer() with MoveFile did not remove the source file")
			}

			// Verify the content of the moved file
			targetContent, err := mockFS.ReadFile(expectedTargetPath)
			if err != nil {
				t.Fatalf("Failed to read moved file: %v", err)
			}

			if string(targetContent) != string(sourceContent) {
				t.Errorf("handleFileTransfer() moved content does not match source content")
			}
		}
	})
}

func TestPerformFileOperation(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFS()

	tests := []struct {
		name          string
		operation     FileOperationType
		sourceExists  bool
		destExists    bool
		expectedError bool
	}{
		{
			name:          "Copy file - source exists",
			operation:     CopyFile,
			sourceExists:  true,
			destExists:    true,
			expectedError: false,
		},
		{
			name:          "Move file - source exists",
			operation:     MoveFile,
			sourceExists:  true,
			destExists:    true,
			expectedError: false,
		},
		{
			name:          "Copy file - source doesn't exist",
			operation:     CopyFile,
			sourceExists:  false,
			destExists:    false,
			expectedError: true,
		},
		{
			name:          "Move file - source doesn't exist",
			operation:     MoveFile,
			sourceExists:  false,
			destExists:    false,
			expectedError: true,
		},
		{
			name:          "Invalid operation",
			operation:     FileOperationType(99), // Invalid operation
			sourceExists:  true,
			destExists:    false,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup source and destination file paths
			sourceFile := "/source/test.txt"
			destFile := "/target/dest.txt"

			// Ensure directories exist
			mockFS.MkdirAll(filepath.Dir(sourceFile), 0o755)
			mockFS.MkdirAll(filepath.Dir(destFile), 0o755)

			// Create source file if needed for this test case
			if tt.sourceExists {
				sourceContent := []byte("Test content for file operation")
				mockFS.WriteFile(sourceFile, sourceContent, 0o644)
			}

			// Test the function
			err := performFileOperationWithFS(sourceFile, destFile, tt.operation, mockFS)

			// Check if error matches expectation
			if (err != nil) != tt.expectedError {
				t.Errorf("performFileOperation() error = %v, expectedError = %v", err, tt.expectedError)
			}

			// Verify destination file exists if operation should succeed
			if !tt.expectedError {
				if !mockFS.FileExists(destFile) {
					t.Errorf("performFileOperation() destination file doesn't exist after operation")
				}

				// For move operation, source should no longer exist
				if tt.operation == MoveFile {
					if mockFS.FileExists(sourceFile) {
						t.Errorf("performFileOperation() source file still exists after move")
					}
				}

				// For copy operation, source should still exist
				if tt.operation == CopyFile {
					if !mockFS.FileExists(sourceFile) {
						t.Errorf("performFileOperation() source file doesn't exist after copy")
					}
				}
			}
		})
	}
}
