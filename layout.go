// Układ i rysowanie paska: czysta logika, bez wywołań systemowych.
// Zwraca listę prostokątów i napisów, którą okno Windows tylko maluje.
package main

import (
	"fmt"
	"math"
	"time"
)

type RGB struct{ R, G, B uint8 }

const (
	OpRect = iota
	OpText
)

const (
	FontRoman = iota
	FontMinute
	FontClock
	FontSecond
	fontCount
)

type Op struct {
	Kind           int
	X0, Y0, X1, Y1 int
	C              RGB
	S              string
	Font           int
}

// Metrics: wymiary w pikselach dla danej skali (rozmiar × DPI).
// Szerokości kolumn trzymają proporcję tarcz 9 : 6 : 3.
type Metrics struct {
	S                            float64
	Pad, Gap, WH, WM, WS, Footer int
	T                            int // grubość kreski
	FontH                        [fontCount]int
}

func px(v float64) int { return int(math.Round(v)) }

func NewMetrics(s float64) Metrics {
	m := Metrics{S: s}
	m.Pad, m.Gap = px(3*s), px(2*s)
	m.WH, m.WM, m.WS = px(30*s), px(20*s), px(10*s)
	m.Footer = px(44 * s)
	m.T = px(s)
	if m.T < 1 {
		m.T = 1
	}
	m.FontH = [fontCount]int{px(13 * s), px(11 * s), px(17 * s), px(12 * s)}
	return m
}

func (m Metrics) Width() int { return 2*m.Pad + m.WH + m.WM + m.WS + 2*m.Gap }

func (m Metrics) colX() (xh, xm, xs int) {
	xh = m.Pad
	xm = xh + m.WH + m.Gap
	xs = xm + m.WM + m.Gap
	return
}

// SecondsRect: obszar kolumny sekund, jedyny odświeżany wiele razy na sekundę.
func (m Metrics) SecondsRect(h int) (x0, y0, x1, y1 int) {
	_, _, xs := m.colX()
	return xs, 0, xs + m.WS, h - m.Footer
}

// ---- kolory -------------------------------------------------------------

func oklch(l, c, hdeg float64) RGB {
	h := hdeg * math.Pi / 180
	a, b := c*math.Cos(h), c*math.Sin(h)
	l_ := l + 0.3963377774*a + 0.2158037573*b
	m_ := l - 0.1055613458*a - 0.0638541728*b
	s_ := l - 0.0894841775*a - 1.2914855480*b
	L, M, S := l_*l_*l_, m_*m_*m_, s_*s_*s_
	lin := [3]float64{
		4.0767416621*L - 3.3077115913*M + 0.2309699292*S,
		-1.2684380046*L + 2.6097574011*M - 0.3413193965*S,
		-0.0041960863*L - 0.7034186147*M + 1.7076147010*S,
	}
	var out [3]uint8
	for i, v := range lin {
		v = math.Max(0, math.Min(1, v))
		if v <= 0.0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		out[i] = uint8(math.Round(v * 255))
	}
	return RGB{out[0], out[1], out[2]}
}

func blend(a, b RGB, t float64) RGB { // t = udział koloru a
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x)*t + float64(y)*(1-t))) }
	return RGB{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B)}
}

var (
	colBg       = RGB{0x11, 0x15, 0x16}
	colTrackH   = RGB{0x2D, 0x3A, 0x38}
	colTrackM   = RGB{0xD8, 0xD5, 0xC6}
	colTrackS   = RGB{0x1D, 0x25, 0x24}
	colInkLight = RGB{0xF3, 0xF1, 0xE6}
	colInkDark  = RGB{0x15, 0x1C, 0x1B}
	colMarker   = RGB{0xFF, 0xFD, 0xF5}
	colEdge     = RGB{0x0B, 0x0E, 0x0E}
	colText     = RGB{0xE6, 0xE8, 0xE2}
	colMuted    = RGB{0x95, 0xA1, 0x9D}

	palH, palM, palS [12]RGB // te same barwy co na tarczach zegara
)

func init() {
	for k := 0; k < 12; k++ {
		hue := float64((k*30 + 25) % 360)
		palH[k] = oklch(0.56, 0.13, hue)
		palM[k] = oklch(0.79, 0.11, hue)
		palS[k] = oklch(0.63, 0.15, hue)
	}
}

var roman = [12]string{"XII", "I", "II", "III", "IIII", "V", "VI", "VII", "VIII", "IX", "X", "XI"}
var fives = [12]string{"00", "05", "10", "15", "20", "25", "30", "35", "40", "45", "50", "55"}

// ---- rysowanie ----------------------------------------------------------

type colSpec struct {
	x, w    int
	v       float64 // postęp w wycinkach, 0–12
	pal     *[12]RGB
	track   RGB
	ink     RGB
	ticks   int                 // liczba kresek na całej kolumnie
	tickLen func(i int) float64 // długość kreski jako część szerokości; 1 = pełna
	labels  *[12]string
	font    int
}

