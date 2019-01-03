package airspy

import (
	"fmt"
	"github.com/mattn/go-pointer"
	"github.com/racerxdl/spy2go/spytypes"
	"github.com/racerxdl/spy2go/spywrap"
	"log"
	"unsafe"
)

var libVersion = "x.x.x"

// GetLibraryVersion Returns the native library version.
// Requires Initialize to be called.
func GetLibraryVersion() string {
	return libVersion
}

// Initialize initializes the native airspy library
// It is required to call this once when starting the application
func Initialize() {
	result := spywrap.Airspy_init()
	if result != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(result))
	}

	var libvt = spywrap.NewAirspy_lib_version_t()

	spywrap.Airspy_lib_version(libvt)

	libVersion = fmt.Sprintf("%d.%d.%d", libvt.GetMajor_version(), libvt.GetMinor_version(), libvt.GetRevision())
}

// DeInitialize cleans up the native library
// It is required to call this before closing the application
func DeInitialize() {
	spywrap.Airspy_exit()
}

type Device struct {
	instance      spywrap.Struct_SS_airspy_device
	boardId       uint8
	serial        uint64
	partId        uint64
	name          string
	versionString string
	sampleRates   []uint32

	centerFrequency uint32
	sampleRate      uint32

	lnaGain uint8
	vgaGain uint8
	mixGain uint8
	cb      spytypes.Callback
}

func MakeAirspyDevice(serial uint64) *Device {
	var res spywrap.Airspy_open_result_t
	if serial != 0 {
		res = spywrap.OpenDevice()
	} else {
		res = spywrap.OpenDeviceBySerial(serial)
	}

	var r = res.GetResult()
	var v = res.GetDevice()
	var dev = Device{
		instance: v,
	}

	spywrap.FreeOpenResult(res)

	if r != spywrap.AirspySuccess || v == nil {
		panic(spywrap.GetAirspyError(r))
	}

	// region Get Board ID
	var bid = make([]uint8, 1)

	r = spywrap.Airspy_board_id_read(v, bid)
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}

	dev.boardId = bid[0]
	// endregion
	// region Get Version String
	versionString := make([]byte, 255)

	r = spywrap.Airspy_version_string_read(v, versionString, 255)
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}

	dev.versionString = spywrap.CharStringToString(versionString)
	// endregion
	// region Get Serial and Part Number
	s := spywrap.NewAirspy_read_partid_serialno_t()

	r = spywrap.Airspy_board_partid_serialno_read(v, s)

	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}

	dev.serial = spywrap.SerialNumber(s.GetSerial_no())
	dev.partId = spywrap.PartNumber(s.GetPart_id())
	dev.name = fmt.Sprintf("Airspy(%d) 0x%x", dev.boardId, dev.serial)
	// endregion
	// region Get Available SampleRates
	sampleRates := make([]uint32, 1)

	r = spywrap.Airspy_get_samplerates(v, sampleRates, 0)

	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}

	sampleRates = make([]uint32, sampleRates[0])

	r = spywrap.Airspy_get_samplerates(v, sampleRates, uint(len(sampleRates)))

	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}

	dev.sampleRates = sampleRates
	// endregion
	// region Defaults
	dev.SetSampleType(spywrap.AirspySampleFloat32Iq).SetCenterFrequency(106300000)
	dev.SetSampleRate(sampleRates[0]).SetLNAGain(8).SetMixerGain(5).SetVGAGain(5)
	// endregion

	return &dev
}

func (f *Device) Close() {
	spywrap.Airspy_close(f.instance)
}

//
func (f *Device) GetName() string {
	return f.name
}

func (f *Device) GetSerial() uint64 {
	return f.serial
}

func (f *Device) GetPartId() uint64 {
	return f.partId
}

