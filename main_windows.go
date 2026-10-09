// Zegar Pasek: pionowy pasek czasu przy krawędzi ekranu Windows.
// Trzy kolumny (godziny, minuty, sekundy) ładują się kolorami.
// Prawy przycisk myszy na pasku otwiera menu.
package main

import (
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// ---- Win32 --------------------------------------------------------------

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassExW       = user32.NewProc("RegisterClassExW")
	pCreateWindowExW        = user32.NewProc("CreateWindowExW")
	pDefWindowProcW         = user32.NewProc("DefWindowProcW")
	pDestroyWindow          = user32.NewProc("DestroyWindow")
	pGetMessageW            = user32.NewProc("GetMessageW")
	pTranslateMessage       = user32.NewProc("TranslateMessage")
	pDispatchMessageW       = user32.NewProc("DispatchMessageW")
	pPostQuitMessage        = user32.NewProc("PostQuitMessage")
	pPostMessageW           = user32.NewProc("PostMessageW")
	pSetTimer               = user32.NewProc("SetTimer")
	pKillTimer              = user32.NewProc("KillTimer")
	pInvalidateRect         = user32.NewProc("InvalidateRect")
	pBeginPaint             = user32.NewProc("BeginPaint")
	pEndPaint               = user32.NewProc("EndPaint")
	pGetClientRect          = user32.NewProc("GetClientRect")
	pSetWindowPos           = user32.NewProc("SetWindowPos")
	pShowWindow             = user32.NewProc("ShowWindow")
	pGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	pSystemParametersInfoW  = user32.NewProc("SystemParametersInfoW")
	pCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	pAppendMenuW            = user32.NewProc("AppendMenuW")
	pTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	pDestroyMenu            = user32.NewProc("DestroyMenu")
	pGetCursorPos           = user32.NewProc("GetCursorPos")
	pSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	pLoadCursorW            = user32.NewProc("LoadCursorW")
	pSetProcessDPIAware     = user32.NewProc("SetProcessDPIAware")
	pGetDC                  = user32.NewProc("GetDC")
	pReleaseDC              = user32.NewProc("ReleaseDC")
	pFillRect               = user32.NewProc("FillRect")
	pDrawTextW              = user32.NewProc("DrawTextW")
	pRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")
	pTrackMouseEvent        = user32.NewProc("TrackMouseEvent")
	pGetWindowRect          = user32.NewProc("GetWindowRect")
	pSetWindowRgn           = user32.NewProc("SetWindowRgn")

	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pCreateFontW            = gdi32.NewProc("CreateFontW")
	pGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")
	pGetStockObject         = gdi32.NewProc("GetStockObject")
	pCreateEllipticRgn      = gdi32.NewProc("CreateEllipticRgn")
	pSetDIBitsToDevice      = gdi32.NewProc("SetDIBitsToDevice")

	pSHAppBarMessage = shell32.NewProc("SHAppBarMessage")

	pRegCreateKeyExW  = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	pRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	pRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	pRegCloseKey      = advapi32.NewProc("RegCloseKey")

	pCreateMutexW     = kernel32.NewProc("CreateMutexW")
	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

const (
	wsPopup         = 0x80000000
	wsExToolWindow  = 0x00000080
	wsExTopmost     = 0x00000008
	wsExNoActivate  = 0x08000000
	swHide          = 0
	wmMouseActivate = 0x0021
	maNoActivate    = 3
	wmMouseMove     = 0x0200
	wmMouseHover    = 0x02A1
	wmMouseLeave    = 0x02A3
	tmeHover        = 0x0001
	tmeLeave        = 0x0002
	hoverDelayMs    = 200
	swShowNoActive  = 4
	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	hwndBottom      = uintptr(1)
	wmNull          = 0x0000
	wmDestroy       = 0x0002
	wmActivate      = 0x0006
	wmPaint         = 0x000F
	wmEraseBkgnd    = 0x0014
	wmSettingChange = 0x001A
	wmWinPosChanged = 0x0047
	wmDisplayChange = 0x007E
	wmTimer         = 0x0113
	wmRButtonUp     = 0x0205
	wmAppBar        = 0x8000 + 1 // WM_APP + 1: powiadomienia paska

	smCxScreen     = 0
	smCyScreen     = 1
	spiGetWorkArea = 0x0030
	spiSetWorkArea = 0x002F

	mfString    = 0x0000
	mfChecked   = 0x0008
	mfSeparator = 0x0800
	tpmRightBtn = 0x0002
	tpmNoNotify = 0x0080
	tpmRetCmd   = 0x0100

	srcCopy      = 0x00CC0020
	bkTransp     = 1
	dtCenter     = 0x0001
	dtVCenter    = 0x0004
	dtSingleLine = 0x0020
	dtNoClip     = 0x0100
	dtNoPrefix   = 0x0800
	logPixelsX   = 88
	idcArrow     = 32512
	clearType    = 5

	abmNew           = 0
	abmRemove        = 1
	abmQueryPos      = 2
	abmSetPos        = 3
	abmActivate      = 6
	abmWinPosChanged = 9
	abnPosChanged    = 1
	abnFullScreenApp = 2
	abeLeft          = 0
	abeRight         = 2

	hkeyCurrentUser = uintptr(0x80000001)
	keyReadWrite    = 0x20019 | 0x20006
	regSZ           = 1
	regDWORD        = 4

	errAlreadyExists = 183
)

var hwndTopmost = ^uintptr(0) // (HWND)-1

type rect struct{ Left, Top, Right, Bottom int32 }
type point struct{ X, Y int32 }

type msgT struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type paintStruct struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     rect
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type appBarData struct {
	CbSize           uint32
	HWnd             uintptr
	UCallbackMessage uint32
	UEdge            uint32
	Rc               rect
	LParam           uintptr
}

type trackMouseEvent struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   uintptr
	DwHoverTime uint32
}

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// ---- stan ---------------------------------------------------------------

