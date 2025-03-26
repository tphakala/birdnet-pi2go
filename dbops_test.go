package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Helper function to set up a test database
func setupTestDB(t *testing.T) (db *gorm.DB, dbPath string) {
	t.Helper()

	// Create a temporary directory for the database
	tempDir := t.TempDir()
	dbPath = filepath.Join(tempDir, "test.db")

	// Open a connection to the test database
	newLogger := logger.New(
		nil, // Don't log to stdout during tests
		logger.Config{
			SlowThreshold: 1 * time.Second,
			LogLevel:      logger.Silent,
			Colorful:      false,
		},
	)

	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(&Note{}); err != nil {
		t.Fatalf("Failed to migrate test schema: %v", err)
	}

	// Register cleanup function
	t.Cleanup(func() {
		// Get the underlying SQL DB
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	})

	return db, dbPath
}

func TestConvertDetectionToNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		detection *Detection
		want      Note
	}{
		{
			name: "Basic detection conversion",
			detection: &Detection{
				Date:       "2023-01-01",
				Time:       "12:34:56",
				SciName:    "Corvus corax",
				ComName:    "Common Raven",
				Confidence: 0.95,
				Lat:        42.123,
				Lon:        -71.456,
				Cutoff:     0.5,
				Sens:       1.0,
			},
			want: Note{
				Date:           "2023-01-01",
				Time:           "12:34:56",
				ScientificName: "Corvus corax",
				CommonName:     "Common Raven",
				Confidence:     0.95,
				Latitude:       42.123,
				Longitude:      -71.456,
				Threshold:      0.5,
				Sensitivity:    1.0,
				ClipName:       "corvus_corax_95p_20230101T123456Z.wav",
				Verified:       "unverified",
			},
		},
		{
			name: "RFC3339 format date",
			detection: &Detection{
				Date:       "2023-02-15T00:00:00Z", // RFC3339 format
				Time:       "10:30:45",
				SciName:    "Turdus merula",
				ComName:    "Common Blackbird",
				Confidence: 0.87,
				Lat:        51.507,
				Lon:        -0.128,
				Cutoff:     0.6,
				Sens:       1.2,
			},
			want: Note{
				Date:           "2023-02-15", // Should be converted to simple date format
				Time:           "10:30:45",
				ScientificName: "Turdus merula",
				CommonName:     "Common Blackbird",
				Confidence:     0.87,
				Latitude:       51.507,
				Longitude:      -0.128,
				Threshold:      0.6,
				Sensitivity:    1.2,
				ClipName:       "turdus_merula_87p_20230215T103045Z.wav",
				Verified:       "unverified",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertDetectionToNote(tt.detection)

			// Compare field by field for better error messages
			if got.Date != tt.want.Date {
				t.Errorf("convertDetectionToNote().Date = %v, want %v", got.Date, tt.want.Date)
			}
			if got.Time != tt.want.Time {
				t.Errorf("convertDetectionToNote().Time = %v, want %v", got.Time, tt.want.Time)
			}
			if got.ScientificName != tt.want.ScientificName {
				t.Errorf("convertDetectionToNote().ScientificName = %v, want %v", got.ScientificName, tt.want.ScientificName)
			}
			if got.CommonName != tt.want.CommonName {
				t.Errorf("convertDetectionToNote().CommonName = %v, want %v", got.CommonName, tt.want.CommonName)
			}
			if got.Confidence != tt.want.Confidence {
				t.Errorf("convertDetectionToNote().Confidence = %v, want %v", got.Confidence, tt.want.Confidence)
			}
			if got.Latitude != tt.want.Latitude {
				t.Errorf("convertDetectionToNote().Latitude = %v, want %v", got.Latitude, tt.want.Latitude)
			}
			if got.Longitude != tt.want.Longitude {
				t.Errorf("convertDetectionToNote().Longitude = %v, want %v", got.Longitude, tt.want.Longitude)
			}
			if got.Threshold != tt.want.Threshold {
				t.Errorf("convertDetectionToNote().Threshold = %v, want %v", got.Threshold, tt.want.Threshold)
			}
			if got.Sensitivity != tt.want.Sensitivity {
				t.Errorf("convertDetectionToNote().Sensitivity = %v, want %v", got.Sensitivity, tt.want.Sensitivity)
			}
			if got.ClipName != tt.want.ClipName {
				t.Errorf("convertDetectionToNote().ClipName = %v, want %v", got.ClipName, tt.want.ClipName)
			}
			if got.Verified != tt.want.Verified {
				t.Errorf("convertDetectionToNote().Verified = %v, want %v", got.Verified, tt.want.Verified)
			}
		})
	}
}

