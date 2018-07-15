package spy2go

import "unsafe"

const SoftwareID = "Spy2Go 1.0"

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
	StreamModeIQOnly  = StreamTypeIQ
	//StreamModeAFOnly  = StreamTypeAF
	StreamModeFFTOnly = StreamTypeFFT
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

type ComplexInt16 struct {
	real int16
	imag int16
}

type ComplexUInt8 struct {
	real uint8
	imag uint8
}

type Complex64SamplesCallback func(data []complex64)
type Complex32SamplesCallback func(data []ComplexInt16)
type Complex16SamplesCallback func(data []ComplexUInt8)
type FFTSamplesCallback func(data []uint8)
type DeviceSyncCallback func()

type CallbackBase struct {
	OnFloatIQ Complex64SamplesCallback
	OnInt16IQ Complex32SamplesCallback
	OnUInt8IQ Complex16SamplesCallback
	OnFFT FFTSamplesCallback
	OnDeviceSync DeviceSyncCallback
}