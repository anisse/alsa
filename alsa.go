//oto-compatible (mostly) CGO-less alsa library
package alsa

import (
	"fmt"
	"math/bits"
	"syscall"
	"unsafe"
)

type device struct {
	fd         uintptr
	channels   uint32
	format     uint32
	sampleRate uint32
}

type snd_pcm_info struct {
	device                uint32
	subdevice             uint32
	stream                int32
	card                  uint32
	id                    [64]uint8
	name                  [80]uint8
	subname               [32]uint8
	dev_class             int32
	dev_subclass          int32
	subdevices_count      uint32
	subdevices_avail      uint32
	union_snd_pcm_sync_id [4]uint32
	reserved              [64]byte
}

const (
	sndMaskSize       = (256 + 31) / 32
	sndTotalMasks     = 3
	sndTotalIntervals = 12
)

type snd_mask struct {
	bits [sndMaskSize]uint32
}

const (
	intervalOpenmin = 1 << iota
	intervalOpenmax
	intervalInteger
	intervalEmpty
)

type snd_interval struct {
	min      uint32
	max      uint32
	bitfield uint32 // contains openmin, openmax, integer, empty
}

type snd_pcm_uframes uint //TODO: this is "unsigned long", maybe to fix that we must use per-arch uint32/uint64 ; we don't use it, but we risk an overflow here
type snd_pcm_sframes int

type snd_pcm_hw_params struct {
	flags     uint32
	masks     [sndTotalMasks]snd_mask
	mres      [5]snd_mask
	intervals [sndTotalIntervals]snd_interval
	ires      [9]snd_interval
	rmask     uint32
	cmask     uint32
	info      uint32
	msbits    uint32
	rate_num  uint32
	rate_den  uint32
	fifo_size snd_pcm_uframes
	reserved  [64]byte
}

type snd_pcm_sw_params struct {
	tstamp_mode       int32
	period_step       uint32
	sleep_min         uint32
	avail_min         snd_pcm_uframes
	xfer_align        snd_pcm_uframes
	start_threshold   snd_pcm_uframes
	stop_threshold    snd_pcm_uframes
	silence_threshold snd_pcm_uframes
	silence_size      snd_pcm_uframes
	boundary          snd_pcm_uframes
	proto             uint32
	tstamp_type       uint32
	reserved          [56]byte
}

type snd_xferi struct {
	result snd_pcm_sframes
	buf    uintptr
	frames snd_pcm_uframes
}

const (
	sndrvPcmIoctlInfo         = 0x81204101
	sndrvPcmIoctlPrepare      = 0x004140
	sndrvPcmIoctlHwParams     = 0xc2604111
	sndrvPcmIoctlSwParams     = 0xc0884113
	sndrvPcmIoctlWriteiFrames = 0x40184150
)

const (
	//TODO: add more formats, but this might need converting
	// my HDA codec does not support anything besides S16_LE, S20_LE and S24_LE
	//pcmFormatS8      = 0
	pcmFormatS16Le = 2
	//pcmFormatFloatLe = 14
)

const (
	maskParamAccess = iota
	maskParamFormat
	maskParamSubformat
)

const (
	accessMmapInterleaved = iota
	accessMmapNoninterleaved
	accessMmapComplex
	accessRwInterleaved
	accessRwNoninterleaved
)

const (
	paramSampleBits = iota
	paramFrameBits
	paramChannels
	paramRate
	paramPeriodTime
	paramPeriodSize
	paramPeriodBytes
	paramPeriods
	paramBufferTime
	paramBufferSize
	paramBufferBytes
	paramTickTime
)

func ioctl(fd, req, data uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, data)
	if errno != 0 {
		return errno
	}
	return nil
}

func openDevice(devName string) (*device, error) {
	f, err := syscall.Open(devName, syscall.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot open alsa device %s: %s", devName, err.Error())
	}

	var info snd_pcm_info
	err = ioctl(uintptr(f), sndrvPcmIoctlInfo, uintptr(unsafe.Pointer(&info)))
	if err != nil {
		return nil, fmt.Errorf("cannot get pcm info for dev %s: %s", devName, err.Error())
	}
	/*
		fmt.Println(info)
		fmt.Println(string(info.id[:]))
		fmt.Println(string(info.name[:]))
		fmt.Println(string(info.subname[:]))
	*/

	return &device{fd: uintptr(f)}, nil
}

func (p *snd_pcm_hw_params) setInteger(param, val uint32) {
	p.intervals[param].min = val
	p.intervals[param].max = val
	p.intervals[param].bitfield = intervalInteger
}

func (p *snd_pcm_hw_params) setMin(param, val uint32) {
	p.intervals[param].min = val
}

