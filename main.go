package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
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

	lines := readExistingLines(fn)

	var f io.WriteCloser
	var err error
	if fn != "" && !dryRun {
		f, err = os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open file for writing: %s\n", err)
			return
		}
		defer f.Close()
	}

	processInput(lines, f, quietMode, dryRun)
}

func readExistingLines(fn string) sync.Map {
	var lines sync.Map
	if fn == "" {
		return lines
	}

	r, err := os.Open(fn)
	if err != nil {
		return lines
	}
	defer r.Close()

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			lines.Store(line, true)
		}
	}
	return lines
}

func processInput(lines sync.Map, f io.Writer, quietMode, dryRun bool) {
	sc := bufio.NewScanner(os.Stdin)
	var wg sync.WaitGroup
	inputChan := make(chan string)

	// Start worker goroutines
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go processLine(inputChan, &lines, f, quietMode, dryRun, &wg)
	}

	// Read input and send to workers
	for sc.Scan() {
		inputChan <- sc.Text()
	}
	close(inputChan)

	wg.Wait()
}

func processLine(inputChan <-chan string, lines *sync.Map, f io.Writer, quietMode, dryRun bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for line := range inputChan {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, exists := lines.LoadOrStore(line, true); !exists {
			if !quietMode {
				fmt.Println(line)
			}
			if !dryRun && f != nil {
				fmt.Fprintf(f, "%s\n", line)
			}
		}
	}
}
