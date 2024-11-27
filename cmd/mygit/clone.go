package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type file struct {
	name string
	hash []byte
	data []byte
}

type directory struct {
	name        string
	hash        []byte
	data        []byte
	files       []*file
	directories []*directory
}

type delta struct {
	base []byte
	data []byte
}

type commit struct {
	hash []byte
	data []byte
}

var files = make(map[string]*file)
var directories = make(map[string]*directory)
var deltas = make([]*delta, 0)
var commits = make([]*commit, 0)
var lastTimestamp int64
var lastTree string

const (
	FILE = iota
	DIRECTORY
)

func getRef(url string) string {
	var buf *bytes.Buffer = new(bytes.Buffer)
	buf = bytes.NewBufferString("0014command=ls-refs\n0000")
	client := http.DefaultClient
	req, err := http.NewRequest(
		"POST", url+"/git-upload-pack", buf,
	)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Git-Protocol", "version=2")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatal(resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	refs := string(body)
	ref := strings.Split(strings.Split(refs, "\n")[0], " ")[0][4:]
	return ref
}

func getBody(url, ref string) []byte {
	buf := bytes.NewBufferString("0011command=fetch00010032want " + ref + "\n0000")
	client := http.DefaultClient
	req, err := http.NewRequest(
		"POST", url+"/git-upload-pack", buf,
	)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Git-Protocol", "version=2")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatal(resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

func readPack(body []byte) []byte {
	re := regexp.MustCompile(`[[:xdigit:]]{4}.PACK`)
	index := re.FindIndex(body)[0]
	size, err := strconv.ParseInt(
		string(body[index:index+4]), 16, 64,
	)
	if err != nil {
		log.Fatal(err)
	}
	pack := body[index : index+int(size)-20]
	return pack
}

func parseCommit(reader *bytes.Reader) {
	zreader, err := zlib.NewReader(reader)
	if err != nil {
		log.Fatal(err)
	}
	buf := new(bytes.Buffer)
	io.Copy(buf, zreader)

	header := []byte(fmt.Sprintf("commit %v\x00", buf.Len()))
	blob := append(header, buf.Bytes()...)

	h := sha1.New()
	h.Write(blob)
	checksum := h.Sum(nil)

	commits = append(commits, &commit{checksum, buf.Bytes()})

	var timestamp int64
	var tree string
	lines := strings.Split(buf.String(), "\n")
	for _, line := range lines {
		words := strings.Split(line, " ")
		if words[0] == "tree" {
			tree = words[1]
		} else if words[0] == "author" {
			var err error
			timestamp, err = strconv.ParseInt(words[3], 10, 64)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			continue
		}
	}

	if timestamp > lastTimestamp {
		lastTree = tree
	}
}

func parseTree(reader *bytes.Reader, compressed bool) {
	buf := new(bytes.Buffer)
	if compressed {
		zreader, err := zlib.NewReader(reader)
		if err != nil {
			log.Fatal(err)
		}
		io.Copy(buf, zreader)
	} else {
		io.Copy(buf, reader)
	}

	data := buf.Bytes()
	re := regexp.MustCompile(`\d{5}`)
	indices := re.FindAllIndex(data, -1)
	boundaries := make([]int, 0)
	for _, index := range indices {
		boundaries = append(boundaries, index[0])
	}
	boundaries = append(boundaries, len(data))
	slices := make([][]byte, 0)
	for i := range boundaries[:len(boundaries)-1] {
		slices = append(slices, data[boundaries[i]:boundaries[i+1]])
	}
	entries := make([]string, 0)
	for _, slice := range slices {
		border := bytes.Index(slice, []byte{0})
		line := [][]byte{slice[0:border], slice[border+1:]}
		entry := fmt.Sprint(string(line[0]), fmt.Sprintf(" %x", line[1]))
		entries = append(entries, entry)
	}

	header := []byte(fmt.Sprintf("tree %v\x00", buf.Len()))
	blob := append(header, buf.Bytes()...)

	h := sha1.New()
	h.Write(blob)
	checksum := h.Sum(nil)

	nodeFiles := make([]*file, 0)
	nodeDirs := make([]*directory, 0)
	for _, entry := range entries {
		line := strings.Split(entry, " ")
		mode, name, checksum := line[0], line[1], line[2]
		if mode == "100644" {
			if val, ok := files[checksum]; ok {
				val.name = name
			} else {
				cs, err := hex.DecodeString(checksum)
				if err != nil {
					log.Fatal(err)
				}
				files[checksum] = &file{name, cs, nil}
			}
			nodeFiles = append(nodeFiles, files[checksum])
		} else if mode == "40000" {
			if val, ok := directories[checksum]; ok {
				val.name = name
			} else {
				cs, err := hex.DecodeString(checksum)
				if err != nil {
					log.Fatal(err)
				}
				directories[checksum] = &directory{name, cs, data, nil, nil}
			}
			nodeDirs = append(nodeDirs, directories[checksum])
		} else {
			log.Fatal("Invalid entry format:", mode)
		}
	}

	if val, ok := directories[hex.EncodeToString(checksum)]; ok {
		val.files = nodeFiles
		val.directories = nodeDirs
	} else {
		dir := &directory{"", checksum, data, nodeFiles, nodeDirs}
		directories[hex.EncodeToString(checksum)] = dir
	}
}

func parseBlob(reader *bytes.Reader) {
	zreader, err := zlib.NewReader(reader)
	if err != nil {
		log.Fatal(err)
	}
	buf := new(bytes.Buffer)
	io.Copy(buf, zreader)

	header := []byte(fmt.Sprintf("blob %v\x00", buf.Len()))
	blob := append(header, buf.Bytes()...)

	h := sha1.New()
	h.Write(blob)
	checksum := h.Sum(nil)

	if val, ok := files[hex.EncodeToString(checksum)]; ok {
		val.data = buf.Bytes()
	} else {
		files[hex.EncodeToString(checksum)] = &file{"", checksum, buf.Bytes()}
	}
}

func parseDelta(reader *bytes.Reader) {
	base := make([]byte, 20)
	if _, err := reader.Read(base); err != nil {
		log.Fatal(err)
	}
	zreader, err := zlib.NewReader(reader)
	if err != nil {
		log.Fatal(err)
	}
	buf := new(bytes.Buffer)
	io.Copy(buf, zreader)

	deltas = append(deltas, &delta{base, buf.Bytes()})
}

func resolveDelta(delta *delta) {
	var mode int
	if val, ok := files[hex.EncodeToString(delta.base)]; ok {
		if len(val.data) == 0 {
			return
		}
		mode = FILE
	} else if val, ok := directories[hex.EncodeToString(delta.base)]; ok {
		if len(val.data) == 0 {
			return
		}
		mode = DIRECTORY
	} else {
		return
	}

	reader := bytes.NewReader(delta.data)
	var err error
	if _, err = binary.ReadUvarint(reader); err != nil {
		log.Fatal(err)
	}
	if _, err = binary.ReadUvarint(reader); err != nil {
		log.Fatal(err)
	}

	data := make([]byte, 0)

	for {
		var b byte
		if b, err = reader.ReadByte(); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		if b&(1<<7) == 0 {
			instructionData := make([]byte, int(b))
			if err = binary.Read(reader, binary.LittleEndian, instructionData); err != nil {
				log.Fatal(err)
			}
			data = append(data, instructionData...)
		} else {
			offsetData := make([]byte, 0)
			for i := 0; i < 4; i++ {
				if b&(1<<i) == 0 {
					offsetData = append(offsetData, 0)
				} else {
					b, err := reader.ReadByte()
					if err != nil {
						log.Fatal(err)
					}
					offsetData = append(offsetData, b)
				}
			}
			offset := binary.LittleEndian.Uint32(offsetData)
			sizeData := make([]byte, 0)
			for i := 0; i < 3; i++ {
				if b&(1<<(i+4)) == 0 {
					sizeData = append(sizeData, 0)
				} else {
					b, err := reader.ReadByte()
					if err != nil {
						log.Fatal(err)
					}
					sizeData = append(sizeData, b)
				}
			}
			sizeData = append(sizeData, 0)
			size := binary.LittleEndian.Uint32(sizeData)
			if size == 0 {
				size = 0x10000
			}

			var instructionData []byte
			if mode == FILE {
				file := files[hex.EncodeToString(delta.base)]
				instructionData = file.data[offset : offset+size]
			} else {
				dir := directories[hex.EncodeToString(delta.base)]
				instructionData = dir.data[offset : offset+size]
			}

			data = append(data, instructionData...)
		}
	}

	if mode == FILE {
		header := []byte(fmt.Sprintf("blob %v\x00", len(data)))
		blob := append(header, data...)

		h := sha1.New()
		h.Write(blob)
		checksum := h.Sum(nil)

		if val, ok := files[hex.EncodeToString(checksum)]; ok {
			val.data = data
		} else {
			files[hex.EncodeToString(checksum)] = &file{"", checksum, data}
		}
	} else {
		reader := bytes.NewReader(data)
		parseTree(reader, false)
	}
}

func resolveTree(tree *directory) {
	if tree.name != "" {
		if err := os.Mkdir(tree.name, 0750); err != nil {
			log.Fatal(err)
		}
		if err := os.Chdir(tree.name); err != nil {
			log.Fatal(err)
		}
	}
	for _, file := range tree.files {
		f, err := os.Create(file.name)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write(file.data)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
	for _, dir := range tree.directories {
		resolveTree(dir)
	}
}

func buildGit() {
	if err := os.Mkdir(".git", 0750); err != nil {
		log.Fatal(err)
	}
	if err := os.Chdir(".git"); err != nil {
		log.Fatal(err)
	}
	if err := os.Mkdir("objects", 0750); err != nil {
		log.Fatal(err)
	}
	if err := os.Chdir("objects"); err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		hash := hex.EncodeToString(file.hash)
		if _, err := os.Stat(hash[0:2]); os.IsNotExist(err) {
			if err = os.Mkdir(hash[0:2], 0750); err != nil {
				log.Fatal(err)
			}
		}
		if err := os.Chdir(hash[0:2]); err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(hash[2:])
		if err != nil {
			log.Fatal(err)
		}
		header := []byte(fmt.Sprintf("blob %v\x00", len(file.data)))
		blob := append(header, file.data...)
		w := zlib.NewWriter(f)
		w.Write(blob)
		w.Close()
		f.Close()
		if err := os.Chdir(".."); err != nil {
			log.Fatal(err)
		}
	}
	for _, dir := range directories {
		hash := hex.EncodeToString(dir.hash)
		if _, err := os.Stat(hash[0:2]); os.IsNotExist(err) {
			if err := os.Mkdir(hash[0:2], 0750); err != nil {
				log.Fatal(err)
			}
		}
		if err := os.Chdir(hash[0:2]); err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(hash[2:])
		if err != nil {
			log.Fatal(err)
		}
		header := []byte(fmt.Sprintf("tree %v\x00", len(dir.data)))
		blob := append(header, dir.data...)
		w := zlib.NewWriter(f)
		w.Write(blob)
		w.Close()
		f.Close()
		if err := os.Chdir(".."); err != nil {
			log.Fatal(err)
		}
	}
	for _, commit := range commits {
		hash := hex.EncodeToString(commit.hash)
		if err := os.Mkdir(hash[0:2], 0750); err != nil {
			log.Fatal(err)
		}
		if err := os.Chdir(hash[0:2]); err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(hash[2:])
		if err != nil {
			log.Fatal(err)
		}
		header := []byte(fmt.Sprintf("commit %v\x00", len(commit.data)))
		blob := append(header, commit.data...)
		w := zlib.NewWriter(f)
		w.Write(blob)
		w.Close()
		f.Close()
		if err := os.Chdir(".."); err != nil {
			log.Fatal(err)
		}
	}
	if err := os.Chdir("../.."); err != nil {
		log.Fatal(err)
	}
}

func printBlob(blob *file) {
	if blob.name != "" {
		fmt.Println(blob.name)
	} else {
		fmt.Println("No name")
	}
	fmt.Println(hex.EncodeToString(blob.hash))
	if blob.data != nil {
		fmt.Println(string(blob.data[:64]), "...")
	} else {
		fmt.Println("No data")
	}
	fmt.Println()
}

func printTree(tree *directory) {
	if tree.name != "" {
		fmt.Println(tree.name)
	} else {
		fmt.Println("No name")
	}
	fmt.Println(hex.EncodeToString(tree.hash))
	for _, file := range tree.files {
		fmt.Println(file.name, hex.EncodeToString(file.hash))
	}
	for _, dir := range tree.directories {
		fmt.Println(dir.name, hex.EncodeToString(dir.hash))
	}
	fmt.Println()
}

func clone(url, path string) {
	ref := getRef(url)
	body := getBody(url, ref)
	pack := readPack(body)
	reader := bytes.NewReader(pack)
	reader.Seek(9, io.SeekStart)

	var version uint32
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		log.Fatal(err)
	}

	var number uint32
	if err := binary.Read(reader, binary.BigEndian, &number); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < int(number); i++ {
		b, err := reader.ReadByte()
		if err != nil {
			log.Fatal(err)
		}
		type_ := (b >> 4) & ((1 << 3) - 1)
		reader.UnreadByte()

		if _, err := binary.ReadUvarint(reader); err != nil {
			log.Fatal(err)
		}

		switch type_ {
		case 1:
			parseCommit(reader)

		case 2:
			parseTree(reader, true)

		case 3:
			parseBlob(reader)

		case 7:
			parseDelta(reader)

		default:
			log.Fatal("Unknown object type:", type_)
		}
	}

	for _, delta := range deltas {
		resolveDelta(delta)
	}

	if path != "" {
		if err := os.Mkdir(path, 0750); err != nil && !os.IsExist(err) {
			log.Fatal(err)
		}
		if err := os.Chdir(path); err != nil {
			log.Fatal(err)
		}
	} else {
		words := strings.Split(url, "/")
		repo := words[len(words)-1]
		if strings.HasSuffix(repo, ".git") {
			repo = strings.TrimSuffix(repo, ".git")
		}
		if err := os.Mkdir(repo, 0750); err != nil {
			log.Fatal(err)
		}
		if err := os.Chdir(repo); err != nil {
			log.Fatal(err)
		}
	}

	buildGit()
	resolveTree(directories[lastTree])
}
