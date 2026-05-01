package domain

import "io"

type FileReader interface {
	Open() (io.ReadCloser, error)
}

type File struct {
	Filename string
	Size     int64
	Reader   FileReader
}
