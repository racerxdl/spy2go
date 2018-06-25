package spy2go

import "unsafe"

const SoftwareID = "Spy2Go 1.0"

const SpyserverProtocolVersion = ((2) << 24) | ((0) << 16) | (1558)
const SpyserverMaxCommandBodySize = 256
const SpyserverMaxMessageBodySize = 1 << 20
const SpyserverMaxDisplayPixels = 1 << 15
const SpyserverMinDisplayPixels = 100
const SpyserverMaxFFTDBRange = 150
const SpyserverMinFFTDBRange = 10
const SpyserverMaxFFTDBOffset = 100

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
	CmdHello      = 0
	CmdGetSetting = 1
	CmdSetSetting = 2
	CmdPing       = 3
)

const (
	SettingStreamingMode    = 0
	SettingStreamingEnabled = 1
	SettingGain             = 2

	SettingIqFormat     = 100
	SettingIqFrequency  = 101
	SettingIqDecimation = 102

	SettingFFTFormat        = 200
	SettingFFTFrequency     = 201
	SettingFFTDecimation    = 202
	SettingFFTDbOffset      = 203
	SettingFFTDbRange       = 204
	SettingFFTDisplayPixels = 205
)

const (
	StreamTypeStatus = 0
	StreamTypeIQ     = 1
	StreamTypeAF     = 2
	StreamTypeFFT    = 4
)

const (
	StreamModeIQOnly  = StreamTypeIQ
	StreamModeAFOnly  = StreamTypeAF
	StreamModeFFTOnly = StreamTypeFFT
	StreamModeFFTIq   = StreamTypeFFT | StreamTypeIQ
	StreamModeFFTAf   = StreamTypeFFT | StreamTypeAF
)

const (
	StreamFormatDint4      = 0
	StreamFormatUint8      = 1
	StreamFormatInt16      = 2
	StreamFormatInt24      = 3
	StreamFormatFloat      = 4
	StreamFormatCompressed = 5
)

const (
	MsgTypeDeviceInfo  = 0
	MsgTypeClientSync  = 1
	MsgTypePong        = 2
	MsgTypeReadSetting = 3

	MsgTypeUint8IQ      = 100
	MsgTypeInt16IQ      = 101
	MsgTypeInt24IQ      = 102
	MsgTypeFloatIQ      = 103
	MsgTypeCompressedIQ = 104

	MsgTypeUint8AF      = 200
	MsgTypeInt16AF      = 201
	MsgTypeInt24AF      = 202
	MsgTypeFloatAF      = 203
	MsgTypeCompressedAF = 204

	MsgTypeDint4FFT      = 300
	MsgTypeUint8FFT      = 301
	MsgTypeCompressedFFT = 302
)

const (
	ParserAcquiringHeader = iota
	ParserReadingData = iota
)

type ClientHandshake struct {
	ProtocolVersion uint32
	ClientNameLength uint32
}

type CommandHeader struct {
	CommandType uint32
	BodySize uint32
}

type SettingTarget struct {
	StreamType uint32
	SettingType uint32
}

type MessageHeader struct {
	ProtocolID uint32
	MessageType uint32
	StreamType uint32
	SequenceNumber uint32
	BodySize uint32
}

const MessageHeaderSize = uint32(unsafe.Sizeof(MessageHeader{}))

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

type ClientSync struct {
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

type CallbackBase interface {
	OnFloatIQ([]complex64)
	OnInt16IQ([]ComplexInt16)
	OnUInt8IQ([]ComplexUInt8)
	OnFFT([]uint8)
	OnDeviceSync()
}