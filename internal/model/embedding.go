package model

// Embedding is a cached embedding vector of a transaction text.
// TextHash identifies the text the vector was computed from.
type Embedding struct {
	TextHash string
	Vector   []float32
}
