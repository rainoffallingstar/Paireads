package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrintUsage_NoError(t *testing.T) {
	printUsage()
}

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestFindCommonNames(t *testing.T) {
	r1 := map[string]bool{"read1": true, "read2": true, "read3": true}
	r2 := map[string]bool{"read2": true, "read3": true, "read4": true}

	common, unpaired := findCommonNames(r1, r2)

	if !common["read2"] || !common["read3"] {
		t.Error("read2 and read3 should be common")
	}
	if common["read1"] {
		t.Error("read1 should not be common")
	}
	if common["read4"] {
		t.Error("read4 should not be common")
	}
	if !unpaired["read1"] || !unpaired["read4"] {
		t.Error("read1 and read4 should be unpaired")
	}
}

func TestFindCommonNames_AllMatching(t *testing.T) {
	r1 := map[string]bool{"a": true, "b": true}
	r2 := map[string]bool{"a": true, "b": true}

	common, _ := findCommonNames(r1, r2)

	if len(common) != 2 {
		t.Errorf("expected 2 common names, got %d", len(common))
	}
}

func TestFindCommonNames_NoMatch(t *testing.T) {
	r1 := map[string]bool{"a": true}
	r2 := map[string]bool{"b": true}

	common, unpaired := findCommonNames(r1, r2)

	if len(common) != 0 {
		t.Errorf("expected 0 common names, got %d", len(common))
	}
	if len(unpaired) != 2 {
		t.Errorf("expected 2 unpaired names, got %d", len(unpaired))
	}
}

func TestFindCommonNames_EmptyInput(t *testing.T) {
	r1 := map[string]bool{}
	r2 := map[string]bool{}

	common, unpaired := findCommonNames(r1, r2)

	if len(common) != 0 {
		t.Errorf("expected 0 common names from empty input, got %d", len(common))
	}
	if len(unpaired) != 0 {
		t.Errorf("expected 0 unpaired names from empty input, got %d", len(unpaired))
	}
}

func TestFindCommonNames_OneSideEmpty(t *testing.T) {
	r1 := map[string]bool{"a": true, "b": true}
	r2 := map[string]bool{}

	common, unpaired := findCommonNames(r1, r2)

	if len(common) != 0 {
		t.Error("expected no common when one side is empty")
	}
	if len(unpaired) != 2 {
		t.Errorf("expected 2 unpaired, got %d", len(unpaired))
	}
}

func TestSaveReadNamesList(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_readnames.txt")

	names := []string{"read1", "read2", "read3"}
	err := saveReadNamesList(names, tmpFile)
	if err != nil {
		t.Fatalf("saveReadNamesList failed: %v", err)
	}

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("output file does not exist")
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read back: %v", err)
	}
	if len(content) == 0 {
		t.Error("output file is empty")
	}
}

func TestSaveReadNamesList_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty_readnames.txt")

	err := saveReadNamesList([]string{}, tmpFile)
	if err != nil {
		t.Fatalf("saveReadNamesList with empty list failed: %v", err)
	}
}
