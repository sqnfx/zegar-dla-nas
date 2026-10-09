// Podgląd układu do PNG (tylko do sprawdzenia rysunku poza Windows).
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"
)

var glyphs = map[rune][7]string{
	'I': {"###", ".#.", ".#.", ".#.", ".#.", ".#.", "###"},
	'V': {"#...#", "#...#", "#...#", "#...#", ".#.#.", ".#.#.", "..#.."},
	'X': {"#...#", "#...#", ".#.#.", "..#..", ".#.#.", "#...#", "#...#"},
	'0': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'1': {"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	'2': {".###.", "#...#", "....#", "...#.", "..#..", ".#...", "#####"},
	'3': {".###.", "#...#", "....#", "..##.", "....#", "#...#", ".###."},
	'4': {"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	'5': {"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	'6': {".###.", "#....", "####.", "#...#", "#...#", "#...#", ".###."},
	'7': {"#####", "....#", "...#.", "..#..", "..#..", "..#..", "..#.."},
	'8': {".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	'9': {".###.", "#...#", "#...#", ".####", "....#", "....#", ".###."},
	':': {".", "#", ".", ".", ".", "#", "."},
}

func render(m Metrics, w, h int, t time.Time, up bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for _, op := range Ops(m, w, h, t, up, false) {
		c := color.RGBA{op.C.R, op.C.G, op.C.B, 255}
		switch op.Kind {
		case OpRect:
			for y := op.Y0; y < op.Y1; y++ {
				for x := op.X0; x < op.X1; x++ {
					img.SetRGBA(x, y, c)
				}
			}
		case OpText:
			fh := op.Y1 - op.Y0
			s := fh / 9
			if s < 1 {
				s = 1
			}
			tw := 0
			for _, r := range op.S {
				tw += (len(glyphs[r][0]) + 1) * s
			}
			x := (op.X0+op.X1)/2 - tw/2
			y0 := op.Y0 + (fh-7*s)/2
			for _, r := range op.S {
				g := glyphs[r]
				for gy, row := range g {
					for gx, ch := range row {
						if ch == '#' {
							for dy := 0; dy < s; dy++ {
								for dx := 0; dx < s; dx++ {
									img.SetRGBA(x+gx*s+dx, y0+gy*s+dy, c)
								}
							}
						}
					}
				}
				x += (len(g[0]) + 1) * s
			}
		}
	}
	return img
}

func main() {
	m := NewMetrics(2)
	w, h := m.Width(), 1400
	t := time.Date(2026, 10, 8, 10, 44, 37, 400e6, time.Local)
	out := image.NewRGBA(image.Rect(0, 0, w*2+60, h))
	for i, up := range []bool{false, true} {
		im := render(m, w, h, t, up)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.SetRGBA(x+i*(w+60), y, im.RGBAAt(x, y))
			}
		}
	}
	for y := 0; y < h; y++ {
		for x := w; x < w+60; x++ {
			out.SetRGBA(x, y, color.RGBA{70, 80, 90, 255})
		}
	}
	f, _ := os.Create(os.Args[1])
	defer f.Close()
	png.Encode(f, out)

	if len(os.Args) > 2 {
		previewDial(os.Args[2], t)
	}
}

// previewDial zapisuje okrągły zegar do PNG i sprawdza, że szybkie odświeżanie samej
// tarczy sekund daje ten sam obraz co pełne rysowanie.
func previewDial(path string, t time.Time) {
	for _, k := range []float64{96 / 25.4, 1.5 * 96 / 25.4} {
		d := NewDial(k)
		t0 := time.Now()
		d.Render(t, true)
		full := time.Since(t0)
		t2 := t.Add(350 * time.Millisecond)
		t0 = time.Now()
		d.Render(t2, false)
		part := time.Since(t0)
		ref := NewDial(k)
		ref.Render(t2, true)
		diff, worst := 0, 0
		for i := range d.Pix {
			dv := int(d.Pix[i]) - int(ref.Pix[i])
			if dv < 0 {
				dv = -dv
			}
			if dv > 0 {
				diff++
			}
			if dv > worst {
				worst = dv
			}
		}
		fmt.Printf("k=%.2f  %dx%d  pełne %v  sekundy %v  różnice: %d bajtów, największa %d\n", k, d.W, d.H, full, part, diff, worst)
		if k < 4 {
			variants := [][2]bool{{true, false}, {false, true}, {true, true}}
			strip := image.NewRGBA(image.Rect(0, 0, d.W*3, d.H))
			for vi, v := range variants {
				dv := NewDial(k)
				dv.HideMin, dv.HideSec = v[0], v[1]
				dv.Render(t, true)
				dv.Render(t2, false)
				for i := 0; i < dv.W*dv.H; i++ {
					o := strip.PixOffset(vi*d.W+i%dv.W, i/dv.W)
					strip.Pix[o], strip.Pix[o+1], strip.Pix[o+2], strip.Pix[o+3] = dv.Pix[i*4+2], dv.Pix[i*4+1], dv.Pix[i*4], 255
				}
			}
			f, _ := os.Create(path + ".warianty.png")
			png.Encode(f, strip)
			f.Close()
		}
		if k > 4 {
			img := image.NewRGBA(image.Rect(0, 0, d.W, d.H))
			for i := 0; i < d.W*d.H; i++ {
				img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = d.Pix[i*4+2], d.Pix[i*4+1], d.Pix[i*4], 255
			}
			f, _ := os.Create(path)
			png.Encode(f, img)
			f.Close()
		}
	}
}