func TestFindLastEntryInTargetDB(t *testing.T) {
	t.Parallel()

	// Setup test database
	db, _ := setupTestDB(t)

	// Test with empty database
	t.Run("Empty database", func(t *testing.T) {
		note, err := findLastEntryInTargetDB(db)
		if err != nil {
			t.Fatalf("findLastEntryInTargetDB() error = %v", err)
		}
		if note != nil {
			t.Errorf("findLastEntryInTargetDB() with empty DB = %v, want nil", note)
		}
	})

	// Insert some test records
	testNotes := []Note{
		{
			Date:           "2023-01-01",
			Time:           "10:00:00",
			ScientificName: "Test Species 1",
			CommonName:     "Test Bird 1",
		},
		{
			Date:           "2023-01-01",
			Time:           "11:00:00",
			ScientificName: "Test Species 2",
			CommonName:     "Test Bird 2",
		},
		{
			Date:           "2023-01-02",
			Time:           "09:00:00",
			ScientificName: "Test Species 3",
			CommonName:     "Test Bird 3",
		},
	}

	for _, n := range testNotes {
		if err := db.Create(&n).Error; err != nil {
			t.Fatalf("Failed to create test note: %v", err)
		}
	}

	// Test finding the last entry
	t.Run("Database with entries", func(t *testing.T) {
		lastNote, err := findLastEntryInTargetDB(db)
		if err != nil {
			t.Fatalf("findLastEntryInTargetDB() error = %v", err)
		}

		// Check that we got the correct last entry
		if lastNote == nil {
			t.Fatal("findLastEntryInTargetDB() returned nil, expected a note")
		}
		if lastNote.Date != "2023-01-02" || lastNote.Time != "09:00:00" {
			t.Errorf("findLastEntryInTargetDB() = %v, want note with Date='2023-01-02', Time='09:00:00'", lastNote)
		}
	})

	// Test with database error
	t.Run("Database error", func(t *testing.T) {
		// Create a DB with no table
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "invalid.db")
		newLogger := logger.New(nil, logger.Config{LogLevel: logger.Silent})
		invalidDB, _ := gorm.Open(sqlite.Open(dbPath), &gorm.Config{Logger: newLogger})

		// This should return an error since the table doesn't exist
		_, err := findLastEntryInTargetDB(invalidDB)
		if err == nil {
			t.Errorf("findLastEntryInTargetDB() with invalid DB did not return an error")
		}
	})
}

func TestFormulateQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		lastNote        *Note
		wantWhereClause string
		wantParamsLen   int
	}{
		{
			name:            "Nil last note",
			lastNote:        nil,
			wantWhereClause: "",
			wantParamsLen:   0,
		},
		{
			name: "With last note",
			lastNote: &Note{
				Date: "2023-01-01",
				Time: "12:34:56",
			},
			wantWhereClause: "date > ? OR (date = ? AND time > ?)",
			wantParamsLen:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			whereClause, params := formulateQuery(tt.lastNote)

			if whereClause != tt.wantWhereClause {
				t.Errorf("formulateQuery() whereClause = %v, want %v", whereClause, tt.wantWhereClause)
			}

			if len(params) != tt.wantParamsLen {
				t.Errorf("formulateQuery() params length = %v, want %v", len(params), tt.wantParamsLen)
			}

			// Check specific parameters for the case with a last note
			if tt.lastNote != nil {
				if len(params) >= 3 {
					if params[0] != tt.lastNote.Date {
						t.Errorf("formulateQuery() params[0] = %v, want %v", params[0], tt.lastNote.Date)
					}
					if params[1] != tt.lastNote.Date {
						t.Errorf("formulateQuery() params[1] = %v, want %v", params[1], tt.lastNote.Date)
					}
					if params[2] != tt.lastNote.Time {
						t.Errorf("formulateQuery() params[2] = %v, want %v", params[2], tt.lastNote.Time)
					}
				} else {
					t.Errorf("formulateQuery() params too short, got length %v, expected at least 3", len(params))
				}
			}
		})
	}
}

