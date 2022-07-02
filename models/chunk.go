package models

import "encoding/json"

type Chunk struct {
	ChunkID int             `json:"chunk_id"`
	File    string          `json:"file"`
	Params  json.RawMessage `json:"params"`
}

type ChunksBase struct {
	Chunks []*Chunk `json:"chunks"`
}
