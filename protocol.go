package spy2go

import "unsafe"

// SoftwareID the software ID that gets sent to SpyServer in sayHello.
// You can replace the default value for your own application name.
var SoftwareID = "Spy2Go 1.0"

const SpyserverProtocolVersion = ((2) << 24) | ((0) << 16) | (1558)
//const SpyserverMaxCommandBodySize = 256
const SpyserverMaxMessageBodySize = 1 << 20
const SpyserverMaxDisplayPixels = 1 << 15
const SpyserverMinDisplayPixels = 100
const SpyserverMaxFFTDBRange = 150
const SpyserverMinFFTDBRange = 10
const SpyserverMaxFFTDBOffset = 100

const SpyserverMinGain = 0
const SpyserverMaxGain = 16

const DefaultFFTRange = 127
const DefaultDisplayPixels = 2000

const InvalidValue = 0xFFFFFFFF

const (
	DeviceInvalid   = 0
	DeviceAirspyOne = 1
	DeviceAirspyHf  = 2
	DeviceRtlsdr    = 3
)

const (
	DeviceInvalidName = "Invalid Device"
	DeviceAirspyOneName = "Airspy Mini / R2"
	DeviceAirspyHFName = "Airspy HF / HF+"
	DeviceRtlsdrName = "RTLSDR"
)

var DeviceName = map[uint32]string {
	DeviceInvalid: DeviceInvalidName,
	DeviceAirspyOne: DeviceAirspyOneName,
	DeviceAirspyHf: DeviceAirspyHFName,
	DeviceRtlsdr: DeviceRtlsdrName,
}

const (
	cmdHello      = 0
	cmdGetSetting = 1
	cmdSetSetting = 2
	cmdPing       = 3
)

const (
	settingStreamingMode    = 0
	settingStreamingEnabled = 1
	settingGain             = 2

	settingIqFormat     	= 100
	settingIqFrequency  	= 101
	settingIqDecimation 	= 102

	settingFFTFormat        = 200
	settingFFTFrequency     = 201
	settingFFTDecimation    = 202
	settingFFTDbOffset      = 203
	settingFFTDbRange       = 204
	settingFFTDisplayPixels = 205
)

const (
	StreamTypeStatus = 0
	StreamTypeIQ     = 1
	StreamTypeAF     = 2
	StreamTypeFFT    = 4
)

const (
	// StreamModeIQOnly only enables IQ Channel
	StreamModeIQOnly  = StreamTypeIQ
	//StreamModeAFOnly  = StreamTypeAF

	// StreamModeFFTOnly only enables FFT Channel
	StreamModeFFTOnly = StreamTypeFFT

	// StreamModeFFTOnly only enables both IQ and FFT Channels
	StreamModeFFTIQ   = StreamTypeFFT | StreamTypeIQ
	//StreamModeFFTAF   = StreamTypeFFT | StreamTypeAF
)

const (
	//StreamFormatDint4      = 0
	StreamFormatUint8      = 1
	StreamFormatInt16      = 2
	//StreamFormatInt24      = 3
	StreamFormatFloat      = 4
	//StreamFormatCompressed = 5
)

const (
	msgTypeDeviceInfo  = 0
	msgTypeClientSync  = 1
	msgTypePong        = 2
	msgTypeReadSetting = 3

	msgTypeUint8IQ      = 100
	msgTypeInt16IQ      = 101
	//msgTypeInt24IQ      = 102
	msgTypeFloatIQ      = 103
	//msgTypeCompressedIQ = 104

	//msgTypeUint8AF      = 200
	//msgTypeInt16AF      = 201
	//msgTypeInt24AF      = 202
	//msgTypeFloatAF      = 203
	//msgTypeCompressedAF = 204

	//msgTypeDint4FFT      = 300
	msgTypeUint8FFT      = 301
	//msgTypeCompressedFFT = 302
)

const (
	parserAcquiringHeader = iota
	parserReadingData = iota
)

//type clientHandshake struct {
//	ProtocolVersion uint32
//	ClientNameLength uint32
//}

type commandHeader struct {
	CommandType uint32
	BodySize uint32
}

//type settingTarget struct {
//	StreamType uint32
//	SettingType uint32
//}

type messageHeader struct {
	ProtocolID uint32
	MessageType uint32
	StreamType uint32
	SequenceNumber uint32
	BodySize uint32
}

const messageHeaderSize = uint32(unsafe.Sizeof(messageHeader{}))

type DeviceInfo struct {
	DeviceType uint32
	DeviceSerial uint32
	MaximumSampleRate uint32
	MaximumBandwidth uint32
	DecimationStageCount uint32
	GainStageCount uint32
	MaximumGainIndex uint32
	MinimumFrequency uint32
	MaximumFrequency uint32
}

type clientSync struct {
	CanControl uint32
	Gain uint32
	DeviceCenterFrequency uint32
	IQCenterFrequency uint32
	FFTCenterFrequency uint32
	MinimumIQCenterFrequency uint32
	MaximumIQCenterFrequency uint32
	MinimumFFTCenterFrequency uint32
	MaximumFFTCenterFrequency uint32
}

// ComplexInt16 is a Complex Number in a signed 16 bit number
type ComplexInt16 struct {
	Real int16
	Imag int16
}

// ComplexUInt8 is a Complex Number in a unsigned 8 bit number
// In this case the value 0 is in variable half-way (127)
type ComplexUInt8 struct {
	Real uint8
	Imag uint8
}

// Complex64SamplesCallback callback type for Float IQ Samples
type Complex64SamplesCallback func(data []complex64)

// Complex32SamplesCallback callback type for 16 bit signed integer IQ Samples
type Complex32SamplesCallback func(data []ComplexInt16)

// Complex16SamplesCallback callback type for 8 bit unsigned integer IQ Samples
type Complex16SamplesCallback func(data []ComplexUInt8)

// FFTSamplesCallback callback type for 8 bit FFT Bins Samples.
type FFTSamplesCallback func(data []uint8)

// DeviceSyncCallback callback type for Device Sync Packets.
type DeviceSyncCallback func()

// CallbackBase the base struct for the Spyserver Data Callbacks.
// Currently the only IQ Mode enabled is Int16IQ so OnFloatIQ and OnUInt8IQ can be nil safely.
type CallbackBase struct {
	// OnFloatIQ Called when a new set of Float IQ Samples are available (currently not in use)
	OnFloatIQ Complex64SamplesCallback
	// OnInt16IQ Called when a new set of 16 bit signed integer IQ Samples are available (default)
	OnInt16IQ Complex32SamplesCallback
	// OnUInt8IQ Called when a new set of 8 bit unsigned integer IQ Samples are available (currently not in use)
	OnUInt8IQ Complex16SamplesCallback
	// OnFFT Called when a new set of 8 bit FFT bins are available
	OnFFT FFTSamplesCallback
	// OnDeviceSync Called when a Device Sync Packet is received. Any changes from the server will be notified here.
	OnDeviceSync DeviceSyncCallback
}