func TestMergeDatabases(t *testing.T) {
	// Setup source and target databases
	sourceDB, sourceDBPath := setupTestDB(t)
	targetDB, targetDBPath := setupTestDB(t)

	// Insert some test records into the source database
	testNotes := []Note{
		{
			Date:           "2023-01-01",
			Time:           "10:00:00",
			ScientificName: "Test Species 1",
			CommonName:     "Test Bird 1",
			Confidence:     0.9,
			ClipName:       "test1.wav",
		},
		{
			Date:           "2023-01-02",
			Time:           "11:00:00",
			ScientificName: "Test Species 2",
			CommonName:     "Test Bird 2",
			Confidence:     0.8,
			ClipName:       "test2.wav",
		},
	}

	for _, n := range testNotes {
		if err := sourceDB.Create(&n).Error; err != nil {
			t.Fatalf("Failed to create test note in source DB: %v", err)
		}
	}

	// Call the MergeDatabases function
	err := MergeDatabases(sourceDBPath, targetDBPath)
	if err != nil {
		t.Fatalf("MergeDatabases() error = %v", err)
	}

	// Check if records were merged correctly
	var count int64
	if err := targetDB.Model(&Note{}).Count(&count).Error; err != nil {
		t.Fatalf("Failed to count records in target DB: %v", err)
	}

	if count != int64(len(testNotes)) {
		t.Errorf("MergeDatabases() resulted in %d records in target DB, want %d", count, len(testNotes))
	}

	// Check specific records
	for i, expected := range testNotes {
		var actual Note
		if err := targetDB.Where("scientific_name = ?", expected.ScientificName).First(&actual).Error; err != nil {
			t.Errorf("Failed to find record %d in target DB: %v", i, err)
			continue
		}

		// Check a few important fields
		if actual.Date != expected.Date {
			t.Errorf("Record %d: Date = %s, want %s", i, actual.Date, expected.Date)
		}
		if actual.CommonName != expected.CommonName {
			t.Errorf("Record %d: CommonName = %s, want %s", i, actual.CommonName, expected.CommonName)
		}
		if actual.Confidence != expected.Confidence {
			t.Errorf("Record %d: Confidence = %f, want %f", i, actual.Confidence, expected.Confidence)
		}
	}

	// Test merging into a database with existing records
	// Insert a record into the target database
	additionalNote := Note{
		Date:           "2023-01-03",
		Time:           "12:00:00",
		ScientificName: "Test Species 3",
		CommonName:     "Test Bird 3",
		Confidence:     0.7,
		ClipName:       "test3.wav",
	}
	if err := targetDB.Create(&additionalNote).Error; err != nil {
		t.Fatalf("Failed to create additional note in target DB: %v", err)
	}

	// Create a new source database for second merge
	sourceDB2, sourceDBPath2 := setupTestDB(t)
	sourceNote := Note{
		Date:           "2023-01-04",
		Time:           "13:00:00",
		ScientificName: "Test Species 4",
		CommonName:     "Test Bird 4",
		Confidence:     0.6,
		ClipName:       "test4.wav",
	}
	if err := sourceDB2.Create(&sourceNote).Error; err != nil {
		t.Fatalf("Failed to create note in second source DB: %v", err)
	}

	// Merge the second source into the target
	err = MergeDatabases(sourceDBPath2, targetDBPath)
	if err != nil {
		t.Fatalf("Second MergeDatabases() error = %v", err)
	}

	// Check if all records are present
	var finalCount int64
	if err := targetDB.Model(&Note{}).Count(&finalCount).Error; err != nil {
		t.Fatalf("Failed to count final records in target DB: %v", err)
	}

	expectedFinalCount := int64(len(testNotes) + 2) // Original + additional + new source
	if finalCount != expectedFinalCount {
		t.Errorf("After second merge, got %d records, want %d", finalCount, expectedFinalCount)
	}

	// Test error case - non-existent source database
	err = MergeDatabases("/non/existent/path/to/nonexistent.db", targetDBPath)
	if err == nil {
		t.Errorf("MergeDatabases() with non-existent source did not return an error")
	}
}

// Mock functions for dependencies to enable more advanced testing
type mockDetectionTable struct {
	t  *testing.T
	db *gorm.DB
}

func newMockDetectionTable(t *testing.T) (table *mockDetectionTable, dbPath string) {
	t.Helper()

	db, dbPath := setupTestDB(t)

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

	return &mockDetectionTable{t: t, db: db}, dbPath
}

