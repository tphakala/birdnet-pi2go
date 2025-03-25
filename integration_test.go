package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// mockDetectionTable creates a mock "detections" table in the source database
func createMockDetectionTable(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Execute raw SQL to create the table since we're using a specific structure
	err := db.Exec(`
		CREATE TABLE detections (
			Date TEXT,
			Time TEXT,
			Sci_Name TEXT,
			Com_Name TEXT,
			Confidence REAL,
			Lat REAL,
			Lon REAL,
			Cutoff REAL,
			Week INTEGER,
			Sens REAL,
			Overlap REAL,
			File_Name TEXT
		)
	`).Error

	if err != nil {
		t.Fatalf("Failed to create mock detections table: %v", err)
	}
}

// insertMockDetection adds a sample detection record to the source database
func insertMockDetection(t *testing.T, db *gorm.DB, detection *Detection) {
	t.Helper()

	err := db.Exec(
		`INSERT INTO detections (Date, Time, Sci_Name, Com_Name, Confidence, Lat, Lon, Cutoff, Week, Sens, Overlap, File_Name) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		detection.Date, detection.Time, detection.SciName, detection.ComName, detection.Confidence,
		detection.Lat, detection.Lon, detection.Cutoff, detection.Week, detection.Sens,
		detection.Overlap, detection.FileName,
	).Error

	if err != nil {
		t.Fatalf("Failed to insert mock detection: %v", err)
	}
}

func setupIntegrationTest(t *testing.T) (tempDir, sourceDBPath, sourceFilesDir string, testDetections []Detection, testContent []byte) {
	// Create temporary directories for this test
	tempDir = t.TempDir()
	sourceDBPath = filepath.Join(tempDir, "source.db")
	sourceFilesDir = filepath.Join(tempDir, "source_files")

	// Create source files directory structure
	extractedDir := filepath.Join(sourceFilesDir, "Extracted", "By_Date", "2023-01-15", "Test Bird")
	if err := os.MkdirAll(extractedDir, 0o755); err != nil {
		t.Fatalf("Failed to create source directory structure: %v", err)
	}

	// Create a test audio file
	testFileName := "test_audio.wav"
	sourceFilePath := filepath.Join(extractedDir, testFileName)
	testContent = []byte("This is test audio content for integration test")
	if err := os.WriteFile(sourceFilePath, testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}

	// Setup source database with mock data
	newLogger := logger.New(
		nil, // Don't log to stdout during tests
		logger.Config{
			SlowThreshold: 1 * time.Second,
			LogLevel:      logger.Silent,
			Colorful:      false,
		},
	)

	sourceDB, err := gorm.Open(sqlite.Open(sourceDBPath), &gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("Failed to open source database: %v", err)
	}

	// Create the mock detections table in the source database
	createMockDetectionTable(t, sourceDB)

	// Insert test detections
	testDetections = []Detection{
		{
			Date:       "2023-01-15",
			Time:       "13:45:30",
			SciName:    "Testus birdus",
			ComName:    "Test Bird",
			Confidence: 0.85,
			Lat:        42.123,
			Lon:        -71.456,
			Cutoff:     0.5,
			Week:       3,
			Sens:       1.0,
			Overlap:    0.0,
			FileName:   testFileName,
		},
		{
			Date:       "2023-01-16",
			Time:       "09:15:00",
			SciName:    "Avius testus",
			ComName:    "Another Test Bird",
			Confidence: 0.92,
			Lat:        42.456,
			Lon:        -71.789,
			Cutoff:     0.6,
			Week:       3,
			Sens:       1.0,
			Overlap:    0.0,
			FileName:   "another_test.wav", // This file doesn't exist, so it will be skipped
		},
	}

	for i := range testDetections {
		insertMockDetection(t, sourceDB, &testDetections[i])
	}

	return tempDir, sourceDBPath, sourceFilesDir, testDetections, testContent
}

func verifyNoteCount(t *testing.T, targetDBPath string, expectedCount int64) {
	newLogger := logger.New(
		nil,
		logger.Config{
			SlowThreshold: 1 * time.Second,
			LogLevel:      logger.Silent,
			Colorful:      false,
		},
	)

	targetDB, err := gorm.Open(sqlite.Open(targetDBPath), &gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("Failed to open target database: %v", err)
	}

	var count int64
	if err := targetDB.Model(&Note{}).Count(&count).Error; err != nil {
		t.Fatalf("Failed to count notes in target database: %v", err)
	}

	if count != expectedCount {
		t.Errorf("Expected %d notes, got %d", expectedCount, count)
	}
}

func verifyFileOperation(t *testing.T, operation FileOperationType, sourceFilePath, targetFilePath string, testContent []byte) {
	// Check if the target file exists
	if _, err := os.Stat(targetFilePath); os.IsNotExist(err) {
		t.Errorf("File not found at expected location: %s", targetFilePath)
		return
	}

	// Verify the content of the copied/moved file
	targetContent, err := os.ReadFile(targetFilePath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if !bytes.Equal(targetContent, testContent) {
		t.Errorf("File content does not match source content")
	}

	// For move operation, source should no longer exist
	if operation == MoveFile {
		if _, err := os.Stat(sourceFilePath); !os.IsNotExist(err) {
			t.Errorf("Source file still exists after move operation")
		}
		return
	}

	// For copy operation, source should still exist
	if operation == CopyFile {
		if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
			t.Errorf("Source file doesn't exist after copy operation")
		}
	}
}

func TestConvertAndTransferDataIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, sourceDBPath, sourceFilesDir, testDetections, testContent := setupIntegrationTest(t)

	// Run the function we want to test with copy operation
	t.Run("Copy operation", func(t *testing.T) {
		// Create directories for this subtest
		subTestDir := t.TempDir()
		subTargetDBPath := filepath.Join(subTestDir, "target_copy.db")
		subTargetFilesDir := filepath.Join(subTestDir, "target_files_copy")

		// Run the function with copy operation
		convertAndTransferData(sourceDBPath, subTargetDBPath, sourceFilesDir, subTargetFilesDir, CopyFile, false)

		// Verify database records
		verifyNoteCount(t, subTargetDBPath, int64(len(testDetections)))

		// Verify file was copied correctly
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", "2023-01-15T13:45:30")
		expectedYear := parsedDate.Format("2006")
		expectedMonth := parsedDate.Format("01")
		expectedFilename := "testus_birdus_85p_20230115T134530Z.wav"

		sourceFilePath := filepath.Join(sourceFilesDir, "Extracted", "By_Date", "2023-01-15", "Test Bird", testDetections[0].FileName)
		expectedTargetPath := filepath.Join(subTargetFilesDir, expectedYear, expectedMonth, expectedFilename)

		verifyFileOperation(t, CopyFile, sourceFilePath, expectedTargetPath, testContent)
	})

	// Run the function we want to test with move operation
	t.Run("Move operation", func(t *testing.T) {
		// Create a copy of the source file for move operation
		moveSourceDir := filepath.Join(tempDir, "move_source_files")
		moveExtractedDir := filepath.Join(moveSourceDir, "Extracted", "By_Date", "2023-01-15", "Test Bird")
		if err := os.MkdirAll(moveExtractedDir, 0o755); err != nil {
			t.Fatalf("Failed to create move source directory structure: %v", err)
		}

		moveSourceFilePath := filepath.Join(moveExtractedDir, testDetections[0].FileName)
		if err := os.WriteFile(moveSourceFilePath, testContent, 0o644); err != nil {
			t.Fatalf("Failed to create test audio file for move: %v", err)
		}

		// Create directories for this subtest
		subTestDir := t.TempDir()
		subTargetDBPath := filepath.Join(subTestDir, "target_move.db")
		subTargetFilesDir := filepath.Join(subTestDir, "target_files_move")

		// Run the function with move operation
		convertAndTransferData(sourceDBPath, subTargetDBPath, moveSourceDir, subTargetFilesDir, MoveFile, false)

		// Verify database records
		verifyNoteCount(t, subTargetDBPath, int64(len(testDetections)))

		// Verify file was moved correctly
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", "2023-01-15T13:45:30")
		expectedYear := parsedDate.Format("2006")
		expectedMonth := parsedDate.Format("01")
		expectedFilename := "testus_birdus_85p_20230115T134530Z.wav"

		expectedTargetPath := filepath.Join(subTargetFilesDir, expectedYear, expectedMonth, expectedFilename)

		verifyFileOperation(t, MoveFile, moveSourceFilePath, expectedTargetPath, testContent)
	})

	// Test the skip audio transfer functionality
	t.Run("Skip audio transfer", func(t *testing.T) {
		// Create directories for this subtest
		subTestDir := t.TempDir()
		subTargetDBPath := filepath.Join(subTestDir, "target_skip.db")
		subTargetFilesDir := filepath.Join(subTestDir, "target_files_skip")

		// Run the function with skipAudioTransfer=true
		convertAndTransferData(sourceDBPath, subTargetDBPath, sourceFilesDir, subTargetFilesDir, CopyFile, true)

		// Verify database records
		verifyNoteCount(t, subTargetDBPath, int64(len(testDetections)))

		// Check that target directory structure wasn't created
		expectedYear := "2023"
		expectedMonth := "01"
		expectedDirPath := filepath.Join(subTargetFilesDir, expectedYear, expectedMonth)

		if _, err := os.Stat(expectedDirPath); !os.IsNotExist(err) {
			t.Errorf("Target directory was created when skipAudioTransfer=true: %s", expectedDirPath)
		}
	})

	// Test with incremental data (already existing data in target)
	t.Run("Incremental data", func(t *testing.T) {
		// Create directories for this subtest
		subTestDir := t.TempDir()
		subTargetDBPath := filepath.Join(subTestDir, "target_incremental.db")
		subTargetFilesDir := filepath.Join(subTestDir, "target_files_incremental")

		// First run to create initial data
		convertAndTransferData(sourceDBPath, subTargetDBPath, sourceFilesDir, subTargetFilesDir, CopyFile, false)

		// Create a new source DB with additional data
		incrementalDBPath := filepath.Join(subTestDir, "source_incremental.db")
		incrementalDB, err := gorm.Open(sqlite.Open(incrementalDBPath), &gorm.Config{Logger: logger.New(
			nil,
			logger.Config{
				SlowThreshold: 1 * time.Second,
				LogLevel:      logger.Silent,
				Colorful:      false,
			},
		)})
		if err != nil {
			t.Fatalf("Failed to open incremental source database: %v", err)
		}

		// Create the mock detections table and add a new detection
		createMockDetectionTable(t, incrementalDB)
		newDetection := Detection{
			Date:       "2023-01-17", // Later date than previous detections
			Time:       "14:30:00",
			SciName:    "Novus incremental",
			ComName:    "New Incremental Bird",
			Confidence: 0.95,
			Lat:        43.123,
			Lon:        -72.456,
			Cutoff:     0.6,
			Week:       3,
			Sens:       1.0,
			Overlap:    0.0,
			FileName:   "incremental.wav",
		}
		insertMockDetection(t, incrementalDB, &newDetection)

		// Run the function with the new source DB
		convertAndTransferData(incrementalDBPath, subTargetDBPath, sourceFilesDir, subTargetFilesDir, CopyFile, true)

		// Verify combined record count
		verifyNoteCount(t, subTargetDBPath, int64(len(testDetections)+1))

		// Verify the new detection was added
		newLogger := logger.New(nil, logger.Config{LogLevel: logger.Silent})
		targetDB, err := gorm.Open(sqlite.Open(subTargetDBPath), &gorm.Config{Logger: newLogger})
		if err != nil {
			t.Fatalf("Failed to open target database: %v", err)
		}

		var newNote Note
		result := targetDB.Where("scientific_name = ?", newDetection.SciName).First(&newNote)
		if result.Error != nil {
			t.Errorf("New detection was not added to the target database: %v", result.Error)
		}
	})

	// Test error cases
	t.Run("Error cases", func(t *testing.T) {
		// Test with non-existent source database
		subTestDir := t.TempDir()
		nonExistentSource := filepath.Join(subTestDir, "nonexistent.db")
		validTarget := filepath.Join(subTestDir, "target.db")

		// This won't panic but might return silently or log an error
		convertAndTransferData(nonExistentSource, validTarget, "", "", CopyFile, true)

		// Verify no database was created
		if _, err := os.Stat(validTarget); !os.IsNotExist(err) {
			t.Errorf("Database was created with non-existent source")
		}
	})
}
