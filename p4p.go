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
	W float64
	H float64
	// Always uses pt as page size unit if true.
	UnitIsPt bool
}

func A1() PageSize {
	return PageSize{W: 1683.78, H: 2383.94, UnitIsPt: true}
}

func A2() PageSize {
	return PageSize{W: 1190.55, H: 1683.78, UnitIsPt: true}
}

func A3() PageSize {
	return PageSize{W: 841.89, H: 1190.55, UnitIsPt: true}
}

func A4() PageSize {
	return PageSize{W: 595.28, H: 841.89, UnitIsPt: true}
}

func A5() PageSize {
	return PageSize{W: 420.94, H: 595.28, UnitIsPt: true}
}

func A6() PageSize {
	return PageSize{W: 297.64, H: 420.94, UnitIsPt: true}
}

func Legal() PageSize {
	return PageSize{W: 612, H: 1008, UnitIsPt: true}
}

func Letter() PageSize {
	return PageSize{W: 612, H: 792, UnitIsPt: true}
}

func Tabloid() PageSize {
	return PageSize{W: 792, H: 1224, UnitIsPt: true}
}

// Rotates the page size by 90 degrees (switching to landscape on default page sizes).
func (s PageSize) Rotate() PageSize {
	return PageSize{W: s.H, H: s.W, UnitIsPt: s.UnitIsPt}
}

func (s PageSize) normalize(u Unit) PageSize {
	if s.UnitIsPt {
		var res PageSize
		f := float64(u)
		res.W = s.W / f
		res.H = s.H / f
		return res
	}
	return s
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

type P4P struct {
	pdf          *gofpdf.Fpdf
	imageIndex   int
	normPageSize PageSize
	unit         Unit
}

func New(unit Unit, pageSize PageSize) *P4P {
	var unitStr string
	switch unit {
	case Point:
		unitStr = "pt"
	case Millimeter:
		unitStr = "mm"
	case Centimeter:
		unitStr = "cm"
	case Inch:
		unitStr = "in"
	default:
		panic("unhandled case")
	}

	size := pageSize.normalize(unit)

	return &P4P{
		pdf: gofpdf.NewCustom(&gofpdf.InitType{
			OrientationStr: "P",
			UnitStr:        unitStr,
			Size:           gofpdf.SizeType{Wd: size.W, Ht: size.H},
		}),
		normPageSize: size,
		unit:         unit,
	}
}

// Returns the page size in the units of the P4P object.
func (p *P4P) PageSize() (w, h float64) {
	return p.normPageSize.W, p.normPageSize.H
}

// Returns layout in the units of the P4P object.
func (p *P4P) CalcImageLayout(imgWidthPx, imgHeightPx int, opts ImageOptions) (x, y, w, h float64) {
	pgW, pgH := p.PageSize()

	f := float64(p.unit)
	imgW := float64(imgWidthPx) / f
	imgH := float64(imgHeightPx) / f

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

	return
}

// Returns crop coordinates that can be passed into SubImage.
// Cropping only becomes necessary if the image is larger than the page.
func (p *P4P) CalcImageCropCoords(imgWidthPx, imgHeightPx int, opts ImageOptions) (x1, y1, x2, y2 int, mustCrop bool) {
	pgW, pgH := p.PageSize()
	lX, lY, lW, lH := p.CalcImageLayout(imgWidthPx, imgHeightPx, opts)

	// Convert to pixels (=pt).
	pxW := lW / float64(imgWidthPx)
	pxH := lH / float64(imgHeightPx)
	imgX1, imgY1 := lX/pxW, lY/pxH
	imgX2, imgY2 := imgX1+float64(imgWidthPx), imgY1+float64(imgHeightPx)
	pgWPx, pgHPx := pgW/pxW, pgH/pxH

	x1, y1, x2, y2 = 0, 0, imgWidthPx, imgHeightPx

	if imgX1 < 0 {
		x1 = int(-imgX1)
		mustCrop = true
	}
	if imgY1 < 0 {
		y1 = int(-imgY1)
		mustCrop = true
	}
	if imgX2 > pgWPx+imgX1 {
		x2 = int(pgWPx - imgX1)
		mustCrop = true
	}
	if imgY2 > pgHPx+imgY1 {
		y2 = int(pgHPx - imgY1)
		mustCrop = true
	}

	return
}

func (p *P4P) addImage(typ string, r io.Reader, opts ImageOptions) {
	name := "p4p_image_" + strconv.Itoa(p.imageIndex)
	p.imageIndex++
	p.pdf.AddPage()
	info := p.pdf.RegisterImageOptionsReader(
		name,
		gofpdf.ImageOptions{
			ImageType:             typ,
			AllowNegativePosition: true,
		},
		r,
	)

	f := float64(p.unit)
	// Convert image size from the units of the P4P object into pixels.
	imgWPx, imgHPx := int(info.Width()*f), int(info.Height()*f)

	x, y, w, h := p.CalcImageLayout(imgWPx, imgHPx, opts)

	p.pdf.ImageOptions(name, x, y, w, h, false, gofpdf.ImageOptions{
		ImageType:             typ,
		AllowNegativePosition: true,
	}, 0, "")
}

func (p *P4P) AddImage(img image.Image, opts ImageOptions) error {
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
	p.addImage(typ, &b, opts)
	return nil
}

func (p *P4P) AddImageFile(path string, opts ImageOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	p.addImage(strings.TrimPrefix(filepath.Ext(path), "."), f, opts)
	return nil
}

func (p *P4P) Write(w io.Writer) error {
	return p.pdf.Output(w)
}

func (p *P4P) WriteFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return p.Write(f)
}
