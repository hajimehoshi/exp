package main

import (
	"image"

	"github.com/hajimehoshi/ebiten"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type PointF struct {
	X, Y float64
}

type Line struct {
	P0 PointF
	P1 PointF
}

func (l *Line) Intersect(y float64) ([]float64, bool) {
	if l.P0.Y == y {
		return nil, false
	}
	if l.P1.Y == y {
		return nil, false
	}
	if l.P0.Y == l.P1.Y {
		return nil, true
	}
	if l.P0.Y < y && l.P1.Y < y {
		return nil, true
	}
	if l.P0.Y > y && l.P1.Y > y {
		return nil, true
	}
	m := (l.P1.X - l.P0.X) / (l.P1.Y - l.P0.Y)
	x := m*(y-l.P0.Y) + l.P0.X
	return []float64{x}, true
}

type Intersecter interface {
	Intersect(y float64) ([]float64, bool)
}

type Path struct {
	intersecters []Intersecter
}

func (p Path) Intersect(y float64) ([]float64, bool) {
	r := []float64{}
	for _, i := range p.intersecters {
		xs, ok := i.Intersect(y)
		if !ok {
			return nil, ok
		}
		r = append(r, xs...)
	}
	return r, true
}

var offscreen = image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))

func colorAt(path *Path, x, y int) uint8 {
	// This function emulates a fragment shader.

	color := 0.0
	const div = 4.0
	for j := 0; j < div; j++ {
		var intersections []float64
		offset := float64(j) / div
		for yy := float64(y) + offset; yy < float64(y)+offset+(1.0/div); yy += 1.0 / 256.0 {
			is, ok := path.Intersect(yy)
			if !ok {
				continue
			}
			intersections = is
			break
		}
		if len(intersections) == 0 {
			continue
		}
		idx := 0
		for xx := 0; xx < x && len(intersections) > idx; xx++ {
			for len(intersections) > idx && float64(xx+1) > intersections[idx] {
				idx++
			}
		}
		val := 0.0
		last := float64(x)
		for len(intersections) > idx && float64(x+1) > intersections[idx] {
			if idx%2 != 0 {
				val += intersections[idx] - last
			}
			last = intersections[idx]
			idx++
		}
		if idx%2 != 0 {
			val += float64(x+1) - last
		}
		color += val / div
	}

	return uint8(color * 255)
}

func (p *Path) appendPolygon(points ...PointF) {
	if len(points) == 0 {
		return
	}
	for i := 0; i < len(points)-1; i++ {
		p.intersecters = append(p.intersecters, &Line{points[i], points[i+1]})
	}
	p.intersecters = append(p.intersecters, &Line{points[len(points)-1], points[0]})
}

func (p *Path) appendRect(x, y, length float64) {
	p0 := PointF{x, y}
	p1 := PointF{x, y + 1}
	p2 := PointF{x + length, y + 1}
	p3 := PointF{x + length, y}
	p.intersecters = append(
		p.intersecters,
		&Line{p0, p1},
		&Line{p1, p2},
		&Line{p2, p3},
		&Line{p3, p0},
	)
}

var count = 0

func update(screen *ebiten.Image) error {
	path := &Path{}
	p0 := PointF{10, 20}
	p1 := PointF{20, 30}
	p2 := PointF{40, 35}
	p3 := PointF{30, 25}
	path.appendPolygon(p0, p1, p2, p3)
	path.appendRect(130, 30, 100)
	path.appendRect(130.5, 40+float64(count)/15.0, 100)

	for j := 0; j < screenHeight; j++ {
		for i := 0; i < screenWidth; i++ {
			c := colorAt(path, i, j)
			p := 4 * (j*screenWidth + i)
			offscreen.Pix[p] = c
			offscreen.Pix[p+1] = c
			offscreen.Pix[p+2] = c
			offscreen.Pix[p+3] = c
		}
	}

	count++
	if ebiten.IsRunningSlowly() {
		return nil
	}

	screen.ReplacePixels(offscreen.Pix)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Fill"); err != nil {
		panic(err)
	}
}
