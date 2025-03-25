package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// FuzzGenerateClipName tests the GenerateClipName function with fuzzed inputs
func FuzzGenerateClipName(f *testing.F) {
	// Add seed corpus
	f.Add("2023-01-15", "13:45:30", "Corvus corax", "Common Raven", 0.85)
	f.Add("2023-02-28", "23:59:59", "Parus major", "Great Tit", 0.95)
	f.Add("2022-12-31", "00:00:00", "Sitta europaea", "Eurasian Nuthatch", 0.75)

	// Fuzz test
	f.Fuzz(func(t *testing.T, date, timeStr, sciName, comName string, confidence float64) {
		// Skip invalid inputs that would cause parse errors
		if !isValidDate(date) || !isValidTime(timeStr) {
			t.Skip("Invalid date or time format")
		}

		// Constrain confidence to reasonable range
		confidence = constrainFloat(confidence, 0.0, 1.0)

		// Create a detection with the fuzzed values
		detection := Detection{
			Date:       date,
			Time:       timeStr,
			SciName:    sciName,
			ComName:    comName,
			Confidence: confidence,
		}

		// Call the function
		clipName := GenerateClipName(&detection)

		// Verify the result
		if clipName == "" {
			// Function should never return empty string unless parsing fails
			if isValidDate(date) && isValidTime(timeStr) {
				t.Errorf("GenerateClipName returned empty string for valid inputs")
			}
			return
		}

		// Basic validation of the clip name format
		if !strings.HasSuffix(clipName, ".wav") {
			t.Errorf("Generated clip name doesn't end with .wav: %s", clipName)
		}

		// Validate format: lowercase_scientific_name_confidenceP_YYYYMMDDTHHMMSSZ.wav
		parts := strings.Split(clipName, "_")
		if len(parts) < 3 {
			t.Errorf("Generated clip name has incorrect format: %s", clipName)
			return
		}

		// Check filename extension
		fileExt := filepath.Ext(clipName)
		if fileExt != ".wav" {
			t.Errorf("Expected .wav file extension, got %s", fileExt)
		}

		// Check if the scientific name was correctly formatted (lowercase, spaces to underscores)
		formattedSciName := strings.ToLower(strings.ReplaceAll(sciName, " ", "_"))
		formattedSciName = stripNonAlphanumeric(formattedSciName)

		// Allow for the case where the scientific name might be empty or invalid
		if formattedSciName != "" && !strings.Contains(clipName, formattedSciName) {
			t.Errorf("Formatted scientific name not found in clip name. Expected: %s, got: %s",
				formattedSciName, clipName)
		}

		// Check if confidence percentage is included
		confStr := fmt.Sprintf("%dp", int(confidence*100))
		if !strings.Contains(clipName, confStr) {
			t.Errorf("Confidence percentage not found in clip name. Expected: %s, got: %s",
				confStr, clipName)
		}

		// Check if the clip name contains a timestamp in the correct format
		// Extract the timestamp part (should be the last part before .wav)
		lastPart := parts[len(parts)-1]
		timeStampPart := strings.TrimSuffix(lastPart, ".wav")

		// Timestamp should be in format YYYYMMDDTHHMMSSZ
		timeRegex := regexp.MustCompile(`^\d{8}T\d{6}Z$`)
		if !timeRegex.MatchString(timeStampPart) {
			t.Errorf("Invalid timestamp format in clip name: %s", timeStampPart)
		}
	})
}

