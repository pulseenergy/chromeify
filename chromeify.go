package main

import (
	"fmt"
	"os"
	"errors"
	"bytes"
	"log"
	"flag"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"html/template"
	"github.com/quirkey/magick"
)

type Theme struct {
	name string
	topLeft image.Image
	top image.Image
	topRight image.Image
	left image.Image
	right image.Image
	bottomLeft image.Image
	bottom image.Image
	bottomRight image.Image
}

func defaultTheme() (theme Theme, err error) {
	topLeft, err := loadImage("data/top_left.png")
	if err != nil {
		return
	}
	topRight, err := loadImage("data/top_right.png")
	if err != nil {
		return
	}
	top, err := loadImage("data/top_center.png")
	if err != nil {
		return
	}
	border, err := loadImage("data/1x1_border.png")
	if err != nil {
		return
	}
	return Theme{"default", topLeft, top, topRight, border, border, border, border, border}, nil
}

func drawOffset(dst draw.Image, src image.Image, offset image.Point) {
	draw.Draw(dst, src.Bounds().Add(offset), src, image.ZP, draw.Src)
}

func (theme Theme) Decorate(in image.Image) image.Image {
	outerWidth := theme.left.Bounds().Dx() + in.Bounds().Dx() + theme.right.Bounds().Dx()
	outerHeight := theme.top.Bounds().Dy() + in.Bounds().Dy() + theme.bottom.Bounds().Dy()

	img := image.NewRGBA(image.Rect(0, 0, outerWidth, outerHeight))

	// pink fill shows any gaps
//	pink := color.RGBA{255, 0, 255, 255}
//	draw.Draw(img, img.Bounds(), &image.Uniform{pink}, image.ZP, draw.Src)
	drawOffset(img, in, image.Pt(theme.left.Bounds().Dx(), theme.top.Bounds().Dy()))

	// top-left
	drawOffset(img, theme.topLeft, image.ZP)

	// top
	for offset := image.Pt(theme.topLeft.Bounds().Dx(), 0); offset.X < outerWidth - theme.topRight.Bounds().Dx(); offset.X += theme.top.Bounds().Dx() {
		drawOffset(img, theme.top, offset)
	}

	// top-right
	offset := image.Pt(outerWidth - theme.topRight.Bounds().Dx(), 0)
	drawOffset(img, theme.topRight, offset)

	// left
	for offset := image.Pt(0, theme.topLeft.Bounds().Dy()); offset.Y < outerHeight - theme.bottomLeft.Bounds().Dy(); offset.Y += theme.left.Bounds().Dy() {
		drawOffset(img, theme.left, offset)
	}

	// right
	for offset := image.Pt(outerWidth - theme.right.Bounds().Dx(), theme.topRight.Bounds().Dy()); offset.Y < outerHeight - theme.bottomRight.Bounds().Dy(); offset.Y += theme.right.Bounds().Dy() {
		drawOffset(img, theme.right, offset)
	}

	// bottom-left
	offset = image.Pt(0, outerHeight - theme.bottomLeft.Bounds().Dy())
	drawOffset(img, theme.bottomLeft, offset)

	// bottom
	for offset := image.Pt(theme.bottomLeft.Bounds().Dx(), outerHeight - theme.bottom.Bounds().Dy()); offset.X < outerWidth - theme.bottomRight.Bounds().Dx(); offset.X += theme.bottom.Bounds().Dx() {
		drawOffset(img, theme.bottom, offset)
	}

	// bottom-right
	offset = image.Pt(outerWidth - theme.bottomRight.Bounds().Dx(), outerHeight - theme.bottomRight.Bounds().Dy())
	drawOffset(img, theme.bottomRight, offset)

	return img
}

func loadImage(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func writeImage(filename string, img image.Image) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

var indexTemplate = template.Must(template.ParseFiles("data/index.html"))

func index(res http.ResponseWriter, req *http.Request) {
	indexTemplate.Execute(res, "hello")
}

func decorate(res http.ResponseWriter, req *http.Request) {
	var err error
	status := http.StatusInternalServerError

	defer func () {
		if err != nil {
			http.Error(res, err.Error(), status)
		}
	}()

	maxMem := int64(1) << 22 // 4mb
	err = req.ParseMultipartForm(maxMem)
	if err != nil {
		return
	}
	m := req.MultipartForm
	images := m.File["image"]
	if len(images) != 1 {
		err = errors.New("expected one image");
		status = http.StatusBadRequest
		return
	}
	theme, err := defaultTheme()
	if err != nil {
		return
	}
	f, err := images[0].Open()
	if err != nil {
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return
	}
	out := theme.Decorate(img)
	if len(m.Value["dropshadow"]) == 1 && m.Value["dropshadow"][0] == "true" {
		b, err := dropshadow(out)
		if err != nil {
			return
		}
		res.Header().Set("Content-Type", "image/png")
		_, err = res.Write(b)
		if err != nil {
			return
		}
	} else {
		res.Header().Set("Content-Type", "image/png")
		err = png.Encode(res, out)
		if err != nil {
			return
		}
	}
}

func dropshadow(img image.Image) ([]byte, error) {
	var b bytes.Buffer
	err := png.Encode(&b, img)
	if err != nil {
		return nil, err
	}
	im, err := magick.NewFromBlob(b.Bytes(), "png")
	if err != nil {
		return nil, err
	}
	err = im.Shadow("#000", 30, 5, 0, 0)
	if err != nil {
		return nil, err
	}
	return im.ToBlob("png")
}

func main() {
	addr := flag.String("addr", "", "http service address")
	file := flag.String("file", "", "decorate specified file; writes to output.png")
	flag.Parse()

	if *file != "" {
		fmt.Printf("decorating file: %s\n", *file)

		theme, err := defaultTheme()
		if err != nil {
			log.Fatal(err)
		}
		page, err := loadImage(*file)
		if err != nil {
			log.Fatal(err)
		}
		img := theme.Decorate(page)
		err = writeImage("output.png", img)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		http.Handle("/", http.HandlerFunc(index))
		http.Handle("/decorate", http.HandlerFunc(decorate))

		fmt.Printf("listening on: %s\n", *addr)
		err := http.ListenAndServe(*addr, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}
