package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"rajidou/internal/netx"
)

// Source map in this file:
// - load key constants: rajiko/modules/static.js
// - GenGPS/GenDeviceInfo behavior: rajiko/modules/util.js
// - station->area resolution behavior: rajiko/modules/constants.js
// KeyMaterial is the app credential bundle consumed by auth/header builders.
// Values are treated as immutable once loaded and copied by value to callers.
type KeyMaterial struct {
	AppVersion   string
	AppID        string
	AppKeyBase64 string
}

var (
	keyMaterialCache *KeyMaterial
	keyMu            sync.Mutex
)

// LoadRajikoAppKey returns the current Radiko app credential set.
// It memoizes values in-process to avoid repeatedly rebuilding the same struct.
// The lock protects lazy initialization and keeps concurrent calls consistent.
func LoadRajikoAppKey() KeyMaterial {
	keyMu.Lock()
	defer keyMu.Unlock()
	if keyMaterialCache != nil {
		return *keyMaterialCache
	}
	keyMaterialCache = &KeyMaterial{AppVersion: RadikoAppVersion, AppID: RadikoAppID, AppKeyBase64: RadikoAppKeyBase64}
	return *keyMaterialCache
}

var areaCoordinates = [][2]float64{
	{43.064615, 141.346807}, {40.824308, 140.739998}, {39.703619, 141.152684}, {38.268837, 140.8721}, {39.718614, 140.102364},
	{38.240436, 140.363633}, {37.750299, 140.467551}, {36.341811, 140.446793}, {36.565725, 139.883565}, {36.390668, 139.060406},
	{35.856999, 139.648849}, {35.605057, 140.123306}, {35.689488, 139.691706}, {35.447507, 139.642345}, {37.902552, 139.023095},
	{36.695291, 137.211338}, {36.594682, 136.625573}, {36.065178, 136.221527}, {35.664158, 138.568449}, {36.651299, 138.180956},
	{35.391227, 136.722291}, {34.97712, 138.383084}, {35.180188, 136.906565}, {34.730283, 136.508588}, {35.004531, 135.86859},
	{35.021247, 135.755597}, {34.686297, 135.519661}, {34.691269, 135.183071}, {34.685334, 135.832742}, {34.225987, 135.167509},
	{35.503891, 134.237736}, {35.472295, 133.0505}, {34.661751, 133.934406}, {34.39656, 132.459622}, {34.185956, 131.470649},
	{34.065718, 134.55936}, {34.340149, 134.043444}, {33.841624, 132.765681}, {33.559706, 133.531079}, {33.606576, 130.418297},
	{33.249442, 130.299794}, {32.744839, 129.873756}, {32.789827, 130.741667}, {33.238172, 131.612619}, {31.911096, 131.423893},
	{31.560146, 130.557978}, {26.2124, 127.680932},
}

// GenGPS returns a plausible GPS coordinate for the given area ID (JP1..JP47).
// It uses prefecture-capital seed coordinates and adds small random jitter so
// repeated requests do not look identical.
func GenGPS(areaID string) (string, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(areaID, "JP"))
	if err != nil || n < 1 || n > len(areaCoordinates) {
		return "", fmt.Errorf("invalid area id: %s", areaID)
	}
	pos := areaCoordinates[n-1]
	lat := pos[0] + (randFloat()/40.0)*randSign()
	lon := pos[1] + (randFloat()/40.0)*randSign()
	return fmt.Sprintf("%.6f,%.6f,gps", lat, lon), nil
}

// GenDeviceInfo builds Android-like client metadata used by Radiko endpoints.
// userID is generated from 16 random bytes and hex-encoded to match upstream
// format assumptions seen in the original client flow.
func GenDeviceInfo(appVersion string) (appVer, userID, userAgent, device string) {
	model := "Google Pixel 6"
	sdk := "34"
	device = sdk + ".GQML3"
	userAgent = fmt.Sprintf("Dalvik/2.1.0 (Linux; U; Android 14.0.0; %s/AP2A.240805.005.S4)", model)
	uid := make([]byte, 16)
	_, _ = rand.Read(uid)
	userID = hex.EncodeToString(uid)
	return appVersion, userID, userAgent, device
}

