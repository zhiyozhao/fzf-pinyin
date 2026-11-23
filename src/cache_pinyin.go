package fzf

// SearchPrefix is the sound subset of Search for syllable-aware matching.
// Suffix closure does not hold: zhongg matches 中国, but hongg does not.
func (cc *ChunkCache) SearchPrefix(chunk *Chunk, key string) *ChunkBitmap {
	if len(key) == 0 || !chunk.IsFull() {
		return nil
	}
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	qc, ok := cc.cache[chunk]
	if !ok {
		return nil
	}
	for end := len(key) - 1; end > 0; end-- {
		if bitmap, found := (*qc)[key[:end]]; found {
			return &bitmap
		}
	}
	return nil
}
