package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	rootDir := "." // корневая директория проекта
	outFile, err := os.Create("project_go_files.txt")
	if err != nil {
		log.Fatalf("cannot create output file: %v", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Ищем только .go файлы, исключаем *.pb.go
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".pb.go") {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				log.Printf("cannot read file %s: %v", path, err)
				return nil
			}

			_, _ = writer.WriteString(fmt.Sprintf("File: %s\n", path))
			_, _ = writer.WriteString(string(content))
			_, _ = writer.WriteString("\n--------------------\n")
		}

		return nil
	})

	if err != nil {
		log.Fatalf("error walking the path: %v", err)
	}

	fmt.Println("All .go files (except *.pb.go) have been written to project_go_files.txt")
}