func randFloat() float64 {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	n := uint64(0)
	for _, x := range b {
		n = (n << 8) | uint64(x)
	}
	return float64(n%1000000) / 1000000.0
}

func randSign() float64 {
	if randFloat() > 0.5 {
		return 1
	}
	return -1
}

// stationAreaCache stores stationID -> areaID mappings discovered from
// per-area station list XMLs. stationAreaWarm ensures full warm-up runs once.
var stationAreaCache sync.Map
var stationAreaWarm sync.Once

// WarmStationAreaCache preloads station-area mappings for all 47 JP areas.
// Warm-up is best-effort: individual area fetch failures are ignored so callers
// can still proceed with fallback probing in ResolveStationAreaID.
func WarmStationAreaCache(ctx context.Context, net *netx.Client) {
	stationAreaWarm.Do(func() {
		var wg sync.WaitGroup
		wg.Add(47)
		for n := 1; n <= 47; n++ {
			areaID := fmt.Sprintf("JP%d", n)
			go func(area string) {
				defer wg.Done()
				warmAreaStations(ctx, net, area)
			}(areaID)
		}
		wg.Wait()
	})
}

// ResolveStationAreaID returns the JP area ID for a station code.
// It checks cache first, then performs one-time global warm-up, then falls back
// to incremental per-area fetches to recover from partial warm-up failures.
func ResolveStationAreaID(ctx context.Context, net *netx.Client, stationID string) (string, error) {
	if v, ok := stationAreaCache.Load(stationID); ok {
		return v.(string), nil
	}
	WarmStationAreaCache(ctx, net)
	if v, ok := stationAreaCache.Load(stationID); ok {
		return v.(string), nil
	}
	for n := 1; n <= 47; n++ {
		areaID := fmt.Sprintf("JP%d", n)
		if warmAreaStations(ctx, net, areaID) {
			if v, ok := stationAreaCache.Load(stationID); ok {
				return v.(string), nil
			}
		}
	}
	return "", fmt.Errorf("cannot resolve area id for station: %s", stationID)
}

// warmAreaStations fetches one area station list and updates stationAreaCache.
// It returns whether the area fetch succeeded; callers use this signal to decide
// whether a second cache lookup is meaningful.
func warmAreaStations(ctx context.Context, net *netx.Client, areaID string) bool {
	url := fmt.Sprintf("https://radiko.jp/v3/station/list/%s.xml", areaID)
	status, xml, err := net.GetText(ctx, url, nil)
	if err != nil || status < 200 || status >= 300 {
		return false
	}
	ids := extractStationIDsFromAreaXML(xml)
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, exists := stationAreaCache.Load(id); !exists {
			stationAreaCache.Store(id, areaID)
		}
	}
	return true
}

// extractStationIDsFromAreaXML scans raw XML text for <id>...</id> entries.
// Parsing is intentionally minimal and assumes Radiko area XML stays small and
// structurally stable; this helper avoids introducing a full XML decoder.
func extractStationIDsFromAreaXML(xml string) []string {
	out := make([]string, 0, 32)
	offset := 0
	for {
		start := strings.Index(xml[offset:], "<id>")
		if start < 0 {
			break
		}
		start += offset + len("<id>")
		end := strings.Index(xml[start:], "</id>")
		if end < 0 {
			break
		}
		end += start
		id := strings.TrimSpace(xml[start:end])
		if id != "" {
			out = append(out, id)
		}
		offset = end + len("</id>")
	}
	return out
}

func sleepContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(v, hi))
}

func newRequest(ctx context.Context, method, url string, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}