func (p *snd_pcm_hw_params) setMask(mask, val uint32) {
	p.masks[mask].bits[0] = 0
	p.masks[mask].bits[1] = 0
	p.masks[mask].bits[val>>5] |= (1 << (val & 0x1f))
}
func formatBits(format uint32) uint32 {
	switch format {
	default:
		fallthrough
	case pcmFormatS16Le:
		return 16
		/*
			case pcmFormatS8:
				return 8
			case pcmFormatFloatLe:
				return 32
		*/
	}
}
func (dev *device) sampleSize() uint32 {
	return formatBits(uint32(dev.format)) / 8 * dev.channels
}
func (dev *device) setConfig(channels, rate uint32) error {
	const (
		format      = pcmFormatS16Le
		periodCount = 4
		periodSize  = 1024
	)

	hwParams := &snd_pcm_hw_params{
		flags: 0,
		rmask: 0xFFFFFFFF,
		cmask: 0,
		info:  0xFFFFFFFF,
	}
	for i := 0; i < len(hwParams.masks); i++ {
		hwParams.masks[i].bits[0] = 0xFFFFFFFF
		hwParams.masks[i].bits[1] = 0xFFFFFFFF
	}
	for i := 0; i < len(hwParams.intervals); i++ {
		hwParams.intervals[i].min = 0
		hwParams.intervals[i].max = 0xFFFFFFFF
	}
	hwParams.setInteger(paramChannels, channels)
	hwParams.setInteger(paramRate, rate)
	hwParams.setMask(maskParamFormat, format)
	hwParams.setMask(maskParamSubformat, 0)
	/* Other, default values */
	hwParams.setMin(paramPeriodSize, periodSize)
	hwParams.setInteger(paramSampleBits, formatBits(format))
	hwParams.setInteger(paramFrameBits, formatBits(format)*channels)
	hwParams.setInteger(paramPeriods, periodCount)
	hwParams.setMask(maskParamAccess, accessRwInterleaved)

	err := ioctl(dev.fd, sndrvPcmIoctlHwParams, uintptr(unsafe.Pointer(hwParams)))
	if err != nil {
		return fmt.Errorf("could not set hw params: %s", err.Error())
	}

	hwPeriodCount := hwParams.intervals[paramPeriods].max
	hwPeriodSize := hwParams.intervals[paramPeriodSize].max
	threshold := snd_pcm_uframes(hwPeriodSize * hwPeriodCount)

	swParams := &snd_pcm_sw_params{
		period_step:     1,
		avail_min:       1,
		tstamp_mode:     1,
		start_threshold: threshold,
		stop_threshold:  threshold,
		boundary:        threshold << uint(bits.LeadingZeros(uint(threshold))),
	}

	err = ioctl(dev.fd, sndrvPcmIoctlSwParams, uintptr(unsafe.Pointer(swParams)))
	if err != nil {
		return fmt.Errorf("could not set sw params: %s", err.Error())
	}

	err = ioctl(dev.fd, sndrvPcmIoctlPrepare, 0)
	if err != nil {
		return fmt.Errorf("could not prepare device: %s", err.Error())
	}

	return nil
}

func (dev *device) write(frames []byte) (int, error) {
	xfer := &snd_xferi{
		frames: snd_pcm_uframes(len(frames) / int(dev.sampleSize())),
		buf:    uintptr(unsafe.Pointer(&frames[0])),
	}
	err := ioctl(dev.fd, sndrvPcmIoctlWriteiFrames, uintptr(unsafe.Pointer(xfer)))
	written := int(xfer.result) * int(dev.sampleSize())
	if err == syscall.EPIPE || err == syscall.EAGAIN {
		return written, err
	} else if err != nil {
		return written, fmt.Errorf("writing frame data: %s", err.Error())
	}

	return written, nil
}

type Player struct {
	*device
}

//TODO: bufferSize is ignored, pass it down as period_size
func NewPlayer(sampleRate, channelNum, bytesPerSample, bufferSizeInBytes int) (*Player, error) {
	if channelNum != 2 {
		return nil, fmt.Errorf("only two channel mode is supported, please convert your samples")
	}
	if bytesPerSample != 2 {
		return nil, fmt.Errorf("only signed 16 bit PCM samples are supported, please convert your samples")
	}
	if sampleRate != 44100 {
		return nil, fmt.Errorf("only 44100 Hz is supported, please resample")
	}
	dev, err := openDevice("/dev/snd/pcmC0D0p")
	if err != nil {
		return nil, err
	}
	dev.format = pcmFormatS16Le
	dev.channels = uint32(channelNum)
	dev.sampleRate = uint32(sampleRate)
	err = dev.setConfig(dev.channels, dev.sampleRate)
	if err != nil {
		return nil, err
	}
	return &Player{dev}, nil
}
func (p *Player) Close() error {
	return syscall.Close(int(p.fd))
}
func (p *Player) Write(data []byte) (int, error) {
	return p.write(data)
}
