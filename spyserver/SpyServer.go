// spy2go is a Spyserver Go Client that supports receiving custom IQ and FFT data
// It is currently Work in Progress but it already works.
package spyserver

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/racerxdl/spy2go/spytypes"
	"log"
	"net"
	"time"
)

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// Spyserver connection handler.
// Use MakeSpyserver or MakeSpyserverFullHS to create an instance.
type Spyserver struct {
	fullhostname string
	callback     spytypes.Callback
	client       net.Conn

	terminated     bool
	routineRunning bool
	gotDeviceInfo  bool
	gotSyncInfo    bool
	streamingMode  uint32
	gain           uint32

	availableSampleRates []uint32

	parserPhase        uint32
	deviceInfo         deviceInfo
	header             messageHeader
	lastSequenceNumber uint32
	droppedBuffers     uint32
	downStreamBytes    uint64
	parserPosition     uint32
	bodyBuffer         []uint8
	headerBuffer       []uint8

	Streaming      bool
	CanControl     bool
	IsConnected    bool
	DroppedBuffers uint32

	MinimumTunableFrequency uint32
	MaximumTunableFrequency uint32
	DeviceCenterFrequency   uint32
	channelCenterFrequency  uint32
	DisplayCenterFrequency  uint32

	currentSampleRate           uint32
	currentDisplaySampleRate    uint32
	channelDecimationStageCount uint32
	displayDecimationStageCount uint32
	displayOffset               int32
	displayRange                int32
	displayPixels               uint32

	msgChannel chan []uint8
}

// MakeSpyserverByFullHS creates an instance of Spyserver by giving hostname + port.
// Example: MakeSpyserverByFullHS("airspy.com:5555")
func MakeSpyserverByFullHS(fullhostname string) *Spyserver {
	var s = &Spyserver{
		fullhostname:         fullhostname,
		callback:             nil,
		terminated:           false,
		gotDeviceInfo:        false,
		gotSyncInfo:          false,
		parserPhase:          parserAcquiringHeader,
		Streaming:            false,
		CanControl:           false,
		IsConnected:          false,
		availableSampleRates: []uint32{},
		headerBuffer:         make([]uint8, messageHeaderSize),

		displayOffset:               0,
		displayRange:                defaultFFTRange,
		displayPixels:               defaultDisplayPixels,
		streamingMode:               StreamModeIQOnly,
		displayDecimationStageCount: 1,
	}
	s.cleanup()
	return s
}

// MakeSpyserver creates an instance of Spyserver by giving hostname and port as separated parameters.
// Example: MakeSpyserver("airspy.com", 5555)
func MakeSpyserver(hostname string, port int) *Spyserver {
	var s = &Spyserver{
		fullhostname:         fmt.Sprintf("%s:%d", hostname, port),
		callback:             nil,
		terminated:           false,
		gotDeviceInfo:        false,
		gotSyncInfo:          false,
		parserPhase:          parserAcquiringHeader,
		Streaming:            false,
		CanControl:           false,
		IsConnected:          false,
		availableSampleRates: []uint32{},
		headerBuffer:         make([]uint8, messageHeaderSize),

		displayOffset:               0,
		displayRange:                defaultFFTRange,
		displayPixels:               defaultDisplayPixels,
		streamingMode:               StreamModeIQOnly,
		displayDecimationStageCount: 1,
	}
	s.cleanup()
	return s
}

// region Private Methods

// sayHello sends a Hello Command to the server, with the Software ID (in this case, spy2go)
func (f *Spyserver) sayHello() bool {
	var totalLength = 4
	var softwareVersionBytes = []byte(SoftwareID)
	totalLength += len(softwareVersionBytes)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(SpyserverProtocolVersion))
	binary.Write(buf, binary.LittleEndian, softwareVersionBytes)

	return f.sendCommand(cmdHello, buf.Bytes())
}

// cleanup Cleans up all variables and returns to its default states.
func (f *Spyserver) cleanup() {
	f.deviceInfo.DeviceType = DeviceInvalid
	f.deviceInfo.DeviceSerial = 0
	f.deviceInfo.DecimationStageCount = 0
	f.deviceInfo.GainStageCount = 0
	f.deviceInfo.MaximumSampleRate = 0
	f.deviceInfo.MaximumBandwidth = 0
	f.deviceInfo.MaximumGainIndex = 0
	f.deviceInfo.MinimumFrequency = 0
	f.deviceInfo.MaximumFrequency = 0

	f.gain = 0
	f.CanControl = false
	f.gotDeviceInfo = false
	f.gotSyncInfo = false

	f.lastSequenceNumber = 0xFFFFFFFF
	f.droppedBuffers = 0
	f.downStreamBytes = 0
	f.parserPhase = parserAcquiringHeader
	f.parserPosition = 0

	f.Streaming = false
	f.terminated = true
}

