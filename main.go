// Copyright 2024 Jelly Terra <jellyterra@symboltics.com>
// Use of this source code form is governed under the MIT license.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	err := _main()
	if err != nil {
		fmt.Println(err)
	}
}

func _main() error {
	var (
		isRecursive = flag.Bool("R", false, "Recursive scan.")
		minSize     = flag.Int64("min-size", 1024*1024, "Minimum size.")
	)

	flag.Usage = func() {
		fmt.Println("Filesystem Deduplication Utility by (C) 2024 Jelly Terra <jellyterra@symboltics.com>")
		fmt.Println("\nUsage: fs-dedup [OPTIONS]", "(FILE)...")
		flag.PrintDefaults()
	}

	flag.Parse()

	var filePaths []string

	if *isRecursive {
		for _, filePath := range flag.Args() {
			stat, err := os.Stat(filePath)
			switch {
			case err != nil:
				return err
			case stat.IsDir():
				err = filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
					switch {
					case err != nil:
						return err
					case info.Mode().IsRegular():
						// Accept.
						filePaths = append(filePaths, path)
					case info.IsDir():
						// Exclude.
					default:
						// Ignore.
						fmt.Println("Ignoring", path)
					}
					return nil
				})
				if err != nil {
					return err
				}
			case stat.Mode().IsRegular():
				filePaths = append(filePaths, filePath)
			default:
				fmt.Println("Ignoring", filePath)
			}
		}
	} else {
		for _, filePath := range flag.Args() {
			stat, err := os.Stat(filePath)
			switch {
			case err != nil:
				return err
			case stat.IsDir():
				return fmt.Errorf("%s is a directory", filePath)
			case stat.Mode().IsRegular():
				filePaths = append(filePaths, filePath)
			default:
				fmt.Println("Ignoring", filePath)
			}
		}
	}

	sizeMap, err := SeekBySize(filePaths, *minSize)
	if err != nil {
		return err
	}

	fmt.Println(len(sizeMap), "files scanned.")

	var (
		dupSets [][]string

		// Total size of duplications.
		totalDup int64
	)

	for size, sizeSet := range sizeMap {
		// Find the same files out.
		sumMap, errMap := SeekByChecksum(sizeSet)
		if len(errMap) != 0 {
			for filePath, err := range errMap {
				fmt.Println(filePath, err)
			}
			return nil
		}

		for sum, sumSet := range sumMap {
			dupSets = append(dupSets, sumSet)

			totalDup += int64(len(sumSet)-1) * size

			for _, path := range sumSet {
				fmt.Println(path)
			}
			fmt.Printf("Checksum (SHA256): [%x]. Size: [%d] bytes.\n\n", []byte(sum), size)
		}
	}

	fmt.Println("Starting deduplication:", totalDup, "bytes in total.")

	for _, set := range dupSets {
		errMap := Dedup(set)
		if len(errMap) != 0 {
			fmt.Println("Ref-linking error:", errMap)
		}
	}

	fmt.Println("Done.")

	return nil
}
