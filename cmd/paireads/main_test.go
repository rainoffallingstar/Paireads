package main

import (
	"os"
	"path/filepath"
	"testing"

	bamnative "github.com/PeeperLab/xenofilter/bamnative"
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

func TestClassifyCompletePairsRequiresUniquePrimaryMates(t *testing.T) {
	statuses := make(map[string]readPairStatus)
	addRecord := func(record *bamnative.Record) {
		status := statuses[record.Name]
		updateReadPairStatus(&status, record)
		statuses[record.Name] = status
	}

	addRecord(&bamnative.Record{Name: "complete", RefID: 0, Flags: bamnative.FlagPaired | bamnative.FlagFirstInPair})
	addRecord(&bamnative.Record{Name: "complete", RefID: 0, Flags: bamnative.FlagPaired | bamnative.FlagSecondInPair})
	addRecord(&bamnative.Record{Name: "complete", RefID: 0, Flags: bamnative.FlagPaired | bamnative.FlagFirstInPair | bamnative.FlagSupplementary})

	addRecord(&bamnative.Record{Name: "duplicate_r1", RefID: 0, Flags: bamnative.FlagPaired | bamnative.FlagFirstInPair})
	addRecord(&bamnative.Record{Name: "duplicate_r1", RefID: 0, Flags: bamnative.FlagPaired | bamnative.FlagFirstInPair})
	addRecord(&bamnative.Record{Name: "duplicate_r1", RefID: 0, Flags: bamnative.FlagPaired | bamnative.FlagSecondInPair})

	addRecord(&bamnative.Record{Name: "supplementary_only", RefID: 0, Flags: bamnative.FlagPaired | bamnative.FlagFirstInPair})
	addRecord(&bamnative.Record{Name: "supplementary_only", RefID: 0, Flags: bamnative.FlagPaired | bamnative.FlagSecondInPair | bamnative.FlagSupplementary})

	completePairs, rejectedNames := classifyCompletePairs(statuses)
	if !completePairs["complete"] {
		t.Fatal("expected complete primary R1/R2 pair to be retained")
	}
	if completePairs["duplicate_r1"] || completePairs["supplementary_only"] {
		t.Fatalf("ambiguous or supplementary-only mates must not be retained: %v", completePairs)
	}
	if len(rejectedNames) != 2 {
		t.Fatalf("expected two rejected names, got %v", rejectedNames)
	}
}

func TestUpdateReadPairStatusRejectsInvalidMateFlags(t *testing.T) {
	status := readPairStatus{}
	updateReadPairStatus(&status, &bamnative.Record{
		Name:  "invalid",
		RefID: 0,
		Flags: bamnative.FlagPaired | bamnative.FlagFirstInPair | bamnative.FlagSecondInPair,
	})
	if !status.invalidPrimary {
		t.Fatal("record marked as both R1 and R2 must be invalid")
	}
}

func TestWriteLines(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	err := saveReadNamesList([]string{"hello world"}, tmpFile)
	if err != nil {
		t.Fatalf("writeLines failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read back: %v", err)
	}

	expected := "hello world\n"
	if string(content) != expected {
		t.Errorf("expected %q, got %q", expected, string(content))
	}
}
