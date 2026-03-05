// Command to filter BAM files keeping only properly paired reads
// This replaces the samtools/picard workflow for filtering unpaired reads
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	bamnative "github.com/PeeperLab/xenofilter/bamnative"
)

var Version = "0.1.0"

func main() {
	for _, a := range os.Args[1:] {
		if a == "--version" || a == "-V" {
			fmt.Printf("paireads %s\n", Version)
			return
		}
	}

	// Pre-scan for --coord-sort flag before positional argument parsing.
	coordSort := false
	cleanArgs := make([]string, 0, len(os.Args)-1)
	for _, a := range os.Args[1:] {
		if a == "--coord-sort" {
			coordSort = true
		} else {
			cleanArgs = append(cleanArgs, a)
		}
	}

	if len(cleanArgs) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Single BAM mode: paireads [--coord-sort] <input.bam> <output.bam>
	if len(cleanArgs) == 2 && strings.HasSuffix(cleanArgs[0], ".bam") && !strings.HasPrefix(cleanArgs[1], "-") {
		runSingleBAMMode(cleanArgs[0], cleanArgs[1], coordSort)
		return
	}

	// Dual BAM mode: paireads [--coord-sort] <R1.bam> <R2.bam> <output_prefix>
	if len(cleanArgs) < 3 {
		printUsage()
		os.Exit(1)
	}
	runDualBAMMode(cleanArgs[0], cleanArgs[1], cleanArgs[2], coordSort)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  Single BAM mode:  paireads [--coord-sort] <input.bam> <output.bam>")
	fmt.Println("  Dual BAM mode:    paireads [--coord-sort] <R1.bam> <R2.bam> <output_prefix>")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --coord-sort    Coordinate-sort the output BAM and create a BAI index.")
	fmt.Println("                  Default (without this flag): name-sort the output, no index.")
	fmt.Println()
	fmt.Println("Single BAM mode:")
	fmt.Println("  Filters a merged paired-end BAM file, keeping only reads with proper pairs.")
	fmt.Println("  Reads that appear only once (unpaired) are filtered out.")
	fmt.Println()
	fmt.Println("Dual BAM mode:")
	fmt.Println("  1. Extracts read names from both R1 and R2 BAM files")
	fmt.Println("  2. Finds reads that exist in BOTH files (properly paired)")
	fmt.Println("  3. Filters both R1 and R2 to keep only paired reads")
	fmt.Println("  4. Sorts (and optionally indexes) the output BAM files")
	fmt.Println()
	fmt.Println("Output files (single mode, without --coord-sort):")
	fmt.Println("  <output.bam>     - Filtered BAM with only paired reads (name-sorted)")
	fmt.Println()
	fmt.Println("Output files (single mode, with --coord-sort):")
	fmt.Println("  <output.bam>     - Filtered BAM with only paired reads (coordinate-sorted)")
	fmt.Println("  <output.bam>.bai - BAI index")
	fmt.Println()
	fmt.Println("Output files (dual mode, without --coord-sort):")
	fmt.Println("  <output_prefix>_R1.bam  - Filtered R1 BAM (name-sorted)")
	fmt.Println("  <output_prefix>_R2.bam  - Filtered R2 BAM (name-sorted)")
	fmt.Println("  <output_prefix>_filtered_readnames.txt")
	fmt.Println()
	fmt.Println("Output files (dual mode, with --coord-sort):")
	fmt.Println("  <output_prefix>_R1.bam     - Filtered R1 BAM (coordinate-sorted)")
	fmt.Println("  <output_prefix>_R1.bam.bai - BAI index for R1")
	fmt.Println("  <output_prefix>_R2.bam     - Filtered R2 BAM (coordinate-sorted)")
	fmt.Println("  <output_prefix>_R2.bam.bai - BAI index for R2")
	fmt.Println("  <output_prefix>_filtered_readnames.txt")
}

