package main

import (
	"image"

	colors "gopkg.in/go-playground/colors.v1"
)

func imageBaseColor(img image.Image) *colors.RGBColor {
	var R, G, B, A float64

	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r, g, b, a := img.At(x, y).RGBA()
			R += float64(r) / float64(255*img.Bounds().Dy()*img.Bounds().Dy())
			G += float64(g) / float64(255*img.Bounds().Dy()*img.Bounds().Dy())
			B += float64(b) / float64(255*img.Bounds().Dy()*img.Bounds().Dy())
			A += float64(a) / float64(255*img.Bounds().Dy()*img.Bounds().Dy())
		}
	}

	clr, _ := colors.RGB(uint8(R), uint8(G), uint8(B))

	return clr
}
