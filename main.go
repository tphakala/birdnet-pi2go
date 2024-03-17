package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// FileOperationType defines the type of operation to perform on the audio files.
type FileOperationType int

const (
	CopyFile FileOperationType = iota
	MoveFile
)

func main() {
	// Define command-line flags.
	var (
		sourceDBPath      string = "birds.db"   // BirdNET-Pi database.
		targetDBPath      string = "birdnet.db" // BirdNET-Go database.
		sourceFilesDir    string                // BirdNET-Pi audio files directory.
		targetFilesDir    string = "clips"      // BirdNET-Go audio files directory.
		operationFlag     string = "copy"       // copy or move audio clips
		skipAudioTransfer bool   = false        // skip copying audio files
	)

	// Register flags.
	flag.StringVar(&sourceDBPath, "source-db", sourceDBPath, "Path to the BirdNET-Pi SQLite database.")
	flag.StringVar(&targetDBPath, "target-db", targetDBPath, "Path to the BirdNET-Go SQLite database.")
	flag.StringVar(&sourceFilesDir, "source-dir", "", "Directory path for BirdNET-Pi BirdSongs.")
	flag.StringVar(&targetFilesDir, "target-dir", targetFilesDir, "Directory path for BirdNET-Go clips.")
	flag.StringVar(&operationFlag, "operation", "", "Operation to perform on audio files: 'copy' or 'move'.")
	flag.BoolVar(&skipAudioTransfer, "skip-audio-transfer", skipAudioTransfer, "Skip transferring audio files and only perform database migration. true/false.")

	// Parse the provided flags.
	flag.Parse()

	// Ensure database paths are provided; other parameters are optional.
	if operationFlag == "" {
		fmt.Println("birdnet-pi2go: Convert birdnet-pi data to birdnet-go.")
		fmt.Println("This tool is provided 'AS IS', without warranty of any kind. Please ensure you have backed up your data before using this tool.")
		fmt.Println("Usage:")
		flag.PrintDefaults() // Print default help messages.
		os.Exit(1)           // Exit after displaying help message.
	}

	// Initialize file operation type.
	var operation FileOperationType

	// Determine the file operation based on the operation flag, if directories are provided.

	switch operationFlag {
	case "move":
		if !skipAudioTransfer {
			if sourceFilesDir == "" {
				log.Fatal("Source directory is required for move operation.")
			}
			// Confirm that the user has backed up their data before proceeding with the move operation.
			fmt.Print("Have you backed up your data and wish to proceed with the move operation? (yes/no): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal("Failed to read response:", err)
			}
			if strings.TrimSpace(strings.ToLower(response)) != "yes" {
				fmt.Println("Operation aborted by the user. Ensure data is backed up before attempting to move files.")
				os.Exit(1)
			}
		}
		operation = MoveFile
	case "copy":
		if !skipAudioTransfer {
			if sourceFilesDir == "" {
				log.Fatal("Source directory is required for copy operation.")
			}
			// Check disk space before copying, if required.
			enoughSpace, err := checkDiskSpace(sourceFilesDir, targetFilesDir)
			if err != nil {
				log.Fatalf("Failed to check disk space: %v", err)
			}
			if !enoughSpace {
				log.Fatal("Not enough space on target volume to perform copy operation.")
			}
		}
		operation = CopyFile
	case "merge":
		// Merge existing BirdNET-Go database to migrated data.
		MergeDatabases(sourceDBPath, targetDBPath)
		return
	default:
		log.Fatal("Invalid operation. Use 'copy' or 'move'.") // Handle invalid operation value.
	}

	// Call the conversion and transfer function with the parsed parameters.
	// If sourceFilesDir and targetFilesDir are empty, file operations are skipped.
	convertAndTransferData(sourceDBPath, targetDBPath, sourceFilesDir, targetFilesDir, operation, skipAudioTransfer)
}

// calculateDirSize calculates the total size of all files within a directory.
func calculateDirSize(dirPath string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(dirPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size() // Add file size if it's not a directory.
		}
		return nil
	})
	return totalSize, err // Return the total size and any error encountered.
}

// checkDiskSpace checks if the target directory has enough free space for transferring files from the source directory.
func checkDiskSpace(sourceDir, targetDir string) (bool, error) {
	sourceSize, err := calculateDirSize(sourceDir)
	if err != nil {
		return false, err
	}

	freeSpace, err := getFreeSpace(targetDir)
	if err != nil {
		return false, err
	}

	return uint64(sourceSize) <= freeSpace, nil
}
