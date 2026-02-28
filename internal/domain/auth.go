package domain

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"rajidou/internal/netx"
)

// Source map in this file:
// - retrieve token flow: rajiko/modules/auth.js + rajiko/background.js
// - local token cache is CLI adaptation replacing extension storage.
// TokenCacheItem stores a Radiko token and the auth request timestamp (ms).
type TokenCacheItem struct {
	Token       string `json:"token"`
	RequestTime int64  `json:"requestTime"`
}

// AuthClient performs Radiko auth1/auth2 and caches auth tokens per area.
// The cache is an optimization for CLI usage and mirrors extension storage
// semantics with a local JSON file and TTL-based reuse.
type AuthClient struct {
	net        *netx.Client
	cachePath  string
	cacheOnce  sync.Once
	cacheMu    sync.Mutex
	tokenCache map[string]TokenCacheItem
}

// NewAuthClient builds an AuthClient with file-backed token cache settings.
func NewAuthClient(net *netx.Client) *AuthClient {
	return &AuthClient{
		net:        net,
		cachePath:  filepath.Join(".cache", "auth-tokens.json"),
		tokenCache: map[string]TokenCacheItem{},
	}
}

// RetrieveToken returns a valid auth token for the given area.
// It first reuses a recent cached token, then executes the Radiko auth1/auth2
// handshake. The auth1 response provides a key range that is sliced from the
// app key to build the auth2 partial key required by the protocol.
func (a *AuthClient) RetrieveToken(ctx context.Context, areaID string) (string, error) {
	a.ensureCacheLoaded()

	a.cacheMu.Lock()
	if c, ok := a.tokenCache[areaID]; ok && time.Now().UnixMilli()-c.RequestTime < 42e5 {
		t := c.Token
		a.cacheMu.Unlock()
		return t, nil
	}
	a.cacheMu.Unlock()

	keyMaterial := LoadRajikoAppKey()
	appVersion, userID, userAgent, device := GenDeviceInfo(keyMaterial.AppVersion)

	auth1Req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://radiko.jp/v2/api/auth1", nil)
	if err != nil {
		return "", err
	}
	auth1Req.Header.Set("X-Radiko-App", keyMaterial.AppID)
	auth1Req.Header.Set("X-Radiko-App-Version", appVersion)
	auth1Req.Header.Set("X-Radiko-Device", device)
	auth1Req.Header.Set("X-Radiko-User", userID)

	auth1Resp, err := a.net.Do(auth1Req)
	if err != nil {
		return "", err
	}
	defer auth1Resp.Body.Close()
	if auth1Resp.StatusCode < 200 || auth1Resp.StatusCode >= 300 {
		return "", fmt.Errorf("auth1 failed: %d", auth1Resp.StatusCode)
	}

	token := auth1Resp.Header.Get("x-radiko-authtoken")
	offsetStr := auth1Resp.Header.Get("x-radiko-keyoffset")
	lengthStr := auth1Resp.Header.Get("x-radiko-keylength")
	if token == "" || offsetStr == "" || lengthStr == "" {
		return "", fmt.Errorf("auth1 response is missing token or key range headers")
	}
	// Header parsing keeps auth flow simple; range validity is enforced below.
	offset, _ := strconv.Atoi(offsetStr)
	length, _ := strconv.Atoi(lengthStr)

	fullKey, err := base64.StdEncoding.DecodeString(keyMaterial.AppKeyBase64)
	if err != nil {
		return "", err
	}
	if offset < 0 || offset+length > len(fullKey) || length <= 0 {
		return "", fmt.Errorf("invalid key range from auth1")
	}
	partial := base64.StdEncoding.EncodeToString(fullKey[offset : offset+length])
	gps, err := GenGPS(areaID)
	if err != nil {
		return "", err
	}

	auth2Req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://radiko.jp/v2/api/auth2", nil)
	if err != nil {
		return "", err
	}
	auth2Req.Header.Set("X-Radiko-App", keyMaterial.AppID)
	auth2Req.Header.Set("X-Radiko-App-Version", appVersion)
	auth2Req.Header.Set("X-Radiko-Device", device)
	auth2Req.Header.Set("X-Radiko-User", userID)
	auth2Req.Header.Set("X-Radiko-AuthToken", token)
	auth2Req.Header.Set("X-Radiko-Partialkey", partial)
	auth2Req.Header.Set("X-Radiko-Location", gps)
	auth2Req.Header.Set("X-Radiko-Connection", "wifi")
	auth2Req.Header.Set("User-Agent", userAgent)

	auth2Resp, err := a.net.Do(auth2Req)
	if err != nil {
		return "", err
	}
	defer auth2Resp.Body.Close()
	if auth2Resp.StatusCode != 200 {
		return "", fmt.Errorf("auth2 failed: %d", auth2Resp.StatusCode)
	}

	a.cacheMu.Lock()
	a.tokenCache[areaID] = TokenCacheItem{Token: token, RequestTime: time.Now().UnixMilli()}
	a.cacheMu.Unlock()
	_ = a.saveCache()
	return token, nil
}

func (a *AuthClient) ensureCacheLoaded() {
	a.cacheOnce.Do(func() {
		raw, err := os.ReadFile(a.cachePath)
		if err != nil {
			// Missing/unreadable cache is non-fatal; auth flow can still proceed.
			return
		}
		m := map[string]TokenCacheItem{}
		if err := json.Unmarshal(raw, &m); err != nil {
			// Ignore malformed cache and continue with an empty in-memory cache.
			return
		}
		a.cacheMu.Lock()
		a.tokenCache = m
		a.cacheMu.Unlock()
	})
}

func (a *AuthClient) saveCache() error {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(a.cachePath), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(a.tokenCache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.cachePath, b, 0o644)
}
