package types

type CircleAttributes struct {
	Cx           float64 `json:"cx"`
	Cy           float64 `json:"cy"`
	Radius       float64 `json:"radius"`
	StrokeWidth  int     `json:"strokeWidth"`
	StrokeColor  string  `json:"strokeColor"`
	FillColorHex string  `json:"fillColor"`
}
