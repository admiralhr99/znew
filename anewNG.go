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
	var quietMode bool
	var dryRun bool
	var trim bool
	flag.BoolVar(&quietMode, "q", false, "quiet mode (no output at all)")
	flag.BoolVar(&dryRun, "d", false, "don't append anything to the file, just print the new lines to stdout")
	flag.BoolVar(&trim, "t", false, "trim leading and trailing whitespace before comparison")
	flag.Parse()

	fn := flag.Arg(0)

	lines := make(map[string]struct{})
	var lineMutex sync.Mutex

	var f io.WriteCloser

	if fn != "" {
		r, err := os.Open(fn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open file for reading: %s\n", err)
			return
		}
		defer r.Close()

		sc := bufio.NewScanner(r)
		for sc.Scan() {
			line := sc.Text()
			if trim {
				line = strings.TrimSpace(line)
			}
			lineMutex.Lock()
			lines[line] = struct{}{}
			lineMutex.Unlock()
		}

		if !dryRun {
			f, err = os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open file for writing: %s\n", err)
				return
			}
			defer f.Close()
		}
	}

	sc := bufio.NewScanner(os.Stdin)
	var wg sync.WaitGroup
	for sc.Scan() {
		wg.Add(1)
		go func(line string) {
			defer wg.Done()
			if trim {
				line = strings.TrimSpace(line)
			}
			lineMutex.Lock()
			if _, exists := lines[line]; !exists {
				lines[line] = struct{}{}
				lineMutex.Unlock()
				if !quietMode {
					fmt.Println(line)
				}
				if !dryRun && fn != "" {
					if _, err := fmt.Fprintf(f, "%s\n", line); err != nil {
						fmt.Fprintf(os.Stderr, "failed to write to file: %s\n", err)
					}
				}
			} else {
				lineMutex.Unlock()
			}
		}(sc.Text())
	}
	wg.Wait()
}
