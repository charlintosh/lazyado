package screens

//go:generate go run ../../../scripts/logo_gen -in ../../../docs/images/logo.png -out splash_logo_gen.go

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	zzzPhases   = 16
	zzzRowShift = 0.8
	zzzGlowMax  = 0.85
)

type pixel struct {
	c  color.RGBA
	ok bool
}

type logoArt struct {
	size int
	rows []string
	glow []string
	zzz  [zzzPhases][]string
}

func buildLogo(size int, mono bool) *logoArt {
	grid := logoGrids[size]
	art := &logoArt{size: size}
	glow := rgba(styles.ColorLogoGlow)
	for r := 0; r+1 < len(grid); r += 2 {
		top, bottom := grid[r], grid[r+1]
		base := renderRow(top, bottom, r, func(ch byte, _ int) pixel { return logoPixel(ch, mono) })
		art.rows = append(art.rows, base)
		art.glow = append(art.glow, renderRow(top, bottom, r, func(ch byte, _ int) pixel {
			if p := logoPixel(ch, mono); p.ok && !mono {
				return pixel{c: glow, ok: true}
			}
			return logoPixel(ch, mono)
		}))
		for phase := range art.zzz {
			if mono || !strings.ContainsRune(top+bottom, 'z') {
				art.zzz[phase] = append(art.zzz[phase], base)
				continue
			}
			art.zzz[phase] = append(art.zzz[phase], renderRow(top, bottom, r, func(ch byte, y int) pixel {
				if ch == 'z' {
					return pixel{c: zzzColor(phase, y), ok: true}
				}
				return logoPixel(ch, mono)
			}))
		}
	}
	return art
}

func (l *logoArt) height() int {
	return len(l.rows)
}

func (l *logoArt) view(reveal float64, phase int) string {
	edge := int(math.Floor(reveal))
	blank := strings.Repeat(" ", l.size)
	lines := make([]string, len(l.rows))
	for r := range l.rows {
		switch {
		case r < edge && phase >= 0:
			lines[r] = l.zzz[phase%zzzPhases][r]
		case r < edge:
			lines[r] = l.rows[r]
		case r == edge:
			lines[r] = l.glow[r]
		default:
			lines[r] = blank
		}
	}
	return strings.Join(lines, "\n")
}

func logoPixel(ch byte, mono bool) pixel {
	var c color.Color
	switch ch {
	case 'a':
		c = styles.ColorLogoBox
	case 'b', 'z':
		c = styles.ColorLogoSloth
	case 'c':
		c = styles.ColorLogoEdge
	case 'w':
		c = styles.ColorLogoPaper
	default:
		return pixel{}
	}
	switch {
	case mono && (ch == 'c' || ch == 'w'):
		return pixel{}
	case mono:
		return pixel{c: rgba(styles.ColorLogoSloth), ok: true}
	}
	return pixel{c: rgba(c), ok: true}
}

func zzzColor(phase, y int) color.RGBA {
	wave := (math.Sin(2*math.Pi*float64(phase)/zzzPhases+float64(y)*zzzRowShift) + 1) / 2
	return mix(rgba(styles.ColorLogoSloth), rgba(styles.ColorLogoGlow), wave*zzzGlowMax)
}

func renderRow(top, bottom string, y int, at func(ch byte, y int) pixel) string {
	var b strings.Builder
	runStart := 0
	for x := 1; x <= len(top); x++ {
		if x < len(top) && at(top[x], y) == at(top[runStart], y) && at(bottom[x], y+1) == at(bottom[runStart], y+1) {
			continue
		}
		b.WriteString(halfBlocks(at(top[runStart], y), at(bottom[runStart], y+1), x-runStart))
		runStart = x
	}
	return b.String()
}

func halfBlocks(top, bottom pixel, n int) string {
	style := lipgloss.NewStyle()
	glyph := " "
	switch {
	case top.ok && bottom.ok && top.c == bottom.c:
		glyph, style = "█", style.Foreground(top.c)
	case top.ok && bottom.ok:
		glyph, style = "▀", style.Foreground(top.c).Background(bottom.c)
	case top.ok:
		glyph, style = "▀", style.Foreground(top.c)
	case bottom.ok:
		glyph, style = "▄", style.Foreground(bottom.c)
	default:
		return strings.Repeat(glyph, n)
	}
	return style.Render(strings.Repeat(glyph, n))
}

func rgba(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func mix(from, to color.RGBA, k float64) color.RGBA {
	lerp := func(a, b uint8) uint8 { return uint8(math.Round(float64(a) + (float64(b)-float64(a))*k)) }
	return color.RGBA{R: lerp(from.R, to.R), G: lerp(from.G, to.G), B: lerp(from.B, to.B), A: 255}
}