func column(ops *[]Op, m Metrics, top, h int, up bool, c colSpec) {
	my := func(t float64) int {
		p := px(t * float64(h))
		if up {
			return top + h - p
		}
		return top + p
	}
	rect := func(x0, ya, x1, yb int, col RGB) {
		if ya > yb {
			ya, yb = yb, ya
		}
		if yb > ya && x1 > x0 {
			*ops = append(*ops, Op{Kind: OpRect, X0: x0, Y0: ya, X1: x1, Y1: yb, C: col})
		}
	}
	// wycinki: część załadowana w kolorze, reszta w kolorze toru
	for k := 0; k < 12; k++ {
		f := math.Max(0, math.Min(1, c.v-float64(k)))
		a, mid, b := my(float64(k)/12), my((float64(k)+f)/12), my(float64(k+1)/12)
		rect(c.x, a, c.x+c.w, mid, c.pal[k])
		rect(c.x, mid, c.x+c.w, b, c.track)
	}
	// podziałka
	per := c.ticks / 12
	for i := 0; i < c.ticks; i++ {
		frac := c.tickLen(i)
		pos := float64(i) * 12 / float64(c.ticks) // w wycinkach
		bg := c.track
		if pos < c.v {
			bg = c.pal[i/per]
		}
		col := c.ink
		if frac < 1 {
			col = blend(c.ink, bg, 0.6)
		} else {
			col = blend(c.ink, bg, 0.85)
		}
		y := my(float64(i) / float64(c.ticks))
		if up {
			rect(c.x, y-m.T, c.x+px(frac*float64(c.w)), y, col)
		} else {
			rect(c.x, y, c.x+px(frac*float64(c.w)), y+m.T, col)
		}
	}
	// opisy tuż za kreską początku wycinka
	if c.labels != nil {
		fh := m.FontH[c.font]
		off := m.T + px(2*m.S)
		for k := 0; k < 12; k++ {
			y := my(float64(k) / 12)
			y0 := y + off
			if up {
				y0 = y - off - fh
			}
			*ops = append(*ops, Op{Kind: OpText, X0: c.x, Y0: y0, X1: c.x + c.w, Y1: y0 + fh, C: c.ink, S: c.labels[k], Font: c.font})
		}
	}
	// znacznik "teraz": biała linia z ciemnym obrysem, czytelna na każdym kolorze
	y := my(c.v / 12)
	if y < top+m.T {
		y = top + m.T
	}
	if y > top+h-m.T {
		y = top + h - m.T
	}
	rect(c.x, y-2*m.T, c.x+c.w, y+2*m.T, colEdge)
	rect(c.x, y-m.T, c.x+c.w, y+m.T, colMarker)
}

// Ops buduje listę rysunkową. secondsOnly = tylko kolumna sekund (szybkie odświeżanie).
func Ops(m Metrics, w, h int, t time.Time, up, secondsOnly bool) []Op {
	ops := make([]Op, 0, 256)
	top := m.Pad
	ch := h - m.Pad - m.Footer
	if ch < 24 {
		ch = 24
	}
	xh, xm, xs := m.colX()

	sec := float64(t.Second()) + float64(t.Nanosecond())/1e9
	min := float64(t.Minute()) + sec/60
	hr := float64(t.Hour()%12) + min/60

	secCol := colSpec{x: xs, w: m.WS, v: sec / 5, pal: &palS, track: colTrackS, ink: colInkLight, ticks: 60,
		tickLen: func(i int) float64 {
			if i%5 == 0 {
				return 1
			}
			return 0.4
		}}

	if secondsOnly {
		x0, y0, x1, y1 := m.SecondsRect(h)
		ops = append(ops, Op{Kind: OpRect, X0: x0, Y0: y0, X1: x1, Y1: y1, C: colBg})
		column(&ops, m, top, ch, up, secCol)
		return ops
	}

	ops = append(ops, Op{Kind: OpRect, X0: 0, Y0: 0, X1: w, Y1: h, C: colBg})
	column(&ops, m, top, ch, up, colSpec{x: xh, w: m.WH, v: hr, pal: &palH, track: colTrackH, ink: colInkLight, ticks: 48,
		tickLen: func(i int) float64 { // kwadranse
			switch {
			case i%4 == 0:
				return 1
			case i%2 == 0:
				return 0.4
			}
			return 0.22
		}, labels: &roman, font: FontRoman})
	column(&ops, m, top, ch, up, colSpec{x: xm, w: m.WM, v: min / 5, pal: &palM, track: colTrackM, ink: colInkDark, ticks: 60,
		tickLen: func(i int) float64 {
			if i%5 == 0 {
				return 1
			}
			return 0.3
		}, labels: &fives, font: FontMinute})
	column(&ops, m, top, ch, up, secCol)

	// odczyt kontrolny pod kolumnami
	fy := h - m.Footer + px(5*m.S)
	ops = append(ops, Op{Kind: OpText, X0: 0, Y0: fy, X1: w, Y1: fy + m.FontH[FontClock], C: colText,
		S: fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute()), Font: FontClock})
	fy += m.FontH[FontClock] + px(4*m.S)
	ops = append(ops, Op{Kind: OpText, X0: 0, Y0: fy, X1: w, Y1: fy + m.FontH[FontSecond], C: colMuted,
		S: fmt.Sprintf("%02d", t.Second()), Font: FontSecond})
	return ops
}
