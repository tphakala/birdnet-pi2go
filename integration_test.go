package main

import (
	"fmt"
	"log"
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

func setupIntegrationTest(t *testing.T) (sourceDBPath, sourceFilesDir string, testDetections []Detection, testContent []byte) {
	// Create temporary directories for this test
	tempDir := t.TempDir()
	sourceDBPath = filepath.Join(tempDir, "source.db")
	sourceFilesDir = filepath.Join(tempDir, "source_files")

	// Create source files directory structure - this will still happen on real disk
	// since we need the DB to be a real file, but we'll use MockFS for the file transfer
	extractedDir := filepath.Join(sourceFilesDir, "Extracted", "By_Date", "2023-01-15", "Test Bird")
	if err := os.MkdirAll(extractedDir, 0o755); err != nil {
		t.Fatalf("Failed to create source directory structure: %v", err)
	}

	// Create a test audio file for integration
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

	return sourceDBPath, sourceFilesDir, testDetections, testContent
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
	// Use a mock filesystem for file operations
	mockFS := NewMockFS()

	// Set up the source file first
	mockFS.MkdirAll(filepath.Dir(sourceFilePath), 0o755)
	mockFS.WriteFile(sourceFilePath, testContent, 0o644)

	// Create target directory
	mockFS.MkdirAll(filepath.Dir(targetFilePath), 0o755)

	// Perform the operation
	err := performFileOperationWithFS(sourceFilePath, targetFilePath, operation, mockFS)
	if err != nil {
		t.Fatalf("Failed to perform file operation: %v", err)
	}

	// Check if the target file exists
	if !mockFS.FileExists(targetFilePath) {
		t.Errorf("File not found at expected location: %s", targetFilePath)
		return
	}

	// Verify the content of the copied/moved file
	targetContent, err := mockFS.ReadFile(targetFilePath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if string(targetContent) != string(testContent) {
		t.Errorf("File content does not match source content")
	}

	// For move operation, source should no longer exist
	if operation == MoveFile {
		if mockFS.FileExists(sourceFilePath) {
			t.Errorf("Source file still exists after move operation")
		}
		return
	}

	// For copy operation, source should still exist
	if operation == CopyFile {
		if !mockFS.FileExists(sourceFilePath) {
			t.Errorf("Source file doesn't exist after copy operation")
		}
	}
}

// Create a mock implementation of convertAndTransferData that uses the mock filesystem
func convertAndTransferDataWithMockFS(sourceDBPath, targetDBPath, sourceFilesDir, targetFilesDir string, operation FileOperationType, skipAudioTransfer bool, mockFS FileSystem) {
	// This is a modified version of convertAndTransferData that uses a mock filesystem
	newLogger := createGormLogger()

	// Check if source database file exists
	if _, err := os.Stat(sourceDBPath); os.IsNotExist(err) {
		log.Printf("Source database file does not exist: %s", sourceDBPath)
		return
	}

	sourceDB, err := gorm.Open(sqlite.Open(sourceDBPath), &gorm.Config{Logger: newLogger})
	if err != nil {
		log.Fatalf("source db open: %v", err)
	}

	// Check if detections table exists
	var count int64
	err = sourceDB.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='detections'").Count(&count).Error
	if err != nil || count == 0 {
		log.Printf("detections table not found in source database: %s", sourceDBPath)
		return
	}

	targetDB := initializeAndMigrateTargetDB(targetDBPath, newLogger)

	lastNote, err := findLastEntryInTargetDB(targetDB)
	if err != nil {
		log.Fatalf("Error finding last entry in target database: %v", err)
	}

	whereClause, params := formulateQuery(lastNote)
	totalCount := getTotalRecordCount(sourceDB, whereClause, params...)
	fmt.Println("Total records to process:", totalCount)

	// Process records with the mock filesystem for file operations
	processRecordsWithMockFS(sourceDB, targetDB, totalCount, sourceFilesDir, targetFilesDir, operation, skipAudioTransfer, whereClause, params, mockFS)
	fmt.Println("Data conversion and file transfer completed successfully.")
}

// Process records using the mock filesystem
func processRecordsWithMockFS(sourceDB, targetDB *gorm.DB, totalCount int, sourceFilesDir, targetFilesDir string, operation FileOperationType, skipAudioTransfer bool, whereClause string, params []any, mockFS FileSystem) {
	const batchSize = 1000 // Define the size of each batch

	for offset := 0; offset < totalCount; offset += batchSize {
		batchDetections := fetchBatch(sourceDB, offset, batchSize, whereClause, params)
		fmt.Printf("Processing batch %d-%d of %d\n", offset+1, offset+len(batchDetections), totalCount)

		for i := range batchDetections {
			// Process each detection with the mock filesystem
			note := convertDetectionToNote(&batchDetections[i])
			if err := targetDB.Create(&note).Error; err != nil {
				log.Printf("Error inserting note: %v", err)
			}

			if !skipAudioTransfer {
				handleFileTransferWithFS(&batchDetections[i], sourceFilesDir, targetFilesDir, operation, mockFS)
			}
		}
	}
}

// Update the TestConvertAndTransferDataIntegration to use the mock filesystem
func TestConvertAndTransferDataIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sourceDBPath, sourceFilesDir, testDetections, testContent := setupIntegrationTest(t)

	// Create a mock filesystem and pre-populate it with source files
	mockFS := NewMockFS()

	// Pre-populate the mock filesystem with the source files from the setupIntegrationTest
	// Create the source directory structure in the mock filesystem
	testFileName := testDetections[0].FileName
	testDate := testDetections[0].Date
	testBirdName := testDetections[0].ComName
	sourceFilePath := filepath.Join(sourceFilesDir, "Extracted", "By_Date", testDate, testBirdName, testFileName)

	mockFS.MkdirAll(filepath.Dir(sourceFilePath), 0o755)
	mockFS.WriteFile(sourceFilePath, testContent, 0o644)

	// Run the function we want to test with copy operation
	t.Run("Copy operation", func(t *testing.T) {
		// Create directories for this subtest
		subTestDir := t.TempDir()
		subTargetDBPath := filepath.Join(subTestDir, "target_copy.db")
		subTargetFilesDir := filepath.Join(subTestDir, "target_files_copy")

		// Run the function with copy operation - using the mock filesystem
		convertAndTransferDataWithMockFS(sourceDBPath, subTargetDBPath, sourceFilesDir, subTargetFilesDir, CopyFile, false, mockFS)

		// Verify database records
		verifyNoteCount(t, subTargetDBPath, int64(len(testDetections)))

		// Verify file was copied correctly
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", "2023-01-15T13:45:30")
		expectedYear := parsedDate.Format("2006")
		expectedMonth := parsedDate.Format("01")
		expectedFilename := "testus_birdus_85p_20230115T134530Z.wav"

		expectedTargetPath := filepath.Join(subTargetFilesDir, expectedYear, expectedMonth, expectedFilename)

		// Check if the file exists in the mock filesystem
		if !mockFS.FileExists(expectedTargetPath) {
			t.Errorf("File not copied to expected location in mock filesystem: %s", expectedTargetPath)
		}
	})

	// Run the test for the skip audio transfer functionality
	t.Run("Skip audio transfer", func(t *testing.T) {
		// Create directories for this subtest
		subTestDir := t.TempDir()
		subTargetDBPath := filepath.Join(subTestDir, "target_skip.db")
		subTargetFilesDir := filepath.Join(subTestDir, "target_files_skip")

		// Run the function with skipAudioTransfer=true
		convertAndTransferDataWithMockFS(sourceDBPath, subTargetDBPath, sourceFilesDir, subTargetFilesDir, CopyFile, true, mockFS)

		// Verify database records
		verifyNoteCount(t, subTargetDBPath, int64(len(testDetections)))

		// The year/month directories should not be created since we're skipping file transfers
		expectedYear := "2023"
		expectedMonth := "01"
		expectedDirPath := filepath.Join(subTargetFilesDir, expectedYear, expectedMonth)

		// Check that the directory doesn't exist in the mock filesystem
		if mockFS.DirExists(expectedDirPath) {
			t.Errorf("Target directory was created in mock filesystem when skipAudioTransfer=true: %s", expectedDirPath)
		}
	})
}
