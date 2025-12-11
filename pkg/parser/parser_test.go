package parser

import (
	"path/filepath"
	"runtime"
	"testing"
)

func getTestFilePath(filename string) string {
	// Get the directory of this test file
	_, file, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(file)
	// Go up 3 levels to project root, then into testfiles
	projectRoot := filepath.Join(testDir, "..", "..")
	return filepath.Join(projectRoot, "testfiles", filename)
}

func TestParseWSDOTPassStatus_Closed_Rain(t *testing.T) {
	p := New()
	status, err := p.ParseWSDOTPassStatus(getTestFilePath("closed_wsdot_stevens_pass_2025_12_10_rain.html"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.East != "Pass Closed" {
		t.Errorf("Expected East='Pass Closed', got '%s'", status.East)
	}

	if status.West != "Pass Closed" {
		t.Errorf("Expected West='Pass Closed', got '%s'", status.West)
	}

	if !status.IsClosed {
		t.Error("Expected IsClosed=true for closed pass")
	}

	if status.Conditions == "" {
		t.Error("Expected conditions text but got empty string")
	}

	t.Logf("✓ Closed pass detected correctly")
	t.Logf("  East: %s", status.East)
	t.Logf("  West: %s", status.West)
	t.Logf("  IsClosed: %v", status.IsClosed)
	t.Logf("  Conditions: %s", status.Conditions)
}

func TestParseWSDOTPassStatus_Closed_Snow(t *testing.T) {
	p := New()
	status, err := p.ParseWSDOTPassStatus(getTestFilePath("closed_wsdot_stevens_pass.html"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.East != "Pass Closed" {
		t.Errorf("Expected East='Pass Closed', got '%s'", status.East)
	}

	if status.West != "Pass Closed" {
		t.Errorf("Expected West='Pass Closed', got '%s'", status.West)
	}

	if !status.IsClosed {
		t.Error("Expected IsClosed=true for closed pass")
	}

	t.Logf("✓ Closed pass (snow) detected correctly")
	t.Logf("  East: %s", status.East)
	t.Logf("  West: %s", status.West)
}

func TestParseWSDOTPassStatus_Open(t *testing.T) {
	p := New()
	status, err := p.ParseWSDOTPassStatus(getTestFilePath("open_wsdot_stevens_pass_2024_01_10.html"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.East == "Pass Closed" {
		t.Errorf("Expected East != 'Pass Closed', got '%s'", status.East)
	}

	if status.West == "Pass Closed" {
		t.Errorf("Expected West != 'Pass Closed', got '%s'", status.West)
	}

	if status.IsClosed {
		t.Error("Expected IsClosed=false for open pass")
	}

	t.Logf("✓ Open pass detected correctly")
	t.Logf("  East: %s", status.East)
	t.Logf("  West: %s", status.West)
	t.Logf("  IsClosed: %v", status.IsClosed)
}

func TestParseWSDOTPassStatus_ClosedEast(t *testing.T) {
	p := New()
	status, err := p.ParseWSDOTPassStatus(getTestFilePath("closed_east_wsdot_stevens_pass.html"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.East != "Pass Closed" {
		t.Errorf("Expected East='Pass Closed', got '%s'", status.East)
	}

	if status.West == "Pass Closed" {
		t.Errorf("Expected West != 'Pass Closed', got '%s'", status.West)
	}

	if !status.IsClosed {
		t.Error("Expected IsClosed=true when east is closed")
	}

	t.Logf("✓ East closed detected correctly")
	t.Logf("  East: %s", status.East)
	t.Logf("  West: %s", status.West)
	t.Logf("  IsClosed: %v", status.IsClosed)
}

func TestParseWSDOTPassStatus_ClosedWest(t *testing.T) {
	p := New()
	status, err := p.ParseWSDOTPassStatus(getTestFilePath("closed_west_wsdot_stevens_pass.html"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.East == "Pass Closed" {
		t.Errorf("Expected East != 'Pass Closed', got '%s'", status.East)
	}

	if status.West != "Pass Closed" {
		t.Errorf("Expected West='Pass Closed', got '%s'", status.West)
	}

	if !status.IsClosed {
		t.Error("Expected IsClosed=true when west is closed")
	}

	t.Logf("✓ West closed detected correctly")
	t.Logf("  East: %s", status.East)
	t.Logf("  West: %s", status.West)
	t.Logf("  IsClosed: %v", status.IsClosed)
}

