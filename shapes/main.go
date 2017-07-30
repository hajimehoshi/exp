package main

import (
	"image"
	"math"
	"sort"

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

func (l *Line) Start() PointF {
	return l.P0
}

func (l *Line) End() PointF {
	return l.P1
}

type Arc struct {
	X          float64
	Y          float64
	Radius     float64
	StartAngle float64
	EndAngle   float64
	//AntiClockwise bool
}

func normalizeAngle(a float64, start, end float64) float64 {
	for a < start {
		a += 2 * math.Pi
	}
	for end <= a {
		a -= 2 * math.Pi
	}
	return a
}

func (a *Arc) Intersect(y float64) ([]float64, bool) {
	if a.Radius <= 0 {
		return nil, true
	}
	if a.Y-a.Radius > y {
		return nil, true
	}
	if a.Y+a.Radius < y {
		return nil, true
	}
	t := math.Pow(a.Radius, 2) - math.Pow(y-a.Y, 2)
	x0 := -math.Sqrt(t) + a.X
	x1 := math.Sqrt(t) + a.X
	if x0 == x1 {
		return nil, false
	}
	as := a.StartAngle
	ae := a.EndAngle
	a0 := normalizeAngle(math.Asin((y - a.Y) / a.Radius), 0, 2 * math.Pi)
	a1 := normalizeAngle(-a0 + math.Pi, 0, 2 * math.Pi)
	if a0 < math.Pi/2 || 3*math.Pi/2 <= a0 {
		tmp := a0
		a0 = a1
		a1 = tmp
	}
	a0 = normalizeAngle(a0, as, ae)
	a1 = normalizeAngle(a1, as, ae)
	r := []float64{}
	if a0 == as || a0 == ae || a1 == as || a1 == ae {
		return nil, false
	}
	if as < a0 && a0 < ae {
		r = append(r, x0)
	}
	if as < a1 && a1 < ae {
		r = append(r, x1)
	}
	return r, true
}

func (a *Arc) Start() PointF {
	return PointF{
		X: a.Radius*math.Cos(a.StartAngle) + a.X,
		Y: a.Radius*math.Sin(a.StartAngle) + a.Y,
	}
}

func (a *Arc) End() PointF {
	return PointF{
		X: a.Radius*math.Cos(a.EndAngle) + a.X,
		Y: a.Radius*math.Sin(a.EndAngle) + a.Y,
	}
}

type Intersecter interface {
	Intersect(y float64) ([]float64, bool)
	Start() PointF
	End() PointF
}

type Path struct {
	intersecters []Intersecter
	pos          PointF
}

func (p Path) Intersect(y float64) ([]float64, bool) {
	is := p.intersecters
	if len(is) == 0 {
		return nil, true
	}
	r := []float64{}
	for _, i := range is {
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
		sort.Float64s(intersections)
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

func (p *Path) moveTo(x, y float64) {
	p.pos = PointF{x, y}
}

func (p *Path) lineTo(x, y float64) {
	p.intersecters = append(p.intersecters, &Line{p.pos, PointF{x, y}})
	p.pos = PointF{x, y}
}

func (p *Path) rect(x, y, length float64) {
	p0 := PointF{x, y}
	p1 := PointF{x, y + 1}
	p2 := PointF{x + length, y + 1}
	p3 := PointF{x + length, y}
	p.intersecters = append(
		p.intersecters,
		&Line{p0, p1},
		&Line{p1, p2},
		&Line{p2, p3},
		&Line{p3, p0})
	p.pos = p0
}

func (p *Path) arc(x, y, radius, startAngle, endAngle float64) {
	arc := &Arc{
		X:          x,
		Y:          y,
		Radius:     radius,
		StartAngle: startAngle,
		EndAngle:   endAngle,
	}
	if p.pos != arc.Start() {
		line := &Line{p.pos, arc.Start()}
		p.intersecters = append(p.intersecters, line)
	}
	p.intersecters = append(p.intersecters, arc)
	p.pos = arc.End()
}

var count = 0

func update(screen *ebiten.Image) error {
	path := &Path{}
	p0 := PointF{10, 20}
	p1 := PointF{20, 30}
	p2 := PointF{40, 35}
	p3 := PointF{30, 25}
	path.moveTo(p0.X, p0.Y)
	for _, p := range []PointF{p0, p1, p2, p3, p0} {
		path.lineTo(p.X, p.Y)
	}
	path.rect(130, 30, 100)
	path.rect(130.5, 40+float64(count)/15.0, 100)

	path.moveTo(50, 50)
	path.arc(50, 50, 30, math.Pi / 4, 2 * math.Pi + math.Pi / 8)
	path.lineTo(50, 50)

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