// onConnect is executed just after a connection is made with spyserver and got a synchronization info.
// It updates all settings on spyserver
func (f *Spyserver) onConnect() {
	f.setSetting(settingStreamingMode, []uint32{f.streamingMode})
	f.setSetting(settingIqFormat, []uint32{StreamFormatInt16})
	f.setSetting(settingFFTFormat, []uint32{StreamFormatUint8})
	f.setSetting(settingFFTDisplayPixels, []uint32{f.displayPixels})
	f.setSetting(settingFFTDbOffset, []uint32{uint32(f.displayOffset - 50)})
	f.setSetting(settingFFTDbRange, []uint32{uint32(f.displayRange)})
	f.setSetting(settingFFTDecimation, []uint32{1})

	var sampleRates = make([]uint32, f.deviceInfo.DecimationStageCount)
	for i := uint32(0); i < f.deviceInfo.DecimationStageCount; i++ {
		var decim = uint32(1 << i)
		sampleRates[i] = uint32(float32(f.deviceInfo.MaximumSampleRate) / float32(decim))
	}
	f.availableSampleRates = sampleRates
}

// setSetting changes a setting in Spyserver
func (f *Spyserver) setSetting(settingType uint32, params []uint32) bool {
	var argBytes = make([]uint8, 0)

	if len(params) > 0 {
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, settingType)
		for i := 0; i < len(params); i++ {
			binary.Write(buf, binary.LittleEndian, params[i])
		}
		argBytes = buf.Bytes()
	}

	return f.sendCommand(cmdSetSetting, argBytes)
}

