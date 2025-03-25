package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkGenerateClipName measures performance of clip name generation
func BenchmarkGenerateClipName(b *testing.B) {
	// Test data
	detection := Detection{
		Date:       "2023-01-15",
		Time:       "13:45:30",
		SciName:    "Corvus corax",
		ComName:    "Common Raven",
		Confidence: 0.95,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		GenerateClipName(&detection)
	}
}

// BenchmarkCopyFile measures performance of the file copy operation
func BenchmarkCopyFile(b *testing.B) {
	// Create temporary directory for benchmark
	tempDir, err := os.MkdirTemp("", "benchmark-copy-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create sample files of different sizes
	sizes := []int{
		1024,     // 1 KB
		10240,    // 10 KB
		102400,   // 100 KB
		1048576,  // 1 MB
		10485760, // 10 MB
	}

	for _, size := range sizes {
		b.Run(filepath.Base(tempDir)+"-"+byteCountIEC(int64(size)), func(b *testing.B) {
			// Create source file with specified size
			sourceFile := filepath.Join(tempDir, "source-"+byteCountIEC(int64(size)))
			data := make([]byte, size)
			for i := 0; i < size; i++ {
				data[i] = byte(i % 256)
			}
			if err := os.WriteFile(sourceFile, data, 0o644); err != nil {
				b.Fatalf("Failed to create source file: %v", err)
			}

			// Destination file path
			destFile := filepath.Join(tempDir, "dest-"+byteCountIEC(int64(size)))

			// Measure file copy performance
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Reset destination file for each iteration
				if i > 0 {
					os.Remove(destFile)
				}

				if err := copyFile(sourceFile, destFile); err != nil {
					b.Fatalf("Copy failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConvertDetectionToNote measures performance of detection to note conversion
func BenchmarkConvertDetectionToNote(b *testing.B) {
	// Test data
	detection := &Detection{
		Date:       "2023-01-15",
		Time:       "13:45:30",
		SciName:    "Corvus corax",
		ComName:    "Common Raven",
		Confidence: 0.95,
		Lat:        42.123,
		Lon:        -71.456,
		Cutoff:     0.5,
		Week:       3,
		Sens:       1.0,
		Overlap:    0.0,
		FileName:   "test.wav",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		convertDetectionToNote(detection)
	}
}

// BenchmarkHandleFileTransfer measures performance of the entire file transfer process
func BenchmarkHandleFileTransfer(b *testing.B) {
	// Skip in short mode as this can be time-consuming
	if testing.Short() {
		b.Skip("Skipping in short mode")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "benchmark-transfer-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceRoot := filepath.Join(tempDir, "source")
	targetRoot := filepath.Join(tempDir, "target")

	// Create directories
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		b.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		b.Fatalf("Failed to create target dir: %v", err)
	}

	// Test with different file sizes
	sizes := []int{
		1024,    // 1 KB
		102400,  // 100 KB
		1048576, // 1 MB
	}

	for _, size := range sizes {
		b.Run("Transfer-"+byteCountIEC(int64(size)), func(b *testing.B) {
			// Create test detection
			detection := Detection{
				Date:       "2023-01-15",
				Time:       "13:45:30",
				SciName:    "Testus birdus",
				ComName:    "Test Bird",
				Confidence: 0.85,
				FileName:   "test-" + byteCountIEC(int64(size)) + ".wav",
			}

			// Create source file structure
			extractedDir := filepath.Join(sourceRoot, "Extracted", "By_Date", detection.Date, detection.ComName)
			if err := os.MkdirAll(extractedDir, 0o755); err != nil {
				b.Fatalf("Failed to create extracted dir: %v", err)
			}

			// Create source file with test content
			sourceFilePath := filepath.Join(extractedDir, detection.FileName)
			data := make([]byte, size)
			for i := 0; i < size; i++ {
				data[i] = byte(i % 256)
			}
			if err := os.WriteFile(sourceFilePath, data, 0o644); err != nil {
				b.Fatalf("Failed to create source file: %v", err)
			}

			// Expected target paths for cleanup
			parsedDate, _ := time.Parse("2006-01-02T15:04:05", detection.Date+"T"+detection.Time)
			expectedYear, expectedMonth := parsedDate.Format("2006"), parsedDate.Format("01")
			expectedFileName := "testus_birdus_85p_20230115T134530Z.wav"

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Clean target before each iteration
				if i > 0 {
					// Clean up the specific file instead of the entire year directory
					targetPath := filepath.Join(targetRoot, expectedYear, expectedMonth, expectedFileName)
					os.Remove(targetPath)
				}

				// Run the file transfer
				handleFileTransfer(&detection, sourceRoot, targetRoot, CopyFile)
			}
		})
	}
}

// BenchmarkHandleFileTransferParallel measures performance with parallel transfers
func BenchmarkHandleFileTransferParallel(b *testing.B) {
	// Skip in short mode
	if testing.Short() {
		b.Skip("Skipping in short mode")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "benchmark-parallel-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceRoot := filepath.Join(tempDir, "source")
	targetRoot := filepath.Join(tempDir, "target")

	// Create directories
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		b.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		b.Fatalf("Failed to create target dir: %v", err)
	}

	// Create test detections (multiple birds on same day)
	detections := []Detection{
		{
			Date:       "2023-01-15",
			Time:       "13:45:30",
			SciName:    "Testus birdus",
			ComName:    "Test Bird",
			Confidence: 0.85,
			FileName:   "test1.wav",
		},
		{
			Date:       "2023-01-15",
			Time:       "14:15:00",
			SciName:    "Avius testus",
			ComName:    "Another Bird",
			Confidence: 0.90,
			FileName:   "test2.wav",
		},
		{
			Date:       "2023-01-15",
			Time:       "15:30:45",
			SciName:    "Parallelus benchmarkus",
			ComName:    "Benchmark Bird",
			Confidence: 0.95,
			FileName:   "test3.wav",
		},
	}

	// Create source files
	for i := range detections {
		extractedDir := filepath.Join(sourceRoot, "Extracted", "By_Date", detections[i].Date, detections[i].ComName)
		if err := os.MkdirAll(extractedDir, 0o755); err != nil {
			b.Fatalf("Failed to create extracted dir: %v", err)
		}

		sourceFilePath := filepath.Join(extractedDir, detections[i].FileName)
		data := make([]byte, 10240) // 10KB files
		if err := os.WriteFile(sourceFilePath, data, 0o644); err != nil {
			b.Fatalf("Failed to create source file: %v", err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine selects a different detection in round-robin fashion
		i := 0
		for pb.Next() {
			idx := i % len(detections)
			detection := detections[idx]
			handleFileTransfer(&detection, sourceRoot, targetRoot, CopyFile)
			i++
		}
	})
}

// Helper function to format byte sizes
func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%d %cB", b/div, "KMGTPE"[exp])
}
