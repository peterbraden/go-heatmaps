package heatmap

import (
	"encoding/binary"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/peterbraden/go-geo"
	"github.com/twpayne/go-polyline"
	"image"
	"image/color"
	"math"
)

func toCanvasPt(height, width int, bbox geo.BBox, lat, long float64) (x, y int) {
	x = int(math.Round(((long-bbox.W)/(bbox.E-bbox.W))*float64(width) + 0.5))
	y = int(math.Round(float64(height) - ((lat-bbox.S)/(bbox.N-bbox.S))*float64(height) + 0.5))
	return
}

func Hex(s string) colorful.Color {
	c, err := colorful.Hex(s)
	if err != nil {
		panic("MustParseHex: " + err.Error())
	}
	return c
}

// This table contains the "keypoints" of the colorgradient you want to generate.
// The position of each keypoint has to live in the range [0,1]
type GradientTable []struct {
	Col   colorful.Color
	Alpha float64
	Pos   float64
}

func floatToRGBA(opacity float32) color.RGBA {
	bits := math.Float32bits(opacity)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)
	return color.RGBA{bytes[0], bytes[1], bytes[2], bytes[3]}
}
func rgbaToFloat(c color.RGBA) float32 {
	bits := binary.LittleEndian.Uint32([]uint8{c.R, c.G, c.B, c.A})
	float := math.Float32frombits(bits)
	return float
}

func drawPixel(x, y int, opacity float32, img *image.RGBA) {
	c := img.At(x, y).(color.RGBA)
	cval := rgbaToFloat(c)
	cval += opacity
	c = floatToRGBA(cval)
	img.Set(x, y, c)
}

func pixel(x, y int, opacity float32, img *image.RGBA) {
	drawPixel(x, y, opacity, img)
	drawPixel(x+1, y, opacity/4., img)
	drawPixel(x-1, y, opacity/4., img)
	drawPixel(x, y+1, opacity/4., img)
	drawPixel(x, y-1, opacity/4., img)
}

func blackline(x0, y0, x1, y1 int, img *image.RGBA, opacity uint8) {
	xiolinWu(x0, y0, x1, y1, img, opacity)
}

func fpart(x float64) float64 {
	return x - math.Floor(x)
}

func rfpart(x float64) float64 {
	return 1.0 - fpart(x)
}

func xiolinWu(x0, y0, x1, y1 int, img *image.RGBA, opacity uint8) {
	steep := Abs(x0-x1) < Abs(y0-y1)
	if steep {
		x0, y0 = y0, x0
		x1, y1 = y1, x1
	}
	if x0 > x1 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
	}

	dx := x1 - x0
	dy := y1 - y0
	fopacity := float64(opacity)

	gradient := float64(dy) / float64(dx)
	if dx == 0 {
		gradient = 1.0
	}

	xpxl1 := x0
	xpxl2 := x1
	intery := float64(y0) // Intersect Y

	// main loop
	if steep {
		for x := xpxl1; x <= xpxl2-1; x++ {
			pixel(int(intery), x, float32(rfpart(intery)*fopacity), img)
			pixel(int(intery)+1, x, float32(fpart(intery)*fopacity), img)
			intery = intery + gradient
		}
	} else {
		for x := xpxl1; x <= xpxl2-1; x++ {
			pixel(x, int(intery), float32(rfpart(intery)*fopacity), img)
			pixel(x, int(intery)+1, float32(fpart(intery)*fopacity), img)
			intery = intery + gradient
		}
	}
}

func bresenham(x0, y0, x1, y1 int, img *image.RGBA, opacity float32) {
	steep := Abs(x0-x1) < Abs(y0-y1)
	if steep {
		x0, y0 = y0, x0
		x1, y1 = y1, x1
	}
	if x0 > x1 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
	}

	dx := x1 - x0
	dy := y1 - y0

	derr := Abs(dy) * 2
	err2 := 0
	y := y0

	for x := x0; x <= x1; x++ {
		if steep {
			pixel(y, x, opacity, img)
		} else {
			pixel(x, y, opacity, img)
		}
		err2 += derr
		if err2 > dx {
			if y1 > y0 {
				y = y + 1
			} else {
				y = y - 1
			}
			err2 = err2 - dx*2
		}
	}
}

// This is the meat of the gradient computation. It returns a HCL-blend between
// the two colors around `t`.
// Note: It relies heavily on the fact that the gradient keypoints are sorted.
func (self GradientTable) GetInterpolatedColorFor(t float64) color.Color {
	for i := 0; i < len(self)-1; i++ {
		c1 := self[i]
		c2 := self[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			// We are in between c1 and c2. Go blend them!
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			alpha := ((1 - t) * c1.Alpha) + (t * c2.Alpha)
			r, g, b, _ := c1.Col.BlendHcl(c2.Col, t).Clamped().RGBA()
			// GOLANG IS SHIT
			return color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(alpha * 255)}
		}
	}

	// Nothing found? Means we're at (or past) the last gradient keypoint.
	return self[len(self)-1].Col
}
func inImage(x, y, size int) bool {
	return x > 0 && x < size && y > 0 && y < size
}

func HeatMap(colors GradientTable, polylines []string, bbox geo.BBox, size int, opacity uint8) *image.RGBA {
	dest := image.NewRGBA(image.Rect(0, 0, size, size))
	if len(polylines) == 0 {
		return dest // Early bailout for empty image
	}

	for i := range polylines {
		ps, _, _ := polyline.DecodeCoords([]byte(polylines[i]))
		if len(ps) > 1 && len(ps[0]) > 1 {
			cp0x, cp0y := toCanvasPt(size, size, bbox, ps[0][0], ps[0][1])
			prevcpx, prevcpy := cp0x, cp0y
			for i := range ps {
				cpx, cpy := toCanvasPt(size, size, bbox, ps[i][0], ps[i][1])
				// Don't render consecutive points that are the same - big perf difference
				if cpx != prevcpx || cpy != prevcpy {
					// Don't render if it's not in the tile - big perf difference when zoomed in
					if inImage(prevcpx, prevcpy, size) || inImage(cpx, cpy, size) {
						blackline(prevcpx, prevcpy, cpx, cpy, dest, opacity)
					}
					prevcpx = cpx
					prevcpy = cpy
				}
			}
		}
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			color := rgbaToFloat(dest.At(x, y).(color.RGBA))
			if color > 0 {
				c := colors.GetInterpolatedColorFor(float64(color / 255.0))
				dest.Set(x, y, c)
			}
		}
	}
	return dest
}
