// file fileops.go
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// handleFileTransfer manages the process of transferring a file from a source to a target directory
// based on a detection event. It involves renaming the file according to a specified format,
// creating necessary subdirectories in the target location, and performing the file transfer operation.
func handleFileTransfer(detection Detection, sourceFilesDir, targetFilesDir string, operation FileOperationType) {
	// Custom layout to parse the detection date and time.
	const customLayout = "2006-01-02T15:04:05"
	dateTime := detection.Date + "T" + detection.Time

	parsedDate, err := time.Parse(customLayout, dateTime)
	if err != nil {
		log.Printf("Error parsing combined date and time: %v", err)
		return // Ensure further processing is halted upon error.
	}
	/*
		// Format the date and time for the filename in the format YYYYMMDDTHHMMSSZ.
		formattedDateTime := parsedDate.Format("20060102T150405Z")

		// Format the scientific name for the filename: lowercase, spaces to underscores, remove hyphens and colons.
		sciNameFormatted := strings.ToLower(strings.ReplaceAll(detection.SciName, " ", "_"))

		// Generate the new filename with the formatted date, time, and confidence level.
		confidencePercentage := fmt.Sprintf("%dp", int(detection.Confidence*100))
		newFileName := fmt.Sprintf("%s_%s_%s.wav", sciNameFormatted, confidencePercentage, formattedDateTime)
	*/

	// Determine the year and month for subdirectory structuring within the target directory.
	year, month := parsedDate.Format("2006"), parsedDate.Format("01")
	subDirPath := filepath.Join(targetFilesDir, year, month)

	// Ensure the target subdirectories exist or create them.
	if err := os.MkdirAll(subDirPath, os.ModePerm); err != nil {
		log.Printf("Failed to create subdirectories: %v", err)
		return
	}

	// Construct the full target file path with the new filename.
	newFileName := GenerateClipName(detection)
	targetFilePath := filepath.Join(subDirPath, newFileName)
	//fmt.Println(targetFilePath)

	// Construct the source directory and file paths.
	sourceDirPath := filepath.Join(sourceFilesDir, "Extracted", "By_Date", detection.Date, detection.ComName)
	sourceFilePath := filepath.Join(sourceDirPath, detection.FileName)

	// Check if the source file exists before attempting transfer.
	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		//log.Printf("Source file does not exist, skipping copy: %s", sourceFilePath)
		return
	} else {
		// Perform the file operation (copy or move).
		if err := performFileOperation(sourceFilePath, targetFilePath, operation); err != nil {
			log.Printf("File operation error: %v", err)
		}
	}
}

func GenerateClipName(detection Detection) string {
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
	sciNameFormatted := strings.ToLower(strings.ReplaceAll(detection.SciName, " ", "_"))

	// Generate the new filename with the formatted date, time, and confidence level.
	confidencePercentage := fmt.Sprintf("%dp", int(detection.Confidence*100))
	newFileName := fmt.Sprintf("%s_%s_%s.wav", sciNameFormatted, confidencePercentage, formattedDateTime)

	return newFileName
}

// performFileOperation abstracts the logic for copying or moving files based on the specified operation.
func performFileOperation(sourceFilePath, targetFilePath string, operation FileOperationType) error {
	switch operation {
	case CopyFile:
		return copyFile(sourceFilePath, targetFilePath)
	case MoveFile:
		return moveFile(sourceFilePath, targetFilePath)
	default:
		return fmt.Errorf("unsupported file operation")
	}
}

// copyFile handles the copying of a file from the source path to the destination path.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err // Handle file opening error.
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err // Handle destination file creation error.
	}
	defer destinationFile.Close()

	// Perform the actual file copy operation.
	_, err = io.Copy(destinationFile, sourceFile)
	return err // Return the result of the copy operation.
}

// moveFile handles moving a file from the source path to the destination path.
func moveFile(src, dst string) error {
	// Attempt to rename (move) the file directly.
	return os.Rename(src, dst)
}

// generateTargetFileNameAndPath generates a new file name based on the naming convention
// and constructs a relative path for use as ClipName and for file operations.
func generateTargetFileNameAndPath(date, commonName string, confidence float64, originalFileName, targetBaseDir string) (string, string) {
	// Generate the new file name based on the birdnet-go naming convention.
	confidencePercentage := fmt.Sprintf("%dp", int(confidence*100))
	newFileName := fmt.Sprintf("%s_%s_%s%s", commonName, confidencePercentage, date, filepath.Ext(originalFileName))

	// Construct the target file path including the 'clips/' directory.
	targetFilePath := filepath.Join(targetBaseDir, newFileName)

	// Assuming the targetBaseDir includes the 'clips/' part, otherwise adjust as needed.
	// Extract the part of the targetFilePath that should be stored in the database.
	clipName := strings.TrimPrefix(targetFilePath, targetBaseDir)
	clipName = strings.TrimPrefix(clipName, "/") // Ensure no leading slash.

	return newFileName, clipName
}
