package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
)

func download(client *storage.Client, root string, objects chan string, errorsChan chan error, wg *sync.WaitGroup) {
	defer wg.Done()

	b := client.Bucket("sentry-profiles")
	for objectName := range objects {
		parts := strings.Split(objectName, "/")
		count := len(parts)
		dirPath := fmt.Sprintf("%s/%s/%s", root, parts[count-3], parts[count-2])

		if _, err := os.Stat(dirPath); errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(dirPath, 0755)
			if err != nil {
				errorsChan <- err
				continue
			}
		}

		objectName := fmt.Sprintf("%s/%s/%s", parts[count-3], parts[count-2], parts[count-1])
		fileName := fmt.Sprintf("%s.json", objectName)
		path := fmt.Sprintf("%s/%s", root, fileName)

		if _, err := os.Stat(path); err == nil {
			continue
		}

		f, err := os.Create(path)
		if err != nil {
			errorsChan <- err
			continue
		}

		ctx := context.Background()
		rc, err := b.Object(objectName).NewReader(ctx)
		if err != nil {
			errorsChan <- err
			continue
		}

		if _, err := io.Copy(f, rc); err != nil {
			errorsChan <- err
			continue
		}

		err = rc.Close()
		if err != nil {
			errorsChan <- err
			continue
		}

		err = f.Close()
		if err != nil {
			errorsChan <- err
			continue
		}

		log.Println(objectName)
	}
}

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		fmt.Println("./downloader <file of relative object paths> <destination directory>")
		return
	}

	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer storageClient.Close()

	objectPathList := args[0]
	destination := args[1]
	file, err := os.Open(objectPathList)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var wg sync.WaitGroup

	objects := make(chan string)
	errorsChan := make(chan error)
	for i := 0; i < 128; i++ {
		wg.Add(1)
		go download(storageClient, destination, objects, errorsChan, &wg)
	}

	go func() {
		for err := range errorsChan {
			log.Println(err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		objects <- scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	close(objects)
	wg.Wait()
	close(errorsChan)
}
