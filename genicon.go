//go:build ignore

// genicon draws Setzer's tray mark and writes icon.png (macOS template) and
// icon.ico (Windows). Run with: go run genicon.go
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

func main() {
	const S = 44
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	black := color.RGBA{0, 0, 0, 255}

	// Three stacked bars — a "set type / lines of text" mark.
	barW, barH, gap := 28, 6, 7
	x0, y := (S-barW)/2, 8
	for i := 0; i < 3; i++ {
		w := barW
		if i == 2 {
			w = barW * 2 / 3 // a shorter last line
		}
		draw.Draw(img, image.Rect(x0, y, x0+w, y+barH), &image.Uniform{black}, image.Point{}, draw.Src)
		y += barH + gap
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		panic(err)
	}
	must(os.WriteFile("icon.png", pngBuf.Bytes(), 0o644))

	// Minimal ICO wrapping the PNG (Windows Vista+ accepts PNG-compressed icons).
	var ico bytes.Buffer
	w16 := func(v uint16) { binary.Write(&ico, binary.LittleEndian, v) }
	w8 := func(v uint8) { binary.Write(&ico, binary.LittleEndian, v) }
	w32 := func(v uint32) { binary.Write(&ico, binary.LittleEndian, v) }
	w16(0)             // reserved
	w16(1)             // type: icon
	w16(1)             // image count
	w8(S)              // width
	w8(S)              // height
	w8(0)              // palette
	w8(0)              // reserved
	w16(1)             // planes
	w16(32)            // bits per pixel
	w32(uint32(pngBuf.Len()))
	w32(22) // offset = 6 (dir) + 16 (entry)
	ico.Write(pngBuf.Bytes())
	must(os.WriteFile("icon.ico", ico.Bytes(), 0o644))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
