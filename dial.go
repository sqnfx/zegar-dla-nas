// Okrągły zegar trzech tarcz rysowany programowo (z wygładzaniem krawędzi) do bufora pikseli.
// Te same wymiary co na stronie: 1 jednostka = 1 mm, tarcze 9, 6 i 3 cm.
package main

import (
	"math"
	"time"
)

const (
	// Zegar siedzi w okrągłej "lupie": kwadrat 110 mm, środek tarcz w środku kwadratu.
	dialOriginX, dialOriginY = 55.0, 55.0
	dialWmm, dialHmm         = 110.0, 110.0
	rHour, rMin, rSec        = 45.0, 30.0, 15.0
	rLensIn, rLensOut        = 53.0, 55.0 // oprawka lupy
)

type Dial struct {
	W, H   int
	K      float64 // piksele na milimetr
	cx, cy float64
	Pix    []byte // BGRA, wiersze od góry
	static []byte // tarcze godzin i minut, odświeżane raz na sekundę
	clip   [4]int // x0, y0, x1, y1: obszar, w którym wolno rysować

	HideMin, HideSec bool // ukryte tarcze środkowe: minutowa i sekundowa
}

func NewDial(k float64) *Dial {
	d := &Dial{K: k, W: int(math.Ceil(dialWmm * k)), H: int(math.Ceil(dialHmm * k))}
	d.cx, d.cy = dialOriginX*k, dialOriginY*k
	d.Pix = make([]byte, d.W*d.H*4)
	d.static = make([]byte, d.W*d.H*4)
	d.clip = [4]int{0, 0, d.W, d.H}
	return d
}

func (d *Dial) px(x, y int, c RGB, a float64) {
	if a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	p := d.Pix[(y*d.W+x)*4:]
	p[0] = uint8(float64(p[0]) + (float64(c.B)-float64(p[0]))*a + 0.5)
	p[1] = uint8(float64(p[1]) + (float64(c.G)-float64(p[1]))*a + 0.5)
	p[2] = uint8(float64(p[2]) + (float64(c.R)-float64(p[2]))*a + 0.5)
	p[3] = 255
}

// box zamienia prostokąt w mm (względem środka tarcz) na piksele, przycięte do obszaru rysowania.
func (d *Dial) box(x0, y0, x1, y1 float64) (ax, ay, bx, by int) {
	ax = int(math.Floor(d.cx + x0*d.K - 1))
	ay = int(math.Floor(d.cy + y0*d.K - 1))
	bx = int(math.Ceil(d.cx + x1*d.K + 1))
	by = int(math.Ceil(d.cy + y1*d.K + 1))
	ax, ay = max(ax, d.clip[0]), max(ay, d.clip[1])
	bx, by = min(bx, d.clip[2]), min(by, d.clip[3])
	return
}

// disc: tarcza o promieniu r (mm) obrócona tak, że pod znacznikiem u góry jest wartość v (0–12 wycinków).
func (d *Dial) disc(r, v float64, pal *[12]RGB, track RGB) {
	at := func(u float64) RGB {
		u = math.Mod(u+12, 12)
		if u < v {
			return pal[int(u)]
		}
		return track
	}
	rp := r * d.K
	ax, ay, bx, by := d.box(-r, -r, r, r)
	for y := ay; y < by; y++ {
		dy := float64(y) + 0.5 - d.cy
		for x := ax; x < bx; x++ {
			dx := float64(x) + 0.5 - d.cx
			dist := math.Hypot(dx, dy)
			cov := rp - dist + 0.5
			if cov <= 0 {
				continue
			}
			u := math.Mod(math.Atan2(dx, -dy)*6/math.Pi+v+24, 12) // kąt w wycinkach po 30°
			w := 1.9099 / math.Max(dist, 1)                       // szerokość piksela w wycinkach
			b := math.Round(u)
			if math.Abs(u-v) < math.Abs(u-b) {
				b = v
			}
			var col RGB
			if math.Abs(u-b) < w/2 { // piksel na granicy dwóch kolorów: mieszamy proporcjonalnie
				col = blend(at(b-1e-6), at(b+1e-6), (b-(u-w/2))/w)
			} else {
				col = at(u)
			}
			d.px(x, y, col, cov)
		}
	}
}

// shadow: miękki cień pod tarczą o promieniu r, żeby tarcze wyglądały na ułożone jedna na drugiej.
func (d *Dial) shadow(r float64) {
	const blur, off = 1.4, 0.6
	ax, ay, bx, by := d.box(-r-blur, -r-blur+off, r+blur, r+blur+off)
	for y := ay; y < by; y++ {
		dy := (float64(y)+0.5-d.cy)/d.K - off
		for x := ax; x < bx; x++ {
			dx := (float64(x) + 0.5 - d.cx) / d.K
			t := (math.Hypot(dx, dy) - (r - blur)) / (2 * blur)
			if t >= 1 {
				continue
			}
			if t < 0 {
				t = 0
			}
			d.px(x, y, RGB{}, 0.45*(1-t*t*(3-2*t)))
		}
	}
}

