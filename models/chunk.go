package models

import "encoding/json"

type Chunk struct {
	ChunkID int             `json:"chunk_id"`
	Params  json.RawMessage `json:"params"`
}

type ChunksBase struct {
	Chunks []*Chunk `json:"chunks"`
}