const (
	regPath  = `Software\ZegarPasek`
	runPath  = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue = "ZegarPasek"
)

var sizeFactor = [3]float64{0.8, 1.0, 1.3}

var cfg = struct {
	left    bool // krawędź: false = prawa
	up      bool // kierunek ładowania: true = od dołu, jak nalewana woda
	size    int  // 0 mały, 1 średni, 2 duży
	reserve bool // rezerwuj miejsce (okna nie wchodzą pod pasek)
	hover   bool // po najechaniu pokaż okrągły zegar w lupie
	dialMin bool // w lupie: pokaż tarczę minut
	dialSec bool // w lupie: pokaż tarczę sekund
}{up: true, size: 1, reserve: true, hover: true, dialMin: true, dialSec: true}

var (
	hwnd           uintptr
	metrics        Metrics
	fonts          [fontCount]uintptr
	brushes        = map[RGB]uintptr{}
	memDC, memBmp  uintptr
	memOld         uintptr
	bufW, bufH     int
	dirtyAll       = true
	lastSecond     = -1
	registered     bool // zarejestrowany jako pasek pulpitu (AppBar)
	fullscreenApp  bool
	inLayout       bool
	msgTaskbarMade uint32

	hDial       uintptr // okrągłe okienko-lupa z zegarem trzech tarcz
	dial        *Dial
	dialVisible bool
	tracking    bool // czy pasek czeka na zdarzenia najechania / zjechania myszy
)

// ---- ustawienia w rejestrze --------------------------------------------

func regOpen(path string) (uintptr, bool) {
	var h uintptr
	r, _, _ := pRegCreateKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(u16(path))), 0, 0, 0,
		keyReadWrite, 0, uintptr(unsafe.Pointer(&h)), 0)
	return h, r == 0
}

