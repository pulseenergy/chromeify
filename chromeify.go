package main

import (
	//"fmt"
	"os"
	"log"
	"image"
	//"image/color"
	"image/draw"
	"image/png"
)

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

func main() {
	topLeft, err := loadImage("data/top_left.png")
	if err != nil {
		log.Fatal(err)
	}
	topRight, err := loadImage("data/top_right.png")
	if err != nil {
		log.Fatal(err)
	}
	top, err := loadImage("data/top_center.png")
	if err != nil {
		log.Fatal(err)
	}
	left, err := loadImage("data/1x1_border.png")
	if err != nil {
		log.Fatal(err)
	}
	right := left
	bottom := left
	bottomLeft := left
	bottomRight := left

	page, err := loadImage("data/sample_page.png")
	if err != nil {
		log.Fatal(err)
	}

	outerWidth := left.Bounds().Dx() + page.Bounds().Dx() + right.Bounds().Dx()
	outerHeight := top.Bounds().Dy() + page.Bounds().Dy() + bottom.Bounds().Dy()

	img := image.NewRGBA(image.Rect(0, 0, outerWidth, outerHeight))

	// pink fill shows any gaps
//	pink := color.RGBA{255, 0, 255, 255}
//	draw.Draw(img, img.Bounds(), &image.Uniform{pink}, image.ZP, draw.Src)

	draw.Draw(img, page.Bounds().Add(image.Pt(left.Bounds().Dx(), top.Bounds().Dy())), page, image.ZP, draw.Src)

	// top-left
	offset := image.ZP
	draw.Draw(img, topLeft.Bounds().Add(offset), topLeft, image.ZP, draw.Src)

	// top
	for offset := topLeft.Bounds().Dx(); offset < outerWidth - topRight.Bounds().Dx(); offset += top.Bounds().Dx() {
		r := top.Bounds().Add(image.Pt(offset, 0))
		draw.Draw(img, r, top, image.ZP, draw.Src)
	}

	// top-right
	offset = image.Pt(outerWidth - topRight.Bounds().Dx(), 0)
	draw.Draw(img, topRight.Bounds().Add(offset), topRight, image.ZP, draw.Src)

	// left
	for offset := topLeft.Bounds().Dy(); offset < outerHeight - bottomLeft.Bounds().Dy(); offset += left.Bounds().Dy() {
		r := left.Bounds().Add(image.Pt(0, offset))
		draw.Draw(img, r, left, image.ZP, draw.Src)
	}

	// right
	for offset := topRight.Bounds().Dy(); offset < outerHeight - bottomRight.Bounds().Dy(); offset += right.Bounds().Dy() {
		r := right.Bounds().Add(image.Pt(outerWidth - right.Bounds().Dx(), offset))
		draw.Draw(img, r, right, image.ZP, draw.Src)
	}

	// bottom-left
	offset = image.Pt(0, outerHeight - bottomLeft.Bounds().Dy())
	draw.Draw(img, bottomLeft.Bounds().Add(offset), bottomLeft, image.ZP, draw.Src)

	// bottom
	for offset := bottomLeft.Bounds().Dx(); offset < outerWidth - bottomRight.Bounds().Dx(); offset += bottom.Bounds().Dx() {
		r := bottom.Bounds().Add(image.Pt(offset, outerHeight - bottom.Bounds().Dy()))
		draw.Draw(img, r, bottom, image.ZP, draw.Src)
	}

	// bottom-right
	offset = image.Pt(outerWidth - bottomRight.Bounds().Dx(), outerHeight - bottomRight.Bounds().Dy())
	draw.Draw(img, bottomRight.Bounds().Add(offset), topRight, image.ZP, draw.Src)


	writeImage("output.png", img)
}
