// Copyright 2024 Jelly Terra <jellyterra@symboltics.com>
// Use of this source code form is governed under the MIT license.

package main

import (
	"crypto/sha256"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"sync"
)

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, f)
	return string(hash.Sum(nil)), err
}

func SeekBySize(filePaths []string, minSize int64) (map[int64][]string, error) {
	sizeMap := make(map[int64][]string, len(filePaths))
	for _, filePath := range filePaths {
		stat, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}
		if stat.Size() < minSize {
			continue
		}
		sizeMap[stat.Size()] = append(sizeMap[stat.Size()], filePath)
	}
	for k, set := range sizeMap {
		if len(set) == 1 {
			delete(sizeMap, k)
		}
	}
	return sizeMap, nil
}

func SeekByChecksum(filePaths []string) (map[string][]string, map[string]error) {
	sumMap := make(map[string][]string, len(filePaths)>>1)
	errMap := make(map[string]error)

	type Result struct {
		Path string
		Sum  string
		Err  error
	}

	resultChan := make(chan Result, 32)

	waitCollector := make(chan bool)
	go func() {
		defer close(waitCollector)
		for {
			result, ok := <-resultChan
			switch {
			case !ok:
				return
			case result.Err != nil:
				errMap[result.Path] = result.Err
			default:
				sumMap[result.Sum] = append(sumMap[result.Sum], result.Path)
			}
		}
	}()

	var wg sync.WaitGroup
	for _, filePath := range filePaths {
		wg.Add(1)
		go func() {
			sum, err := HashFile(filePath)
			resultChan <- Result{filePath, sum, err}
			wg.Done()
		}()
	}
	wg.Wait()
	close(resultChan)
	<-waitCollector

	for k, set := range sumMap {
		if len(set) == 1 {
			delete(sumMap, k)
		}
	}
	return sumMap, errMap
}

func Dedup(dupSet []string) map[string]error {
	errMap := make(map[string]error)

	origin := dupSet[0]

	originFile, err := unix.Open(origin, unix.O_RDONLY, 0)
	if err != nil {
		return nil
	}
	defer unix.Close(originFile)

	do := func(dest string) error {
		destFile, err := unix.Open(dest, unix.O_WRONLY, 0)
		if err != nil {
			return err
		}
		defer unix.Close(destFile)

		return unix.IoctlFileClone(destFile, originFile)
	}

	for _, path := range dupSet[1:] {
		err := do(path)
		if err != nil {
			errMap[path] = err
		}
	}

	return errMap
}
