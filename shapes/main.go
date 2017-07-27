package main

import (
	"image"

	"github.com/hajimehoshi/ebiten"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type PointF struct {
	X, Y float64
}

type Line struct {
	P0 PointF
	P1 PointF
}

func (l *Line) Cross(y float64) []float64 {
	if l.P0.Y == l.P1.Y {
		if l.P0.Y == y {
			return []float64{l.P0.X}
		}
		return nil
	}
	if l.P0.Y < y && l.P1.Y <= y {
		return nil
	}
	if l.P0.Y > y && l.P1.Y >= y {
		return nil
	}
	m := (l.P1.X - l.P0.X) / (l.P1.Y - l.P0.Y)
	x := m*(y-l.P0.Y) + l.P0.X
	return []float64{x}
}

type Crosser interface {
	Cross(y float64) []float64
}

type Path []Crosser

func (p Path) Cross(y float64) []float64 {
	r := []float64{}
	for _, c := range p {
		r = append(r, c.Cross(y)...)
	}
	return r
}

var offscreen = image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))

func update(screen *ebiten.Image) error {
	p0 := PointF{10, 20}
	p1 := PointF{20, 30}
	p2 := PointF{40, 35}
	p3 := PointF{30, 25}
	path := Path{
		&Line{p0, p1},
		&Line{p1, p2},
		&Line{p2, p3},
		&Line{p3, p0},
	}

	for j := 0; j < screenHeight; j++ {
		crossed := path.Cross(float64(j))
		if len(crossed) == 0 {
			continue
		}
		if len(crossed) % 2 == 1 {
			crossed = crossed[:len(crossed)-1]
		}
		idx := 0
		for i := 0; i < screenWidth; i++ {
			for len(crossed) > idx && float64(i) >= crossed[idx] {
				idx++
			}
			p := 4 * (j * screenWidth + i)
			if idx % 2 == 1 {
				offscreen.Pix[p] = 0xff
				offscreen.Pix[p+1] = 0xff
				offscreen.Pix[p+2] = 0xff
				offscreen.Pix[p+3] = 0xff
			} else {
				offscreen.Pix[p] = 0
				offscreen.Pix[p+1] = 0
				offscreen.Pix[p+2] = 0
				offscreen.Pix[p+3] = 0
			}
		}
	}

	if ebiten.IsRunningSlowly() {
		return nil
	}

	screen.ReplacePixels(offscreen.Pix)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Fill"); err != nil {
		panic(err)
	}
}
