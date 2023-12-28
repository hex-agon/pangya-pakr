package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"golang.org/x/text/encoding/korean"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func main() {
	flag.Parse()

	if flag.NArg() < 2 {
		log.Println("Usage: pakr <input-dir> <output-pak>")
		return
	}

	root := flag.Arg(0)
	target := flag.Arg(1)

	if target == "" {
		target = filepath.Join(filepath.Dir(root), "pack.pak")
	}
	entryPaths := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if path == root {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entryPaths = append(entryPaths, filepath.ToSlash(relPath))
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	entries := make([]PakEntry, len(entryPaths))
	log.Printf("Packing a total of %d entries to %s", len(entryPaths), target)

	pakFile, err := newPak(target)
	defer pakFile.Close()

	if err != nil {
		log.Fatal(err)
	}

	// Write all the entries and update their metadata
	offset := 0
	for i, entryPath := range entryPaths {
		resolvedPath := filepath.Join(root, entryPath)

		entryInfo, err := os.Stat(resolvedPath)
		if err != nil {
			log.Fatal(err)
		}

		if entryInfo.IsDir() {
			log.Printf("Packing directory '%s'...", entryPath)
			entries[i] = PakEntry{
				PathLength:         byte(len(entryPath)),
				Compression:        EntryCompressionDirectory,
				Offset:             uint32(offset),
				CompressedFileSize: 0,
				FileSize:           0,
			}
		} else {
			log.Printf("Packing file '%s'...", entryPath)
			entryFile, err := os.Open(resolvedPath)
			if err != nil {
				log.Fatal(err)
			}
			written, err := io.Copy(pakFile, entryFile)
			if err != nil {
				log.Fatal(err)
			}
			entries[i] = PakEntry{
				PathLength:         byte(len(entryPath)),
				Compression:        EntryCompressionNone,
				Offset:             uint32(offset),
				CompressedFileSize: uint32(written),
				FileSize:           uint32(written),
			}
			offset += int(written)
		}
	}

	// Write the pak entry list
	for i, entryPath := range entryPaths {
		entry := entries[i]
		entryBytes, err := encodePakEntryForList(entryPath, entry)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := pakFile.Write(entryBytes); err != nil {
			log.Fatal(err)
		}
	}

	// Write the trailer
	trailer := PakTrailer{
		FileListOffset: uint32(offset),
		EntryCount:     uint32(len(entryPaths)),
		Version:        0x12,
	}
	buffer := new(bytes.Buffer)
	err = binary.Write(buffer, binary.LittleEndian, trailer)
	if err != nil {
		log.Fatal(err)
	}
	_, err = pakFile.Write(buffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	// Print the pak info
	stat, _ := os.Stat(target)
	fmt.Printf(`<fileinfo fname="%s" fdir="" fsize="%d" fcrc="%d" fdate="%s" ftime="%s" pname="%s.zip" psize="%d" />`,
		target,
		int32(stat.Size()),
		int32(pakFile.crc32),
		stat.ModTime().Format("2006-02-1"),
		stat.ModTime().Format("15-04-05"),
		"void",
		0,
	)

}

var krEncoder = korean.EUCKR.NewEncoder()

func encodePakEntryForList(entryPath string, entry PakEntry) ([]byte, error) {
	buffer := new(bytes.Buffer)

	entry.Compression ^= 0x80

	if err := binary.Write(buffer, binary.LittleEndian, entry); err != nil {
		return nil, err
	}

	krEntryPath, err := krEncoder.Bytes([]byte(entryPath))
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	buffer.Write(krEntryPath)
	buffer.WriteByte(0x00) // null terminate the entry
	return buffer.Bytes(), nil
}
