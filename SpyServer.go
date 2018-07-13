package spy2go

import (
	"log"
	"net"
	"fmt"
	"time"
	"encoding/binary"
	"bytes"
)

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

type Spyserver struct {

	hostname string
	port int
	callback *CallbackBase
	client net.Conn

	terminated bool
	routineRunning bool
	gotDeviceInfo bool
	gotSyncInfo bool
	streamingMode uint32
	gain uint32

	availableSampleRates []uint32

	parserPhase uint32
	deviceInfo DeviceInfo
	header messageHeader
	lastSequenceNumber uint32
	droppedBuffers uint32
	downStreamBytes uint64
	parserPosition uint32
	bodyBuffer []uint8
	headerBuffer []uint8

	Streaming bool
	CanControl bool
	IsConnected bool
	DroppedBuffers uint32

	MinimumTunableFrequency uint32
	MaximumTunableFrequency uint32
	DeviceCenterFrequency   uint32
	channelCenterFrequency  uint32
	DisplayCenterFrequency  uint32

	currentSampleRate uint32
	channelDecimationStageCount uint32
	displayOffset uint32
	displayRange uint32
	displayPixels uint32

	msgChannel chan []uint8
}

func MakeSpyserver(hostname string, port int) *Spyserver {
	return &Spyserver{
		hostname: hostname,
		port: port,
		callback: nil,
		terminated: false,
		gotDeviceInfo: false,
		gotSyncInfo: false,
		parserPhase: parserAcquiringHeader,
		Streaming: false,
		CanControl: false,
		IsConnected: false,
		availableSampleRates: []uint32{},
		headerBuffer: make([]uint8, messageHeaderSize),

		displayOffset: 0,
		displayRange: DefaultFFTRange,
		displayPixels: DefaultDisplayPixels,
		streamingMode: StreamModeIQOnly,
	}
}

