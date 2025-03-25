// Package main provides functionality for handling database operations in the BirdNet-Pi2Go project.
// file dbops.go
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Note represents a single observation data point.
type Note struct {
	// Standard GORM Model fields: ID, Date, etc.
	ID             uint   `gorm:"primaryKey"`
	Date           string `gorm:"index:idx_notes_date_commonname_confidence"`
	Time           string
	ScientificName string  `gorm:"index"`
	CommonName     string  `gorm:"index;index:idx_notes_date_commonname_confidence"`
	Confidence     float64 `gorm:"index:idx_notes_date_commonname_confidence"`
	Latitude       float64
	Longitude      float64
	Threshold      float64
	Sensitivity    float64
	ClipName       string
	Verified       string `gorm:"type:varchar(20);default:'unverified'"` // Status of the note verification
}

// Detection represents a detection event, directly mapping to the database structure.
type Detection struct {
	// Fields map directly to database columns with additional annotations for GORM.
	Date       string  `gorm:"column:Date"`
	Time       string  `gorm:"column:Time"`
	SciName    string  `gorm:"column:Sci_Name"`
	ComName    string  `gorm:"column:Com_Name"`
	Confidence float64 `gorm:"column:Confidence"`
	Lat        float64 `gorm:"column:Lat"`
	Lon        float64 `gorm:"column:Lon"`
	Cutoff     float64 `gorm:"column:Cutoff"`
	Week       int     `gorm:"column:Week"`
	Sens       float64 `gorm:"column:Sens"`
	Overlap    float64 `gorm:"column:Overlap"`
	FileName   string  `gorm:"column:File_Name"`
}

// TableName overrides the default table name.
func (Detection) TableName() string {
	return "detections"
}

// convertAndTransferData handles the main logic for data conversion and transfer.
func convertAndTransferData(sourceDBPath, targetDBPath, sourceFilesDir, targetFilesDir string, operation FileOperationType, skipAudioTransfer bool) {
	newLogger := createGormLogger()

	sourceDB, err := gorm.Open(sqlite.Open(sourceDBPath), &gorm.Config{Logger: newLogger})
	if err != nil {
		log.Fatalf("source db open: %v", err)
	}

	targetDB := initializeAndMigrateTargetDB(targetDBPath, newLogger)

	lastNote, err := findLastEntryInTargetDB(targetDB)
	if err != nil {
		log.Fatalf("Error finding last entry in target database: %v", err)
	}

	whereClause, params := formulateQuery(lastNote)
	totalCount := getTotalRecordCount(sourceDB, whereClause, params...)
	fmt.Println("Total records to process:", totalCount)

	processRecordsInBatches(sourceDB, targetDB, totalCount, sourceFilesDir, targetFilesDir, operation, skipAudioTransfer, whereClause, params)
	fmt.Println("Data conversion and file transfer completed successfully.")
}

// initializeAndMigrateTargetDB prepares the target database for data insertion.
func initializeAndMigrateTargetDB(targetDBPath string, newLogger logger.Interface) *gorm.DB {
	targetDB, err := gorm.Open(sqlite.Open(targetDBPath), &gorm.Config{Logger: newLogger})
	if err != nil {
		log.Fatalf("target db open: %v", err)
	}

	// Enable foreign key constraint enforcement for SQLite
	if err := targetDB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		log.Printf("failed to enable foreign key support in SQLite: %v", err)

		return nil
	}

	// Set SQLite to use MEMORY journal mode, reduces sdcard wear and improves performance
	if err := targetDB.Exec("PRAGMA journal_mode = MEMORY").Error; err != nil {
		log.Printf("failed to enable MEMORY journal mode in SQLite: %v", err)

		return nil
	}

	// Set SQLite to use NORMAL synchronous mode
	if err := targetDB.Exec("PRAGMA synchronous = OFF").Error; err != nil {
		log.Printf("failed to set synchronous mode in SQLite: %v", err)

		return nil
	}

	// Set SQLIte to use MEMORY temp store mode
	if err := targetDB.Exec("PRAGMA temp_store = MEMORY").Error; err != nil {
		log.Printf("failed to set temp store mode in SQLite: %v", err)
		return nil
	}

	// Increase cache size
	if err := targetDB.Exec("PRAGMA cache_size = -128000").Error; err != nil {
		log.Printf("failed to set cache size in SQLite: %v", err)
		return nil
	}

	// Perform auto-migration to create the table if it does not exist.
	if err := targetDB.AutoMigrate(&Note{}); err != nil {
		log.Fatalf("automigrate: %v", err)
	}

	return targetDB
}

