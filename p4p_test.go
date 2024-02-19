package p4p_test

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"testing"

	p4p "github.com/pic4pdf/lib-p4p"
)

func TestWriteFile(t *testing.T) {
	g := p4p.NewGenerator(p4p.A4())
	// Images by Renee French
	imgFiles := []string{"gophers/gopher.png", "gophers/gopher1.jpg", "gophers/gopher2.png"}
	for _, path := range imgFiles {
		if err := g.AddImageFile(path, p4p.ImageOptions{
			Mode: p4p.Fit,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i, path := range imgFiles {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := g.AddImage(img, p4p.ImageOptions{
			Mode:  p4p.Center,
			Scale: []float64{0.5, 1, 1.5}[i],
		}); err != nil {
			t.Fatal(err)
		}
	}
	{
		_, _, _, _, x1, y1, x2, y2, crop := p4p.Render(p4p.A4(), p4p.Point, 316, 317, p4p.ImageOptions{
			Mode: p4p.Fill,
		})
		if !crop {
			t.Fatal("did not detect that image must be cropped")
		}
		if x1 != 45 || y1 != 0 || x2 != 270 || y2 != 317 {
			t.Fatal("wrong crop coordinates, got:", x1, y1, x2, y2)
		}
	}
	for _, path := range imgFiles {
		if err := g.AddImageFile(path, p4p.ImageOptions{
			Mode: p4p.Fill,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.WriteFile("test.pdf"); err != nil {
		t.Fatal(err)
	}
}
