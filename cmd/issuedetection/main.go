package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	gojson "github.com/goccy/go-json"
	"github.com/pierrec/lz4"

	"github.com/getsentry/vroom/internal/occurrence"
	"github.com/getsentry/vroom/internal/profile"
)

const (
	workersCount int = 512
)

func main() {
	debug := flag.Bool("debug", false, "activate debug logs")
	root := flag.String("path", ".", "path to a profile or a directory with profiles")

	flag.Parse()

	if *debug {
		opts := slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		handler := slog.NewTextHandler(os.Stdout, &opts)
		slog.SetDefault(slog.New(handler))
	}

	f, err := os.Open(*root)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	pathChannel := make(chan string, workersCount)
	errChannel := make(chan error)

	go func() {
		for err := range errChannel {
			log.Println(err)
		}
	}()

	var wg sync.WaitGroup

	for w := 0; w < workersCount; w++ {
		wg.Add(1)
		go AnalyzeProfile(pathChannel, errChannel, &wg)
	}

	err = filepath.WalkDir(*root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		pathChannel <- path
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	close(pathChannel)
	wg.Wait()
	close(errChannel)
}

func AnalyzeProfile(pathChannel chan string, errChan chan error, wg *sync.WaitGroup) {
	defer wg.Done()

	for path := range pathChannel {
		f, err := os.Open(path)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				errChan <- err
			}
			continue
		}
		zr := lz4.NewReader(f)
		var p profile.Profile
		err = gojson.NewDecoder(zr).Decode(&p)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				errChan <- err
			}
			continue
		}
		callTrees, err := p.CallTrees()
		if err != nil {
			errChan <- err
			continue
		}
		for _, o := range occurrence.Find(p, callTrees) {
			fmt.Println( // nolint
				o.Event.Platform,
				o.Event.ProjectID,
				o.EvidenceData["profile_id"],
				o.EvidenceData["frame_duration_ns"],
				o.IssueTitle,
				o.Subtitle,
			)
		}
	}
}
