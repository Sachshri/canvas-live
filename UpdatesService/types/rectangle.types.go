package types

type RectangleAttributes struct {
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	StrokeWidth  int     `json:"strokeWidth"`
	StrokeColor  string  `json:"strokeColor"`
	FillColorHex string  `json:"fillColor"`
}
