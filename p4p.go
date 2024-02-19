package p4p

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// Base unit is Pt.
type Unit float64

const (
	Point      Unit = 1
	Millimeter Unit = Centimeter / 10
	Centimeter Unit = Inch / 2.54
	Inch       Unit = 72
)

type PageSize struct {
	W    float64
	H    float64
	Unit Unit
}

func A1() PageSize {
	return PageSize{W: 1683.78, H: 2383.94, Unit: Point}
}

func A2() PageSize {
	return PageSize{W: 1190.55, H: 1683.78, Unit: Point}
}

func A3() PageSize {
	return PageSize{W: 841.89, H: 1190.55, Unit: Point}
}

func A4() PageSize {
	return PageSize{W: 595.28, H: 841.89, Unit: Point}
}

func A5() PageSize {
	return PageSize{W: 420.94, H: 595.28, Unit: Point}
}

func A6() PageSize {
	return PageSize{W: 297.64, H: 420.94, Unit: Point}
}

func Legal() PageSize {
	return PageSize{W: 612, H: 1008, Unit: Point}
}

func Letter() PageSize {
	return PageSize{W: 612, H: 792, Unit: Point}
}

func Tabloid() PageSize {
	return PageSize{W: 792, H: 1224, Unit: Point}
}

// Rotates the page size by 90 degrees (switching to landscape on default page sizes).
func (s PageSize) Rotate() PageSize {
	return PageSize{W: s.H, H: s.W, Unit: s.Unit}
}

// Converts the given page size into the same page size represented by a different unit
func (s PageSize) Convert(to Unit) PageSize {
	conv := float64(s.Unit) / float64(to)
	return PageSize{
		W:    s.W * conv,
		H:    s.H * conv,
		Unit: to,
	}
}

type Mode int

const (
	// Center image on page; default DPI: 72.
	Center Mode = iota
	// Scale image to the maximum size where it remains entirely visible.
	Fit
	// Scale image to the size where it takes up the whole page; will chop off edge parts of the image.
	Fill
)

type ImageOptions struct {
	Mode Mode
	// Scale the image's size before positioning; works with all layouts (default: 1).
	Scale float64
}

// Returns an the image layout if rendered onto a the specified page in specified units.
// Cropping coordinates are in pixels on the image. Cropping is only necessary if crop returns true.
func Render(pageSize PageSize, unit Unit, imgWidthPx, imgHeightPx int, opts ImageOptions) (x, y, w, h float64, cropX1, cropY1, cropX2, cropY2 int, crop bool) {
	pgSz := pageSize.Convert(unit)
	pgW, pgH := pgSz.W, pgSz.H

	imgW := float64(imgWidthPx) / float64(unit)
	imgH := float64(imgHeightPx) / float64(unit)

	// Calculate coords.
	{
		switch opts.Mode {
		case Center:
			w, h = imgW, imgH
		case Fit:
			if imgW/imgH > pgW/pgH {
				w, h = pgW, pgW*imgH/imgW
			} else {
				w, h = pgH*imgW/imgH, pgH
			}
		case Fill:
			if imgW/imgH < pgW/pgH {
				w, h = pgW, pgW*imgH/imgW
			} else {
				w, h = pgH*imgW/imgH, pgH
			}
		}

		if opts.Scale > 0 {
			w *= opts.Scale
			h *= opts.Scale
		}

		switch opts.Mode {
		case Center, Fit, Fill:
			x, y = pgW/2-w/2, pgH/2-h/2
		}
	}

	// Calculate cropping coords.
	{
		// Size of an image pixel in units
		pxW := w / float64(imgWidthPx)
		pxH := h / float64(imgHeightPx)
		// Image coords on page in image pixels
		imgX1, imgY1 := x/pxW, y/pxH
		imgX2, imgY2 := imgX1+float64(imgWidthPx), imgY1+float64(imgHeightPx)
		// Page size in image pixels
		pgWPx, pgHPx := pgW/pxW, pgH/pxH

		cropX1, cropY1, cropX2, cropY2 = 0, 0, imgWidthPx, imgHeightPx

		if imgX1 < 0 {
			cropX1 = int(-imgX1)
			crop = true
		}
		if imgY1 < 0 {
			cropY1 = int(-imgY1)
			crop = true
		}
		if imgX2 > pgWPx+imgX1 {
			cropX2 = int(pgWPx - imgX1)
			crop = true
		}
		if imgY2 > pgHPx+imgY1 {
			cropY2 = int(pgHPx - imgY1)
			crop = true
		}
	}

	return
}

type Generator struct {
	pdf        *gofpdf.Fpdf
	imageIndex int
	pageSize   PageSize
}

func NewGenerator(pageSize PageSize) *Generator {
	pageSizePt := pageSize.Convert(Point)
	return &Generator{
		pdf: gofpdf.NewCustom(&gofpdf.InitType{
			OrientationStr: "P",
			UnitStr:        "pt",
			Size:           gofpdf.SizeType{Wd: pageSizePt.W, Ht: pageSizePt.H},
		}),
		pageSize: pageSizePt,
	}
}

func (g *Generator) addImage(typ string, r io.Reader, opts ImageOptions) {
	name := "p4p_image_" + strconv.Itoa(g.imageIndex)
	g.imageIndex++
	g.pdf.AddPage()

	opt := gofpdf.ImageOptions{
		ImageType:             typ,
		AllowNegativePosition: true,
	}

	info := g.pdf.RegisterImageOptionsReader(
		name,
		opt,
		r,
	)

	x, y, w, h, _, _, _, _, _ := Render(g.pageSize, Point, int(info.Width()), int(info.Height()), opts)

	g.pdf.ImageOptions(name, x, y, w, h, false, opt, 0, "")
}

func (g *Generator) AddImage(img image.Image, opts ImageOptions) error {
	hasAlpha := true
	if opImg, ok := img.(interface {
		Opaque() bool
	}); ok {
		hasAlpha = !opImg.Opaque()
	}
	var typ string
	var b bytes.Buffer
	if hasAlpha {
		typ = "png"
		if err := png.Encode(&b, img); err != nil {
			return err
		}
	} else {
		typ = "jpeg"
		if err := jpeg.Encode(&b, img, nil); err != nil {
			return err
		}
	}
	g.addImage(typ, &b, opts)
	return nil
}

func (g *Generator) AddImageFile(path string, opts ImageOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	g.addImage(strings.TrimPrefix(filepath.Ext(path), "."), f, opts)
	return nil
}

func (g *Generator) Write(w io.Writer) error {
	return g.pdf.Output(w)
}

func (g *Generator) WriteFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return g.Write(f)
}
