package p4p

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

// Base unit is Pt
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
	// Always uses pt as page size unit if true
	UnitIsPt bool
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

func A2() PageSize {
	return PageSize{W: 1190.55, H: 1683.78, UnitIsPt: true}
}

func A1() PageSize {
	return PageSize{W: 1683.78, H: 2383.94, UnitIsPt: true}
}

func Letter() PageSize {
	return PageSize{W: 612, H: 792, UnitIsPt: true}
}

func Legal() PageSize {
	return PageSize{W: 612, H: 1008, UnitIsPt: true}
}

func Tabloid() PageSize {
	return PageSize{W: 792, H: 1224, UnitIsPt: true}
}

// Rotates the page by 90 degrees (switching to landscape on default page sizes).
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
	Center Mode = iota
	Fit
)

type ImageOptions struct {
	Mode Mode
}

type P4P struct {
	pdf          *gofpdf.Fpdf
	imageIndex   int
	normPageSize PageSize
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
	}
}

func (p *P4P) addImage(typ string, r io.Reader, opts ImageOptions) {
	name := "p4p_image_" + strconv.Itoa(p.imageIndex)
	p.imageIndex++
	p.pdf.AddPage()
	info := p.pdf.RegisterImageOptionsReader(
		name,
		gofpdf.ImageOptions{
			ImageType:             typ,
			ReadDpi:               true,
			AllowNegativePosition: true,
		},
		r,
	)

	imgW, imgH := info.Width(), info.Height()
	pgW, pgH := p.normPageSize.W, p.normPageSize.H

	var w, h float64
	switch opts.Mode {
	case Center:
		w, h = imgW, imgH
	case Fit:
		if imgW/imgH > pgW/pgH {
			w, h = pgW, pgW*imgH/imgW
		} else {
			w, h = pgH*imgW/imgH, pgH
		}
	}

	var x, y float64
	switch opts.Mode {
	case Center, Fit:
		x, y = pgW/2-w/2, pgH/2-h/2
	}

	p.pdf.Image(name, x, y, w, h, false, "", 0, "")
}

func (p *P4P) AddImage(img image.Image, opts ImageOptions) error {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		return err
	}
	p.addImage("jpeg", &b, opts)
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