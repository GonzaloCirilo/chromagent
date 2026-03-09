package chroma

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	ChromaSDKURL      = "http://localhost:54235/razer/chromasdk"
	HeartbeatInterval = 10 * time.Second
)

// BGR color helper — Chroma uses BGR, not RGB.
func BGR(r, g, b uint8) int {
	return int(b)<<16 | int(g)<<8 | int(r)
}

// Some handy presets, sir.
var (
	ColorOff        = 0
	ColorRed        = BGR(255, 0, 0)
	ColorGreen      = BGR(0, 255, 0)
	ColorBlue       = BGR(0, 0, 255)
	ColorYellow     = BGR(255, 255, 0)
	ColorCyan       = BGR(0, 255, 255)
	ColorMagenta    = BGR(255, 0, 255)
	ColorOrange     = BGR(255, 165, 0)
	ColorWhite      = BGR(255, 255, 255)
	ColorPurple     = BGR(128, 0, 255)
	ColorArcReactor = BGR(80, 160, 255)
)

// AppInfo is sent when initializing the Chroma SDK session.
type AppInfo struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Author          Author   `json:"author"`
	DeviceSupported []string `json:"device_supported"`
	Category        string   `json:"category"`
}

type Author struct {
	Name    string `json:"name"`
	Contact string `json:"contact"`
}

// InitResponse is returned from the SDK init POST.
type InitResponse struct {
	SessionID int    `json:"sessionid"`
	URI       string `json:"uri"`
}

// Client manages the Chroma SDK REST session.
type Client struct {
	baseURI    string
	httpClient *http.Client
	stopHB     chan struct{}
	mu         sync.Mutex
}

// NewClient initializes a Chroma SDK session and starts the heartbeat.
func NewClient() (*Client, error) {
	appInfo := AppInfo{
		Title:           "Chroma Hook for AI agents",
		Description:     "Razer Chroma RGB lighting driven by AI agent hook events",
		Author:          Author{Name: "chromagent", Contact: "https://github.com/GonzaloCirilo/chromagent"},
		DeviceSupported: []string{"keyboard", "mouse", "mousepad", "headset", "keypad", "chromalink"},
		Category:        "application",
	}

	body, err := json.Marshal(appInfo)
	if err != nil {
		return nil, fmt.Errorf("marshal app info: %w", err)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Post(ChromaSDKURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("init chroma SDK: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			_ = fmt.Errorf("closing response body: %w", err)
		}
	}(resp.Body)

	respBody, _ := io.ReadAll(resp.Body)
	var initResp InitResponse
	if err := json.Unmarshal(respBody, &initResp); err != nil {
		return nil, fmt.Errorf("parse init response: %w (body: %s)", err, string(respBody))
	}

	if initResp.URI == "" {
		return nil, fmt.Errorf("empty URI in init response: %s", string(respBody))
	}

	c := &Client{
		baseURI:    initResp.URI,
		httpClient: httpClient,
		stopHB:     make(chan struct{}),
	}

	// The SDK needs a moment after init before it accepts effect commands.
	time.Sleep(500 * time.Millisecond)

	go c.heartbeatLoop()
	return c, nil
}

func (c *Client) heartbeatLoop() {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.heartbeat()
		case <-c.stopHB:
			return
		}
	}
}

func (c *Client) heartbeat() {
	req, _ := http.NewRequest(http.MethodPut, c.baseURI+"/heartbeat", nil)
	c.httpClient.Do(req) //nolint: ignore heartbeat errors
}

// Close tears down the session.
func (c *Client) Close() {
	close(c.stopHB)
	req, _ := http.NewRequest(http.MethodDelete, c.baseURI, nil)
	c.httpClient.Do(req) //nolint
}

// ---------- Effect helpers ----------

// StaticEffect sets a single color across a device type.
func (c *Client) StaticEffect(device DeviceType, color int) error {
	payload := map[string]interface{}{
		"effect": EffectStatic,
		"param": map[string]interface{}{
			"color": color,
		},
	}
	return c.putEffect(device, payload)
}

