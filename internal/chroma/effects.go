package chroma

import (
	"time"
)

// Flash briefly lights up all devices with a color, then clears.
func (c *Client) Flash(color int, duration time.Duration) {
	c.StaticAll(color)
	time.Sleep(duration)
	c.ClearAll()
}

// Pulse fades a color in and out by stepping through brightness levels.
// steps controls smoothness, cycleDuration is the total on→off time.
func (c *Client) Pulse(r, g, b uint8, steps int, cycleDuration time.Duration) {
	stepDelay := cycleDuration / time.Duration(steps*2)

	// Fade in
	for i := 0; i <= steps; i++ {
		frac := float64(i) / float64(steps)
		cr := uint8(float64(r) * frac)
		cg := uint8(float64(g) * frac)
		cb := uint8(float64(b) * frac)
		c.StaticAll(BGR(cr, cg, cb))
		time.Sleep(stepDelay)
	}
	// Fade out
	for i := steps; i >= 0; i-- {
		frac := float64(i) / float64(steps)
		cr := uint8(float64(r) * frac)
		cg := uint8(float64(g) * frac)
		cb := uint8(float64(b) * frac)
		c.StaticAll(BGR(cr, cg, cb))
		time.Sleep(stepDelay)
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
