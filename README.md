# 🔀 paireads

> A pure Go tool for filtering paired-end BAM reads - no external dependencies required!

## ✨ Features

- 🚀 **Pure Go Implementation** - No samtools, picard, or C dependencies
- ⚡ **Fast & Efficient** - Stream processing with minimal memory overhead
- 🔍 **Explicit Filtering Contracts** - Complete primary mate groups in single mode, shared primary mapped names in dual mode
- 📊 **Comprehensive Output** - Filtered BAM files + read name lists
- 🗜️ **BGZF Support** - Built-in BAM compression/decompression
- 📈 **Selectable Output Order** - Name-sorted by default; coordinate-sorted with BAI using `--coord-sort`

## 📦 Installation

```bash
# Clone the repository
git clone https://github.com/rainoffallingstar/paireads.git
cd paireads

# Build the tool
go build -o paireads ./cmd/paireads
```

## 🎯 Usage

```bash
# Single merged BAM: retain complete unique primary R1/R2 mate groups
paireads [--coord-sort] <input.bam> <output.bam>

# Separate BAMs: retain unique primary mapped names present in both files
paireads [--coord-sort] <R1.bam> <R2.bam> <output_prefix>
```

Without `--coord-sort`, BAM outputs are query-name sorted and no BAI is produced. With `--coord-sort`, BAM outputs are coordinate-sorted and indexed.

### Example

```bash
paireads --coord-sort input_R1.bam input_R2.bam filtered
```

This will generate:

```
filtered_R1.bam                  # Matched primary R1 records
filtered_R1.bam.bai              # BAI index for R1
filtered_R2.bam                  # Matched primary R2 records
filtered_R2.bam.bai              # BAI index for R2
filtered_filtered_readnames.txt  # Names present in only one input
```

## 📋 What It Does

`paireads` replaces the traditional samtools/picard workflow:

```bash
# ❌ Old way (multiple tools)
samtools view R1.bam | awk '{print $1}' | sort > readnames.txt
picard FilterSamReads I=R1.bam O=R1_filtered.bam READ_LIST_FILE=readnames.txt FILTER=excludeReadList
picard SortSam I=R1_filtered.bam O=R1_sorted.bam SORT_ORDER=coordinate
picard BuildBamIndex I=R1_sorted.bam

# ✅ New way (one tool)
paireads R1.bam R2.bam filtered
```

## 🔄 Workflow

```
┌─────────────┐      ┌─────────────┐
│   R1.bam    │      │   R2.bam    │
│  (1000 reads)│     │  (1200 reads)│
└──────┬──────┘      └──────┬──────┘
       │                    │
       │  External name     │
       │  sorting           │
       ▼                    ▼
┌──────────────────────────────────┐
│   Stream Shared Read Names       │
│   ┌─────────────────────────┐   │
│   │ R1 ∩ R2 = 800 matched   │   │
│   │ R1 - R2 = 200 unmatched │   │
│   │ R2 - R1 = 400 unmatched │   │
│   └─────────────────────────┘   │
└──────────┬───────────────────────┘
           │
           ▼
┌──────────────────────┐   ┌──────────────────────┐
│  Filtered R1.bam     │   │  Filtered R2.bam     │
│ (800 matched names)  │   │ (800 matched names)  │
└──────────────────────┘   └──────────────────────┘
           │                           │
           ▼                           ▼
      Sorted & Indexed           Sorted & Indexed
```

## 📊 Output Summary

After running dual mode with `--coord-sort`, the output is summarized like this:

```
Processing paired-end BAM files:
  R1: input_R1.bam
  R2: input_R2.bam
  Output prefix: filtered

[1/6] Name-sorting R1 input...
[2/6] Name-sorting R2 input...
[3/6] Streaming matched read names...
  Found 1000 unique primary names in R1
  Found 1200 unique primary names in R2
  Found 800 matched read names
  Found 600 unmatched read names
[6/6] Coordinate-sorting and indexing output BAM files...

Done!

Summary:
  R1 unique primary names: 1000
  R2 unique primary names: 1200
  Matched read names (kept): 800
  Unmatched read names (filtered out): 600
```

The tool uses primary mapped records to decide eligibility and writes only those records. Secondary, supplementary, and unmapped records are not retained. In dual mode, a shared name means name intersection only; it does not assert SAM `FlagProperPair` or mate-coordinate correctness.

## 🛠️ Technical Details

### Memory Usage

- **Bounded External Sorting**: Each query-name sort uses a 64 MiB memory limit and temporary disk runs
- **Group Streaming**: Single mode retains only the current read-name group; dual mode retains one group per input
- **Transactional Publication**: BAM, BAI, and filtered-name outputs are staged and published with rollback

### File Format

- **BAM**: Binary Alignment/Map format with BGZF compression
- **BAI**: BAM Index for random access
- **Pure Go**: Uses custom BGZF reader/writer (no external C libraries)

### Dependencies

```go
require github.com/rainoffallingstar/bamdriver-go v0.1.2-0.20260721055359-a22f77784fc4
```

## 🧪 Testing

Create test data with overlapping reads:

```go
// R1: read1, read2, read3, read4, read5
// R2: read3, read4, read5, read6, read7
// Expected: read3, read4, read5 are matched names
```

Run the tool:

```bash
paireads test_R1.bam test_R2.bam test_output
```

Expected output:
- Matched names kept: `read3`, `read4`, `read5`
- Unmatched names filtered: `read1`, `read2`, `read6`, `read7`

## 📁 Project Structure

```
paireads/
├── cmd/
│   └── paireads/
│       └── main.go           # CLI application
├── bamnative/
│   ├── bamnative.go          # BAM reader/writer
│   ├── writer.go             # BAM writer
│   ├── index.go              # BAI index creation
│   └── sort.go               # BAM coordinate sorting
├── internal/
│   └── bgzip/
│       ├── bgzip.go          # BGZF decompression
│       └── writer.go         # BGZF compression
├── go.mod
├── go.sum
└── README.md
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [klauspost/compress](https://github.com/klauspost/compress) - Fast gzip compression
- BAM format specification by [SAMtools](https://samtools.github.io/hts-specs/)

## 📞 Contact

For issues and questions, please open an issue on GitHub.

---

Made with ❤️ by [rainoffallingstar](https://github.com/rainoffallingstar)
