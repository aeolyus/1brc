package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
)

// Set chunk size
const chunkSize int64 = 64 * 1024 * 1024

var input = flag.String("input", "", "file to read")
var jobs = flag.Int("jobs", runtime.NumCPU(), "number of concurrent jobs")
var cpuprofile = flag.String("cpuprofile", "", "file to read cpu profile to ")

func main() {
	flag.Parse()
	// Profiling
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Printf("could not create CPU profile: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Printf("could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}
	// Open the file
	file, err := os.Open(*input)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file info:", err)
		return
	}
	fileSize := fileInfo.Size()

	// Get number of chunks and chunks per reader
	chunks := (fileSize + chunkSize - 1) / chunkSize
	chunksPerReader := (chunks + int64(*jobs) - 1) / int64(*jobs)

	out := make(chan []byte)
	var wg sync.WaitGroup
	for i := 0; i < *jobs; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			start := int64(i) * chunksPerReader * chunkSize
			end := start + chunksPerReader*chunkSize
			end = min(end, fileSize)
			// Buffer to read chunks into
			buf := make([]byte, chunkSize)

			// Read file by chunks
			for pos, n := start, 0; pos < end; pos += int64(n) {
				// Read a chunk
				n, err = file.ReadAt(buf, pos)
				if err != nil && !errors.Is(err, io.EOF) {
					panic(err)
				}

				// Don't read past chunk limits
				n = min(n, int(end-pos))
				buf = buf[:n]

				// If no bytes were read, break the loop
				if n == 0 {
					break
				}

				// If not the first chunk in the file and is
				// first chunk in this worker, read from after
				// the first new line
				//
				// aaa;1.2
				// bbb;3.4
				// ccc;5.6
				//
				// The above example may be split into chunks
				// as follows below
				//
				// aaa;1.2
				//  +--- chunk split here
				//  |
				//  v
				// bbb;3.4
				// ccc;5.6
				//
				// worker 1          | worker 2
				// [(chunk1, chunk2) | (chunk3, chunk4)]
				// ...aaa;1.2\nbb    | b;3.4\nccc;...
				//
				// In this case, we want worker 1 to read the
				// full line of bbb and worker 2 to start
				// reading at ccc.
				if pos != 0 && pos == start {
					i := bytes.Index(buf, []byte{'\n'})
					buf = buf[i+1:]
				}
				_, err := file.Seek(pos+int64(n), 0)
				reader := bufio.NewReader(file)
				overflow, err := reader.ReadBytes('\n')
				if err != nil && !errors.Is(err, io.EOF) {
					panic(err)
				}

				send := make([]byte, len(buf))
				copy(send, buf)
				send = append(send, overflow...)
				n += len(overflow)

				out <- send

			}
		}(i)
	}

	done := make(chan bool)
	go func() {
		for data := range out {
			fmt.Print(string(data))
		}
		done <- true
	}()
	wg.Wait()
	close(out)
	<-done
}
