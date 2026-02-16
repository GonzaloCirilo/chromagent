package chroma

// EffectType represents a Chroma SDK effect type string used in JSON payloads.
type EffectType string

const (
	EffectNone           EffectType = "CHROMA_NONE"
	EffectStatic         EffectType = "CHROMA_STATIC"
	EffectCustom         EffectType = "CHROMA_CUSTOM"
	EffectCustom2        EffectType = "CHROMA_CUSTOM2"
	EffectCustomKey      EffectType = "CHROMA_CUSTOM_KEY"
	EffectWave           EffectType = "CHROMA_WAVE"
	EffectBreathing      EffectType = "CHROMA_BREATHING"
	EffectReactive       EffectType = "CHROMA_REACTIVE"
	EffectSpectrumCycle  EffectType = "CHROMA_SPECTRUMCYCLING"
	EffectBlinking       EffectType = "CHROMA_BLINKING"
	EffectStarlight      EffectType = "CHROMA_STARLIGHT"
)

// DeviceType represents a supported Chroma device category.
type DeviceType string

const (
	DeviceKeyboard  DeviceType = "keyboard"
	DeviceMouse     DeviceType = "mouse"
	DeviceMousepad  DeviceType = "mousepad"
	DeviceHeadset   DeviceType = "headset"
	DeviceKeypad    DeviceType = "keypad"
	DeviceChromaLink DeviceType = "chromalink"
)

// AllDevices is the full set of supported device types.
var AllDevices = []DeviceType{
	DeviceKeyboard,
	DeviceMouse,
	DeviceMousepad,
	DeviceHeadset,
	DeviceKeypad,
	DeviceChromaLink,
}

// Grid dimensions for each device type.
const (
	KeyboardRows    = 6
	KeyboardCols    = 22
	KeyboardV2Rows  = 8
	KeyboardV2Cols  = 24

	MouseRows = 9
	MouseCols = 7

	HeadsetLEDs = 5

	MousepadLEDs   = 15
	MousepadV2LEDs = 20

	KeypadRows = 4
	KeypadCols = 5

	ChromaLinkLEDs = 5
)

// MouseLED2 represents mouse LED positions for CUSTOM2 effects (RZLED2).
type MouseLED2 int

const (
	MouseLEDScrollWheel MouseLED2 = 0x0203
	MouseLEDLogo        MouseLED2 = 0x0703
	MouseLEDBacklight   MouseLED2 = 0x0403
	MouseLEDLeftSide1   MouseLED2 = 0x0100
	MouseLEDLeftSide2   MouseLED2 = 0x0101
	MouseLEDLeftSide3   MouseLED2 = 0x0102
	MouseLEDLeftSide4   MouseLED2 = 0x0103
	MouseLEDLeftSide5   MouseLED2 = 0x0104
	MouseLEDLeftSide6   MouseLED2 = 0x0105
	MouseLEDLeftSide7   MouseLED2 = 0x0106
	MouseLEDBottom1     MouseLED2 = 0x0200
	MouseLEDBottom2     MouseLED2 = 0x0201
	MouseLEDBottom3     MouseLED2 = 0x0202
	MouseLEDBottom4     MouseLED2 = 0x0204
	MouseLEDBottom5     MouseLED2 = 0x0205
	MouseLEDRightSide1  MouseLED2 = 0x0600
	MouseLEDRightSide2  MouseLED2 = 0x0601
	MouseLEDRightSide3  MouseLED2 = 0x0602
	MouseLEDRightSide4  MouseLED2 = 0x0603
	MouseLEDRightSide5  MouseLED2 = 0x0604
	MouseLEDRightSide6  MouseLED2 = 0x0605
	MouseLEDRightSide7  MouseLED2 = 0x0606
)

// EffectResponse is the response from a PUT effect request.
type EffectResponse struct {
	Result int    `json:"result"`
	ID     string `json:"id,omitempty"`
}
