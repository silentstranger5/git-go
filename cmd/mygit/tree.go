package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func lsTree(hash string, namesOnly bool) {
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
	var data bytes.Buffer
	_, err = io.Copy(&data, r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading data: %s\n", err)
		os.Exit(1)
	}
	var bs []byte
	tree := make([]string, 0)
	bs, err = data.ReadBytes(byte(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading data: %s\n", err)
		os.Exit(1)
	}
	header := string(bs)
	header = strings.TrimSuffix(header, "\x00")
	if strings.Split(header, " ")[0] != "tree" {
		fmt.Fprintf(os.Stderr, "File is not a tree\n")
		os.Exit(1)
	}
	tree = append(tree, header)
	for data.Len() > 0 {
		bs, err = data.ReadBytes(byte(' '))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading data: %s\n", err)
			os.Exit(1)
		}
		mode := string(bs)
		mode = strings.TrimSuffix(mode, " ")
		var ftype string
		if mode == "040000" {
			ftype = "tree"
		} else if mode == "100644" {
			ftype = "blob"
		}
		bs, err = data.ReadBytes(byte(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading data: %s\n", err)
			os.Exit(1)
		}
		name := string(bs)
		strings.TrimSuffix(name, "\x00")
		bs = data.Next(20)
		hash := fmt.Sprintf("%x", bs)
		entry := []string{
			mode, ftype, hash, name,
		}
		var row string
		if namesOnly {
			row = name
		} else {
			row = strings.Join(entry, " ")
		}
		tree = append(tree, row)
	}
	for _, row := range tree {
		fmt.Println(row)
	}
}

func writeTree(path string) string {
	var tree string
	entries, err := os.ReadDir(path)
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == ".git" {
			continue
		}
		if entry.IsDir() {
			mode := "040000"
			hash := writeTree(path + "/" + entry.Name())
			tree += fmt.Sprintf("%s %s\x00%s", mode, entry.Name(), string(hash[:]))
		} else {
			mode := "100644"
			data, err := os.ReadFile(path + "/" + entry.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
				os.Exit(1)
			}
			hash := sha1.Sum(data)
			tree += fmt.Sprintf("%s %s\x00%s", mode, entry.Name(), string(hash[:]))
		}
	}
	data := []byte(tree)
	data = append([]byte(fmt.Sprintf("tree %d", len(data)) + "\x00"), data...)
	shash := sha1.Sum(data)
	hsum  := sha256.Sum256(data)
	lhash := fmt.Sprintf("%x", hsum[:])
	folder := fmt.Sprintf(".git/objects/%s", lhash[:2])
	filepath := fmt.Sprintf("%s/%s", folder, lhash[2:])
	err = os.Mkdir(folder, 0750)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		os.Exit(1)
	}
	file, err := os.Create(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()
	w := zlib.NewWriter(file)
	_, err = w.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compressing data: %s\n", err)
		os.Exit(1)
	}
	w.Close()
	fmt.Println(lhash)
	return string(shash[:])
}

func commitTree(hash, parent, message string) {
	_, err := os.Stat(".git/config")
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Git is not configured. Try 'config'\n")
		os.Exit(1)
	}
	b, err := os.ReadFile(".git/config")
	s := string(b)
	conf := strings.Split(s, "\n")
	conf = conf[:len(conf)-1]
	name := strings.Split(conf[0], "=")[1]
	email := strings.Split(conf[1], "=")[1]
	var commit string
	commit += fmt.Sprintf("tree %s\x00", hash)
	if parent != "" {
		commit += fmt.Sprintf("parent %s\x00", parent)
	}
	commit += fmt.Sprintf("author %s %s %d\x00", name, email, time.Now().Unix())
	commit += fmt.Sprintf("%s\x00", message)
	data := []byte(commit)
	data = append([]byte(fmt.Sprintf("commit %d\x00", len(data))), data...)
	hashsum := fmt.Sprintf("%x", sha256.Sum256(data))
	path := fmt.Sprintf(".git/objects/%s", hashsum[:2])
	filepath := fmt.Sprintf("%s/%s", path, hashsum[2:])
	err = os.Mkdir(path, 0750)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		os.Exit(1)
	}
	file, err := os.Create(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()
	w := zlib.NewWriter(file)
	_, err = w.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compressing data: %s\n", err)
		os.Exit(1)
	}
	w.Close()
	fmt.Println(hashsum)
}

