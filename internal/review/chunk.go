package review

import "strings"

// MaxDiffChunkSize is the character limit before splitting a diff into chunks.
const MaxDiffChunkSize = 30000

// DiffChunk represents a portion of a diff for independent review.
type DiffChunk struct {
	// Files included in this chunk
	Files []string
	// The diff text for these files
	Diff string
}

// ChunkDiff splits a unified diff into file-level chunks if it exceeds MaxDiffChunkSize.
// If the diff is small enough, returns a single chunk.
func ChunkDiff(diff string) []DiffChunk {
	if len(diff) <= MaxDiffChunkSize {
		return []DiffChunk{{
			Files: FilesInDiff(diff),
			Diff:  diff,
		}}
	}

	fileDiffs := splitByFile(diff)
	var chunks []DiffChunk
	var currentFiles []string
	var currentDiff strings.Builder

	for _, fd := range fileDiffs {
		// If adding this file would exceed the limit, flush current chunk
		if currentDiff.Len() > 0 && currentDiff.Len()+len(fd.diff) > MaxDiffChunkSize {
			chunks = append(chunks, DiffChunk{
				Files: currentFiles,
				Diff:  currentDiff.String(),
			})
			currentFiles = nil
			currentDiff.Reset()
		}
		currentFiles = append(currentFiles, fd.file)
		currentDiff.WriteString(fd.diff)
	}

	// Flush remaining
	if currentDiff.Len() > 0 {
		chunks = append(chunks, DiffChunk{
			Files: currentFiles,
			Diff:  currentDiff.String(),
		})
	}

	return chunks
}

type fileDiff struct {
	file string
	diff string
}

// splitByFile splits a unified diff into per-file sections.
func splitByFile(diff string) []fileDiff {
	var result []fileDiff
	lines := strings.Split(diff, "\n")
	var current strings.Builder
	var currentFile string

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			// Flush previous file
			if currentFile != "" {
				result = append(result, fileDiff{file: currentFile, diff: current.String()})
				current.Reset()
			}
			currentFile = "" // will be set by +++ line
		}
		if after, ok := strings.CutPrefix(line, "+++ b/"); ok {
			currentFile = after
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}

	if currentFile != "" {
		result = append(result, fileDiff{file: currentFile, diff: current.String()})
	}

	return result
}
