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

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	// Check for single BAM mode (new feature)
	if len(os.Args) == 3 && strings.HasSuffix(os.Args[1], ".bam") && !strings.HasPrefix(os.Args[2], "-") {
		// Single BAM mode: paireads <input.bam> <output.bam>
		bamPath := os.Args[1]
		outputPath := os.Args[2]
		runSingleBAMMode(bamPath, outputPath)
		return
	}

	// Original dual-BAM mode
	if len(os.Args) < 4 {
		printUsage()
		os.Exit(1)
	}

	r1Path := os.Args[1]
	r2Path := os.Args[2]
	outputPrefix := os.Args[3]

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

	// Step 4: Filter R1 BAM to keep only paired reads
	fmt.Println("\n[4/6] Filtering R1 BAM to keep only paired reads...")
	if err := filterBAM(r1Path, outputR1, commonNames); err != nil {
		fmt.Printf("Error filtering R1 BAM: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Filter R2 BAM to keep only paired reads
	fmt.Println("\n[5/6] Filtering R2 BAM to keep only paired reads...")
	if err := filterBAM(r2Path, outputR2, commonNames); err != nil {
		fmt.Printf("Error filtering R2 BAM: %v\n", err)
		os.Exit(1)
	}

	// Step 6: Sort and index both output files
	fmt.Println("\n[6/6] Sorting and indexing output BAM files...")
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

	fmt.Println("\nDone!")
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  R1 reads: %d\n", len(r1Names))
	fmt.Printf("  R2 reads: %d\n", len(r2Names))
	fmt.Printf("  Paired reads (kept): %d\n", len(commonNames))
	fmt.Printf("  Unpaired reads (filtered out): %d\n", len(unpairedNames))
	fmt.Printf("\nOutput files:\n")
	fmt.Printf("  %s\n", outputR1)
	fmt.Printf("  %s.bai\n", outputR1)
	fmt.Printf("  %s\n", outputR2)
	fmt.Printf("  %s.bai\n", outputR2)
	fmt.Printf("  %s\n", readnamesPath)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  Single BAM mode:  paireads <input.bam> <output.bam>")
	fmt.Println("  Dual BAM mode:    paireads <R1.bam> <R2.bam> <output_prefix>")
	fmt.Println()
	fmt.Println("Single BAM mode:")
	fmt.Println("  Filters a merged paired-end BAM file, keeping only reads with proper pairs.")
	fmt.Println("  Reads that appear only once (unpaired) are filtered out.")
	fmt.Println()
	fmt.Println("Dual BAM mode:")
	fmt.Println("  1. Extracts read names from both R1 and R2 BAM files")
	fmt.Println("  2. Finds reads that exist in BOTH files (properly paired)")
	fmt.Println("  3. Filters both R1 and R2 to keep only paired reads")
	fmt.Println("  4. Sorts and indexes the output BAM files")
	fmt.Println()
	fmt.Println("Output files (single mode):")
	fmt.Println("  <output.bam> - Filtered BAM with only paired reads (name-sorted)")
	fmt.Println("  <output.bam.bai> - BAI index")
	fmt.Println()
	fmt.Println("Output files (dual mode):")
	fmt.Println("  <output_prefix>_R1.bam  - Filtered R1 BAM with only paired reads")
	fmt.Println("  <output_prefix>_R1.bam.bai - BAI index for R1")
	fmt.Println("  <output_prefix>_R2.bam  - Filtered R2 BAM with only paired reads")
	fmt.Println("  <output_prefix>_R2.bam.bai - BAI index for R2")
	fmt.Println("  <output_prefix>_filtered_readnames.txt - List of filtered (unpaired) read names")
}

// runSingleBAMMode processes a single merged paired-end BAM file
// and filters out unpaired reads (reads that appear only once)
func runSingleBAMMode(bamPath, outputPath string) {
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

	// Step 2: Identify paired and unpaired reads
	pairedNames := make(map[string]bool)
	unpairedNames := make([]string, 0)

	for name, count := range readNameCounts {
		if count == 2 {
			// Properly paired (appears twice: R1 and R2)
			pairedNames[name] = true
		} else if count == 1 {
			// Unpaired (appears only once)
			unpairedNames = append(unpairedNames, name)
		}
		// count > 2 might indicate multimapping, keep them for now
	}

	fmt.Printf("  Paired reads (count=2): %d\n", len(pairedNames))
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

	// Step 3: Filter BAM to keep only paired reads
	fmt.Println("\n[2/3] Filtering BAM to keep only paired reads...")
	kept, err := filterBAMWithCount(bamPath, outputPath, pairedNames)
	if err != nil {
		fmt.Printf("Error filtering BAM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Kept %d out of %d records\n", kept, totalRecords)

	// Step 4: Create BAI index (file is already name-sorted)
	fmt.Println("\n[3/3] Creating BAM index...")
	if err := bamnative.BuildIndex(outputPath); err != nil {
		fmt.Printf("Warning: Failed to create index: %v\n", err)
	} else {
		fmt.Printf("  BAM index created: %s.bai\n", outputPath)
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
	fmt.Printf("  %s.bai\n", outputPath)
	if len(unpairedNames) > 0 {
		fmt.Printf("  %s\n", readnamesPath)
	}
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

		// Store the read name
		names[rec.Name] = true
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("no reads found in BAM file")
	}

	return names, nil
}

// extractReadNameCounts extracts read names and counts their occurrences
// Used for single BAM mode to identify paired vs unpaired reads
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

// findCommonNames finds read names that exist in both sets
// Also returns the unpaired (filtered) read names
func findCommonNames(r1, r2 map[string]bool) (common, unpaired map[string]bool) {
	common = make(map[string]bool)
	unpaired = make(map[string]bool)

	// Iterate through the smaller set for efficiency
	if len(r1) < len(r2) {
		for name := range r1 {
			if r2[name] {
				common[name] = true
			} else {
				unpaired[name] = true
			}
		}
		// Add reads only in r2 to unpaired
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
		// Add reads only in r1 to unpaired
		for name := range r1 {
			if !r2[name] {
				unpaired[name] = true
			}
		}
	}

	return common, unpaired
}

// saveReadNames saves read names to a file (one per line)
func saveReadNames(names map[string]bool, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Sort names for consistent output
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

// saveReadNamesList saves a list of read names to a file (one per line)
func saveReadNamesList(names []string, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Sort names for consistent output
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

// filterBAM reads a BAM file and writes only records with matching read names
func filterBAM(inputPath, outputPath string, keepNames map[string]bool) error {
	// Open input
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

	// Create output writer
	writer, err := bamnative.NewWriter(outputPath, header)
	if err != nil {
		return fmt.Errorf("failed to create BAM writer: %w", err)
	}
	defer writer.Close()

	// Filter and write records
	kept := 0
	total := 0

	for {
		rec, err := reader.Read()
		if err != nil {
			break
		}

		total++

		// Keep only records with matching names
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

// filterBAMWithCount reads a BAM file and writes only records with matching read names
// Returns the number of kept records
func filterBAMWithCount(inputPath, outputPath string, keepNames map[string]bool) (int, error) {
	// Open input
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

	// Create output writer
	writer, err := bamnative.NewWriter(outputPath, header)
	if err != nil {
		return 0, fmt.Errorf("failed to create BAM writer: %w", err)
	}
	defer writer.Close()

	// Filter and write records
	kept := 0

	for {
		rec, err := reader.Read()
		if err != nil {
			break
		}

		// Keep only records with matching names
		if keepNames[rec.Name] {
			if err := writer.Write(rec); err != nil {
				return kept, fmt.Errorf("failed to write record: %w", err)
			}
			kept++
		}
	}

	return kept, nil
}

// sortAndIndex sorts and indexes a BAM file
func sortAndIndex(bamPath string) error {
	// Create temporary sorted file path
	sortedPath := strings.TrimSuffix(bamPath, ".bam") + "_sorted.bam"

	// Sort the BAM file
	if err := bamnative.Sort(bamPath, &bamnative.SortOptions{OutputPath: sortedPath}); err != nil {
		return fmt.Errorf("failed to sort BAM: %w", err)
	}

	// Close the writer and move file
	if err := os.Remove(bamPath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	if err := os.Rename(sortedPath, bamPath); err != nil {
		return fmt.Errorf("failed to rename sorted file: %w", err)
	}

	// Create BAI index
	if err := bamnative.BuildIndex(bamPath); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}