func regGetDword(h uintptr, name string, def uint32) uint32 {
	var v, typ uint32
	size := uint32(4)
	r, _, _ := pRegQueryValueExW.Call(h, uintptr(unsafe.Pointer(u16(name))), 0,
		uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&v)), uintptr(unsafe.Pointer(&size)))
	if r != 0 || typ != regDWORD {
		return def
	}
	return v
}

func regSetDword(h uintptr, name string, v uint32) {
	pRegSetValueExW.Call(h, uintptr(unsafe.Pointer(u16(name))), 0, regDWORD, uintptr(unsafe.Pointer(&v)), 4)
}

func b2u(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func loadConfig() {
	h, ok := regOpen(regPath)
	if !ok {
		return
	}
	defer pRegCloseKey.Call(h)
	cfg.left = regGetDword(h, "Left", 0) != 0
	cfg.up = regGetDword(h, "FillUp", 1) != 0
	cfg.reserve = regGetDword(h, "Reserve", 1) != 0
	cfg.hover = regGetDword(h, "Hover", 1) != 0
	cfg.dialMin = regGetDword(h, "DialMinutes", 1) != 0
	cfg.dialSec = regGetDword(h, "DialSeconds", 1) != 0
	if s := regGetDword(h, "Size", 1); s <= 2 {
		cfg.size = int(s)
	}
}

func saveConfig() {
	h, ok := regOpen(regPath)
	if !ok {
		return
	}
	defer pRegCloseKey.Call(h)
	regSetDword(h, "Left", b2u(cfg.left))
	regSetDword(h, "FillUp", b2u(cfg.up))
	regSetDword(h, "Reserve", b2u(cfg.reserve))
	regSetDword(h, "Hover", b2u(cfg.hover))
	regSetDword(h, "DialMinutes", b2u(cfg.dialMin))
	regSetDword(h, "DialSeconds", b2u(cfg.dialSec))
	regSetDword(h, "Size", uint32(cfg.size))
}

func autostartEnabled() bool {
	h, ok := regOpen(runPath)
	if !ok {
		return false
	}
	defer pRegCloseKey.Call(h)
	r, _, _ := pRegQueryValueExW.Call(h, uintptr(unsafe.Pointer(u16(runValue))), 0, 0, 0, 0)
	return r == 0
}

func setAutostart(on bool) {
	h, ok := regOpen(runPath)
	if !ok {
		return
	}
	defer pRegCloseKey.Call(h)
	if !on {
		pRegDeleteValueW.Call(h, uintptr(unsafe.Pointer(u16(runValue))))
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	val, err := syscall.UTF16FromString(`"` + exe + `"`)
	if err != nil {
		return
	}
	pRegSetValueExW.Call(h, uintptr(unsafe.Pointer(u16(runValue))), 0, regSZ,
		uintptr(unsafe.Pointer(&val[0])), uintptr(len(val)*2))
}

// ---- położenie okna -----------------------------------------------------

func systemDPI() int {
	dc, _, _ := pGetDC.Call(0)
	if dc == 0 {
		return 96
	}
	d, _, _ := pGetDeviceCaps.Call(dc, logPixelsX)
	pReleaseDC.Call(0, dc)
	if d < 48 {
		return 96
	}
	return int(d)
}

func recreateFonts() {
	if memDC != 0 {
		// nie usuwamy kroju, który jest jeszcze wybrany w buforze rysowania
		stock, _, _ := pGetStockObject.Call(13) // SYSTEM_FONT
		pSelectObject.Call(memDC, stock)
	}
	for i, f := range fonts {
		if f != 0 {
			pDeleteObject.Call(f)
			fonts[i] = 0
		}
	}
	face := u16("Bahnschrift Condensed") // wąski krój systemowy; gdy go brak, Windows dobierze zastępczy
	for i := 0; i < fontCount; i++ {
		h := int32(-metrics.FontH[i])
		fonts[i], _, _ = pCreateFontW.Call(uintptr(h), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, clearType, 0,
			uintptr(unsafe.Pointer(face)))
	}
}

func appBar(msg uint32, abd *appBarData) uintptr {
	abd.CbSize = uint32(unsafe.Sizeof(*abd))
	abd.HWnd = hwnd
	r, _, _ := pSHAppBarMessage.Call(uintptr(msg), uintptr(unsafe.Pointer(abd)))
	return r
}

func narrow(rc *rect, w int32) {
	if cfg.left {
		rc.Right = rc.Left + w
	} else {
		rc.Left = rc.Right - w
	}
}

// applyLayout ustawia pasek przy wybranej krawędzi; w trybie rezerwacji zgłasza go
// systemowi jako pasek pulpitu, żeby zmaksymalizowane okna kończyły się obok niego.
func applyLayout() {
	if inLayout || hwnd == 0 {
		return
	}
	inLayout = true
	defer func() { inLayout = false }()
	hideDial()

	metrics = NewMetrics(sizeFactor[cfg.size] * float64(systemDPI()) / 96)
	recreateFonts()
	w := int32(metrics.Width())

	if cfg.reserve && !registered {
		abd := appBarData{UCallbackMessage: wmAppBar}
		registered = appBar(abmNew, &abd) != 0
	} else if !cfg.reserve && registered {
		var abd appBarData
		appBar(abmRemove, &abd)
		registered = false
	}

	var rc rect
	if registered {
		cx, _, _ := pGetSystemMetrics.Call(smCxScreen)
		cy, _, _ := pGetSystemMetrics.Call(smCyScreen)
		abd := appBarData{UEdge: abeRight, Rc: rect{0, 0, int32(cx), int32(cy)}}
		if cfg.left {
			abd.UEdge = abeLeft
		}
		narrow(&abd.Rc, w)
		appBar(abmQueryPos, &abd) // system odsuwa pasek od paska zadań i innych pasków
		narrow(&abd.Rc, w)
		appBar(abmSetPos, &abd)
		rc = abd.Rc
	} else {
		pSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&rc)), 0)
		narrow(&rc, w)
	}

	flags := uintptr(swpNoActivate)
	after := hwndTopmost
	if fullscreenApp {
		flags |= swpNoZOrder
		after = 0
	}
	pSetWindowPos.Call(hwnd, after, uintptr(rc.Left), uintptr(rc.Top),
		uintptr(rc.Right-rc.Left), uintptr(rc.Bottom-rc.Top), flags)
	dirtyAll = true
	pInvalidateRect.Call(hwnd, 0, 0)
}