// createGormLogger configures and returns a new GORM logger instance.
func createGormLogger() logger.Interface {
	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: 1 * time.Second,
			LogLevel:      logger.Error,
			Colorful:      true,
		},
	)
}

// getTotalRecordCount returns the total number of records in the source database
// that match the given whereClause and parameters.
func getTotalRecordCount(sourceDB *gorm.DB, whereClause string, params ...interface{}) int {
	var totalCount int64
	query := sourceDB.Model(&Detection{})

	if whereClause != "" {
		query = query.Where(whereClause, params...)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		log.Fatalf("Error counting source records: %v", err)
	}

	return int(totalCount)
}

// processRecordsInBatches processes records from the source database in batches,
// converting each record to a Note and optionally transferring files.
func processRecordsInBatches(sourceDB, targetDB *gorm.DB, totalCount int, sourceFilesDir, targetFilesDir string, operation FileOperationType, skipAudioTransfer bool, whereClause string, params []any) {
	const batchSize = 1000 // Define the size of each batch

	for offset := 0; offset < totalCount; offset += batchSize {
		batchDetections := fetchBatch(sourceDB, offset, batchSize, whereClause, params)
		fmt.Printf("Processing batch %d-%d of %d\n", offset+1, offset+len(batchDetections), totalCount)

		for i := range batchDetections {
			processDetection(targetDB, &batchDetections[i], sourceFilesDir, targetFilesDir, operation, skipAudioTransfer)
		}
	}
}

// fetchBatch retrieves a specific batch of Detection records from the source database,
// based on the provided offset and batchSize.
func fetchBatch(sourceDB *gorm.DB, offset, batchSize int, whereClause string, params []any) []Detection {
	var detections []Detection

	query := sourceDB.Model(&Detection{}).Order("date ASC, time ASC").Offset(offset).Limit(batchSize)

	if whereClause != "" {
		query = query.Where(whereClause, params...)
	}

	if err := query.Find(&detections).Error; err != nil {
		log.Fatalf("Error fetching batch: %v", err)
	}

	return detections
}

// processDetection takes a single Detection record, converts it to a Note,
// inserts it into the target database, and optionally handles file transfer
// if audio transfer is not skipped.
func processDetection(targetDB *gorm.DB, detection *Detection, sourceFilesDir, targetFilesDir string, operation FileOperationType, skipAudioTransfer bool) {
	note := convertDetectionToNote(detection)
	if err := targetDB.Create(&note).Error; err != nil {
		log.Printf("Error inserting note: %v", err)
	}

	if !skipAudioTransfer {
		go handleFileTransfer(detection, sourceFilesDir, targetFilesDir, operation)
	}
}

