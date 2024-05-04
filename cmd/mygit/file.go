package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"os"
	"fmt"
	"io"
)

func catFile(hash string) {
	path := fmt.Sprintf(".git/objects/%s/%s", hash[:2], hash[2:])
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()
	r, err := zlib.NewReader(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating zlib reader: %s\n", err)
		os.Exit(1)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading data: %s\n", err)
		os.Exit(1)
	}
	fmt.Print(string(bytes.Split(data, []byte{0})[1]))
}

func hashObject(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading data: %s\n", err)
		os.Exit(1)
	}
	input := []byte(fmt.Sprintf("blob %d\x00", len(data)))
	input = append(input, data...)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	folder := fmt.Sprintf(".git/objects/%s", hash[:2])
	path := fmt.Sprintf("%s/%s", folder, hash[2:])
	err = os.Mkdir(folder, 0750)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		os.Exit(1)
	}
	file, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()
	w := zlib.NewWriter(file)
	_, err = w.Write(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compressing data: %s\n", err)
		os.Exit(1)
	}
	w.Close()
	fmt.Println(hash)
}