func (f *Device) GetAvailableSampleRates() []uint32 {
	return f.sampleRates
}
func (f *Device) GetCenterFrequency() uint32 {
	return f.centerFrequency
}
func (f *Device) GetSampleRate() uint32 {
	return f.sampleRate
}
func (f *Device) SetSampleRate(sampleRate uint32) *Device {
	if f.sampleRate != sampleRate {

		if spywrap.Airspy_is_streaming(f.instance) == spywrap.AirspyTrue {
			return f.Stop().SetSampleRate(sampleRate).Start()
		}

		r := spywrap.Airspy_set_samplerate(f.instance, uint(sampleRate))
		if r != spywrap.AirspySuccess {
			panic(spywrap.GetAirspyError(r))
		}

		f.sampleRate = sampleRate
	}
	return f
}
func (f *Device) SetCenterFrequency(centerFrequency uint32) *Device {
	if centerFrequency < 24000000 {
		centerFrequency = 24000000
	}

	if centerFrequency > 1750000000 {
		centerFrequency = 1750000000
	}

	if f.centerFrequency != centerFrequency {
		r := spywrap.Airspy_set_freq(f.instance, uint(centerFrequency))
		if r != spywrap.AirspySuccess {
			panic(spywrap.GetAirspyError(r))
		}
		f.centerFrequency = centerFrequency
	}
	return f
}
func (f *Device) Start() *Device {

	cb := spywrap.Callback{
		Func: internalCallback,
		Data: f,
	}

	r := spywrap.AirspyStart(f.instance, uintptr(pointer.Save(&cb)))
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}

	return f
}
func (f *Device) Stop() *Device {
	if spywrap.Airspy_is_streaming(f.instance) == spywrap.AirspyTrue {
		spywrap.Airspy_stop_rx(f.instance)
	}
	return f
}
func (f *Device) SetAGC(agc bool) *Device {

	if agc {
		spywrap.Airspy_set_mixer_agc(f.instance, 1)
		spywrap.Airspy_set_lna_agc(f.instance, 1)
	} else {
		spywrap.Airspy_set_mixer_agc(f.instance, 0)
		spywrap.Airspy_set_lna_agc(f.instance, 0)

		f.SetLNAGain(f.lnaGain)
		f.SetMixerGain(f.mixGain)
	}

	return f
}
func (f *Device) SetLNAGain(gain uint8) *Device {
	r := spywrap.Airspy_set_lna_gain(f.instance, gain)
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}
	f.lnaGain = gain
	return f
}
func (f *Device) SetVGAGain(gain uint8) *Device {
	r := spywrap.Airspy_set_vga_gain(f.instance, gain)
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}
	f.vgaGain = gain
	return f
}
func (f *Device) SetMixerGain(gain uint8) *Device {
	r := spywrap.Airspy_set_mixer_gain(f.instance, gain)
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}
	f.mixGain = gain
	return f
}
func (f *Device) SetLinearityGain(gain uint8) *Device {
	r := spywrap.Airspy_set_linearity_gain(f.instance, gain)
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}
	return f
}

func (f *Device) SetBiasT(biast bool) *Device {
	val := uint8(0)
	if biast {
		val = 1
	}

	r := spywrap.Airspy_set_rf_bias(f.instance, val)
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}
	return f
}

func (f *Device) SetSampleType(sampleType int) *Device {
	r := spywrap.Airspy_set_sample_type(f.instance, spywrap.Enum_SS_airspy_sample_type(sampleType))
	if r != spywrap.AirspySuccess {
		panic(spywrap.GetAirspyError(r))
	}
	return f
}

func (f *Device) SetCallback(cb spytypes.Callback) *Device {
	f.cb = cb
	return f
}

func internalCallback(data interface{}, transfer spywrap.Airspy_transfer_t) int {
	f := data.(*Device)
	const arrLen = 1 << 20
	var sampleType = transfer.GetSample_type()
	var samples = transfer.GetSamples()
	var length = transfer.GetSample_count()

	if f.cb != nil {
		var arr interface{}
		var stype int
		switch sampleType {
		case spywrap.AirspySampleFloat32Iq:
			vArr := (*[arrLen]complex64)(unsafe.Pointer(samples))[:length:length]
			tmpArr := make([]complex64, length)
			copy(tmpArr, vArr)
			arr = tmpArr
			stype = spytypes.SamplesComplex64
			break
		case spywrap.AirspySampleFloat32Real:
			vArr := (*[arrLen]float32)(unsafe.Pointer(samples))[:length:length]
			tmpArr := make([]float32, length)
			copy(tmpArr, vArr)
			arr = tmpArr
			stype = spytypes.SamplesFloat32
			break
		case spywrap.AirspySampleInt16Iq:
			vArr := (*[arrLen]spytypes.ComplexInt16)(unsafe.Pointer(samples))[:length:length]
			tmpArr := make([]spytypes.ComplexInt16, length)
			copy(tmpArr, vArr)
			arr = tmpArr
			stype = spytypes.SamplesComplex32
		case spywrap.AirspySampleInt16Real:
			vArr := (*[arrLen]int16)(unsafe.Pointer(samples))[:length:length]
			tmpArr := make([]int16, length)
			copy(tmpArr, vArr)
			arr = tmpArr
			stype = spytypes.SamplesInt16
			break
		case spywrap.AirspySampleUint16Real:
			vArr := (*[arrLen]uint16)(unsafe.Pointer(samples))[:length:length]
			tmpArr := make([]uint16, length)
			copy(tmpArr, vArr)
			arr = tmpArr
			stype = spytypes.SamplesUInt16
		case spywrap.AirspySampleRaw:
			vArr := (*[arrLen]byte)(unsafe.Pointer(samples))[:length:length]
			tmpArr := make([]byte, length)
			copy(tmpArr, vArr)
			arr = tmpArr
			stype = spytypes.SamplesBytes
		default:
			log.Printf("Unknown sample type received!!!!")
			return 1
		}

		f.cb.OnData(stype, arr)
	}
	return 0
}
