package demo

import (
	"math"
	"math/rand"
	"time"
)

// Frame replaces every pixel with f(x, y) under one lock.
func (m *Matrix) Frame(f func(x, y int) (r, g, b byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for y := 0; y < MatrixH; y++ {
		for x := 0; x < MatrixW; x++ {
			i := (y*MatrixW + x) * 3
			m.rgb[i], m.rgb[i+1], m.rgb[i+2] = f(x, y)
		}
	}
}

// Fire animates a rising flame for frames frames at fps. cooling (0-100) is how quickly
// the heat dies on its way up. step is called after every frame and ends the run by
// returning false.
func (m *Matrix) Fire(frames, fps int, cooling float64, step func() bool) {
	heat := make([][]float64, MatrixH)
	for y := range heat {
		heat[y] = make([]float64, MatrixW)
	}
	cool := cooling / 600
	at := func(y, x int) float64 {
		if y >= MatrixH {
			y = MatrixH - 1
		}
		return heat[y][(x+MatrixW)%MatrixW]
	}
	tick := time.NewTicker(time.Second / time.Duration(fps))
	defer tick.Stop()
	for f := 0; f < frames; f++ {
		for x := 0; x < MatrixW; x++ {
			heat[MatrixH-1][x] = 0.6 + 0.4*rand.Float64()
		}
		for y := 0; y < MatrixH-1; y++ {
			for x := 0; x < MatrixW; x++ {
				v := (at(y+1, x-1) + at(y+1, x) + at(y+1, x+1) + at(y+2, x)) / 4
				if v -= rand.Float64() * cool; v < 0 {
					v = 0
				}
				heat[y][x] = v
			}
		}
		m.Frame(func(x, y int) (byte, byte, byte) {
			h := heat[y][x] * 3
			return byte(255 * clamp01(h)), byte(255 * clamp01(h-1)), byte(255 * clamp01(h-2))
		})
		if !step() {
			return
		}
		<-tick.C
	}
}

// Plasma animates the classic sine-sum colour plasma for frames frames at fps.
func (m *Matrix) Plasma(frames, fps int, step func() bool) {
	tick := time.NewTicker(time.Second / time.Duration(fps))
	defer tick.Stop()
	for f := 0; f < frames; f++ {
		t := float64(f) / float64(fps)
		m.Frame(func(x, y int) (byte, byte, byte) {
			fx, fy := float64(x), float64(y)
			v := math.Sin(fx/8+t) + math.Sin(fy/6+1.3*t) + math.Sin((fx+fy)/10+0.7*t) +
				math.Sin(math.Hypot(fx-MatrixW/2, fy-MatrixH/2)/6-t)
			return hue((v + 4) / 8)
		})
		if !step() {
			return
		}
		<-tick.C
	}
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// hue maps h in [0,1) around the colour wheel at full saturation and value.
func hue(h float64) (byte, byte, byte) {
	h = (h - math.Floor(h)) * 6
	x := byte(255 * (1 - math.Abs(math.Mod(h, 2)-1)))
	switch int(h) {
	case 0:
		return 255, x, 0
	case 1:
		return x, 255, 0
	case 2:
		return 0, 255, x
	case 3:
		return 0, x, 255
	case 4:
		return x, 0, 255
	}
	return 255, 0, x
}