// NoEffect turns off all LEDs on a device.
func (c *Client) NoEffect(device DeviceType) error {
	payload := map[string]interface{}{
		"effect": EffectNone,
	}
	return c.putEffect(device, payload)
}

// CustomEffect sets per-key/per-LED colors on a device.
// For grid devices (keyboard, mouse, keypad): pass a 2D array [rows][cols].
// For linear devices (chromalink, headset, mousepad): pass a single row [[led0, led1, ...]].
func (c *Client) CustomEffect(device DeviceType, colors interface{}) error {
	payload := map[string]interface{}{
		"effect": EffectCustom,
		"param":  colors,
	}
	return c.putEffect(device, payload)
}

// CustomEffect2 sets a CHROMA_CUSTOM2 effect (keyboard only).
// colors is the base color grid [rows][cols], keys is the key-specific overlay.
func (c *Client) CustomEffect2(device DeviceType, colors [][]int, keys [][]int) error {
	payload := map[string]interface{}{
		"effect": EffectCustom2,
		"param": map[string]interface{}{
			"color": colors,
			"key":   keys,
		},
	}
	return c.putEffect(device, payload)
}

// CustomKeyEffect sets a CHROMA_CUSTOM_KEY effect (keyboard only).
// colors is the base color grid, keys is the per-key overlay.
func (c *Client) CustomKeyEffect(colors [][]int, keys [][]int) error {
	payload := map[string]interface{}{
		"effect": EffectCustomKey,
		"param": map[string]interface{}{
			"color": colors,
			"key":   keys,
		},
	}
	return c.putEffect(DeviceKeyboard, payload)
}


// PostEffect creates an effect via POST and returns the effect ID for later use.
func (c *Client) PostEffect(device DeviceType, payload interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal effect: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.baseURI, device)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("post effect to %s: %w", device, err)
	}
	defer resp.Body.Close()

	var effectResp EffectResponse
	if err := json.NewDecoder(resp.Body).Decode(&effectResp); err != nil {
		return "", fmt.Errorf("parse effect response: %w", err)
	}
	if effectResp.Result != 0 {
		return "", fmt.Errorf("chroma SDK error: result=%d", effectResp.Result)
	}
	return effectResp.ID, nil
}

// SetEffect applies a previously created effect by its ID.
func (c *Client) SetEffect(effectID string) error {
	payload := map[string]interface{}{
		"id": effectID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal set effect: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURI+"/effect", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("set effect: %w", err)
	}
	defer resp.Body.Close()
	return c.checkResult(resp)
}

// DeleteEffect removes a previously created effect by its ID.
func (c *Client) DeleteEffect(effectID string) error {
	payload := map[string]interface{}{
		"id": effectID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal delete effect: %w", err)
	}

	req, err := http.NewRequest(http.MethodDelete, c.baseURI+"/effect", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete effect: %w", err)
	}
	defer resp.Body.Close()
	return c.checkResult(resp)
}

// putEffect sends a PUT request to apply an effect immediately.
func (c *Client) putEffect(device DeviceType, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal effect: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.baseURI, device)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("put effect to %s: %w", device, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var effectResp EffectResponse
	if err := json.Unmarshal(respBody, &effectResp); err != nil {
		return nil
	}
	if effectResp.Result != 0 {
		return fmt.Errorf("chroma SDK error: result=%d", effectResp.Result)
	}
	return nil
}

// checkResult parses the response body and returns an error if the SDK reports a non-zero result.
func (c *Client) checkResult(resp *http.Response) error {
	var effectResp EffectResponse
	if err := json.NewDecoder(resp.Body).Decode(&effectResp); err != nil {
		return nil // best-effort: don't fail if response isn't parseable
	}
	if effectResp.Result != 0 {
		return fmt.Errorf("chroma SDK error: result=%d", effectResp.Result)
	}
	return nil
}