// ---- rysowanie ----------------------------------------------------------

func brush(c RGB) uintptr {
	if b, ok := brushes[c]; ok {
		return b
	}
	b, _, _ := pCreateSolidBrush.Call(uintptr(c.R) | uintptr(c.G)<<8 | uintptr(c.B)<<16)
	brushes[c] = b
	return b
}

func ensureBuffer(hdc uintptr, w, h int) {
	if memDC != 0 && w == bufW && h == bufH {
		return
	}
	if memDC != 0 {
		pSelectObject.Call(memDC, memOld)
		pDeleteObject.Call(memBmp)
		pDeleteDC.Call(memDC)
	}
	memDC, _, _ = pCreateCompatibleDC.Call(hdc)
	memBmp, _, _ = pCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	memOld, _, _ = pSelectObject.Call(memDC, memBmp)
	pSetBkMode.Call(memDC, bkTransp)
	bufW, bufH = w, h
	dirtyAll = true
}

func execOps(ops []Op) {
	for i := range ops {
		op := &ops[i]
		r := rect{int32(op.X0), int32(op.Y0), int32(op.X1), int32(op.Y1)}
		switch op.Kind {
		case OpRect:
			pFillRect.Call(memDC, uintptr(unsafe.Pointer(&r)), brush(op.C))
		case OpText:
			pSelectObject.Call(memDC, fonts[op.Font])
			pSetTextColor.Call(memDC, uintptr(op.C.R)|uintptr(op.C.G)<<8|uintptr(op.C.B)<<16)
			pDrawTextW.Call(memDC, uintptr(unsafe.Pointer(u16(op.S))), ^uintptr(0),
				uintptr(unsafe.Pointer(&r)), dtCenter|dtVCenter|dtSingleLine|dtNoClip|dtNoPrefix)
		}
	}
}

