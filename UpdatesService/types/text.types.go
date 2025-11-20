package types

type TextAttributes struct {
	Value     string  `json:"value"`
	Bx        float64 `json:"bx"`
	By        float64 `json:"by"`
	TextColor string  `json:"textColor"`
	FontWidth int     `json:"fontWidth"`
	Font      string  `json:"font"`
	BoxWidth  float64 `json:"width"`
	BoxHeight float64 `json:"height"`
	// Padding            int     `json:"padding"`
	BorderStrokeWidth  int    `json:"strokeWidth"`
	BorderStrokeColor  string `json:"strokeColor"`
	BorderFillColorHex string `json:"fillColor"`
}
