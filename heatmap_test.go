package heatmap

import (
	"github.com/peterbraden/go-geo"
	"github.com/twpayne/go-polyline"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

func TestGradient(t *testing.T) {

	var keypoints = GradientTable{
		{Hex("#0000FF"), 0.4, 0.0},
		{Hex("#FF0000"), 1, 0.33},
		{Hex("#FFFF00"), 1, 0.66},
		{Hex("#FFFFFF"), 1, 1},
	}

	keypoints.GetInterpolatedColorFor(float64(0.2) / float64(0xFFFF))
}

func TestHeatmap(t *testing.T) {

	var colors = GradientTable{
		{Hex("#0000FF"), 0.4, 0.0},
		{Hex("#FF0000"), 1, 0.33},
		{Hex("#FFFF00"), 1, 0.66},
		{Hex("#FFFFFF"), 1, 1},
	}

	var bbox = geo.BBox{N: 36., S: 34, W: 10., E: 12.}
	var polylines = make([]string, 0)

	for i := 0; i < 10; i++ {
		var points = make([][]float64, 0)
		for p := 0; p < 100; p++ {
			var point = []float64{
				rand.Float64()*(bbox.E-bbox.W) + bbox.W,
				rand.Float64()*(bbox.N-bbox.S) + bbox.S,
			}
			points = append(points, point)
		}
		polylines = append(polylines, string(polyline.EncodeCoords(points)))
	}
	var img = HeatMap(colors, polylines, bbox, 256, 0x4)

	out, err := os.Create("./test.png")
	if err != nil {
		t.Fatal()
	}
	err = png.Encode(out, img)
	if err != nil {
		t.Fatal()
	}

}