// FuzzConvertDetectionToNote tests the convertDetectionToNote function with fuzzed inputs
func FuzzConvertDetectionToNote(f *testing.F) {
	// Add seed corpus
	f.Add("2023-01-15", "13:45:30", "Corvus corax", "Common Raven", 0.85, 42.123, -71.456, 0.5, 1.0)
	f.Add("2023-02-28", "23:59:59", "Parus major", "Great Tit", 0.95, 51.507, -0.128, 0.6, 1.2)
	f.Add("2022-12-31", "00:00:00", "Sitta europaea", "Eurasian Nuthatch", 0.75, 48.856, 2.352, 0.4, 0.8)

	// Fuzz test
	f.Fuzz(func(t *testing.T, date, timeStr, sciName, comName string, confidence, lat, lon, cutoff, sens float64) {
		// Skip invalid inputs that would cause parse errors
		if !isValidDate(date) || !isValidTime(timeStr) {
			t.Skip("Invalid date or time format")
		}

		// Constrain float values to reasonable ranges
		confidence = constrainFloat(confidence, 0.0, 1.0)
		lat = constrainFloat(lat, -90.0, 90.0)
		lon = constrainFloat(lon, -180.0, 180.0)
		cutoff = constrainFloat(cutoff, 0.0, 1.0)
		sens = constrainFloat(sens, 0.0, 10.0)

		// Create a detection with the fuzzed values
		detection := &Detection{
			Date:       date,
			Time:       timeStr,
			SciName:    sciName,
			ComName:    comName,
			Confidence: confidence,
			Lat:        lat,
			Lon:        lon,
			Cutoff:     cutoff,
			Sens:       sens,
		}

		// Call the function
		note := convertDetectionToNote(detection)

		// Verify the result follows invariants
		if isValidDate(date) {
			// The date should be in YYYY-MM-DD format
			parsedDate, err := time.Parse("2006-01-02", note.Date)
			if err != nil {
				t.Errorf("Converted note has invalid date format: %s", note.Date)
			} else {
				// Date should be within reasonable range
				now := time.Now()
				if parsedDate.Year() < 1900 || parsedDate.After(now.AddDate(1, 0, 0)) {
					t.Errorf("Converted note has unreasonable date: %s", note.Date)
				}
			}
		}

		// Time should be preserved
		if note.Time != timeStr {
			t.Errorf("Time was changed: expected %s, got %s", timeStr, note.Time)
		}

		// Scientific and common names should be preserved
		if note.ScientificName != sciName {
			t.Errorf("Scientific name was changed: expected %s, got %s", sciName, note.ScientificName)
		}
		if note.CommonName != comName {
			t.Errorf("Common name was changed: expected %s, got %s", comName, note.CommonName)
		}

		// Numeric values should be preserved
		if note.Confidence != confidence {
			t.Errorf("Confidence was changed: expected %f, got %f", confidence, note.Confidence)
		}
		if note.Latitude != lat {
			t.Errorf("Latitude was changed: expected %f, got %f", lat, note.Latitude)
		}
		if note.Longitude != lon {
			t.Errorf("Longitude was changed: expected %f, got %f", lon, note.Longitude)
		}
		if note.Threshold != cutoff {
			t.Errorf("Threshold was changed: expected %f, got %f", cutoff, note.Threshold)
		}
		if note.Sensitivity != sens {
			t.Errorf("Sensitivity was changed: expected %f, got %f", sens, note.Sensitivity)
		}

		// Clip name should be generated with expected format
		if note.ClipName == "" {
			t.Errorf("ClipName is empty")
		} else if !strings.HasSuffix(note.ClipName, ".wav") {
			t.Errorf("ClipName does not end with .wav: %s", note.ClipName)
		}

		// Verified status should be "unverified"
		if note.Verified != "unverified" {
			t.Errorf("Verified status should be 'unverified', got: %s", note.Verified)
		}
	})
}

// FuzzFormulateQuery tests the formulateQuery function with fuzzed inputs
func FuzzFormulateQuery(f *testing.F) {
	// Add seed corpus
	f.Add("2023-01-15", "13:45:30")
	f.Add("2023-02-28", "23:59:59")
	f.Add("", "")

	// Fuzz test
	f.Fuzz(func(t *testing.T, date, timeStr string) {
		var lastNote *Note

		// Create a test note if we have valid data
		if date != "" || timeStr != "" {
			lastNote = &Note{
				Date: date,
				Time: timeStr,
			}
		}

		// Call the function
		whereClause, params := formulateQuery(lastNote)

		// Verify the results
		if lastNote == nil {
			// For nil note, should return empty clause and nil params
			if whereClause != "" {
				t.Errorf("Expected empty where clause for nil note, got: %s", whereClause)
			}
			if len(params) != 0 {
				t.Errorf("Expected empty params for nil note, got: %v", params)
			}
		} else {
			// For valid note, should return WHERE clause and params
			expectedClause := "date > ? OR (date = ? AND time > ?)"
			if whereClause != expectedClause {
				t.Errorf("Expected where clause %q, got: %q", expectedClause, whereClause)
			}

			if len(params) != 3 {
				t.Errorf("Expected 3 params, got: %d", len(params))
			} else {
				// Params should be [date, date, time]
				if params[0] != date {
					t.Errorf("Expected first param to be %q, got: %q", date, params[0])
				}
				if params[1] != date {
					t.Errorf("Expected second param to be %q, got: %q", date, params[1])
				}
				if params[2] != timeStr {
					t.Errorf("Expected third param to be %q, got: %q", timeStr, params[2])
				}
			}
		}
	})
}

// Helper functions for fuzzing tests

// isValidDate checks if a string can be parsed as a valid date
func isValidDate(date string) bool {
	// Check common date formats
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	for _, layout := range layouts {
		_, err := time.Parse(layout, date)
		if err == nil {
			return true
		}
	}

	return false
}

// isValidTime checks if a string can be parsed as a valid time
func isValidTime(timeStr string) bool {
	_, err := time.Parse("15:04:05", timeStr)
	return err == nil
}

// constrainFloat limits a float value to the specified range
func constrainFloat(value, minVal, maxVal float64) float64 {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

// stripNonAlphanumeric removes all non-alphanumeric characters except underscore
func stripNonAlphanumeric(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return reg.ReplaceAllString(s, "")
}
