package main

import (
	"flag"
	"fmt"
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
			if end > fileSize {
				end = fileSize
			}
			// Buffer to read chunks into
			buf := make([]byte, chunkSize)

			n := 0
			// Read file by chunks
			for pos := start; pos < end; pos += int64(n) {
				// Read a chunk
				n, err = file.ReadAt(buf, pos)
				if err != nil && err.Error() != "EOF" {
					fmt.Println("Error:", err)
					break
				}

				// If no bytes were read, break the loop
				if n == 0 {
					break
				}

				send := make([]byte, n)
				copy(send, buf[:n])
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
