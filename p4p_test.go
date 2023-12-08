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
		if err := p.AddImage(img, p4p.ImageOptions{
			Mode:  p4p.Center,
			Scale: []float64{0.5, 1, 1.5}[i],
		}); err != nil {
			t.Fatal(err)
		}
	}
	{
		x1, y1, x2, y2, mustCrop := p.CalcImageCropCoords(316, 317, p4p.ImageOptions{
			Mode: p4p.Fill,
		})
		if !mustCrop {
			t.Fatal("did not detect that image must be cropped")
		}
		if x1 != 58 || y1 != 18 || x2 != 257 || y2 != 298 {
			t.Fatal("wrong crop coordinates, got:", x1, y1, x2, y2)
		}
	}
	for _, path := range imgFiles {
		if err := p.AddImageFile(path, p4p.ImageOptions{
			Mode: p4p.Fill,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.WriteFile("test.pdf"); err != nil {
		t.Fatal(err)
	}
}