// region Private Methods
func (f *Spyserver) sayHello() bool {
	var totalLength = 4
	var softwareVersionBytes = []byte(SoftwareID)
	totalLength += len(softwareVersionBytes)


	var argBytes = make([]uint8, totalLength)
	binary.LittleEndian.PutUint32(argBytes[0:], SpyserverProtocolVersion)
	for i := 4; i < totalLength; i++ {
		argBytes[i] = softwareVersionBytes[i-4]
	}

	return f.sendCommand(cmdHello, argBytes)
}
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
func (f *Spyserver) onConnect() {
	f.setSetting(settingStreamingMode, []uint32 { f.streamingMode })
	f.setSetting(settingIqFormat, []uint32 { StreamFormatInt16 })
	f.setSetting(settingFFTFormat, []uint32 { StreamFormatUint8 })
	f.setSetting(settingFFTDisplayPixels, []uint32 { f.displayPixels })
	f.setSetting(settingFFTDbOffset, []uint32 { f.displayOffset })
	f.setSetting(settingFFTDbRange, []uint32 { f.displayRange })

	var sampleRates = make([]uint32, f.deviceInfo.DecimationStageCount)
	for i := uint32(0); i < f.deviceInfo.DecimationStageCount; i++ {
		var decim = uint32(1 << i)
		sampleRates[i] = uint32(float32(f.deviceInfo.MaximumSampleRate) / float32(decim))
	}
	f.availableSampleRates = sampleRates
}
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
		BodySize: argsLen,
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

				if clientMajor != serverMajor || clientMinor != serverMinor {
					panic("Server is running an unsupported protocol version.")
				}

				if f.header.BodySize > SpyserverMaxMessageBodySize {
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
		toWrite := min(messageHeaderSize - f.parserPosition, uint32(len(buffer)))
		for i := f.parserPosition; i < toWrite; i++ {
			f.headerBuffer[i] = buffer[i-f.parserPosition]
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
		toWrite := min(f.header.BodySize - f.parserPosition, uint32(len(buffer)))
		for i := f.parserPosition; i < toWrite; i++ {
			f.bodyBuffer[i] = buffer[i-f.parserPosition]
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
	var dInfo = DeviceInfo{}

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
	f.channelCenterFrequency = clientSync.IQCenterFrequency
	f.DisplayCenterFrequency = clientSync.FFTCenterFrequency

	if f.streamingMode == StreamModeFFTOnly || f.streamingMode == StreamModeFFTIQ {
		f.MinimumTunableFrequency = clientSync.MinimumFFTCenterFrequency
		f.MaximumTunableFrequency = clientSync.MaximumFFTCenterFrequency
	} else if f.streamingMode == StreamModeIQOnly {
		f.MinimumTunableFrequency = clientSync.MinimumIQCenterFrequency
		f.MaximumTunableFrequency = clientSync.MaximumIQCenterFrequency
	}

	f.gotSyncInfo = true

	if f.callback != nil {
		(*f.callback).OnDeviceSync()
	}
}
func (f *Spyserver) processUInt8Samples() {
	var sampleCount = f.header.BodySize / 2

	if f.callback != nil  && f.callback.OnUInt8IQ != nil{
		var u8arr = make([]ComplexUInt8, sampleCount)
		buf := bytes.NewBuffer(f.bodyBuffer)

		var tmp = make([]uint8, sampleCount * 2)
		binary.Read(buf, binary.LittleEndian, &tmp)

		for i := uint32(0); i < sampleCount; i++ {
			u8arr[i] = ComplexUInt8{
				real: tmp[i*2],
				imag: tmp[i*2+1],
			}
		}

		(*f.callback).OnUInt8IQ(u8arr)
	}
}
func (f *Spyserver) processInt16Samples() {
	var sampleCount = f.header.BodySize / 4

	if f.callback != nil && f.callback.OnInt16IQ != nil {
		var c16arr= make([]ComplexInt16, sampleCount)
		buf := bytes.NewBuffer(f.bodyBuffer)

		var tmp = make([]int16, sampleCount * 2)
		binary.Read(buf, binary.LittleEndian, &tmp)

		for i := uint32(0); i < sampleCount; i++ {
			c16arr[i] = ComplexInt16{
				real: tmp[i*2],
				imag: tmp[i*2+1],
			}
		}

		(*f.callback).OnInt16IQ(c16arr)
	}
}
func (f *Spyserver) processFloatSamples() {
	var sampleCount = f.header.BodySize / 8

	if f.callback != nil && f.callback.OnFloatIQ != nil {
		var c64arr= make([]complex64, sampleCount)
		buf := bytes.NewBuffer(f.bodyBuffer)

		for i := uint32(0); i < sampleCount; i++ {
			binary.Read(buf, binary.LittleEndian, &c64arr[i])
		}

		(*f.callback).OnFloatIQ(c64arr)
	}
}
func (f *Spyserver) processUInt8FFT() {
	if f.callback != nil && f.callback.OnFFT != nil {
		(*f.callback).OnFFT(f.bodyBuffer)
	}
}
func (f * Spyserver) handleNewMessage() {
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

	buffer := make([]uint8, 64 * 1024)

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
func (f *Spyserver) GetName() string {
	return DeviceName[f.deviceInfo.DeviceType]
}
func (f *Spyserver) Start() {
	if ! f.Streaming {
		log.Println("Starting streaming")
		f.Streaming = true
		f.downStreamBytes = 0
		f.setStreamState()
	}
}
func (f *Spyserver) Stop() {
	if f.Streaming {
		f.Streaming = false
		f.downStreamBytes = 0
		f.setStreamState()
	}
}
func (f *Spyserver) Connect() {
	if f.routineRunning {
		return
	}

	log.Println("Trying to connect")
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", f.hostname, f.port))
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
	for i := 0; i < 1000 && !hasError; i ++ {
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
		time.Sleep(time.Millisecond)
	}

	f.Disconnect()
	if hasError {
		hasError = false
		panic(errorMsg)
	}

	panic("Server didn't send the device capability and synchronization info.")
}
func (f * Spyserver) Disconnect() {
	log.Println("Disconnecting")
	f.terminated = true
	if f.IsConnected {
		f.client.Close()
	}

	f.routineRunning = false

	f.cleanup()
}
func (f *Spyserver) GetSampleRate() uint32 {
	return f.currentSampleRate
}
func (f *Spyserver) SetSampleRate(sampleRate uint32) uint32 {
	for i := uint32(0); i < f.deviceInfo.DecimationStageCount; i++ {
		if f.availableSampleRates[i] == sampleRate {
			f.channelDecimationStageCount = i
			f.setSetting(settingIqDecimation, []uint32{i})
			return sampleRate
		}
	}

	return 0xFFFFFFFF
}
func (f *Spyserver) GetCenterFrequency() uint32 {
	return f.channelCenterFrequency
}
func (f *Spyserver) SetCenterFrequency(centerFrequency uint32) uint32 {
	if f.channelCenterFrequency != centerFrequency {
		f.setSetting(settingIqFrequency, []uint32{centerFrequency})
		f.channelCenterFrequency = centerFrequency
	}

	return f.channelCenterFrequency
}
func (f *Spyserver) SetDisplayOffset(offset uint32) {
	if f.displayOffset != offset {
		f.displayOffset = offset
		f.setSetting(settingFFTDbOffset, []uint32{offset})
	}
}
func (f *Spyserver) GetDisplayOffset() uint32 {
	return f.displayOffset
}
func (f *Spyserver) SetDisplayRange(dispRange uint32) {
	if f.displayRange != dispRange {
		f.displayRange = dispRange
		f.setSetting(settingFFTDbRange, []uint32{dispRange})
	}
}
func (f *Spyserver) GetDisplayRange() uint32 {
	return f.displayRange
}
func (f *Spyserver) SetDisplayPixels(pixels uint32) {
	if f.displayPixels != pixels {
		f.displayPixels = pixels
		f.setSetting(settingFFTDisplayPixels, []uint32{pixels})
	}
}
func (f *Spyserver) GetDisplayPixels() uint32 {
	return f.displayPixels
}
func (f *Spyserver) SetStreamingMode(streamMode uint32) {
	if f.streamingMode != streamMode {
		f.streamingMode = streamMode
		f.setSetting(settingStreamingMode, []uint32 {streamMode})
	}
}
func (f *Spyserver) GetStreamingMode() uint32 {
	return f.streamingMode
}

func (f *Spyserver) SetCallback(cb *CallbackBase) {
	f.callback = cb
}
func (f *Spyserver) GetAvailableSampleRates() []uint32 {
	return f.availableSampleRates
}
// endregion