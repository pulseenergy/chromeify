package main

import (
	"fmt"
	"os"
	"errors"
	"bytes"
	"log"
	"flag"
	"image"
	//"image/color"
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

func (theme Theme) Decorate(in image.Image) image.Image {
	outerWidth := theme.left.Bounds().Dx() + in.Bounds().Dx() + theme.right.Bounds().Dx()
	outerHeight := theme.top.Bounds().Dy() + in.Bounds().Dy() + theme.bottom.Bounds().Dy()

	img := image.NewRGBA(image.Rect(0, 0, outerWidth, outerHeight))

	// pink fill shows any gaps
//	pink := color.RGBA{255, 0, 255, 255}
//	draw.Draw(img, img.Bounds(), &image.Uniform{pink}, image.ZP, draw.Src)
	draw.Draw(img, in.Bounds().Add(image.Pt(theme.left.Bounds().Dx(), theme.top.Bounds().Dy())), in, image.ZP, draw.Src)

	// top-left
	offset := image.ZP
	draw.Draw(img, theme.topLeft.Bounds().Add(offset), theme.topLeft, image.ZP, draw.Src)

	// top
	for offset := theme.topLeft.Bounds().Dx(); offset < outerWidth - theme.topRight.Bounds().Dx(); offset += theme.top.Bounds().Dx() {
		r := theme.top.Bounds().Add(image.Pt(offset, 0))
		draw.Draw(img, r, theme.top, image.ZP, draw.Src)
	}

	// top-right
	offset = image.Pt(outerWidth - theme.topRight.Bounds().Dx(), 0)
	draw.Draw(img, theme.topRight.Bounds().Add(offset), theme.topRight, image.ZP, draw.Src)

	// left
	for offset := theme.topLeft.Bounds().Dy(); offset < outerHeight - theme.bottomLeft.Bounds().Dy(); offset += theme.left.Bounds().Dy() {
		r := theme.left.Bounds().Add(image.Pt(0, offset))
		draw.Draw(img, r, theme.left, image.ZP, draw.Src)
	}

	// right
	for offset := theme.topRight.Bounds().Dy(); offset < outerHeight - theme.bottomRight.Bounds().Dy(); offset += theme.right.Bounds().Dy() {
		r := theme.right.Bounds().Add(image.Pt(outerWidth - theme.right.Bounds().Dx(), offset))
		draw.Draw(img, r, theme.right, image.ZP, draw.Src)
	}

	// bottom-left
	offset = image.Pt(0, outerHeight - theme.bottomLeft.Bounds().Dy())
	draw.Draw(img, theme.bottomLeft.Bounds().Add(offset), theme.bottomLeft, image.ZP, draw.Src)

	// bottom
	for offset := theme.bottomLeft.Bounds().Dx(); offset < outerWidth - theme.bottomRight.Bounds().Dx(); offset += theme.bottom.Bounds().Dx() {
		r := theme.bottom.Bounds().Add(image.Pt(offset, outerHeight - theme.bottom.Bounds().Dy()))
		draw.Draw(img, r, theme.bottom, image.ZP, draw.Src)
	}

	// bottom-right
	offset = image.Pt(outerWidth - theme.bottomRight.Bounds().Dx(), outerHeight - theme.bottomRight.Bounds().Dy())
	draw.Draw(img, theme.bottomRight.Bounds().Add(offset), theme.topRight, image.ZP, draw.Src)

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

	maxMem := int64(1) << 23 // 8mb
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
		var b bytes.Buffer
		err = png.Encode(&b, out)
		if err != nil {
			return
		}
		im, err := magick.NewFromBlob(b.Bytes(), "png")
		if err != nil {
			return
		}
		err = im.Shadow("#000", 30, 5, 0, 0)
		if err != nil {
			return
		}
		out, err := im.ToBlob("png")
		if err != nil {
			return
		}
		res.Header().Set("Content-Type", "image/png")
		_, err = res.Write(out)
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
