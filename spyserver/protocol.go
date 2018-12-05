package spyserver

import "unsafe"

// SoftwareID the software ID that gets sent to SpyServer in sayHello.
// You can replace the default value for your own application name.
var SoftwareID = "Spy2Go 1.0"

// SpyserverProtocolVersion packed into a integer.
// Defined by ((major) << 24) | ((minor) << 16) | (revision)
const SpyserverProtocolVersion = ((2) << 24) | ((0) << 16) | (1558)

//const SpyserverMaxCommandBodySize = 256
const spyserverMaxMessageBodySize = 1 << 20

// SpyserverMaxDisplayPixels is the max possible pixels for Spyserver FFT width
const SpyserverMaxDisplayPixels = 1 << 15

// SpyserverMinDisplayPixels is the min possible pixels for Spyserver FFT width
const SpyserverMinDisplayPixels = 100

// SpyserverMaxFFTDBRange is the maximum dB Value for FFT Range
const SpyserverMaxFFTDBRange = 150

// SpyserverMinFFTDBRange is the minimum dB Value for FFT Range
const SpyserverMinFFTDBRange = 10

const SpyserverMaxFFTDBOffset = 100

const defaultFFTRange = 127
const defaultDisplayPixels = 2000

// InvalidValue is a constant that is returned in any of the functions that can receive invalid values.
const InvalidValue = 0xFFFFFFFF

// DeviceIds IDs of the devices in spyserver
const (
	DeviceInvalid   = 0
	DeviceAirspyOne = 1
	DeviceAirspyHf  = 2
	DeviceRtlsdr    = 3
)

// DeviceNames names of the devices
const (
	DeviceInvalidName   = "Invalid Device"
	DeviceAirspyOneName = "Airspy Mini / R2"
	DeviceAirspyHFName  = "Airspy HF / HF+"
	DeviceRtlsdrName    = "RTLSDR"
)

// DeviceName list of device names by their ids
var DeviceName = map[uint32]string{
	DeviceInvalid:   DeviceInvalidName,
	DeviceAirspyOne: DeviceAirspyOneName,
	DeviceAirspyHf:  DeviceAirspyHFName,
	DeviceRtlsdr:    DeviceRtlsdrName,
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

	settingIqFormat     = 100
	settingIqFrequency  = 101
	settingIqDecimation = 102

	settingFFTFormat        = 200
	settingFFTFrequency     = 201
	settingFFTDecimation    = 202
	settingFFTDbOffset      = 203
	settingFFTDbRange       = 204
	settingFFTDisplayPixels = 205
)

// StreamTypes is a enum that defines which stream types the spyserver supports.
const (
	//StreamTypeStatus = 0

	StreamTypeIQ = 1

	//StreamTypeAF     = 2

	StreamTypeFFT = 4
)

const (
	// StreamModeIQOnly only enables IQ Channel
	StreamModeIQOnly = StreamTypeIQ

	//StreamModeAFOnly  = StreamTypeAF

	// StreamModeFFTOnly only enables FFT Channel
	StreamModeFFTOnly = StreamTypeFFT

	// StreamModeFFTOnly only enables both IQ and FFT Channels
	StreamModeFFTIQ = StreamTypeFFT | StreamTypeIQ

	//StreamModeFFTAF   = StreamTypeFFT | StreamTypeAF
)

const (
	//StreamFormatDint4      = 0

	StreamFormatUint8 = 1
	StreamFormatInt16 = 2

	//StreamFormatInt24      = 3

	StreamFormatFloat = 4

	//StreamFormatCompressed = 5
)

const (
	msgTypeDeviceInfo  = 0
	msgTypeClientSync  = 1
	msgTypePong        = 2
	msgTypeReadSetting = 3

	msgTypeUint8IQ = 100
	msgTypeInt16IQ = 101

	//msgTypeInt24IQ      = 102

	msgTypeFloatIQ = 103

	//msgTypeCompressedIQ = 104

	//msgTypeUint8AF      = 200
	//msgTypeInt16AF      = 201
	//msgTypeInt24AF      = 202
	//msgTypeFloatAF      = 203
	//msgTypeCompressedAF = 204

	//msgTypeDint4FFT      = 300

	msgTypeUint8FFT = 301

	//msgTypeCompressedFFT = 302
)

const (
	parserAcquiringHeader = iota
	parserReadingData     = iota
)

//type clientHandshake struct {
//	ProtocolVersion uint32
//	ClientNameLength uint32
//}

type commandHeader struct {
	CommandType uint32
	BodySize    uint32
}

//type settingTarget struct {
//	StreamType uint32
//	SettingType uint32
//}

type messageHeader struct {
	ProtocolID     uint32
	MessageType    uint32
	StreamType     uint32
	SequenceNumber uint32
	BodySize       uint32
}

const messageHeaderSize = uint32(unsafe.Sizeof(messageHeader{}))

type deviceInfo struct {
	DeviceType           uint32
	DeviceSerial         uint32
	MaximumSampleRate    uint32
	MaximumBandwidth     uint32
	DecimationStageCount uint32
	GainStageCount       uint32
	MaximumGainIndex     uint32
	MinimumFrequency     uint32
	MaximumFrequency     uint32
	Resolution           uint32
	MinimumIQDecimation  uint32
	ForcedIQFormat       uint32
}

type clientSync struct {
	CanControl                uint32
	Gain                      uint32
	DeviceCenterFrequency     uint32
	IQCenterFrequency         uint32
	FFTCenterFrequency        uint32
	MinimumIQCenterFrequency  uint32
	MaximumIQCenterFrequency  uint32
	MinimumFFTCenterFrequency uint32
	MaximumFFTCenterFrequency uint32
}
