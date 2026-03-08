package chroma

import (
	"log"
	"sync"
	"time"
)

// StaticAll sets the same static color across all supported devices concurrently.
// ChromaLink accessories use CHROMA_CUSTOM (explicit LED array) because some
// devices (headphone stands, mouse docks) ignore CHROMA_STATIC.
func (c *Client) StaticAll(color int) error {
	var wg sync.WaitGroup
	errs := make([]error, len(AllDevices))
	for i, d := range AllDevices {
		wg.Add(1)
		go func(i int, d DeviceType) {
			defer wg.Done()
			if d == DeviceChromaLink {
				// ChromaLink uses CHROMA_CUSTOM with a flat 1D array (not 2D).
				leds := make([]int, ChromaLinkLEDs)
				for j := range leds {
					leds[j] = color
				}
				errs[i] = c.CustomEffect(d, leds)
			} else {
				errs[i] = c.StaticEffect(d, color)
			}
		}(i, d)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			log.Printf("[chroma] StaticAll error on %s: %v", AllDevices[i], err)
			return err
		}
	}
	return nil
}

// ClearAll turns off all devices concurrently.
func (c *Client) ClearAll() error {
	var wg sync.WaitGroup
	errs := make([]error, len(AllDevices))
	for i, d := range AllDevices {
		wg.Add(1)
		go func(i int, d DeviceType) {
			defer wg.Done()
			errs[i] = c.NoEffect(d)
		}(i, d)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Flash briefly lights up all devices with a color, then clears.
func (c *Client) Flash(color int, duration time.Duration) {
	c.StaticAll(color)
	time.Sleep(duration)
	c.ClearAll()
}

// Pulse fades a color in and out by stepping through brightness levels.
// steps controls smoothness, cycleDuration is the total on→off time.
// Timing accounts for HTTP latency so the actual duration matches intent.
func (c *Client) Pulse(r, g, b uint8, steps int, cycleDuration time.Duration) {
	stepDelay := cycleDuration / time.Duration(steps*2)

	step := func(frac float64) {
		t := time.Now()
		cr := uint8(float64(r) * frac)
		cg := uint8(float64(g) * frac)
		cb := uint8(float64(b) * frac)
		c.StaticAll(BGR(cr, cg, cb))
		if remaining := stepDelay - time.Since(t); remaining > 0 {
			time.Sleep(remaining)
		}
	}

	// Fade in
	for i := 0; i <= steps; i++ {
		step(float64(i) / float64(steps))
	}
	// Fade out
	for i := steps; i >= 0; i-- {
		step(float64(i) / float64(steps))
	}
}

// AlertFlash does a rapid triple-flash in a color — attention-grabbing, sir.
func (c *Client) AlertFlash(color int) {
	for i := 0; i < 3; i++ {
		c.StaticAll(color)
		time.Sleep(150 * time.Millisecond)
		c.ClearAll()
		time.Sleep(100 * time.Millisecond)
	}
}
