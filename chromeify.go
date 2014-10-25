package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/quirkey/magick"
	"html/template"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	topLeft, err := loadInternalImage("data/top_left.png")
	if err != nil {
		return
	}
	topRight, err := loadInternalImage("data/top_right.png")
	if err != nil {
		return
	}
	top, err := loadInternalImage("data/top_center.png")
	if err != nil {
		return
	}
	border, err := loadInternalImage("data/1x1_border.png")
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

func loadInternalImage(filename string) (image.Image, error) {
	b, err := Asset(filename)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	return img, err
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

// adapted from from template.ParseFiles
func templateParseAssets(filenames ...string) (*template.Template, error) {
	var t *template.Template
	for _, filename := range filenames {
		b, err := Asset(filename)
		if err != nil {
			return nil, err
		}
		s := string(b)
		name := filepath.Base(filename)
		if t == nil {
			t = template.New(name)
		}
		var tmpl *template.Template
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = template.New(name)
		}
		_, err = tmpl.Parse(s)
		if err != nil {
			return nil, err
		}
		return tmpl, nil
	}
	return t, nil
}

var indexTemplate = template.Must(templateParseAssets("data/index.html"))

func indexHandler(res http.ResponseWriter, req *http.Request) {
	indexTemplate.Execute(res, "hello")
}

func decorateHandler(res http.ResponseWriter, req *http.Request) {
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
		b, err := applyDropshadow(out)
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

func applyDropshadow(img image.Image) ([]byte, error) {
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
	var dropshadow bool
	decorateFlags := flag.NewFlagSet("decorate", flag.ExitOnError)
	decorateFlags.BoolVar(&dropshadow, "dropshadow", false, "draw drop shadow")

	var addr string
	serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)
	serveFlags.StringVar(&addr, "addr", ":8080", "http service address")

	var command string
	if (len(os.Args) > 1) {
		command = os.Args[1]
	}
	switch (command) {
	case "decorate":
		decorateFlags.Parse(os.Args[2:])
		if (decorateFlags.NArg() != 2) {
			fmt.Fprintf(os.Stderr, "Usage: %s decorate <input> <output>\n", os.Args[0])
			decorateFlags.PrintDefaults()
			os.Exit(1)
		}
		input := decorateFlags.Arg(0)
		output := decorateFlags.Arg(1)

		theme, err := defaultTheme()
		if err != nil {
			log.Fatal(err)
		}
		page, err := loadImage(input)
		if err != nil {
			log.Fatal(err)
		}
		img := theme.Decorate(page)
		if dropshadow {
			b, err := applyDropshadow(img)
			if err != nil {
				log.Fatal(err)
			}
			err = ioutil.WriteFile(output, b, 0666)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			err = writeImage(output, img)
			if err != nil {
				log.Fatal(err)
			}
		}
	case "serve":
		serveFlags.Parse(os.Args[2:])

		http.Handle("/", http.HandlerFunc(indexHandler))
		http.Handle("/decorate", http.HandlerFunc(decorateHandler))

		fmt.Printf("listening on: %s\n", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized command %s\n", command)
		fallthrough
	case "", "-h", "--help", "help":
		fmt.Fprintf(os.Stderr, "Usage: %s decorate <input> <output>\n", os.Args[0])
		decorateFlags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Usage: %s serve\n", os.Args[0])
		serveFlags.PrintDefaults()
		os.Exit(1)
	}
}