// sendCommand sends a command to spyserver
func (f *Spyserver) sendCommand(cmd uint32, args []uint8) bool {
	if f.client == nil {
		return false
	}

	var argsLen = uint32(0)
	if args != nil {
		argsLen += uint32(len(args))
	}

	var buff = new(bytes.Buffer)

	var header = commandHeader{
		CommandType: cmd,
		BodySize:    argsLen,
	}

	err := binary.Write(buff, binary.LittleEndian, &header)
	if err != nil {
		panic(err)
	}

	if args != nil {
		for i := 0; i < len(args); i++ {
			buff.WriteByte(byte(args[i]))
		}
	}

	_, err = f.client.Write(buff.Bytes())
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func (f *Spyserver) parseMessage(buffer []uint8) {
	f.downStreamBytes++

	consumed := uint32(0)
	for len(buffer) > 0 && !f.terminated {
		if f.parserPhase == parserAcquiringHeader {
			for f.parserPhase == parserAcquiringHeader && len(buffer) > 0 {
				consumed = f.parseHeader(buffer)
				buffer = buffer[consumed:]
			}

			if f.parserPhase == parserReadingData {
				clientMajor := uint8((SpyserverProtocolVersion >> 24) & 0xFF)
				clientMinor := uint8((SpyserverProtocolVersion >> 16) & 0xFF)

				serverMajor := uint8((f.header.ProtocolID >> 24) & 0xFF)
				serverMinor := uint8((f.header.ProtocolID >> 16) & 0xFF)
				//serverBuild := uint16(f.header.ProtocolID & 0xFFFF)

				if clientMajor != serverMajor || clientMinor != serverMinor {
					panic("Server is running an unsupported protocol version.")
				}

				if f.header.BodySize > spyserverMaxMessageBodySize {
					panic("Server sent more than expected body size.")
				}

				f.bodyBuffer = make([]uint8, f.header.BodySize)
			}
		}

		if f.parserPhase == parserReadingData {
			consumed = f.parseBody(buffer)
			buffer = buffer[consumed:]

			if f.parserPhase == parserAcquiringHeader {
				if f.header.MessageType != msgTypeDeviceInfo && f.header.MessageType != msgTypeClientSync && f.header.MessageType != msgTypeUint8FFT {
					gap := f.header.SequenceNumber - f.lastSequenceNumber - 1
					f.lastSequenceNumber = f.header.SequenceNumber
					f.droppedBuffers += gap
					if gap > 0 {
						log.Printf("Lost %d frames from spyserver!\n", gap)
					}
				}
				f.handleNewMessage()
			}
		}
	}
}

func (f *Spyserver) parseHeader(buffer []uint8) uint32 {
	consumed := uint32(0)

	for len(buffer) > 0 {
		toWrite := min(messageHeaderSize-f.parserPosition, uint32(len(buffer)))
		for i := uint32(0); i < toWrite; i++ {
			f.headerBuffer[i+f.parserPosition] = buffer[i]
		}
		buffer = buffer[toWrite:]
		consumed += toWrite
		f.parserPosition += toWrite

		if f.parserPosition == messageHeaderSize {
			f.parserPosition = 0
			buf := bytes.NewReader(f.headerBuffer)
			err := binary.Read(buf, binary.LittleEndian, &f.header)
			if err != nil {
				panic(err)
			}

			if f.header.BodySize > 0 {
				f.parserPhase = parserReadingData
			}

			return consumed
		}
	}

	return consumed
}

func (f *Spyserver) parseBody(buffer []uint8) uint32 {
	consumed := uint32(0)

	for len(buffer) > 0 {
		toWrite := min(f.header.BodySize-f.parserPosition, uint32(len(buffer)))
		for i := uint32(0); i < toWrite; i++ {
			f.bodyBuffer[i+f.parserPosition] = buffer[i]
		}
		buffer = buffer[toWrite:]
		consumed += toWrite
		f.parserPosition += toWrite

		if f.parserPosition == f.header.BodySize {
			f.parserPosition = 0
			f.parserPhase = parserAcquiringHeader
			return consumed
		}
	}

	return consumed
}

func (f *Spyserver) processDeviceInfo() {
	var dInfo = deviceInfo{}

	buf := bytes.NewReader(f.bodyBuffer)
	err := binary.Read(buf, binary.LittleEndian, &dInfo)
	if err != nil {
		panic(err)
	}

	f.deviceInfo = dInfo
	f.gotDeviceInfo = true
}

func (f *Spyserver) processClientSync() {
	var clientSync = clientSync{}

	buf := bytes.NewReader(f.bodyBuffer)
	err := binary.Read(buf, binary.LittleEndian, &clientSync)
	if err != nil {
		panic(err)
	}

	f.CanControl = clientSync.CanControl != 0
	f.gain = clientSync.Gain
	f.DeviceCenterFrequency = clientSync.DeviceCenterFrequency
	//f.channelCenterFrequency = clientSync.DeviceCenterFrequency
	f.DisplayCenterFrequency = clientSync.FFTCenterFrequency

	if f.streamingMode == StreamModeFFTOnly || f.streamingMode == StreamModeFFTIQ {
		f.MinimumTunableFrequency = clientSync.MinimumFFTCenterFrequency
		f.MaximumTunableFrequency = clientSync.MaximumFFTCenterFrequency
	} else if f.streamingMode == StreamModeIQOnly {
		f.MinimumTunableFrequency = clientSync.MinimumIQCenterFrequency
		f.MaximumTunableFrequency = clientSync.MaximumIQCenterFrequency
	}

	f.gotSyncInfo = true

	//log.Println(clientSync)

	if f.callback != nil {
		f.callback.OnData(spytypes.DeviceSync, nil)
	}
}

func (f *Spyserver) processUInt8Samples() {
	var sampleCount = f.header.BodySize / 2

	if f.callback != nil {
		var u8arr = make([]spytypes.ComplexUInt8, sampleCount)
		buf := bytes.NewBuffer(f.bodyBuffer)

		var tmp = make([]uint8, sampleCount*2)
		binary.Read(buf, binary.LittleEndian, &tmp)

		for i := uint32(0); i < sampleCount; i++ {
			u8arr[i] = spytypes.ComplexUInt8{
				Real: tmp[i*2],
				Imag: tmp[i*2+1],
			}
		}

		f.callback.OnData(spytypes.SamplesComplexUInt8, u8arr)
	}
}

func (f *Spyserver) processInt16Samples() {
	var sampleCount = f.header.BodySize / 4
	//var pairLength = sampleCount * 2
	if f.callback != nil {
		var c16arr = make([]spytypes.ComplexInt16, sampleCount)
		buf := bytes.NewBuffer(f.bodyBuffer)

		var tmp = make([]int16, sampleCount*2)
		binary.Read(buf, binary.LittleEndian, &tmp)

		//for i := 0; i < int(sampleCount * 2); i++ {
		//	var z int16
		//	binary.Read(buf, binary.LittleEndian, &z)
		//	tmp[i] = z
		//}

		for i := uint32(0); i < sampleCount; i++ {
			c16arr[i] = spytypes.ComplexInt16{
				Real: tmp[i*2],
				Imag: tmp[i*2+1],
			}
		}
		f.callback.OnData(spytypes.SamplesComplex32, c16arr)
	}
}

func (f *Spyserver) processFloatSamples() {
	var sampleCount = f.header.BodySize / 8

	if f.callback != nil {
		var c64arr = make([]complex64, sampleCount)
		buf := bytes.NewBuffer(f.bodyBuffer)

		for i := uint32(0); i < sampleCount; i++ {
			binary.Read(buf, binary.LittleEndian, &c64arr[i])
		}

		f.callback.OnData(spytypes.SamplesComplex64, c64arr)
	}
}

func (f *Spyserver) processUInt8FFT() {
	if f.callback != nil {
		f.callback.OnData(spytypes.FFTUInt8, f.bodyBuffer)
	}
}

func (f *Spyserver) handleNewMessage() {
	if f.terminated {
		return
	}

	switch f.header.MessageType {
	case msgTypeDeviceInfo:
		f.processDeviceInfo()
		break
	case msgTypeClientSync:
		f.processClientSync()
		break
	case msgTypeUint8IQ:
		f.processUInt8Samples()
		break
	case msgTypeInt16IQ:
		f.processInt16Samples()
		break
	case msgTypeFloatIQ:
		f.processFloatSamples()
		break
	case msgTypeUint8FFT:
		f.processUInt8FFT()
		break
	}
}

func (f *Spyserver) setStreamState() bool {
	if f.Streaming {
		return f.setSetting(settingStreamingEnabled, []uint32{1})
	} else {
		return f.setSetting(settingStreamingEnabled, []uint32{0})
	}
}

func (f *Spyserver) threadLoop() {
	f.parserPhase = parserAcquiringHeader
	f.parserPosition = 0

	buffer := make([]uint8, 64*1024)

	for f.routineRunning && !f.terminated {
		if f.terminated || !f.routineRunning {
			break
		}

		n, err := f.client.Read(buffer)

		if err != nil {
			if f.routineRunning && !f.terminated {
				log.Println("Error receiving data: ", err)
			}
			break
		}
		if n > 0 {
			var sl = buffer[:n]
			f.parseMessage(sl)
		}
	}
	log.Println("Thread closing")
	f.routineRunning = false
	f.cleanup()
}

// endregion
// region Public Methods

// GetName returns the name of the active device in spyserver
func (f *Spyserver) GetName() string {
	return DeviceName[f.deviceInfo.DeviceType]
}

// Start starts the streaming process (if not already started)
func (f *Spyserver) Start() {
	if !f.Streaming {
		log.Println("Starting streaming")
		f.Streaming = true
		f.downStreamBytes = 0
		f.setStreamState()
	}
}

// Stop stop the streaming process (if started)
func (f *Spyserver) Stop() {
	if f.Streaming {
		f.Streaming = false
		f.downStreamBytes = 0
		f.setStreamState()
	}
}

// Connect initiates the connection with spyserver.
// It panics if the connection fails for some reason.
func (f *Spyserver) Connect() {
	if f.routineRunning {
		return
	}

	log.Println("Trying to connect")
	conn, err := net.Dial("tcp", f.fullhostname)
	if err != nil {
		panic(err)
	}

	f.client = conn
	f.IsConnected = true

	f.sayHello()
	f.cleanup()

	f.terminated = false
	f.gotSyncInfo = false
	f.gotDeviceInfo = false
	f.routineRunning = true

	hasError := false
	errorMsg := ""

	go f.threadLoop()
	log.Println("Connected. Waiting for device info.")
	for i := 0; i < 1000 && !hasError; i++ {
		if f.gotDeviceInfo {
			if f.deviceInfo.DeviceType == DeviceInvalid {
				errorMsg = "Server is up but no device is available"
				hasError = true
				break
			}

			if f.gotSyncInfo {
				f.onConnect()
				return
			}
		}
		time.Sleep(4 * time.Millisecond)
	}

	f.Disconnect()
	if hasError {
		hasError = false
		panic(errorMsg)
	}

	panic("Server didn't send the device capability and synchronization info.")
}

// Disconnect disconnects from current connected spyserver.
func (f *Spyserver) Disconnect() {
	log.Println("Disconnecting")
	f.terminated = true
	if f.IsConnected {
		f.client.Close()
	}

	f.routineRunning = false

	f.cleanup()
}

// GetSampleRate returns the sample rate of the IQ channel in Hertz
func (f *Spyserver) GetSampleRate() uint32 {
	return f.currentSampleRate
}

// SetSampleRate sets the sample rate of the IQ Channel in Hertz
// Check the available sample rates using GetAvailableSampleRates
// Returns InvalidValue in case of a invalid value in the input
func (f *Spyserver) SetSampleRate(sampleRate uint32) uint32 {
	for i := uint32(0); i < f.deviceInfo.DecimationStageCount; i++ {
		if f.availableSampleRates[i] == sampleRate {
			f.channelDecimationStageCount = i
			f.setSetting(settingIqDecimation, []uint32{i})
			f.currentSampleRate = sampleRate
			if (f.streamingMode == StreamModeFFTOnly || f.streamingMode == StreamModeFFTIQ) && f.currentDisplaySampleRate == 0 {
				f.SetDisplaySampleRate(sampleRate)
			}
			return sampleRate
		}
	}

	return InvalidValue
}

// SetDecimationStage sets the sample rate by using the number of decimation stages.
// Each decimation stage decimates by two, then the total decimation will be defined by 2^stages.
// This is the same as SetSampleRate, but SetSampleRate instead, looks at a pre-filled table of all 2^stages
// decimations that the server supports and applies into the original device sample rate.
func (f *Spyserver) SetDecimationStage(decimation uint32) uint32 {
	if decimation > f.deviceInfo.DecimationStageCount {
		return InvalidValue
	}
	f.channelDecimationStageCount = decimation
	f.setSetting(settingIqDecimation, []uint32{decimation})
	f.currentSampleRate = f.availableSampleRates[decimation]

	return decimation
}

// GetCenterFrequency returns the IQ Channel Center Frequency in Hz
func (f *Spyserver) GetCenterFrequency() uint32 {
	return f.channelCenterFrequency
}

// SetCenterFrequency sets the IQ Channel Center Frequency in Hertz and returns it.
func (f *Spyserver) SetCenterFrequency(centerFrequency uint32) uint32 {
	if f.channelCenterFrequency != centerFrequency {
		f.setSetting(settingIqFrequency, []uint32{centerFrequency})
		f.channelCenterFrequency = centerFrequency
		if (f.streamingMode == StreamModeFFTOnly || f.streamingMode == StreamModeFFTIQ) && f.DisplayCenterFrequency == 0 {
			f.SetDisplayCenterFrequency(centerFrequency)
		}
	}

	return f.channelCenterFrequency
}

// GetDisplayCenterFrequency returns the FFT Display Center Frequency in Hertz
func (f *Spyserver) GetDisplayCenterFrequency() uint32 {
	return f.DisplayCenterFrequency
}

// SetDisplayCenterFrequency sets the FFT Channel Center Frequency in Hertz and returns it.
func (f *Spyserver) SetDisplayCenterFrequency(centerFrequency uint32) uint32 {
	if f.DisplayCenterFrequency != centerFrequency {
		f.setSetting(settingFFTFrequency, []uint32{centerFrequency})
		f.DisplayCenterFrequency = centerFrequency
	}

	return f.DisplayCenterFrequency
}

// SetDisplayOffset sets the FFT Display offset in dB
func (f *Spyserver) SetDisplayOffset(offset int32) {
	if f.displayOffset != offset {
		f.displayOffset = offset
		f.setSetting(settingFFTDbOffset, []uint32{uint32(offset)})
	}
}

// GetDisplayOffset returns the FFT Display offset in dB
func (f *Spyserver) GetDisplayOffset() int32 {
	return f.displayOffset
}

// SetDisplayRange sets the FFT Display range in dB
func (f *Spyserver) SetDisplayRange(dispRange int32) {
	if f.displayRange != dispRange {
		f.displayRange = dispRange
		f.setSetting(settingFFTDbRange, []uint32{uint32(dispRange)})
	}
}

// GetDisplayRange returns the FFT Display range in dB
func (f *Spyserver) GetDisplayRange() int32 {
	return f.displayRange
}

// SetDisplayPixels sets the FFT Display width in pixels
func (f *Spyserver) SetDisplayPixels(pixels uint32) {
	if f.displayPixels != pixels {
		f.displayPixels = pixels
		f.setSetting(settingFFTDisplayPixels, []uint32{pixels})
	}
}

// GetDisplayPixels returns the FFT Display width in pixels
func (f *Spyserver) GetDisplayPixels() uint32 {
	return f.displayPixels
}

// SetStreamingMode sets the streaming mode of the server.
// The valid values are StreamModeIQOnly, StreamModeFFTOnly, StreamModeFFTIQ
func (f *Spyserver) SetStreamingMode(streamMode uint32) {
	if f.streamingMode != streamMode {
		f.streamingMode = streamMode
		f.setSetting(settingStreamingMode, []uint32{streamMode})

		if (f.streamingMode == StreamModeFFTOnly || f.streamingMode == StreamModeFFTIQ) && f.DisplayCenterFrequency == 0 {
			f.SetDisplayCenterFrequency(f.GetCenterFrequency())
		}
		if f.streamingMode == StreamModeFFTOnly || f.streamingMode == StreamModeFFTIQ {
			f.setSetting(settingFFTDecimation, []uint32{f.displayDecimationStageCount})
		}
	}
}

// GetStreamingMode returns the streaming mode of the server.
func (f *Spyserver) GetStreamingMode() uint32 {
	return f.streamingMode
}

// SetCallback sets the callbacks for server data
func (f *Spyserver) SetCallback(cb spytypes.Callback) {
	f.callback = cb
}

// GetAvailableSampleRates returns a list of available sample rates for the current connection.
func (f *Spyserver) GetAvailableSampleRates() []uint32 {
	return f.availableSampleRates
}

// SetDisplaySampleRate sets the sample rate of the FFT Channel in Hertz
// Check the available sample rates using GetAvailableSampleRates
// Returns InvalidValue in case of a invalid value in the input
func (f *Spyserver) SetDisplaySampleRate(sampleRate uint32) uint32 {
	for i := uint32(0); i < f.deviceInfo.DecimationStageCount; i++ {
		if f.availableSampleRates[i] == sampleRate {
			f.displayDecimationStageCount = i
			f.setSetting(settingFFTDecimation, []uint32{i})
			f.currentDisplaySampleRate = sampleRate
			return sampleRate
		}
	}

	return InvalidValue
}

// SetDisplayDecimationStage sets the sample rate of the FFT Channel by using the number of decimation stages.
// Each decimation stage decimates by two, then the total decimation will be defined by 2^stages.
// This is the same as SetSampleRate, but SetSampleRate instead, looks at a pre-filled table of all 2^stages
// decimations that the server supports and applies into the original device sample rate.
// Returns InvalidValue in case of a invalid value in the input
func (f *Spyserver) SetDisplayDecimationStage(decimation uint32) uint32 {
	if decimation > f.deviceInfo.DecimationStageCount {
		return InvalidValue
	}
	f.displayDecimationStageCount = decimation
	f.setSetting(settingFFTDecimation, []uint32{decimation})
	f.currentDisplaySampleRate = f.availableSampleRates[decimation]

	return decimation
}

// GetDisplaySampleRate returns the sample rate of FFT Channel in Hertz
func (f *Spyserver) GetDisplaySampleRate() uint32 {
	return f.currentDisplaySampleRate
}

// GetDisplayBandwidth returns the effective bandwidth of the FFT Channel in Hertz.
// For calculating the frequency of each FFT Pixel Column, you should use this as total FFT Bandwidth.
func (f *Spyserver) GetDisplayBandwidth() uint32 {
	return uint32(float32(f.currentDisplaySampleRate) * 0.8)
}

// SetGain sets the gain stage of the server.
// The actual gain in dB varies from device to device.
// Returns InvalidValue in case of a invalid value in the input
func (f *Spyserver) SetGain(gain uint32) uint32 {
	if gain > f.deviceInfo.GainStageCount {
		return InvalidValue
	}
	f.setSetting(settingGain, []uint32{gain})
	f.gain = gain

	return gain
}

// GetGain returns the current gain stage of the server.
func (f *Spyserver) GetGain() uint32 {
	return f.gain
}

// endregion
