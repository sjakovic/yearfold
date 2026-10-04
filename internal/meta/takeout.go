package meta

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// Takeout is the useful part of a Google Takeout JSON sidecar.
type Takeout struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	TakenAt     int64    `json:"takenAt,omitempty"`
	Lat         float64  `json:"lat,omitempty"`
	Lon         float64  `json:"lon,omitempty"`
	People      []string `json:"people,omitempty"`
}

type takeoutRaw struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	PhotoTakenTime struct {
		Timestamp string `json:"timestamp"`
	} `json:"photoTakenTime"`
	GeoData     geo `json:"geoData"`
	GeoDataExif geo `json:"geoDataExif"`
	People      []struct {
		Name string `json:"name"`
	} `json:"people"`
}

type geo struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ParseTakeout decodes a sidecar. ok is false when data is not a photo sidecar.
func ParseTakeout(data []byte) (t Takeout, ok bool) {
	var raw takeoutRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return t, false
	}
	ts, _ := strconv.ParseInt(raw.PhotoTakenTime.Timestamp, 10, 64)
	if ts == 0 && raw.Title == "" {
		return t, false
	}
	t.Title, t.Description, t.TakenAt = raw.Title, raw.Description, ts
	g := raw.GeoData
	if g.Latitude == 0 && g.Longitude == 0 {
		g = raw.GeoDataExif
	}
	t.Lat, t.Lon = g.Latitude, g.Longitude
	for _, p := range raw.People {
		if p.Name != "" {
			t.People = append(t.People, p.Name)
		}
	}
	return t, true
}

const supplemental = ".supplemental-metadata"

var counterRe = regexp.MustCompile(`\((\d+)\)$`)

// MatchSidecar finds the media file a Takeout JSON belongs to among the names
// in the same directory. It handles the Takeout quirks: the
// ".supplemental-metadata" suffix (possibly truncated), names cut at the
// length limit, and the "(1)" duplicate counter that sits before ".json" on
// the sidecar but before the extension on the media file.
func MatchSidecar(jsonName string, mediaNames []string) (string, bool) {
	lower := strings.ToLower(jsonName)
	if !strings.HasSuffix(lower, ".json") {
		return "", false
	}
	base := jsonName[:len(jsonName)-len(".json")]

	counter := ""
	if m := counterRe.FindString(base); m != "" {
		counter = m
		base = base[:len(base)-len(m)]
	}
	for i := len(supplemental); i >= 2; i-- {
		if strings.HasSuffix(base, supplemental[:i]) {
			base = base[:len(base)-i]
			break
		}
	}
	if base == "" {
		return "", false
	}
	if counter != "" {
		if i := strings.LastIndex(base, "."); i > 0 {
			base = base[:i] + counter + base[i:]
		} else {
			base += counter
		}
	}

	var prefixMatch string
	prefixes := 0
	for _, n := range mediaNames {
		if strings.EqualFold(n, base) {
			return n, true
		}
		if strings.HasPrefix(strings.ToLower(n), strings.ToLower(base)) {
			prefixMatch = n
			prefixes++
		}
	}
	// A truncated name is only trusted when it points at exactly one file.
	if prefixes == 1 {
		return prefixMatch, true
	}
	return "", false
}

// EditedOriginal returns the name of the original for a Takeout "-edited"
// variant, which has no sidecar of its own.
func EditedOriginal(mediaName string) (string, bool) {
	i := strings.LastIndex(mediaName, ".")
	if i <= 0 {
		return "", false
	}
	stem, ext := mediaName[:i], mediaName[i:]
	if !strings.HasSuffix(stem, "-edited") {
		return "", false
	}
	return strings.TrimSuffix(stem, "-edited") + ext, true
}
