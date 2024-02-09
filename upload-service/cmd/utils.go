package main

import (
	"math/rand"
	"os"
	"path/filepath"
)

func GenerateID() string {
	const length = 6
	const str = "123456789qwertyuiopasdfghjklzxcvbnm"

	id := make([]byte, length)
	for i := 0; i < length; i++ {
		id[i] = str[rand.Intn(len(str))]
	}

	return string(id)
}

func GetFilesList(dir string) ([]string, error) {
	files := []string{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
