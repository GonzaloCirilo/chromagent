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
		Author:          Author{},
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
			_ = fmt.Errorf("closing reponse body: %w", err)
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
// device: "keyboard", "mouse", "mousepad", "headset", "keypad", "chromalink"
func (c *Client) StaticEffect(device string, color int) error {
	payload := map[string]interface{}{
		"effect": "CHROMA_STATIC",
		"param": map[string]interface{}{
			"color": color,
		},
	}
	return c.putEffect(device, payload)
}

// NoEffect turns off all LEDs on a device.
func (c *Client) NoEffect(device string) error {
	payload := map[string]interface{}{
		"effect": "CHROMA_NONE",
	}
	return c.putEffect(device, payload)
}

// StaticAll sets the same static color across all supported devices.
func (c *Client) StaticAll(color int) error {
	devices := []string{"keyboard", "mouse", "mousepad", "headset", "keypad", "chromalink"}
	var lastErr error
	for _, d := range devices {
		if err := c.StaticEffect(d, color); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ClearAll turns off all devices.
func (c *Client) ClearAll() error {
	devices := []string{"keyboard", "mouse", "mousepad", "headset", "keypad", "chromalink"}
	var lastErr error
	for _, d := range devices {
		if err := c.NoEffect(d); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (c *Client) putEffect(device string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal effect: %w", err)
	}

	url := fmt.Sprintf("%s/chromasdk/%s", c.baseURI, device)
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
	return nil
}
