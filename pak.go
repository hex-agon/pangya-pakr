package main

import (
	"hash/crc32"
	"os"
)

const EntryCompressionNone = 0
const EntryCompressionLZ77 = 1
const EntryCompressionDirectory = 2

var crcTable = crc32.MakeTable(0x04c11db7)

type Pak struct {
	file  *os.File
	crc32 uint32
}

func (p *Pak) Write(bytes []byte) (n int, err error) {
	p.crc32 = crc32.Update(p.crc32, crcTable, bytes)
	return p.file.Write(bytes)
}

func (p *Pak) Close() error {
	return p.file.Close()
}

type PakTrailer struct {
	FileListOffset uint32
	EntryCount     uint32
	Version        byte
}

type PakEntry struct {
	PathLength         byte
	Compression        byte
	Offset             uint32
	CompressedFileSize uint32
	FileSize           uint32
}

func newPak(path string) (*Pak, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &Pak{file: file}, nil
}
