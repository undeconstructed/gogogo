package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

func (g *game) Draw(file string) error {
	img := image.NewNRGBA(image.Rect(0, 0, 1000, 700))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	img.Set(10, 10, color.Black)

	// drawBox := func(x, y, s int, c color.Color) {
	// 	x, y = x-(s/2), y-(s/2)
	// 	draw.Draw(img, image.Rect(x, y, x+s, y+s), &image.Uniform{c}, image.ZP, draw.Src)
	// }

	drawCircle := func(x, y, r int, c color.Color) {
		p := image.Point{x, y}
		draw.DrawMask(img, img.Bounds(), &image.Uniform{c}, image.ZP, &circle{p, r}, image.ZP, draw.Over)
	}

	drawCity := func(x, y int) {
		drawCircle(x, y, 15, color.Black)
		drawCircle(x, y, 10, color.White)
	}
	drawPlace := func(x, y int) {
		drawCircle(x, y, 6, color.Black)
		drawCircle(x, y, 4, color.White)
		drawCircle(x, y, 2, color.Black)
	}
	drawDot := func(x, y int) {
		drawCircle(x, y, 6, color.Black)
		drawCircle(x, y, 4, color.White)
	}
	drawDanger := func(x, y int) {
		drawCircle(x, y, 6, color.Black)
		drawCircle(x, y, 4, color.NRGBA{255, 0, 0, 255})
	}

	for k, dot := range g.dots {
		x, y := 0, 0
		fmt.Sscanf(k, "%d,%d", &x, &y)
		if dot.Place != "" {
			place := g.places[dot.Place]
			if place.City {
				drawCity(x, y)
			} else {
				drawPlace(x, y)
			}
		} else if dot.Danger {
			drawDanger(x, y)
		} else {
			drawDot(x, y)
		}
	}

	out, err := os.Create(file)
	if err != nil {
		return err
	}
	err = png.Encode(out, img)
	if err != nil {
		return err
	}

	return nil
}
