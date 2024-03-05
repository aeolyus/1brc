package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"sync"
)

const chunkSize = 64 * 1024 * 1024 // 64 MiB

var input = flag.String("input", "", "input file path")
var jobs = flag.Int("jobs", runtime.NumCPU(), "number of concurrent jobs")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

type stat struct {
	min   float64
	max   float64
	count float64
	sum   float64
}

type stationStats struct {
	stats    map[string]stat
	stations []string
}

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	err := eval(*input, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

// eval takes a file path, parses the stations statistics, and returns a
// formatted string of the results
func eval(fpath string, w io.Writer) error {
	ss, err := readStats(fpath)
	if err != nil {
		return fmt.Errorf("error parsing statistics: %w", err)
	}
	format(ss, w)
	return nil
}

// format will take a map of station statistics and a sorted list of stations
// and return the properly formatted string output
func format(ss stationStats, w io.Writer) {
	io.WriteString(w, "{")
	for i, station := range ss.stations {
		v := ss.stats[station]
		io.WriteString(w, fmt.Sprintf(
			"%s=%.1f/%.1f/%.1f",
			station, v.min, round(v.sum/v.count), v.max,
		))
		if i < len(ss.stations)-1 {
			io.WriteString(w, ", ")
		}
	}
	io.WriteString(w, "}\n")
}

// readStats reads the input file given the file path and returns a map of
// station statistics and a sorted list of the stations
func readStats(fpath string) (stationStats, error) {
	chunkChan := make(chan []byte)
	statsChan := make(chan map[string]stat)

	go reader(fpath, chunkChan)

	var wg sync.WaitGroup
	for i := 0; i < *jobs; i++ {
		wg.Add(1)
		go worker(&wg, chunkChan, statsChan)
	}

	resultChan := make(chan stationStats)
	go aggregator(statsChan, resultChan)

	wg.Wait()
	close(statsChan)

	return <-resultChan, nil
}

// aggregator reads a stream of maps of stats and aggregates them all before
// sending it down a result channel
func aggregator(
	statsChan <-chan map[string]stat,
	resultChan chan<- stationStats,
) {
	stations := []string{}
	stats := make(map[string]stat)
	for partialStats := range statsChan {
		for k, v := range partialStats {
			if val, ok := stats[k]; ok {
				val.count += v.count
				val.sum += v.sum
				val.min = min(val.min, v.min)
				val.max = max(val.max, v.max)
				stats[k] = val
			} else {
				stats[k] = v
				stations = append(stations, k)
			}
		}
	}

	sort.Strings(stations)

	resultChan <- stationStats{stats, stations}
	close(resultChan)
}

// reader reads a file chunk by chunk and forwards the chunks to a channel
func reader(fpath string, chunkChan chan<- []byte) error {
	f, err := os.Open(fpath)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer f.Close()

	readBuf := make([]byte, chunkSize)
	leftOver := make([]byte, 0, chunkSize)
	for {
		numBytesRead, err := f.Read(readBuf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error reading file: %w", err)
		}

		readBuf = readBuf[:numBytesRead]
		lastLineIdx := bytes.LastIndex(readBuf, []byte{'\n'})
		sendBuf := append(leftOver, readBuf[:lastLineIdx+1]...)
		leftOver = make([]byte, len(readBuf[lastLineIdx+1:]))
		copy(leftOver, readBuf[lastLineIdx+1:])
		chunkChan <- sendBuf
	}
	close(chunkChan)
	return nil
}

// worker processes strings fed to it by the lines channel input and writes its
// stats map results into the stats channel
func worker(
	wg *sync.WaitGroup,
	chunkChan <-chan []byte,
	statsChan chan<- map[string]stat,
) error {
	defer wg.Done()
	stats := make(map[string]stat)
	for chunk := range chunkChan {
		strChunk := string(chunk)
		start := 0
		var station string
		for i, ch := range strChunk {
			if ch == ';' {
				station = strChunk[start:i]
				start = i + 1
			} else if ch == '\n' {
				temp := parseFloat(strChunk[start:i])
				if val, ok := stats[station]; ok {
					val.count++
					val.sum += temp
					val.min = min(val.min, temp)
					val.max = max(val.max, temp)
					stats[station] = val
				} else {
					stats[station] = stat{
						count: 1,
						min:   temp,
						max:   temp,
						sum:   temp,
					}
				}
				start = i + 1
			}
		}
	}
	statsChan <- stats
	return nil
}

// parseFloat is a custom float parser optimized for the given contraint that
// the input is within the range [-99.9, 99.9]
func parseFloat(s string) float64 {
	var neg bool
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	num := 0.0
	if len(s) == 3 {
		num = float64(int(s[0])-int('0')) +
			float64(int(s[2])-int('0'))/10
	} else {
		num = float64(int(s[0])-int('0'))*10 +
			float64(int(s[1])-int('0')) +
			float64(int(s[3])-int('0'))/10
	}
	if neg {
		num = -num
	}
	return num
}

// round rounds a float with IEEE 754 roundTowardPositive to one decimal place
func round(f float64) float64 {
	return math.Ceil(f*10) / 10
}
