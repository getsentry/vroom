package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/getsentry/vroom/internal/occurrence"
	"github.com/getsentry/vroom/internal/profile"
	gojson "github.com/goccy/go-json"
	"github.com/pierrec/lz4"
)

const (
	workersCount int = 512
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("./issuedetection <profiles directory>")
		return
	}

	root := args[0]
	f, err := os.Open(root)
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

	for {
		orgPaths, err := f.Readdir(1024)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Fatal(err)
		}
		for _, orgPath := range orgPaths {
			if !orgPath.IsDir() {
				continue
			}
			path := fmt.Sprintf("%s/%s", root, orgPath.Name())
			orgDir, err := os.Open(path)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Fatal(err)
			}

			projectPaths, err := orgDir.Readdir(1024)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Fatal(err)
			}
			for _, projectPath := range projectPaths {
				if !projectPath.IsDir() {
					continue
				}
				path := fmt.Sprintf("%s/%s/%s", root, orgPath.Name(), projectPath.Name())
				projectDir, err := os.Open(path)
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					log.Fatal(err)
				}
				for {
					profilePaths, err := projectDir.Readdir(1024)
					if err != nil {
						if errors.Is(err, io.EOF) {
							break
						}
						log.Fatal(err)
					}
					for _, profilePath := range profilePaths {
						path := fmt.Sprintf("%s/%s", projectDir.Name(), profilePath.Name())
						pathChannel <- path
					}
				}

				projectDir.Close()
			}

			orgDir.Close()
		}
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
			errChan <- err
			continue
		}
		callTrees, err := p.CallTrees()
		if err != nil {
			errChan <- err
			continue
		}
		for _, o := range occurrence.Find(p, callTrees) {
			fmt.Println(o.Event.Platform, o.Event.ProjectID, o.Event.ID, o.IssueTitle, o.Subtitle)
		}
	}
}