func (m *mockDetectionTable) insertDetections(detections []Detection) {
	for i := range detections {
		err := m.db.Exec(
			`INSERT INTO detections (Date, Time, Sci_Name, Com_Name, Confidence, Lat, Lon, Cutoff, Week, Sens, Overlap, File_Name) 
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			detections[i].Date, detections[i].Time, detections[i].SciName, detections[i].ComName, detections[i].Confidence,
			detections[i].Lat, detections[i].Lon, detections[i].Cutoff, detections[i].Week, detections[i].Sens,
			detections[i].Overlap, detections[i].FileName,
		).Error

		if err != nil {
			m.t.Fatalf("Failed to insert mock detection: %v", err)
		}
	}
}

func TestMergeDatabasesWithRealDB(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for target DB
	tempDir := t.TempDir()
	targetDBPath := filepath.Join(tempDir, "target_test.db")
	sourceDBPath := filepath.Join(tempDir, "source_test.db")

	// Create an empty target database
	targetDB, err := gorm.Open(sqlite.Open(targetDBPath), &gorm.Config{
		Logger: logger.New(
			nil, // Don't log to stdout during tests
			logger.Config{
				SlowThreshold: 1 * time.Second,
				LogLevel:      logger.Silent,
				Colorful:      false,
			},
		),
	})
	if err != nil {
		t.Fatalf("Failed to create target database: %v", err)
	}

	// Migrate schema to create the Note table in target DB
	if err := targetDB.AutoMigrate(&Note{}); err != nil {
		t.Fatalf("Failed to migrate target schema: %v", err)
	}

	// Add a few notes to the target database (to test merging with existing data)
	initialNotes := []Note{
		{
			Date:           "2023-12-01",
			Time:           "09:00:00",
			ScientificName: "Existing Species 1",
			CommonName:     "Existing Bird 1",
			Confidence:     0.8,
			ClipName:       "existing1.wav",
			Verified:       "unverified",
		},
		{
			Date:           "2023-12-02",
			Time:           "10:30:00",
			ScientificName: "Existing Species 2",
			CommonName:     "Existing Bird 2",
			Confidence:     0.7,
			ClipName:       "existing2.wav",
			Verified:       "unverified",
		},
	}

	for _, note := range initialNotes {
		if err := targetDB.Create(&note).Error; err != nil {
			t.Fatalf("Failed to create initial note in target DB: %v", err)
		}
	}

	// Count initial records in target DB
	var initialCount int64
	if err := targetDB.Model(&Note{}).Count(&initialCount).Error; err != nil {
		t.Fatalf("Failed to count initial records: %v", err)
	}

	t.Logf("Initial count in target DB: %d", initialCount)

	// Close the target DB connection to ensure changes are flushed
	targetSqlDB, err := targetDB.DB()
	if err == nil {
		targetSqlDB.Close()
	}

	// Create a custom source database with detections table for testing
	sourceDB, err := gorm.Open(sqlite.Open(sourceDBPath), &gorm.Config{
		Logger: logger.New(
			nil, // Don't log to stdout during tests
			logger.Config{
				SlowThreshold: 1 * time.Second,
				LogLevel:      logger.Silent,
				Colorful:      false,
			},
		),
	})
	if err != nil {
		t.Fatalf("Failed to create source database: %v", err)
	}

	// Create detections table
	err = sourceDB.Exec(`
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
		t.Fatalf("Failed to create detections table: %v", err)
	}

	// Create test detections
	testDetections := []Detection{
		{
			Date:       "2023-01-01",
			Time:       "12:00:00",
			SciName:    "Corvus corax",
			ComName:    "Common Raven",
			Confidence: 0.9,
			Lat:        42.0,
			Lon:        -71.0,
			Cutoff:     0.5,
			Week:       1,
			Sens:       1.0,
			Overlap:    0.0,
			FileName:   "raven_audio.wav",
		},
		{
			Date:       "2023-01-02",
			Time:       "13:30:00",
			SciName:    "Turdus merula",
			ComName:    "Common Blackbird",
			Confidence: 0.85,
			Lat:        42.0,
			Lon:        -71.0,
			Cutoff:     0.5,
			Week:       1,
			Sens:       1.0,
			Overlap:    0.0,
			FileName:   "blackbird_audio.wav",
		},
		{
			Date:       "2023-01-03",
			Time:       "09:15:00",
			SciName:    "Cyanistes caeruleus",
			ComName:    "Eurasian Blue Tit",
			Confidence: 0.78,
			Lat:        42.0,
			Lon:        -71.0,
			Cutoff:     0.5,
			Week:       1,
			Sens:       1.0,
			Overlap:    0.0,
			FileName:   "bluetit_audio.wav",
		},
	}

	// Insert detections into source database
	for _, detection := range testDetections {
		err := sourceDB.Exec(
			`INSERT INTO detections (Date, Time, Sci_Name, Com_Name, Confidence, Lat, Lon, Cutoff, Week, Sens, Overlap, File_Name) 
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			detection.Date, detection.Time, detection.SciName, detection.ComName, detection.Confidence,
			detection.Lat, detection.Lon, detection.Cutoff, detection.Week, detection.Sens,
			detection.Overlap, detection.FileName,
		).Error

		if err != nil {
			t.Fatalf("Failed to insert detection: %v", err)
		}
	}

	// Verify detections were inserted
	var detectionsCount int64
	if err := sourceDB.Raw("SELECT COUNT(*) FROM detections").Count(&detectionsCount).Error; err != nil {
		t.Fatalf("Failed to count detections: %v", err)
	}
	t.Logf("Inserted %d detections in source DB", detectionsCount)

	// Close source DB to flush changes
	sourceSqlDB, err := sourceDB.DB()
	if err == nil {
		sourceSqlDB.Close()
	}

	// Implement a custom merge function to test the conversion and merge logic
	err = customMergeDetectionsToNotes(sourceDBPath, targetDBPath)
	if err != nil {
		t.Fatalf("customMergeDetectionsToNotes() error = %v", err)
	}

	// Reopen the target database to check results
	targetDB, err = gorm.Open(sqlite.Open(targetDBPath), &gorm.Config{
		Logger: logger.New(
			nil, // Don't log to stdout during tests
			logger.Config{
				SlowThreshold: 1 * time.Second,
				LogLevel:      logger.Silent,
				Colorful:      false,
			},
		),
	})
	if err != nil {
		t.Fatalf("Failed to reopen target database: %v", err)
	}

	// Count records after merge
	var finalCount int64
	if err := targetDB.Model(&Note{}).Count(&finalCount).Error; err != nil {
		t.Fatalf("Failed to count final records: %v", err)
	}

	t.Logf("Final count in target DB: %d", finalCount)

	// List all records in target DB for debugging
	var allNotes []Note
	if err := targetDB.Find(&allNotes).Error; err != nil {
		t.Logf("Failed to fetch all notes for debugging: %v", err)
	} else {
		for i, note := range allNotes {
			t.Logf("Note #%d: Date=%s, Time=%s, ScientificName=%s, CommonName=%s",
				i, note.Date, note.Time, note.ScientificName, note.CommonName)
		}
	}

	// Verify that records were merged correctly
	expectedCount := initialCount + int64(len(testDetections))
	if finalCount != expectedCount {
		t.Errorf("Merge resulted in %d records, expected %d", finalCount, expectedCount)
	}

	// Check if original records are still there
	for i, expected := range initialNotes {
		var count int64
		if err := targetDB.Model(&Note{}).Where("scientific_name = ?", expected.ScientificName).Count(&count).Error; err != nil {
			t.Errorf("Failed to verify original record existence: %v", err)
		}
		if count == 0 {
			t.Errorf("Original record #%d '%s' not found after merge", i, expected.ScientificName)
		}
	}

	// Check if merged detections are there (converted to Notes)
	for i, detection := range testDetections {
		var count int64
		if err := targetDB.Model(&Note{}).Where("scientific_name = ?", detection.SciName).Count(&count).Error; err != nil {
			t.Errorf("Failed to verify merged record existence: %v", err)
		}
		if count == 0 {
			t.Errorf("Merged record #%d '%s' not found after merge", i, detection.SciName)
		} else {
			t.Logf("Found %d records in target DB for '%s'", count, detection.SciName)
		}
	}
}

// customMergeDetectionsToNotes implements a merge function that handles
// the conversion from Detection records to Note records for testing
func customMergeDetectionsToNotes(sourceDBPath, targetDBPath string) error {
	// Connect to source database
	sourceDB, err := gorm.Open(sqlite.Open(sourceDBPath), &gorm.Config{
		Logger: logger.New(nil, logger.Config{LogLevel: logger.Silent}),
	})
	if err != nil {
		return fmt.Errorf("failed to open source DB: %w", err)
	}

	// Connect to target database
	targetDB, err := gorm.Open(sqlite.Open(targetDBPath), &gorm.Config{
		Logger: logger.New(nil, logger.Config{LogLevel: logger.Silent}),
	})
	if err != nil {
		return fmt.Errorf("failed to open target DB: %w", err)
	}

	// Get detections from source database
	var detections []Detection
	if err := sourceDB.Raw("SELECT * FROM detections").Scan(&detections).Error; err != nil {
		return fmt.Errorf("failed to get detections from source DB: %w", err)
	}

	// Process each detection - convert to Note and save to target DB
	for _, detection := range detections {
		note := convertDetectionToNote(&detection)
		if err := targetDB.Create(&note).Error; err != nil {
			return fmt.Errorf("failed to insert note: %w", err)
		}
	}

	return nil
}