// runSingleBAMMode processes a single merged paired-end BAM file
// and filters out unpaired reads (reads that appear only once).
func runSingleBAMMode(bamPath, outputPath string, coordSort bool) {
	fmt.Printf("Processing single merged paired-end BAM file:\n")
	fmt.Printf("  Input: %s\n", bamPath)
	fmt.Printf("  Output: %s\n", outputPath)

	// Step 1: Extract read names and count occurrences
	fmt.Println("\n[1/3] Analyzing read pair status...")
	readNameCounts, totalRecords, err := extractReadNameCounts(bamPath)
	if err != nil {
		fmt.Printf("Error analyzing BAM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Total records: %d\n", totalRecords)
	fmt.Printf("  Unique read names: %d\n", len(readNameCounts))

	// Identify paired and unpaired reads
	pairedNames := make(map[string]bool)
	unpairedNames := make([]string, 0)

	for name, count := range readNameCounts {
		if count >= 2 {
			pairedNames[name] = true
		} else {
			unpairedNames = append(unpairedNames, name)
		}
	}

	fmt.Printf("  Paired reads (count>=2): %d\n", len(pairedNames))
	fmt.Printf("  Unpaired reads (count=1): %d\n", len(unpairedNames))

	// Save filtered (unpaired) read names to file
	readnamesPath := strings.TrimSuffix(outputPath, ".bam") + "_filtered_readnames.txt"
	if len(unpairedNames) > 0 {
		if err := saveReadNamesList(unpairedNames, readnamesPath); err != nil {
			fmt.Printf("Warning: Failed to save filtered read names: %v\n", err)
		} else {
			fmt.Printf("  Saved filtered read names to: %s\n", readnamesPath)
		}
	}

	// Step 2: Filter BAM to keep only paired reads
	fmt.Println("\n[2/3] Filtering BAM to keep only paired reads...")
	kept, err := filterBAMWithCount(bamPath, outputPath, pairedNames)
	if err != nil {
		fmt.Printf("Error filtering BAM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Kept %d out of %d records\n", kept, totalRecords)

	// Step 3: Sort (and optionally index)
	if coordSort {
		fmt.Println("\n[3/3] Coordinate-sorting and indexing...")
		if err := sortAndIndex(outputPath); err != nil {
			fmt.Printf("Warning: coord-sort/index failed: %v\n", err)
		} else {
			fmt.Printf("  Sorted and indexed: %s.bai\n", outputPath)
		}
	} else {
		fmt.Println("\n[3/3] Name-sorting output...")
		if err := nameSort(outputPath); err != nil {
			fmt.Printf("Warning: name-sort failed: %v\n", err)
		}
	}

	fmt.Println("\nDone!")
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total input records: %d\n", totalRecords)
	fmt.Printf("  Unique read names: %d\n", len(readNameCounts))
	fmt.Printf("  Paired reads (kept): %d\n", len(pairedNames))
	fmt.Printf("  Unpaired reads (filtered): %d\n", len(unpairedNames))
	fmt.Printf("  Output records: %d\n", kept)
	fmt.Printf("\nOutput files:\n")
	fmt.Printf("  %s\n", outputPath)
	if coordSort {
		fmt.Printf("  %s.bai\n", outputPath)
	}
	if len(unpairedNames) > 0 {
		fmt.Printf("  %s\n", readnamesPath)
	}
}

// runDualBAMMode processes two separate R1/R2 BAM files, keeping only reads
// present in both, then sorts (and optionally indexes) the outputs.
func runDualBAMMode(r1Path, r2Path, outputPrefix string, coordSort bool) {
	outputR1 := outputPrefix + "_R1.bam"
	outputR2 := outputPrefix + "_R2.bam"

	fmt.Printf("Processing paired-end BAM files:\n")
	fmt.Printf("  R1: %s\n", r1Path)
	fmt.Printf("  R2: %s\n", r2Path)
	fmt.Printf("  Output prefix: %s\n", outputPrefix)

	// Step 1: Extract read names from R1
	fmt.Println("\n[1/6] Extracting read names from R1...")
	r1Names, err := extractReadNames(r1Path)
	if err != nil {
		fmt.Printf("Error extracting R1 names: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Found %d reads in R1\n", len(r1Names))

	// Step 2: Extract read names from R2
	fmt.Println("\n[2/6] Extracting read names from R2...")
	r2Names, err := extractReadNames(r2Path)
	if err != nil {
		fmt.Printf("Error extracting R2 names: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Found %d reads in R2\n", len(r2Names))

	// Step 3: Find common read names (paired reads) and unpaired reads
	fmt.Println("\n[3/6] Finding common read names (properly paired)...")
	commonNames, unpairedNames := findCommonNames(r1Names, r2Names)
	fmt.Printf("  Found %d properly paired reads\n", len(commonNames))
	fmt.Printf("  Found %d unpaired reads (to be filtered out)\n", len(unpairedNames))

	// Save filtered (unpaired) read names to file
	readnamesPath := outputPrefix + "_filtered_readnames.txt"
	if err := saveReadNames(unpairedNames, readnamesPath); err != nil {
		fmt.Printf("Warning: Failed to save filtered read names: %v\n", err)
	} else {
		fmt.Printf("  Saved filtered read names to: %s\n", readnamesPath)
	}

	// Step 4: Filter R1 BAM
	fmt.Println("\n[4/6] Filtering R1 BAM to keep only paired reads...")
	if err := filterBAM(r1Path, outputR1, commonNames); err != nil {
		fmt.Printf("Error filtering R1 BAM: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Filter R2 BAM
	fmt.Println("\n[5/6] Filtering R2 BAM to keep only paired reads...")
	if err := filterBAM(r2Path, outputR2, commonNames); err != nil {
		fmt.Printf("Error filtering R2 BAM: %v\n", err)
		os.Exit(1)
	}

	// Step 6: Sort (and optionally index) both outputs
	if coordSort {
		fmt.Println("\n[6/6] Coordinate-sorting and indexing output BAM files...")
		if err := sortAndIndex(outputR1); err != nil {
			fmt.Printf("Warning: R1 sort/index failed: %v\n", err)
		} else {
			fmt.Printf("  R1 sorted and indexed successfully\n")
		}
		if err := sortAndIndex(outputR2); err != nil {
			fmt.Printf("Warning: R2 sort/index failed: %v\n", err)
		} else {
			fmt.Printf("  R2 sorted and indexed successfully\n")
		}
	} else {
		fmt.Println("\n[6/6] Name-sorting output BAM files...")
		if err := nameSort(outputR1); err != nil {
			fmt.Printf("Warning: R1 name-sort failed: %v\n", err)
		} else {
			fmt.Printf("  R1 name-sorted successfully\n")
		}
		if err := nameSort(outputR2); err != nil {
			fmt.Printf("Warning: R2 name-sort failed: %v\n", err)
		} else {
			fmt.Printf("  R2 name-sorted successfully\n")
		}
	}

	fmt.Println("\nDone!")
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  R1 reads: %d\n", len(r1Names))
	fmt.Printf("  R2 reads: %d\n", len(r2Names))
	fmt.Printf("  Paired reads (kept): %d\n", len(commonNames))
	fmt.Printf("  Unpaired reads (filtered out): %d\n", len(unpairedNames))
	fmt.Printf("\nOutput files:\n")
	fmt.Printf("  %s\n", outputR1)
	if coordSort {
		fmt.Printf("  %s.bai\n", outputR1)
	}
	fmt.Printf("  %s\n", outputR2)
	if coordSort {
		fmt.Printf("  %s.bai\n", outputR2)
	}
	fmt.Printf("  %s\n", readnamesPath)
}

// extractReadNames extracts all read names from a BAM file
func extractReadNames(bamPath string) (map[string]bool, error) {
	f, err := os.Open(bamPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open BAM file: %w", err)
	}
	defer f.Close()

	reader, err := bamnative.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create BAM reader: %w", err)
	}

	names := make(map[string]bool)

	for {
		rec, err := reader.Read()
		if err != nil {
			break
		}
		names[rec.Name] = true
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("no reads found in BAM file")
	}

	return names, nil
}

// extractReadNameCounts extracts read names and counts their occurrences.
func extractReadNameCounts(bamPath string) (map[string]int, int, error) {
	f, err := os.Open(bamPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open BAM file: %w", err)
	}
	defer f.Close()

	reader, err := bamnative.NewReader(f)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create BAM reader: %w", err)
	}

	counts := make(map[string]int)
	totalRecords := 0

	for {
		rec, err := reader.Read()
		if err != nil {
			break
		}
		counts[rec.Name]++
		totalRecords++
	}

	if totalRecords == 0 {
		return nil, 0, fmt.Errorf("no records found in BAM file")
	}

	return counts, totalRecords, nil
}

// findCommonNames finds read names that exist in both sets.
// Also returns the unpaired (filtered) read names.
func findCommonNames(r1, r2 map[string]bool) (common, unpaired map[string]bool) {
	common = make(map[string]bool)
	unpaired = make(map[string]bool)

	if len(r1) < len(r2) {
		for name := range r1 {
			if r2[name] {
				common[name] = true
			} else {
				unpaired[name] = true
			}
		}
		for name := range r2 {
			if !r1[name] {
				unpaired[name] = true
			}
		}
	} else {
		for name := range r2 {
			if r1[name] {
				common[name] = true
			} else {
				unpaired[name] = true
			}
		}
		for name := range r1 {
			if !r2[name] {
				unpaired[name] = true
			}
		}
	}

	return common, unpaired
}

// saveReadNames saves a map of read names to a file (one per line, sorted).
func saveReadNames(names map[string]bool, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sortedNames := make([]string, 0, len(names))
	for name := range names {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	writer := bufio.NewWriter(f)
	for _, name := range sortedNames {
		if _, err := writer.WriteString(name + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// saveReadNamesList saves a slice of read names to a file (one per line, sorted).
func saveReadNamesList(names []string, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sortedNames := make([]string, len(names))
	copy(sortedNames, names)
	sort.Strings(sortedNames)

	writer := bufio.NewWriter(f)
	for _, name := range sortedNames {
		if _, err := writer.WriteString(name + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// filterBAM reads a BAM file and writes only records with matching read names.
func filterBAM(inputPath, outputPath string, keepNames map[string]bool) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input BAM: %w", err)
	}
	defer f.Close()

	reader, err := bamnative.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create BAM reader: %w", err)
	}

	header := reader.Header()

	writer, err := bamnative.NewWriter(outputPath, header)
	if err != nil {
		return fmt.Errorf("failed to create BAM writer: %w", err)
	}
	defer writer.Close()

	kept := 0
	total := 0

	for {
		rec, err := reader.Read()
		if err != nil {
			break
		}
		total++
		if keepNames[rec.Name] {
			if err := writer.Write(rec); err != nil {
				return fmt.Errorf("failed to write record: %w", err)
			}
			kept++
		}
	}

	if kept != total {
		fmt.Printf("  Kept %d out of %d records\n", kept, total)
	} else {
		fmt.Printf("  Kept all %d records\n", kept)
	}

	return nil
}

// filterBAMWithCount reads a BAM file and writes only records with matching read names.
// Returns the number of kept records.
func filterBAMWithCount(inputPath, outputPath string, keepNames map[string]bool) (int, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open input BAM: %w", err)
	}
	defer f.Close()

	reader, err := bamnative.NewReader(f)
	if err != nil {
		return 0, fmt.Errorf("failed to create BAM reader: %w", err)
	}

	header := reader.Header()

	writer, err := bamnative.NewWriter(outputPath, header)
	if err != nil {
		return 0, fmt.Errorf("failed to create BAM writer: %w", err)
	}
	defer writer.Close()

	kept := 0

	for {
		rec, err := reader.Read()
		if err != nil {
			break
		}
		if keepNames[rec.Name] {
			if err := writer.Write(rec); err != nil {
				return kept, fmt.Errorf("failed to write record: %w", err)
			}
			kept++
		}
	}

	return kept, nil
}

// sortAndIndex coordinate-sorts a BAM file in-place and creates a BAI index.
func sortAndIndex(bamPath string) error {
	sortedPath := strings.TrimSuffix(bamPath, ".bam") + "_sorted.bam"

	if err := bamnative.Sort(bamPath, &bamnative.SortOptions{OutputPath: sortedPath}); err != nil {
		return fmt.Errorf("failed to sort BAM: %w", err)
	}

	if err := os.Remove(bamPath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	if err := os.Rename(sortedPath, bamPath); err != nil {
		return fmt.Errorf("failed to rename sorted file: %w", err)
	}

	if err := bamnative.BuildIndex(bamPath); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// nameSort name-sorts a BAM file in-place (no index created).
func nameSort(bamPath string) error {
	sortedPath := strings.TrimSuffix(bamPath, ".bam") + "_nsorted.bam"

	if err := bamnative.Sort(bamPath, &bamnative.SortOptions{
		OutputPath: sortedPath,
		ByName:     true,
	}); err != nil {
		return fmt.Errorf("failed to name-sort BAM: %w", err)
	}

	if err := os.Remove(bamPath); err != nil {
		return err
	}

	return os.Rename(sortedPath, bamPath)
}
