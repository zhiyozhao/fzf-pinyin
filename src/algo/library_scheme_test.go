package algo

import "testing"

func TestLibraryRestoresScheme(t *testing.T) {
	before := captureLibraryScheme()
	for _, name := range []string{"path", "history", "default", "path", "default"} {
		if !WithLibraryScheme(name, func() {}) {
			t.Fatal(name)
		}
		if captureLibraryScheme() != before {
			t.Fatalf("%s leaked state", name)
		}
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic not propagated")
			}
		}()
		WithLibraryScheme("path", func() { panic("test") })
	}()
	if captureLibraryScheme() != before {
		t.Fatal("panic leaked state")
	}
	if WithLibraryScheme("bad", func() { t.Fatal("bad scheme ran") }) {
		t.Fatal("bad scheme accepted")
	}
}
