package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"log"
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
	fmt.Printf("Hello, world.\n")
	left, err := loadImage("data/top_left.png")
	if err != nil {
		log.Fatal(err)
	}
	right, err := loadImage("data/top_right.png")
	if err != nil {
		log.Fatal(err)
	}
	center, err := loadImage("data/top_center.png")
	if err != nil {
		log.Fatal(err)
	}
	
	lbounds := left.Bounds()
	rbounds := right.Bounds()
	cbounds := center.Bounds()

	middle := 200
	bounds := image.Rect(0, 0, lbounds.Dx() + rbounds.Dx() + middle, lbounds.Dy())
	fmt.Printf("l: %s, r: %s, out: %s\n", lbounds, rbounds, bounds)

	img := image.NewRGBA(bounds)
	draw.Draw(img, lbounds, left, image.ZP, draw.Src)
	for offset := 0; offset < middle; offset += cbounds.Dx() {
		r := center.Bounds().Add(image.Pt(offset + lbounds.Dx(), 0))
		draw.Draw(img, r, center, image.ZP, draw.Src)
	}
	draw.Draw(img, rbounds.Add(image.Pt(lbounds.Dx() + middle, 0)), right, image.ZP, draw.Src)
	writeImage("output.png", img)
}