// line: odcinek o zaokrąglonych końcach, współrzędne i grubość w mm.
func (d *Dial) line(x0, y0, x1, y1, width float64, c RGB, alpha float64) {
	hw := width * d.K / 2
	px0, py0 := d.cx+x0*d.K, d.cy+y0*d.K
	px1, py1 := d.cx+x1*d.K, d.cy+y1*d.K
	ax, ay, bx, by := d.box(math.Min(x0, x1)-width, math.Min(y0, y1)-width, math.Max(x0, x1)+width, math.Max(y0, y1)+width)
	vx, vy := px1-px0, py1-py0
	l2 := vx*vx + vy*vy
	for y := ay; y < by; y++ {
		for x := ax; x < bx; x++ {
			qx, qy := float64(x)+0.5-px0, float64(y)+0.5-py0
			t := 0.0
			if l2 > 0 {
				t = math.Max(0, math.Min(1, (qx*vx+qy*vy)/l2))
			}
			cov := hw - math.Hypot(qx-t*vx, qy-t*vy) + 0.5
			if cov > 0 {
				d.px(x, y, c, math.Min(cov, 1)*alpha)
			}
		}
	}
}

// tri: wypełniony trójkąt, opcjonalnie powiększony o grow pikseli (na obrys).
func (d *Dial) tri(p [3][2]float64, c RGB, grow float64) {
	var q [3][2]float64
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for i := range p {
		q[i] = [2]float64{d.cx + p[i][0]*d.K, d.cy + p[i][1]*d.K}
		minX, maxX = math.Min(minX, p[i][0]), math.Max(maxX, p[i][0])
		minY, maxY = math.Min(minY, p[i][1]), math.Max(maxY, p[i][1])
	}
	sign := 1.0
	if (q[1][0]-q[0][0])*(q[2][1]-q[0][1])-(q[1][1]-q[0][1])*(q[2][0]-q[0][0]) < 0 {
		sign = -1
	}
	g := grow / d.K
	ax, ay, bx, by := d.box(minX-g, minY-g, maxX+g, maxY+g)
	for y := ay; y < by; y++ {
		for x := ax; x < bx; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			m := math.Inf(1)
			for i := 0; i < 3; i++ {
				a, b := q[i], q[(i+1)%3]
				ex, ey := b[0]-a[0], b[1]-a[1]
				m = math.Min(m, sign*(ex*(fy-a[1])-ey*(fx-a[0]))/math.Hypot(ex, ey))
			}
			d.px(x, y, c, m+grow+0.5)
		}
	}
}

// lens: oprawka lupy (metaliczny pierścień), cień pod nią i odblask na szkle.
func (d *Dial) lens() {
	light, dark := RGB{0xF2, 0xF5, 0xF3}, RGB{0x5E, 0x6A, 0x67}
	ax, ay, bx, by := d.box(-rLensOut, -rLensOut, rLensOut, rLensOut)
	for y := ay; y < by; y++ {
		dy := float64(y) + 0.5 - d.cy
		for x := ax; x < bx; x++ {
			dx := float64(x) + 0.5 - d.cx
			dist := math.Hypot(dx, dy)
			mm := dist / d.K
			if mm < 46.5 {
				continue
			}
			// odblask: jasny łuk w lewej górnej części szkła, między tarczą a oprawką
			if mm > 47.2 && mm < 50.8 {
				ang := math.Atan2(dx, -dy) * 180 / math.Pi // -180..180, zero u góry
				ta := 1 - math.Abs(ang+48)/26              // środek łuku 48° w lewo od góry
				tr := 1 - math.Abs(mm-49)/1.8
				if ta > 0 && tr > 0 {
					d.px(x, y, RGB{255, 255, 255}, 0.5*ta*ta*(3-2*ta)*tr)
				}
			}
			// cień oprawki na szkle
			if in := (rLensIn - mm) / 1.3; in >= 0 && in < 1 {
				d.px(x, y, RGB{}, 0.5*(1-in)*(1-in))
			}
			// oprawka: jaśniejsza od lewej góry, ciemniejsza od prawego dołu
			cov := math.Min(dist-rLensIn*d.K+0.5, 1)
			if cov > 0 {
				t := 0.5 - (dx*0.6+dy*0.8)/(2*dist)
				d.px(x, y, blend(light, dark, t), cov)
			}
		}
	}
}

// ---- znaki kreślone odcinkami -------------------------------------------

type glyph struct {
	w       float64 // szerokość w jednostkach wysokości znaku
	strokes [][][2]float64
}