func paint() {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	var rc rect
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	w, h := int(rc.Right), int(rc.Bottom)
	if w > 0 && h > 0 {
		ensureBuffer(hdc, w, h)
		execOps(Ops(metrics, w, h, time.Now(), cfg.up, !dirtyAll))
		dirtyAll = false
		pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), memDC, 0, 0, srcCopy)
	}
	pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

func onTimer() {
	now := time.Now()
	newSecond := now.Second() != lastSecond
	if dialVisible && dial != nil {
		dial.Render(now, newSecond) // pełne rysowanie raz na sekundę, poza tym tylko tarcza sekund
		blitDial(0)
	}
	if newSecond {
		lastSecond = now.Second()
		dirtyAll = true
		pInvalidateRect.Call(hwnd, 0, 0)
		return
	}
	// między sekundami odświeża się tylko kolumna sekund
	x0, y0, x1, y1 := metrics.SecondsRect(bufH)
	r := rect{int32(x0), int32(y0), int32(x1), int32(y1)}
	pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
}

// ---- lupa z okrągłym zegarem -------------------------------------------

func blitDial(hdc uintptr) {
	if dial == nil || hDial == 0 {
		return
	}
	own := hdc == 0
	if own {
		hdc, _, _ = pGetDC.Call(hDial)
		if hdc == 0 {
			return
		}
	}
	bmi := bitmapInfoHeader{BiWidth: int32(dial.W), BiHeight: -int32(dial.H), BiPlanes: 1, BiBitCount: 32}
	bmi.BiSize = uint32(unsafe.Sizeof(bmi))
	pSetDIBitsToDevice.Call(hdc, 0, 0, uintptr(dial.W), uintptr(dial.H), 0, 0, 0, uintptr(dial.H),
		uintptr(unsafe.Pointer(&dial.Pix[0])), uintptr(unsafe.Pointer(&bmi)), 0)
	if own {
		pReleaseDC.Call(hDial, hdc)
	}
}

// moveDial stawia lupę obok paska, na wysokości kursora.
func moveDial() {
	if dial == nil {
		return
	}
	var bar rect
	var pt point
	pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&bar)))
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	gap := int32(2 * metrics.Pad)
	x := bar.Left - int32(dial.W) - gap
	if cfg.left {
		x = bar.Right + gap
	}
	y := pt.Y - int32(dial.H)/2
	if y > bar.Bottom-int32(dial.H) {
		y = bar.Bottom - int32(dial.H)
	}
	if y < bar.Top {
		y = bar.Top
	}
	pSetWindowPos.Call(hDial, hwndTopmost, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoActivate)
}

func showDial() {
	if !cfg.hover || hDial == 0 || dialVisible {
		return
	}
	k := float64(systemDPI()) / 25.4 // piksele na milimetr: tarcze mają 9, 6 i 3 cm
	if dial == nil || dial.K != k {
		dial = NewDial(k)
		pSetWindowPos.Call(hDial, 0, 0, 0, uintptr(dial.W), uintptr(dial.H), swpNoMove|swpNoZOrder|swpNoActivate)
		rgn, _, _ := pCreateEllipticRgn.Call(0, 0, uintptr(dial.W+1), uintptr(dial.H+1))
		pSetWindowRgn.Call(hDial, rgn, 0) // okno okrągłe; region przejmuje system
	}
	dial.HideMin, dial.HideSec = !cfg.dialMin, !cfg.dialSec
	dial.Render(time.Now(), true)
	moveDial()
	dialVisible = true
	pShowWindow.Call(hDial, swShowNoActive)
	blitDial(0)
}

