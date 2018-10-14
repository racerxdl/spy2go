package spytypes

// ComplexInt16 is a Complex Number in a signed 16 bit number
type ComplexInt16 struct {
	Real int16
	Imag int16
}

// ComplexUInt16 is a Complex Number in a unsigned 16 bit number
type ComplexUInt16 struct {
	Real uint16
	Imag uint16
}

// ComplexUInt8 is a Complex Number in a unsigned 8 bit number
// In this case the value 0 is in variable half-way (127)
type ComplexUInt8 struct {
	Real uint8
	Imag uint8
}

//
//// Complex64SamplesCallback callback type for Float IQ Samples
//type Complex64SamplesCallback func(data []complex64)
//
//// Complex32SamplesCallback callback type for 16 bit signed integer IQ Samples
//type Complex32SamplesCallback func(data []ComplexInt16)
//
//// RealFloatSamplesCallback callback type for Float Real Samples
//type RealFloatSamplesCallback func(data []float32)
//
//// RealFloatSamplesCallback callback type for 16 bit signed integer Real Samples
//type RealI16SamplesCallback func(data []int16)
//
//// RealFloatSamplesCallback callback type for 16 bit unsigned integer Real Samples
//type RealU16SamplesCallback func(data []uint16)
//
//// Complex16SamplesCallback callback type for 8 bit unsigned integer IQ Samples
//type Complex16SamplesCallback func(data []ComplexUInt8)
//
//// RawCallback callback type for raw data from device
//type RawCallback func(data []byte)
//
//// FFTSamplesCallback callback type for 8 bit FFT Bins Samples.
//type FFTSamplesCallback func(data []uint8)
//
//// DeviceSyncCallback callback type for Device Sync Packets.
//type DeviceSyncCallback func()
//
//// CallbackBase the base struct for the Airspy Data Callbacks.
//type CallbackBase struct {
//	// OnFloatIQ Called when a new set of Float IQ Samples are available (currently not in use)
//	OnFloatIQ Complex64SamplesCallback
//	// OnInt16IQ Called when a new set of 16 bit signed integer IQ Samples are available (default)
//	OnInt16IQ Complex32SamplesCallback
//	// OnInt16IQ Called when a new set of 16 bit signed integer Real Samples are available (default)
//	OnInt16Real RealI16SamplesCallback
//	// OnInt16IQ Called when a new set of 16 bit unsigned integer Real Samples are available (default)
//	OnUInt16Real RealU16SamplesCallback
//	// OnRaw Called when a new set of raw data from the device are available
//	OnRaw RawCallback
//}
//
//// SpyserverCallbackBase the base struct for the Spyserver Data Callbacks.
//// Currently the only IQ Mode enabled is Int16IQ so OnFloatIQ and OnUInt8IQ can be nil safely.
//type SpyserverCallbackBase struct {
//	// OnFloatIQ Called when a new set of Float IQ Samples are available (currently not in use)
//	OnFloatIQ Complex64SamplesCallback
//	// OnInt16IQ Called when a new set of 16 bit signed integer IQ Samples are available (default)
//	OnInt16IQ Complex32SamplesCallback
//	// OnUInt8IQ Called when a new set of 8 bit unsigned integer IQ Samples are available (currently not in use)
//	OnUInt8IQ Complex16SamplesCallback
//	// OnFFT Called when a new set of 8 bit FFT bins are available
//	OnFFT FFTSamplesCallback
//	// OnDeviceSync Called when a Device Sync Packet is received. Any changes from the server will be notified here.
//	OnDeviceSync DeviceSyncCallback
//}

const (
	SamplesComplex64 = iota
	SamplesFloat32
	SamplesComplex32
	SamplesInt16
	SamplesUInt16
	SamplesComplexUInt8
	SamplesBytes
	FFTUInt8
	DeviceSync
)

type Callback interface {
	OnData(int, interface{})
}