func ellipse(cx, cy, rx, ry float64) [][2]float64 {
	const n = 18
	pts := make([][2]float64, n+1)
	for i := 0; i <= n; i++ {
		a := 2 * math.Pi * float64(i) / n
		pts[i] = [2]float64{cx + rx*math.Cos(a), cy + ry*math.Sin(a)}
	}
	return pts
}

var glyphSet = func() map[rune]glyph {
	six := [][2]float64{{0.9, 0.12}, {0.7, 0.02}, {0.45, 0}, {0.2, 0.1}, {0.05, 0.3}, {0, 0.6}, {0.05, 0.82}, {0.25, 0.97},
		{0.5, 1}, {0.78, 0.95}, {0.97, 0.8}, {1, 0.66}, {0.92, 0.5}, {0.7, 0.4}, {0.45, 0.38}, {0.2, 0.45}, {0.02, 0.6}}
	nine := make([][2]float64, len(six))
	for i, p := range six {
		nine[i] = [2]float64{1 - p[0], 1 - p[1]}
	}
	const dw = 0.56
	return map[rune]glyph{
		'I': {0.0, [][][2]float64{{{0, 0}, {0, 1}}}},
		'V': {0.62, [][][2]float64{{{0, 0}, {0.5, 1}, {1, 0}}}},
		'X': {0.62, [][][2]float64{{{0, 0}, {1, 1}}, {{1, 0}, {0, 1}}}},
		'0': {dw, [][][2]float64{ellipse(0.5, 0.5, 0.5, 0.5)}},
		'1': {dw, [][][2]float64{{{0.15, 0.24}, {0.58, 0}, {0.58, 1}}}},
		'2': {dw, [][][2]float64{{{0.03, 0.27}, {0.12, 0.1}, {0.32, 0}, {0.68, 0}, {0.9, 0.1}, {0.98, 0.27}, {0.9, 0.45}, {0, 1}, {1, 1}}}},
		'3': {dw, [][][2]float64{{{0.03, 0.18}, {0.2, 0.03}, {0.5, 0}, {0.8, 0.04}, {0.95, 0.2}, {0.9, 0.38}, {0.6, 0.48}, {0.35, 0.48}},
			{{0.6, 0.48}, {0.92, 0.58}, {1, 0.75}, {0.9, 0.93}, {0.55, 1}, {0.2, 0.96}, {0, 0.8}}}},
		'4': {dw, [][][2]float64{{{0.75, 1}, {0.75, 0}, {0, 0.7}, {1, 0.7}}}},
		'5': {dw, [][][2]float64{{{0.92, 0}, {0.12, 0}, {0.05, 0.45}, {0.3, 0.38}, {0.6, 0.38}, {0.9, 0.5}, {1, 0.68}, {0.9, 0.9}, {0.6, 1}, {0.25, 0.97}, {0, 0.82}}}},
		'6': {dw, [][][2]float64{six}},
		'7': {dw, [][][2]float64{{{0, 0}, {1, 0}, {0.4, 1}}}},
		'8': {dw, [][][2]float64{ellipse(0.5, 0.25, 0.42, 0.25), ellipse(0.5, 0.73, 0.5, 0.27)}},
		'9': {dw, [][][2]float64{nine}},
	}
}()

// label: napis wyśrodkowany na promieniu r pod kątem angle (stopnie, zgodnie z ruchem wskazówek od góry),
// obrócony tak, żeby u góry stał pionowo. capH = wysokość znaku w mm.
func (d *Dial) label(s string, angle, r, capH, gap, stroke float64, c RGB) {
	total := 0.0
	for i, ch := range s {
		total += glyphSet[ch].w
		if i > 0 {
			total += gap
		}
	}
	sin, cos := math.Sincos(angle * math.Pi / 180)
	tr := func(lx, ly float64) (float64, float64) { // obrót wokół środka tarcz
		ly -= r
		return lx*cos - ly*sin, lx*sin + ly*cos
	}
	cur := -total / 2
	for _, ch := range s {
		g := glyphSet[ch]
		for _, st := range g.strokes {
			for i := 0; i+1 < len(st); i++ {
				x0, y0 := tr((cur+st[i][0]*g.w)*capH, (st[i][1]-0.5)*capH)
				x1, y1 := tr((cur+st[i+1][0]*g.w)*capH, (st[i+1][1]-0.5)*capH)
				d.line(x0, y0, x1, y1, stroke, c, 1)
			}
		}
		cur += g.w + gap
	}
}

// ticks: n kresek od krawędzi tarczy do środka; spec zwraca długość, grubość (mm) i krycie.
func (d *Dial) ticks(n int, r, rot float64, c RGB, spec func(i int) (length, width, alpha float64)) {
	for i := 0; i < n; i++ {
		length, width, alpha := spec(i)
		sin, cos := math.Sincos((float64(i)*360/float64(n) - rot) * math.Pi / 180)
		ro, ri := r-width/2-0.1, r-length
		d.line(sin*ro, -cos*ro, sin*ri, -cos*ri, width, c, alpha)
	}
}

