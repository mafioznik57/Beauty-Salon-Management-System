package services

import (
	"github.com/uber/h3-go/v4"
)

func LatLngToH3(lat, lng float64, res int) string {
	c := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, res)
	return c.String()
}

func KRing(center string, k int) ([]string, error) {
	var c h3.Cell
	if err := c.UnmarshalText([]byte(center)); err != nil {
		return nil, err
	}
	cells := c.GridDisk(k)
	out := make([]string, 0, len(cells))
	for _, x := range cells {
		out = append(out, x.String())
	}
	return out, nil
}
