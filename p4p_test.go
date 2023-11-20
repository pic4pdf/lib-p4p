package p4p_test

import (
	"testing"

	p4p "github.com/pic4pdf/lib-p4p"
)

func TestWriteFile(t *testing.T) {
	p := p4p.New(p4p.Millimeter, p4p.A4())
	// Images by Renee French
	imgFiles := []string{"gophers/gopher.png", "gophers/gopher1.jpg", "gophers/gopher2.png"}
	for _, path := range imgFiles {
		if err := p.AddImageFile(path, p4p.ImageOptions{
			Mode: p4p.Fit,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.WriteFile("test.pdf"); err != nil {
		t.Fatal(err)
	}
}
