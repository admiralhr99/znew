package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	// Constants for performance optimization
	readBufferSize  = 16 * 1024 * 1024 // 16MB read buffer
	writeBufferSize = 8 * 1024 * 1024  // 8MB write buffer
)

func main() {
	var quietMode, dryRun bool
	flag.BoolVar(&quietMode, "q", false, "quiet mode (no output at all)")
	flag.BoolVar(&dryRun, "d", false, "don't append anything to the file, just print the new lines to stdout")

	// Parse flags after defining them, but before accessing them
	flag.Parse()

	// Get non-flag arguments
	args := flag.Args()
	var fn string
	if len(args) > 0 {
		fn = args[0]
	}

	// Create a map to track existing lines
	// Using map[string]struct{} is more memory efficient than map[string]bool
	existingLines := make(map[string]struct{})

	// Load existing lines if a file was specified
	if fn != "" {
		if err := loadExistingLines(fn, existingLines); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to fully load existing file: %s\n", err)
		}
	}

	// Open file for writing if needed
	var outFile *os.File
	var writer *bufio.Writer
	if fn != "" && !dryRun {
		var err error
		outFile, err = os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open file for writing: %s\n", err)
			return
		}
		defer outFile.Close()

		// Use buffered writer for better performance
		writer = bufio.NewWriterSize(outFile, writeBufferSize)
		defer writer.Flush()
	}

	// Process stdin
	processInput(existingLines, writer, quietMode)
}

// loadExistingLines reads an existing file to populate the line map
func loadExistingLines(fn string, existingLines map[string]struct{}) error {
	file, err := os.Open(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, which is fine
		}
		return err
	}
	defer file.Close()

	// Create a buffered scanner with a large buffer for efficiency
	scanner := bufio.NewScanner(file)
	buf := make([]byte, readBufferSize)
	scanner.Buffer(buf, readBufferSize)

	// Read lines and add to map
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			existingLines[line] = struct{}{}
		}
	}

	return scanner.Err()
}

// processInput handles the stdin
func processInput(existingLines map[string]struct{}, writer *bufio.Writer, quietMode bool) {
	scanner := bufio.NewScanner(os.Stdin)

	// Use a larger buffer for scanner to handle long lines
	buf := make([]byte, readBufferSize)
	scanner.Buffer(buf, readBufferSize)

	// Process each line from stdin
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check if this is a new line
		if _, exists := existingLines[line]; !exists {
			// Add to our tracking map
			existingLines[line] = struct{}{}

			// Output the line if not in quiet mode
			if !quietMode {
				fmt.Println(line)
			}

			// Write to file if we have a writer
			if writer != nil {
				fmt.Fprintln(writer, line)
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %s\n", err)
	}

	// Final flush to ensure everything is written
	if writer != nil {
		writer.Flush()
	}
}
