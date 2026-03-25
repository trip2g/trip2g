package model

import (
	"encoding/binary"
	"math"
)

// NoteChunk is a paragraph-level fragment of a note used for vector search.
// Each chunk carries its own embedding so queries match the most relevant
// fragment rather than the averaged whole-note embedding.
type NoteChunk struct {
	VersionID  int64
	ChunkIndex int
	Content    string    // "{title}\n\n{body}" — same text that was embedded
	Embedding  []float32 // float32 LE, nil until generated
	NotePath   string    // parent note path for dedup and result lookup
}

// Float32SliceToBytes converts []float32 to little-endian []byte for storage.
func Float32SliceToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// BytesToFloat32Slice converts little-endian []byte back to []float32.
func BytesToFloat32Slice(data []byte) []float32 {
	if len(data) == 0 {
		return nil
	}
	floats := make([]float32, len(data)/4)
	for i := range floats {
		floats[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return floats
}
