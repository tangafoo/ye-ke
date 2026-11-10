package scan

// CellPrecision 6 ≈ 1.2 km × 0.6 km at the equator: about the size of a 500 m
// scan, so one sweep covers one scan's worth of ground.
const CellPrecision = 6

const geohashBase32 = "0123456789bcdefghjkmnpqrstuvwxyz"

// Cell is the geohash of a point: the key the sweep ledger is stored under.
func Cell(lat, lng float64) string {
	return geohash(lat, lng, CellPrecision)
}

// CellAt is Cell at another precision (7 ≈ 150 m, for per-block sweeps).
func CellAt(lat, lng float64, precision int) string {
	return geohash(lat, lng, precision)
}

// geohash interleaves longitude and latitude bisections, 5 bits per character.
func geohash(lat, lng float64, precision int) string {
	latLo, latHi := -90.0, 90.0
	lngLo, lngHi := -180.0, 180.0

	out := make([]byte, 0, precision)
	bit, ch := 0, 0
	evenBit := true // even bits encode longitude

	for len(out) < precision {
		if evenBit {
			mid := (lngLo + lngHi) / 2
			if lng >= mid {
				ch = ch<<1 | 1
				lngLo = mid
			} else {
				ch <<= 1
				lngHi = mid
			}
		} else {
			mid := (latLo + latHi) / 2
			if lat >= mid {
				ch = ch<<1 | 1
				latLo = mid
			} else {
				ch <<= 1
				latHi = mid
			}
		}
		evenBit = !evenBit

		if bit++; bit == 5 {
			out = append(out, geohashBase32[ch])
			bit, ch = 0, 0
		}
	}
	return string(out)
}
