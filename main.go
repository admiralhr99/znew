package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
)

func appendLineToFile(filename string, linesChan <-chan string, wg *sync.WaitGroup, linesFile *map[string]struct{}) {
	defer wg.Done()

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {

		f, err = os.Create(filename)
		if err != nil {
			log.Fatalf("failed creating file: %s", err)

		}
	}

	defer func() {
		err := f.Close()
		if err != nil {
			log.Fatalf("failed to close file: %s", err)
		}
	}()
	fileInfo, err := f.Stat()
	if err != nil {
		log.Fatalf("failed to get file info: %s", err)
	}

	lastByte := make([]byte, 1)
	if fileInfo.Size() > 0 {
		offset, err := f.Seek(-1, 2)
		if err != nil {
			fmt.Println("Error seeking file:", err)
			return
		}

		_, err = f.ReadAt(lastByte, offset)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}
		for line := range linesChan {

			if line == "" {
				continue
			} else if lastByte[0] == '\n' {
				_, err = fmt.Fprintln(f, line)

			} else {
				_, err = fmt.Fprintf(f, "\n%s", line)

			}
			if err != nil {
				log.Fatal("failed writing to file: ", err)
			}
			(*linesFile)[line] = struct{}{}
		}

	} else {

		for line := range linesChan {

			if line == "" {
				continue
			}
			if lastByte[0] != '\n' {
				_, err = fmt.Fprintln(f, line)

			} else {
				_, err = fmt.Fprintf(f, "\n%s", line)

			}
			if err != nil {
				log.Fatal("failed writing to file: ", err)
			}
			(*linesFile)[line] = struct{}{}
		}
	}
}

func readLinesFromStdin(linesChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			linesChan <- scanner.Text()
		}
	}
	close(linesChan)
}

func readLinesFromFile(filename string, linesFile *map[string]struct{}) {
	file, err := os.Open(filename)
	if err != nil {
		file, err = os.Create(filename)
		if err != nil {
			log.Fatalf("failed creating file: %s", err)

		}
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("failed to close file: %s\n", err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if line != "" {

			(*linesFile)[line] = struct{}{}
		} else {
			fmt.Println("Your file contains empty lines that might be causing issues. Please remove them and try again.")
			fmt.Println("You can do this with the following command:")
			fmt.Println("sed -i '/^$/d'", filename)
			break
		}
	}

}

func main() {
	var quietMode bool
	var dryRun bool
	var trim bool
	flag.BoolVar(&quietMode, "q", false, "quiet mode (no output at all)")
	flag.BoolVar(&dryRun, "d", false, "don't append anything to the file, just print the new lines to stdout")
	flag.BoolVar(&trim, "t", false, "trim leading and trailing whitespace before comparison")
	fileTwo := flag.String("f", "", "file to compare with stdin")
	flag.Parse()

	linesFile := make(map[string]struct{})
	readLinesFromFile(*fileTwo, &linesFile)

	linesChanRead := make(chan string)  // read from stdin
	linesChanWrite := make(chan string) // write to file
	var wg sync.WaitGroup
	wg.Add(2)
	go readLinesFromStdin(linesChanRead, &wg)
	go appendLineToFile(*fileTwo, linesChanWrite, &wg, &linesFile)

	for line := range linesChanRead {
		if linesChanRead != nil {
			if _, exists := linesFile[line]; !exists {
				if quietMode == true {
					linesChanWrite <- line
				} else if dryRun == true {
					fmt.Println(line)
				} else {
					fmt.Println(line)
					linesChanWrite <- line
				}
			}
		}
	}

	close(linesChanWrite)
	wg.Wait()

}