func (d *Dial) drawSeconds(sec float64) {
	d.disc(rSec, sec/5, &palS, colTrackS)
	d.ticks(60, rSec, sec*6, colInkLight, func(i int) (float64, float64, float64) {
		switch {
		case i%10 == 0:
			return 2.6, 0.35, 1
		case i%5 == 0:
			return 2, 0.28, 1
		}
		return 1.3, 0.22, 0.6
	})
	for s := 0; s < 60; s += 10 {
		d.label(fives[s/5], float64(s)*6-sec*6, 9.2, 2.5, 0.2, 0.36, colInkLight)
	}
	d.line(0, 0, 0, 0, 2.2, colInkLight, 1) // oś
}

// Render rysuje zegar dla chwili t. full=false odświeża tylko tarczę sekund
// (pozostałe są brane z kopii zrobionej przy ostatnim pełnym rysowaniu).
// Ukryta tarcza odsłania to, co leży pod nią: większą tarczę aż do środka.
func (d *Dial) Render(t time.Time, full bool) {
	if !full && d.HideSec {
		return // bez tarczy sekund nic nie zmienia się częściej niż raz na sekundę
	}
	sec := float64(t.Second()) + float64(t.Nanosecond())/1e9
	mins := float64(t.Minute()) + sec/60
	hr := float64(t.Hour()%12) + mins/60

	// obszar tarczy sekund z zapasem na wygładzoną krawędź
	sx0 := max(0, int(d.cx-rSec*d.K-2))
	sy0 := max(0, int(d.cy-rSec*d.K-2))
	sx1 := min(d.W, int(d.cx+rSec*d.K+3))
	sy1 := min(d.H, int(d.cy+rSec*d.K+3))

	if full {
		d.clip = [4]int{0, 0, d.W, d.H}
		for i := 0; i < len(d.Pix); i += 4 {
			d.Pix[i], d.Pix[i+1], d.Pix[i+2], d.Pix[i+3] = colBg.B, colBg.G, colBg.R, 255
		}
		d.shadow(rHour)
		d.disc(rHour, hr, &palH, colTrackH)
		d.ticks(48, rHour, hr*30, colInkLight, func(i int) (float64, float64, float64) { // kwadranse
			switch {
			case i%4 == 0:
				return 4, 0.55, 1
			case i%2 == 0:
				return 2.8, 0.4, 1
			}
			return 1.8, 0.3, 0.6
		})
		for k := 0; k < 12; k++ {
			d.label(roman[k], float64(k)*30-hr*30, 36.6, 4.5, 0.36, 0.62, colInkLight)
		}

		// nieruchome znaczniki odczytu na jednej osi nad dwunastką; tylko dla widocznych tarcz
		marks := [][3][2]float64{{{-2.2, -50.8}, {2.2, -50.8}, {0, -45.3}}}
		if !d.HideMin {
			d.shadow(rMin)
			d.disc(rMin, mins/5, &palM, colTrackM)
			d.ticks(60, rMin, mins*6, colInkDark, func(i int) (float64, float64, float64) {
				if i%5 == 0 {
					return 3, 0.45, 1
				}
				return 1.8, 0.25, 0.6
			})
			for k := 0; k < 12; k++ {
				d.label(fives[k], float64(k)*30-mins*6, 23.3, 2.8, 0.2, 0.42, colInkDark)
			}
			marks = append(marks, [3][2]float64{{-1.5, -33.3}, {1.5, -33.3}, {0, -30.3}})
		}
		if !d.HideSec {
			d.shadow(rSec)
			marks = append(marks, [3][2]float64{{-1.2, -17.7}, {1.2, -17.7}, {0, -15.3}})
		} else {
			// sama oś, z ciemnym obrysem, żeby była widoczna na każdym kolorze
			d.line(0, 0, 0, 0, 3.0, colEdge, 1)
			d.line(0, 0, 0, 0, 2.2, colInkLight, 1)
		}
		for _, p := range marks {
			d.tri(p, colEdge, 0.4*d.K)
			d.tri(p, colMarker, 0)
		}
		d.lens()
		copy(d.static, d.Pix)
		if d.HideSec {
			return
		}
	} else {
		for y := sy0; y < sy1; y++ {
			copy(d.Pix[(y*d.W+sx0)*4:(y*d.W+sx1)*4], d.static[(y*d.W+sx0)*4:(y*d.W+sx1)*4])
		}
	}
	d.clip = [4]int{sx0, sy0, sx1, sy1}
	d.drawSeconds(sec)
	d.clip = [4]int{0, 0, d.W, d.H}
}
