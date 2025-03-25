package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// Define a mock file system to test file operations without actual files
type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) FileExists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	args := m.Called(path, data, perm)
	return args.Error(0)
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	args := m.Called(path, perm)
	return args.Error(0)
}

func (m *MockFileSystem) Remove(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystem) Rename(oldpath, newpath string) error {
	args := m.Called(oldpath, newpath)
	return args.Error(0)
}

// TestHandleFileTransferWithMocks tests the handleFileTransfer function with mocked filesystem
func TestHandleFileTransferWithMocks(t *testing.T) {
	t.Parallel()

	// Setup
	mockFS := new(MockFileSystem)

	// Test case 1: Successful copy operation
	t.Run("Successful copy", func(t *testing.T) {
		t.Parallel()

		// Test data
		detection := Detection{
			Date:       "2023-01-15",
			Time:       "13:45:30",
			SciName:    "Testus birdus",
			ComName:    "Test Bird",
			Confidence: 0.85,
			FileName:   "test_audio.wav",
		}

		sourceDir := "/source"
		targetDir := "/target"

		// Source file path
		sourceDirPath := filepath.Join(sourceDir, "Extracted", "By_Date", detection.Date, detection.ComName)
		sourceFilePath := filepath.Join(sourceDirPath, detection.FileName)

		// Expected target paths
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", detection.Date+"T"+detection.Time)
		expectedYear, expectedMonth := parsedDate.Format("2006"), parsedDate.Format("01")
		subDirPath := filepath.Join(targetDir, expectedYear, expectedMonth)
		expectedFileName := "testus_birdus_85p_20230115T134530Z.wav"
		targetFilePath := filepath.Join(subDirPath, expectedFileName)

		// Setup mock expectations
		mockFS.On("FileExists", sourceFilePath).Return(true)
		mockFS.On("MkdirAll", subDirPath, os.ModePerm).Return(nil)
		mockFS.On("ReadFile", sourceFilePath).Return([]byte("test content"), nil)
		mockFS.On("WriteFile", targetFilePath, []byte("test content"), mock.Anything).Return(nil)

		// Capture original functions and restore after test
		originalStat := osStat
		originalMkdirAll := osMkdirAll
		originalCopyFile := fileCopyFunc

		// Override with mocks
		osStat = func(path string) (os.FileInfo, error) {
			if mockFS.FileExists(path) {
				return nil, nil // Return non-nil FileInfo for existing files
			}
			return nil, os.ErrNotExist
		}

		osMkdirAll = mockFS.MkdirAll

		fileCopyFunc = func(src, dst string) error {
			if !mockFS.FileExists(src) {
				return errors.New("source file does not exist")
			}

			// Simulate reading from source and writing to destination
			data, err := mockFS.ReadFile(src)
			if err != nil {
				return err
			}

			return mockFS.WriteFile(dst, data, 0o644)
		}

		// Restore original functions after test
		defer func() {
			osStat = originalStat
			osMkdirAll = originalMkdirAll
			fileCopyFunc = originalCopyFile
		}()

		// Execute
		handleFileTransfer(&detection, sourceDir, targetDir, CopyFile)

		// Verify
		mockFS.AssertExpectations(t)
	})

	// Test case 2: Source file doesn't exist
	t.Run("Source file doesn't exist", func(t *testing.T) {
		t.Parallel()

		// Test data
		detection := Detection{
			Date:       "2023-01-15",
			Time:       "13:45:30",
			SciName:    "Testus birdus",
			ComName:    "Test Bird",
			Confidence: 0.85,
			FileName:   "missing_file.wav",
		}

		sourceDir := "/source"
		targetDir := "/target"

		// Source file path
		sourceDirPath := filepath.Join(sourceDir, "Extracted", "By_Date", detection.Date, detection.ComName)
		sourceFilePath := filepath.Join(sourceDirPath, detection.FileName)

		// Setup mock expectations
		mockFS.On("FileExists", sourceFilePath).Return(false)

		// Capture original functions and restore after test
		originalStat := osStat

		// Override with mocks
		osStat = func(path string) (os.FileInfo, error) {
			if mockFS.FileExists(path) {
				return nil, nil
			}
			return nil, os.ErrNotExist
		}

		// Restore original function after test
		defer func() {
			osStat = originalStat
		}()

		// Execute - this should not panic and simply return without performing any action
		handleFileTransfer(&detection, sourceDir, targetDir, CopyFile)

		// Verify
		mockFS.AssertExpectations(t)
	})

	// Test case 3: Error creating target directories
	t.Run("Error creating target directories", func(t *testing.T) {
		t.Parallel()

		// Test data
		detection := Detection{
			Date:       "2023-01-15",
			Time:       "13:45:30",
			SciName:    "Testus birdus",
			ComName:    "Test Bird",
			Confidence: 0.85,
			FileName:   "test_audio.wav",
		}

		sourceDir := "/source"
		targetDir := "/target"

		// Source file path
		sourceDirPath := filepath.Join(sourceDir, "Extracted", "By_Date", detection.Date, detection.ComName)
		sourceFilePath := filepath.Join(sourceDirPath, detection.FileName)

		// Expected target paths
		parsedDate, _ := time.Parse("2006-01-02T15:04:05", detection.Date+"T"+detection.Time)
		expectedYear, expectedMonth := parsedDate.Format("2006"), parsedDate.Format("01")
		subDirPath := filepath.Join(targetDir, expectedYear, expectedMonth)

		// Setup mock expectations
		mockFS.On("FileExists", sourceFilePath).Return(true)
		mockFS.On("MkdirAll", subDirPath, os.ModePerm).Return(errors.New("permission denied"))

		// Capture original functions and restore after test
		originalStat := osStat
		originalMkdirAll := osMkdirAll

		// Override with mocks
		osStat = func(path string) (os.FileInfo, error) {
			if mockFS.FileExists(path) {
				return nil, nil
			}
			return nil, os.ErrNotExist
		}

		osMkdirAll = mockFS.MkdirAll

		// Restore original functions after test
		defer func() {
			osStat = originalStat
			osMkdirAll = originalMkdirAll
		}()

		// Execute - this should not panic and handle the error gracefully
		handleFileTransfer(&detection, sourceDir, targetDir, CopyFile)

		// Verify
		mockFS.AssertExpectations(t)
	})
}

// Shadow the OS and file operation functions to allow testing with mocks
var (
	osStat       = os.Stat
	osMkdirAll   = os.MkdirAll
	fileCopyFunc = copyFile
	fileMoveFunc = moveFile
)