// convertDetectionToNote converts a Detection record into a Note record,
// preparing it for insertion into the target database.
func convertDetectionToNote(detection *Detection) Note {
	// Try parsing the date in both RFC3339 and simple date format
	parsedDate, err := time.Parse(time.RFC3339, detection.Date)
	if err != nil {
		// If RFC3339 fails, try simple date format
		parsedDate, err = time.Parse("2006-01-02", detection.Date)
		if err != nil {
			log.Printf("Error parsing date: %v, using original value", err)
		}
	}

	// Only update the date format if parsing was successful
	if err == nil {
		detection.Date = parsedDate.Format("2006-01-02")
	}

	clipName := GenerateClipName(detection)

	return Note{
		Date:           detection.Date,
		Time:           detection.Time,
		ScientificName: detection.SciName,
		CommonName:     detection.ComName,
		Confidence:     detection.Confidence,
		Latitude:       detection.Lat,
		Longitude:      detection.Lon,
		Threshold:      detection.Cutoff,
		Sensitivity:    detection.Sens,
		ClipName:       clipName,
		Verified:       "unverified",
	}
}

// findLastEntryInTargetDB queries the target database for the most recent Note entry.
func findLastEntryInTargetDB(targetDB *gorm.DB) (*Note, error) {
	var lastNote Note
	result := targetDB.Order("date DESC, time DESC").First(&lastNote)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// The database is empty. This is not an error condition for this function.
			return nil, nil // Return nils to indicate no records found gracefully.
		}
		// For other types of errors, return the error as is.
		return nil, result.Error
	}

	return &lastNote, nil
}

// formulateQuery constructs a SQL WHERE clause and its corresponding parameters
// based on the most recent Note entry in the target database.
func formulateQuery(lastNote *Note) (whereClause string, params []any) {
	if lastNote != nil {
		whereClause = "date > ? OR (date = ? AND time > ?)"
		params = []any{lastNote.Date, lastNote.Date, lastNote.Time}
		return whereClause, params
	}

	return "", nil
}

// MergeDatabases merges notes from sourceDB into targetDB.
func MergeDatabases(sourceDBPath, targetDBPath string) error {
	// Check if source database file exists
	if _, err := os.Stat(sourceDBPath); os.IsNotExist(err) {
		return fmt.Errorf("source database file does not exist: %s", sourceDBPath)
	}

	// Connect to the source database.
	sourceDB := initializeAndMigrateTargetDB(sourceDBPath, createGormLogger())

	// Verify we can query the source database (it must be a valid SQLite database with Notes table)
	var count int64
	if err := sourceDB.Model(&Note{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to access source database: %w", err)
	}

	targetDB := initializeAndMigrateTargetDB(targetDBPath, createGormLogger())

	// Determine the total number of notes in the source database.
	var totalNotes int64
	if err := sourceDB.Model(&Note{}).Count(&totalNotes).Error; err != nil {
		return fmt.Errorf("failed to count notes in source database: %w", err)
	}

	// Define the batch size.
	const batchSize = 1000
	// Calculate the number of batches needed.
	numBatches := (totalNotes + batchSize - 1) / batchSize

	for i := int64(0); i < numBatches; i++ {
		// Retrieve a batch of notes from the source database.
		var notes []Note
		if err := sourceDB.Limit(batchSize).Offset(int(i * batchSize)).Find(&notes).Error; err != nil {
			return fmt.Errorf("failed to retrieve batch of notes: %w", err)
		}

		// print progres
		fmt.Printf("Processing batch %d of %d\n", i+1, numBatches)

		// Insert each note in the batch into the target database without the ID field.
		for i := range notes {
			newNote := Note{
				Date:           notes[i].Date,
				Time:           notes[i].Time,
				ScientificName: notes[i].ScientificName,
				CommonName:     notes[i].CommonName,
				Confidence:     notes[i].Confidence,
				Latitude:       notes[i].Latitude,
				Longitude:      notes[i].Longitude,
				Threshold:      notes[i].Threshold,
				Sensitivity:    notes[i].Sensitivity,
				ClipName:       notes[i].ClipName,
				Verified:       notes[i].Verified,
			}

			if err := targetDB.Create(&newNote).Error; err != nil {
				log.Printf("Error inserting note: %v", err)

				continue // Adjust error handling as needed.
			}
		}
	}

	log.Println("Database merge completed successfully with batching.")
	return nil
}
