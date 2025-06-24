package main

import (
	"bytes"
	"errors"

	"github.com/ostafen/digler/internal/format"
)

// simpleScanner implements FileScanner
type simpleScanner struct{}

// Ext returns file extension supported
func (c *simpleScanner) Ext() string {
	return "simple"
}

// Description returns a short description of the format
func (c *simpleScanner) Description() string {
	return "Simple test file format scanner"
}

// Signatures returns file signature byte slices to identify the file
func (c *simpleScanner) Signatures() [][]byte {
	return [][]byte{
		{0xDE, 0xAD, 0xBE, 0xEF}, // example signature
	}
}

// ScanFile tries to detect the signature at the start of the Reader
func (c *simpleScanner) ScanFile(r *format.Reader) (*format.ScanResult, error) {
	signs := c.Signatures()
	buf := make([]byte, len(signs[0]))
	n, err := r.Read(buf)
	if err != nil || n != len(signs[0]) {
		return nil, errors.New("failed to read signature bytes")
	}

	if bytes.Equal(buf, signs[0]) {
		return &format.ScanResult{
			Name: "example.test",
			Ext:  c.Ext(),
			Size: 1024,
		}, nil
	}
	return nil, errors.New("signature mismatch")
}

// Exported constructor function for plugin
func GetScanner() (format.FileScanner, error) {
	return &simpleScanner{}, nil
}
