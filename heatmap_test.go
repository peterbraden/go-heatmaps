package heatmap

import "testing"

func TestGradient(t *testing.T) {

	var keypoints = GradientTable{
		{Hex("#0000FF"), 0.4, 0.0},
		{Hex("#FF0000"), 1, 0.33},
		{Hex("#FFFF00"), 1, 0.66},
		{Hex("#FFFFFF"), 1, 1},
	}

	keypoints.GetInterpolatedColorFor(float64(0.2) / float64(0xFFFF))
}
