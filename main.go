package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
)

var input = flag.String("input", "", "input file path")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

type stat struct {
	min   float64
	max   float64
	count float64
	sum   float64
}

type stationStats struct {
	stats    map[string]*stat
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
func format(ss *stationStats, w io.Writer) {
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
func readStats(fpath string) (*stationStats, error) {
	stations := []string{}
	file, err := os.Open(fpath)
	if err != nil {
		log.Fatal(fmt.Errorf("could not open file: %w", err))
	}
	defer file.Close()

	stats := make(map[string]*stat)

	reader := bufio.NewReader(file)
	for {
		str, err := reader.ReadString('\n')
		str = strings.TrimSpace(str)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("error reading line: %w", err)
		}
		station, temp, err := parseLine(str)
		if err != nil {
			return nil, fmt.Errorf("line parse error: %w", err)
		}
		if val, ok := stats[station]; ok {
			val.count += 1
			val.sum += temp
			val.min = min(val.min, temp)
			val.max = max(val.max, temp)
		} else {
			stations = append(stations, station)
			stats[station] = &stat{
				count: 1,
				min:   temp,
				max:   temp,
				sum:   temp,
			}
		}
	}

	sort.Strings(stations)

	return &stationStats{stats, stations}, nil
}

// parseLine will parse an input string of the format "station;temperature" and
// return the extracted station name as a string and temperature as a float
//
// By avoiding strings.Split, we can avoid allocating a string slice
func parseLine(s string) (string, float64, error) {
	i := strings.Index(s, ";")
	station := s[:i]
	temp, err := strconv.ParseFloat(s[i+1:], 64)
	if err != nil {
		return "", 0, fmt.Errorf("parse float error: %w", err)
	}
	return station, temp, err
}

// round rounds a float with IEEE 754 roundTowardPositive to one decimal place
func round(f float64) float64 {
	return math.Ceil(f*10) / 10
}
