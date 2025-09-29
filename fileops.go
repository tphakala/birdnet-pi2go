// file fileops.go
package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileSystem defines an interface for file operations that can be mocked in tests
type FileSystem interface {
	MkdirAll(path string, perm fs.FileMode) error
	Stat(name string) (fs.FileInfo, error)
	Remove(name string) error
	Create(name string) (io.WriteCloser, error)
	Open(name string) (io.ReadCloser, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	FileExists(name string) bool
}

// OsFS implements FileSystem using the os package
type OsFS struct{}

func (fs OsFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs OsFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (fs OsFS) Remove(name string) error {
	return os.Remove(name)
}

func (fs OsFS) Create(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

func (fs OsFS) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

func (fs OsFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (fs OsFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (fs OsFS) FileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

// DefaultFS is the default filesystem implementation
var DefaultFS FileSystem = OsFS{}

// handleFileTransfer processes a detection record, copying or moving the audio file to the target location
func handleFileTransfer(detection *Detection, sourceFilesDir, targetFilesDir string, operation FileOperationType) {
	handleFileTransferWithFS(detection, sourceFilesDir, targetFilesDir, operation, DefaultFS)
}

// handleFileTransferWithFS processes a detection record, copying or moving the audio file using the provided filesystem implementation
func handleFileTransferWithFS(detection *Detection, sourceFilesDir, targetFilesDir string, operation FileOperationType, fs FileSystem) {
	// Construct the path to the source audio file
	sourceFilePath := filepath.Join(sourceFilesDir, "Extracted", "By_Date", detection.Date, detection.ComName, detection.FileName)

	// Check if the source file exists
	if !fs.FileExists(sourceFilePath) {
		// detection.ComName may have had spaces replaced with underscores and apostrophe's removed
		comNameFormatted := strings.ReplaceAll(detection.ComName, " ", "_")
		comNameFormatted = strings.ReplaceAll(comNameFormatted, "'", "")
		sourceFilePath = filepath.Join(sourceFilesDir, "Extracted", "By_Date", detection.Date, comNameFormatted, detection.FileName)
		if !fs.FileExists(sourceFilePath) {
			log.Printf("Source file not found: %s", sourceFilePath)
			return
		}
	}

	// Generate a new filename that follows the BIRDNET-Pi naming convention
	newFileName := GenerateClipName(detection)

	// Parse the date from the detection to determine target subdirectories
	parsedDate, err := time.Parse("2006-01-02T15:04:05", detection.Date+"T"+detection.Time)
	if err != nil {
		log.Printf("Error parsing date: %v", err)
		return
	}

	// Format the date for target directory structure (year/month)
	year := parsedDate.Format("2006")
	month := parsedDate.Format("01")

	// Construct the full target path
	targetSubDir := filepath.Join(targetFilesDir, year, month)
	targetFilePath := filepath.Join(targetSubDir, newFileName)

	// Ensure target directory exists
	err = fs.MkdirAll(targetSubDir, 0o755)
	if err != nil {
		log.Printf("Failed to create subdirectories: %v", err)
		return
	}

	// Perform the file operation based on the specified operation type
	switch operation {
	case CopyFile:
		// Read the source file
		data, err := fs.ReadFile(sourceFilePath)
		if err != nil {
			log.Printf("Failed to read source file: %v", err)
			return
		}

		// Write to the target file
		err = fs.WriteFile(targetFilePath, data, 0o644)
		if err != nil {
			log.Printf("Failed to write target file: %v", err)
			return
		}

		log.Printf("Copied %s to %s", sourceFilePath, targetFilePath)

	case MoveFile:
		// Read the source file
		data, err := fs.ReadFile(sourceFilePath)
		if err != nil {
			log.Printf("Failed to read source file: %v", err)
			return
		}

		// Write to the target file
		err = fs.WriteFile(targetFilePath, data, 0o644)
		if err != nil {
			log.Printf("Failed to write target file: %v", err)
			return
		}

		// Remove the source file
		err = fs.Remove(sourceFilePath)
		if err != nil {
			log.Printf("Failed to remove source file after move: %v", err)
			// Continue execution even if source removal fails
		}

		log.Printf("Moved %s to %s", sourceFilePath, targetFilePath)

	default:
		log.Printf("Unsupported file operation: %v", operation)
	}
}

// performFileOperationWithFS abstracts the logic for copying or moving files using the provided filesystem
func performFileOperationWithFS(sourceFilePath, targetFilePath string, operation FileOperationType, fs FileSystem) error {
	switch operation {
	case CopyFile:
		return copyFileWithFS(sourceFilePath, targetFilePath, fs)
	case MoveFile:
		return moveFileWithFS(sourceFilePath, targetFilePath, fs)
	default:
		return fmt.Errorf("unsupported file operation")
	}
}

// copyFileWithFS handles the copying of a file using the provided filesystem
func copyFileWithFS(src, dst string, fs FileSystem) error {
	sourceFile, err := fs.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := fs.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Perform the actual file copy operation.
	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

// moveFileWithFS handles moving a file using the provided filesystem
func moveFileWithFS(src, dst string, fs FileSystem) error {
	// First copy the file
	if err := copyFileWithFS(src, dst, fs); err != nil {
		return err
	}
	// Then remove the source
	return fs.Remove(src)
}

// GenerateClipName generates a standardized filename for audio clips.
func GenerateClipName(detection *Detection) string {
	// Custom layout to parse the detection date and time.
	const customLayout = "2006-01-02T15:04:05"

	dateTime := detection.Date + "T" + detection.Time

	parsedDate, err := time.Parse(customLayout, dateTime)
	if err != nil {
		log.Printf("Error parsing combined date and time: %v", err)
		return ""
	}

	// Format the date and time for the filename in the format YYYYMMDDTHHMMSSZ.
	formattedDateTime := parsedDate.Format("20060102T150405Z")

	// Format the scientific name for the filename: lowercase, spaces to underscores, remove hyphens and colons.
	sciNameFormatted := strings.ToLower(detection.SciName)
	sciNameFormatted = strings.ReplaceAll(sciNameFormatted, " ", "_")
	sciNameFormatted = strings.ReplaceAll(sciNameFormatted, "-", "")
	sciNameFormatted = strings.ReplaceAll(sciNameFormatted, ":", "")

	// Generate the new filename with the formatted date, time, and confidence level.
	confidencePercentage := fmt.Sprintf("%dp", int(detection.Confidence*100))
	newFileName := fmt.Sprintf("%s_%s_%s%s", sciNameFormatted, confidencePercentage, formattedDateTime, filepath.Ext(detection.FileName))

	return newFileName
}

// For backward compatibility, keep these functions that use the OS filesystem directly
func performFileOperation(sourceFilePath, targetFilePath string, operation FileOperationType) error {
	return performFileOperationWithFS(sourceFilePath, targetFilePath, operation, DefaultFS)
}

func copyFile(src, dst string) error {
	return copyFileWithFS(src, dst, DefaultFS)
}

func moveFile(src, dst string) error {
	return moveFileWithFS(src, dst, DefaultFS)
}
