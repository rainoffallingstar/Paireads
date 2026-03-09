# 🔀 paireads

> A pure Go tool for filtering paired-end BAM reads - no external dependencies required!

## ✨ Features

- 🚀 **Pure Go Implementation** - No samtools, picard, or C dependencies
- ⚡ **Fast & Efficient** - Stream processing with minimal memory overhead
- 🔍 **Smart Filtering** - Automatically identifies and keeps only properly paired reads
- 📊 **Comprehensive Output** - Filtered BAM files + read name lists
- 🗜️ **BGZF Support** - Built-in BAM compression/decompression
- 📈 **Sorted & Indexed** - Output files are coordinate-sorted with BAI indices

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
paireads <R1.bam> <R2.bam> <output_prefix>
```

### Example

```bash
paireads input_R1.bam input_R2.bam filtered
```

This will generate:

```
filtered_R1.bam                  # Filtered R1 BAM (paired reads only)
filtered_R1.bam.bai              # BAI index for R1
filtered_R2.bam                  # Filtered R2 BAM (paired reads only)
filtered_R2.bam.bai              # BAI index for R2
filtered_filtered_readnames.txt  # List of filtered (unpaired) read names
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
       │  Extract read      │
       │  names             │
       ▼                    ▼
┌──────────────────────────────────┐
│   Find Paired Reads              │
│   ┌─────────────────────────┐   │
│   │ R1 ∩ R2 = 800 paired    │   │
│   │ R1 - R2 = 200 unpaired  │   │
│   │ R2 - R1 = 400 unpaired  │   │
│   └─────────────────────────┘   │
└──────────┬───────────────────────┘
           │
           ▼
┌──────────────────────┐   ┌──────────────────────┐
│  Filtered R1.bam     │   │  Filtered R2.bam     │
│  (800 paired reads)  │   │  (800 paired reads)  │
└──────────────────────┘   └──────────────────────┘
           │                           │
           ▼                           ▼
      Sorted & Indexed           Sorted & Indexed
```

## 📊 Output Summary

After running `paireads`, you'll see a summary like:

```
Processing paired-end BAM files:
  R1: input_R1.bam
  R2: input_R2.bam
  Output prefix: filtered

[1/6] Extracting read names from R1...
  Found 1000 reads in R1

[2/6] Extracting read names from R2...
  Found 1200 reads in R2

[3/6] Finding common read names (properly paired)...
  Found 800 properly paired reads
  Found 600 unpaired reads (to be filtered out)
  Saved filtered read names to: filtered_filtered_readnames.txt

[4/6] Filtering R1 BAM to keep only paired reads...
  Kept 800 out of 1000 records

[5/6] Filtering R2 BAM to keep only paired reads...
  Kept 800 out of 1200 records

[6/6] Sorting and indexing output BAM files...
  R1 sorted and indexed successfully
  R2 sorted and indexed successfully

Done!

Summary:
  R1 reads: 1000
  R2 reads: 1200
  Paired reads (kept): 800
  Unpaired reads (filtered out): 600

Output files:
  filtered_R1.bam
  filtered_R1.bam.bai
  filtered_R2.bam
  filtered_R2.bam.bai
  filtered_filtered_readnames.txt
```

## 🛠️ Technical Details

### Memory Usage

- **Stream Processing**: BAM files are processed sequentially, not loaded entirely into memory
- **Efficient Storage**: Only read names are stored in memory during processing
- **External Sorting**: Large files are sorted using temporary disk storage

### File Format

- **BAM**: Binary Alignment/Map format with BGZF compression
- **BAI**: BAM Index for random access
- **Pure Go**: Uses custom BGZF reader/writer (no external C libraries)

### Dependencies

```go
require (
    github.com/klauspost/compress v1.17.8  // Fast gzip compression
)
```

## 🧪 Testing

Create test data with overlapping reads:

```go
// R1: read1, read2, read3, read4, read5
// R2: read3, read4, read5, read6, read7
// Expected: read3, read4, read5 are paired
```

Run the tool:

```bash
paireads test_R1.bam test_R2.bam test_output
```

Expected output:
- Paired reads kept: `read3`, `read4`, `read5`
- Unpaired reads filtered: `read1`, `read2`, `read6`, `read7`

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