func hideDial() {
	if dialVisible {
		dialVisible = false
		pShowWindow.Call(hDial, swHide)
	}
}

func dialProc(h, msg, wp, lp uintptr) uintptr {
	switch uint32(msg) {
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			blitDial(hdc)
			pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		}
		return 0
	case wmEraseBkgnd:
		return 1
	case wmMouseActivate:
		return maNoActivate
	}
	r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
	return r
}

// ---- menu ---------------------------------------------------------------

const (
	cmdRight = iota + 1
	cmdLeft
	cmdDown
	cmdUp
	cmdSmall
	cmdMedium
	cmdLarge
	cmdReserve
	cmdAutostart
	cmdHover
	cmdDialMin
	cmdDialSec
	cmdClose
)

func showMenu() {
	hideDial()
	menu, _, _ := pCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	item := func(id int, text string, checked bool) {
		f := uintptr(mfString)
		if checked {
			f |= mfChecked
		}
		pAppendMenuW.Call(menu, f, uintptr(id), uintptr(unsafe.Pointer(u16(text))))
	}
	sep := func() { pAppendMenuW.Call(menu, mfSeparator, 0, 0) }

	item(cmdRight, "Prawa krawędź ekranu", !cfg.left)
	item(cmdLeft, "Lewa krawędź ekranu", cfg.left)
	sep()
	item(cmdUp, "Napełniaj od dołu (jak woda)", cfg.up)
	item(cmdDown, "Napełniaj od góry", !cfg.up)
	sep()
	item(cmdSmall, "Rozmiar: mały", cfg.size == 0)
	item(cmdMedium, "Rozmiar: średni", cfg.size == 1)
	item(cmdLarge, "Rozmiar: duży", cfg.size == 2)
	sep()
	item(cmdHover, "Po najechaniu pokaż okrągły zegar w lupie", cfg.hover)
	item(cmdDialMin, "W lupie: tarcza minut", cfg.dialMin)
	item(cmdDialSec, "W lupie: tarcza sekund", cfg.dialSec)
	sep()
	item(cmdReserve, "Rezerwuj miejsce (okna nie wchodzą pod pasek)", cfg.reserve)
	item(cmdAutostart, "Uruchamiaj razem z Windows", autostartEnabled())
	sep()
	item(cmdClose, "Zamknij zegar", false)

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWindow.Call(hwnd)
	cmd, _, _ := pTrackPopupMenu.Call(menu, tpmRightBtn|tpmNoNotify|tpmRetCmd,
		uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	pPostMessageW.Call(hwnd, wmNull, 0, 0)
	pDestroyMenu.Call(menu)

	switch cmd {
	case cmdRight:
		cfg.left = false
	case cmdLeft:
		cfg.left = true
	case cmdDown:
		cfg.up = false
	case cmdUp:
		cfg.up = true
	case cmdSmall:
		cfg.size = 0
	case cmdMedium:
		cfg.size = 1
	case cmdLarge:
		cfg.size = 2
	case cmdReserve:
		cfg.reserve = !cfg.reserve
	case cmdHover:
		cfg.hover = !cfg.hover
	case cmdDialMin:
		cfg.dialMin = !cfg.dialMin
	case cmdDialSec:
		cfg.dialSec = !cfg.dialSec
	case cmdAutostart:
		setAutostart(!autostartEnabled())
		return
	case cmdClose:
		pDestroyWindow.Call(hwnd)
		return
	default:
		return
	}
	saveConfig()
	applyLayout()
}

// ---- okno ---------------------------------------------------------------

func wndProc(h, msg, wp, lp uintptr) uintptr {
	switch uint32(msg) {
	case wmPaint:
		paint()
		return 0
	case wmEraseBkgnd:
		return 1
	case wmTimer:
		onTimer()
		return 0
	case wmRButtonUp:
		showMenu()
		return 0
	case wmMouseMove:
		if cfg.hover {
			if !tracking {
				tme := trackMouseEvent{DwFlags: tmeHover | tmeLeave, HwndTrack: h, DwHoverTime: hoverDelayMs}
				tme.CbSize = uint32(unsafe.Sizeof(tme))
				r, _, _ := pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
				tracking = r != 0
			}
			if dialVisible {
				moveDial() // lupa idzie za kursorem wzdłuż paska
			}
		}
		return 0
	case wmMouseHover:
		showDial()
		return 0
	case wmMouseLeave:
		tracking = false
		hideDial()
		return 0
	case wmAppBar:
		switch wp {
		case abnPosChanged:
			applyLayout()
		case abnFullScreenApp:
			// gra lub film na pełnym ekranie: pasek schodzi pod spód, potem wraca na wierzch
			fullscreenApp = lp != 0
			after := hwndTopmost
			if fullscreenApp {
				after = hwndBottom
			}
			pSetWindowPos.Call(h, after, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
		}
		return 0
	case wmActivate:
		if registered {
			var abd appBarData
			appBar(abmActivate, &abd)
		}
	case wmWinPosChanged:
		if registered {
			var abd appBarData
			appBar(abmWinPosChanged, &abd)
		}
	case wmDisplayChange:
		applyLayout()
	case wmSettingChange:
		if !registered && wp == spiSetWorkArea {
			applyLayout()
		}
	case wmDestroy:
		pKillTimer.Call(h, 1)
		if registered {
			var abd appBarData
			appBar(abmRemove, &abd)
			registered = false
		}
		pPostQuitMessage.Call(0)
		return 0
	default:
		if msgTaskbarMade != 0 && uint32(msg) == msgTaskbarMade {
			// Eksplorator uruchomił się ponownie: trzeba zgłosić pasek od nowa
			registered = false
			applyLayout()
			return 0
		}
	}
	r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
	return r
}

func main() {
	runtime.LockOSThread()

	// jedna kopia naraz
	_, _, e := pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(u16("ZegarPasek.SingleInstance"))))
	if errno, ok := e.(syscall.Errno); ok && errno == errAlreadyExists {
		return
	}

	pSetProcessDPIAware.Call()
	loadConfig()

	r, _, _ := pRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(u16("TaskbarCreated"))))
	msgTaskbarMade = uint32(r)

	inst, _, _ := pGetModuleHandleW.Call(0)
	cursor, _, _ := pLoadCursorW.Call(0, idcArrow)
	className := u16("ZegarPasekWindow")
	wc := wndClassEx{
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     inst,
		HCursor:       cursor,
		LpszClassName: className,
	}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	if a, _, _ := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); a == 0 {
		return
	}

	hwnd, _, _ = pCreateWindowExW.Call(wsExToolWindow|wsExTopmost,
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(u16("Zegar Pasek"))),
		wsPopup, 0, 0, 0, 0, 0, 0, inst, 0)
	if hwnd == 0 {
		return
	}

	// lupa: osobne okno bez ramki, własność paska (znika razem z nim)
	dialClass := u16("ZegarPasekLupa")
	wd := wndClassEx{
		LpfnWndProc:   syscall.NewCallback(dialProc),
		HInstance:     inst,
		HCursor:       cursor,
		LpszClassName: dialClass,
	}
	wd.CbSize = uint32(unsafe.Sizeof(wd))
	if a, _, _ := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wd))); a != 0 {
		hDial, _, _ = pCreateWindowExW.Call(wsExToolWindow|wsExTopmost|wsExNoActivate,
			uintptr(unsafe.Pointer(dialClass)), uintptr(unsafe.Pointer(u16("Zegar trzech tarcz"))),
			wsPopup, 0, 0, 0, 0, hwnd, 0, inst, 0)
	}

	applyLayout()
	pShowWindow.Call(hwnd, swShowNoActive)
	pSetTimer.Call(hwnd, 1, 50, 0)

	var m msgT
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
