package fzf

import (
	"reflect"
	"testing"

	"github.com/junegunn/fzf/src/util"
)

func TestPinyinCacheDoesNotAssumeSuffixClosure(t *testing.T) {
	chunk := &Chunk{count: chunkSize}
	for i := range chunk.items {
		text := "不匹配"
		if i == 0 {
			text = "中国"
		}
		chunk.items[i].text = util.ToChars([]byte(text))
		chunk.items[i].text.Index = int32(i)
	}
	cache := NewChunkCache()
	// A user can type hongg first, then insert z at the start. hongg is not
	// a spelling of 中国; zhongg is. Upstream's suffix cache would lose it.
	before := buildPatternWith(cache, []rune("hongg"))
	if got := before.Match(chunk, nil); len(got) != 0 {
		t.Fatal("invalid prefix inside a syllable")
	}
	if cache.Lookup(chunk, "hongg") == nil {
		t.Fatal("test must populate real bitmap cache")
	}
	got := buildPatternWith(cache, []rune("zhongg")).Match(chunk, nil)
	want := buildPatternWith(NewChunkCache(), []rune("zhongg")).Match(chunk, nil)
	if len(want) != 1 || !reflect.DeepEqual(got, want) {
		t.Fatalf("suffix cache lost recall: got %v want %v", got, want)
	}
}

func TestPinyinCachedAndFreshPatterns(t *testing.T) {
	chunk := &Chunk{count: chunkSize}
	words := []string{"中国", "重庆银行", "音乐", "中x国", "x中国", "方案", "女绿", "北京", "foo.go"}
	for i := range chunk.items {
		text := "没有相关字"
		if i < len(words) {
			text = words[i]
		}
		chunk.items[i].text = util.ToChars([]byte(text))
		chunk.items[i].text.Index = int32(i)
	}
	cache := NewChunkCache()
	for _, q := range []string{"h", "ho", "hong", "hongg", "z", "zh", "zho", "zhon", "zhong", "zhongg", "zhonggu", "zhongguo", "zg", "c", "cq", "cqy", "cqyh", "y", "yi", "yin", "yiny", "yinyue", "zhongg | cq", "zg !^x", "'zg", "^zg$"} {
		got := buildPatternWith(cache, []rune(q)).Match(chunk, nil)
		want := buildPatternWith(NewChunkCache(), []rune(q)).Match(chunk, nil)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("query %q: cached != fresh", q)
		}
	}
}
