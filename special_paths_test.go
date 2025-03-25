package main

import (
	"path/filepath"
	"testing"
)

func TestFilePathsWithSpecialCharacters(t *testing.T) {
	t.Parallel()

	// Create a mock filesystem for testing
	mockFS := NewMockFS()

	// Test cases with various special characters
	testCases := []struct {
		name      string
		birdName  string
		fileName  string
		expectErr bool
	}{
		{
			name:      "Spaces in names",
			birdName:  "Test Bird With Spaces",
			fileName:  "recording with spaces.wav",
			expectErr: false,
		},
		{
			name:      "International characters",
			birdName:  "Höckerschwan", // German: Mute Swan
			fileName:  "aufnahme-höckerschwan.wav",
			expectErr: false,
		},
		{
			name:      "Special symbols",
			birdName:  "Bird (winter)",
			fileName:  "bird-winter!@#.wav",
			expectErr: false,
		},
		{
			name:      "Japanese characters",
			birdName:  "鴨 (カモ)",   // Japanese: Duck
			fileName:  "鳥の録音.wav", // Japanese: Bird recording
			expectErr: false,
		},
		{
			name:      "Arabic characters",
			birdName:  "طائر النحام",    // Arabic: Flamingo
			fileName:  "تسجيل-طائر.wav", // Arabic: Bird recording
			expectErr: false,
		},
		{
			name:      "Mixed characters",
			birdName:  "Pájaro Carpintero 啄木鸟",
			fileName:  "mixed-recording!@$%^.wav",
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test detection with special characters
			detection := &Detection{
				Date:       "2023-01-15",
				Time:       "13:45:30",
				SciName:    tc.birdName,
				ComName:    tc.birdName,
				Confidence: 0.85,
				FileName:   tc.fileName,
			}

			// Setup source directories and file
			sourceDir := "/source/" + tc.name
			targetDir := "/target/" + tc.name
			sourcePath := filepath.Join(sourceDir, "Extracted", "By_Date", detection.Date, detection.ComName, detection.FileName)

			// Create source directory structure and file
			mockFS.MkdirAll(filepath.Dir(sourcePath), 0o755)
			mockFS.WriteFile(sourcePath, []byte("Test audio content"), 0o644)

			// Handle file transfer
			handleFileTransferWithFS(detection, sourceDir, targetDir, CopyFile, mockFS)

			// Create expected target path
			clipName := GenerateClipName(detection)
			expectedTargetPath := filepath.Join(targetDir, "2023", "01", clipName)

			// Verify the file was copied successfully
			if !mockFS.FileExists(expectedTargetPath) {
				if !tc.expectErr {
					t.Errorf("Failed to copy file with special characters: %s", tc.birdName)
				}
			} else {
				// Success case - verify content
				content, err := mockFS.ReadFile(expectedTargetPath)
				if err != nil {
					t.Errorf("Error reading copied file: %v", err)
				} else if string(content) != "Test audio content" {
					t.Errorf("File content doesn't match original")
				}

				// Validate clip name follows expected format
				note := convertDetectionToNote(detection)
				if note.ClipName != clipName {
					t.Errorf("Generated clip name mismatch: got %s, expected %s", note.ClipName, clipName)
				}
			}
		})
	}
}
