package algo

import "sync"

// Library scheme snapshots are separate from CLI initialization. The library
// owns its Go runtime; callers must not concurrently use bare algo functions.
// Keep this adapter in a new file so upstream files remain byte-for-byte intact.
type libraryScheme struct {
	white, delimiter int16
	initial          charClass
	delimiters       string
	classes          [128]charClass
	bonuses          [charNumber + 1][charNumber + 1]int16
}

func captureLibraryScheme() libraryScheme {
	return libraryScheme{bonusBoundaryWhite, bonusBoundaryDelimiter, initialCharClass, delimiterChars, asciiCharClasses, bonusMatrix}
}

func (s libraryScheme) apply() {
	bonusBoundaryWhite, bonusBoundaryDelimiter = s.white, s.delimiter
	initialCharClass, delimiterChars = s.initial, s.delimiters
	asciiCharClasses, bonusMatrix = s.classes, s.bonuses
}

var libraryLock sync.Mutex
var libraryOnce sync.Once
var librarySchemes map[string]libraryScheme

// WithLibraryScheme serializes library operations, selects a frozen scheme,
// and restores the previous globals even on panic. Do not nest calls.
func WithLibraryScheme(name string, fn func()) bool {
	libraryLock.Lock()
	defer libraryLock.Unlock()
	previous := captureLibraryScheme()
	defer previous.apply()
	libraryOnce.Do(func() {
		librarySchemes = make(map[string]libraryScheme)
		for _, name := range []string{"default", "path", "history"} {
			// Init assumes pristine defaults; path mutates more than bonuses.
			libraryScheme{white: 10, delimiter: 9, initial: charWhite, delimiters: "/,:;|"}.apply()
			Init(name)
			librarySchemes[name] = captureLibraryScheme()
		}
	})
	s, ok := librarySchemes[name]
	if ok {
		s.apply()
		fn()
	}
	return ok
}
