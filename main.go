package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

type FOURCC = uint32
type WORD = uint16
type DWORD = uint32
type SHORT = int16
type LONG = int32

type RECT struct {
	Left   SHORT `json:"left"`
	Top    SHORT `json:"top"`
	Right  SHORT `json:"right"`
	Bottom SHORT `json:"bottom"`
}

type _avimainheader struct {
	// fcc                   FOURCC
	// cb                    WORD
	DwMicroSecPerFrame    DWORD    `json:"dwMicroSecPerFrame"`
	DwMaxBytesPerSec      DWORD    `json:"dwMaxBytesPerSec"`
	DwPaddingGranularity  DWORD    `json:"dwPaddingGranularity"`
	DwFlags               DWORD    `json:"dwFlags"`
	DwTotalFrame          DWORD    `json:"dwTotalFrame"`
	DwInitialFrames       DWORD    `json:"dwInitialFrames"`
	DwStreams             DWORD    `json:"dwStreams"`
	DwSuggestedBufferSize DWORD    `json:"dwSuggestedBufferSize"`
	DwWidth               DWORD    `json:"dwWidth"`
	DwHeight              DWORD    `json:"dwHeight"`
	DwReserved            [4]DWORD `json:"dwReserved"`
}

type AVIStreamHeader struct {
	FccType               FOURCC `json:"fccType"`
	FccHandler            FOURCC `json:"fccHandler"`
	DwFlags               DWORD  `json:"dwFlags"`
	WPriority             WORD   `json:"wPriority"`
	WLanguage             WORD   `json:"wLanguage"`
	DwInitialFrames       DWORD  `json:"dwInitialFrames"`
	DwScale               DWORD  `json:"dwScale"`
	DwRate                DWORD  `json:"dwRate"`
	DwStart               DWORD  `json:"dwStart"`
	DwLength              DWORD  `json:"dwLength"`
	DwSuggestedBufferSize DWORD  `json:"dwSuggestedBufferSize"`
	DwQuality             DWORD  `json:"dwQuality"`
	DwSampleSize          DWORD  `json:"dwSampleSize"`
	RcFrame               RECT   `json:"rcFrame"`
}

type BITMAPINFOHEADER struct {
	DwSize         DWORD `json:"biSize"`
	LWidth         LONG  `json:"biWidth"`
	LHeight        LONG  `json:"biHeight"`
	WPlanes        WORD  `json:"biPlanes"`
	WBitCount      WORD  `json:"biBitCount"`
	DwCompression  DWORD `json:"biCompression"`
	DwSizeImage    DWORD `json:"biSizeImage"`
	LXPelsPerMeter LONG  `json:"biXPelsPerMeter"`
	LYPelsPerMeter LONG  `json:"biYPelsPerMeter"`
	DwClrUsed      DWORD `json:"biClrUsed"`
	DwClrImportant DWORD `json:"biClrImportant"`
}

type WAVEFORMATEX struct {
	WFormatTag      WORD  `json:"wFormatTag"`
	NChannels       WORD  `json:"nChannels"`
	NSamplesPerSec  DWORD `json:"nSamplesPerSec"`
	NAvgBytesPerSec DWORD `json:"nAvgBytesPerSec"`
	NBlockAlign     WORD  `json:"nBlockAlign"`
	WBitsPerSample  WORD  `json:"wBitsPerSample"`
	CbSize          WORD  `json:"cbSize"`
}

const RIFF = 0x46464952
const FILE_TYPE = 0x20495641
const FILE_SIZE = 1625548
const LIST = 0x5453494C
const vids = 0x73646976
const auds = 0x73647561
const STRF = 0x66727473
const STRH = 0x68727473
const avih = 0x68697661
const dc = 0x63643030
const wb = 0x62773130

var f *os.File

var objType int
var chanCfg int
var freqIdx int

func chopFourBytes(content *[]byte) uint32 {

	data := (*content)[0:4]
	*content = (*content)[4:]
	return binary.LittleEndian.Uint32(data)

}

func printTostring(content uint32) {

	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, content)
	fmt.Printf("%s\n", b)
}

func printToHex(content uint32, info string) {
	if info != "" {
		fmt.Printf("%s = %X\n", info, content)
		return
	}
	fmt.Printf("%X\n", content)

}

func pretteyPrintStruct(T any) {
	d, _ := json.MarshalIndent(T, "", "  ")
	fmt.Printf("%T\n%s\n", T, d)
}

func parse[T any](buf []byte) *T {
	// Equivalent of: mem::transmute::<*const u8, &'a T>(chunk.content.as_ptr())
	return (*T)(unsafe.Pointer(&buf[0]))
}

func writeTofile(content uint32) {

	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, content)
	if _, err := f.WriteString(fmt.Sprintf("%s", b) + "\n"); err != nil {
		panic(err)
	}
}

// func init() {
// 	var err error
// 	f, err = os.OpenFile("example.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		panic(err)
// 	}
// }

func dumpHeader[T any](body []byte) any {
	var h T
	binary.Read(bytes.NewReader(body), binary.LittleEndian, &h)
	pretteyPrintStruct(h)

	return h

}

func adtsHeader(objType, freqIdx, chanCfg, payloadLen int) []byte {
	frameLen := payloadLen + 7
	profile := objType - 1 // ADTS stores object type minus 1
	return []byte{
		0xFF,
		0xF1, // sync, MPEG-4, layer 0, no CRC
		byte(profile<<6 | freqIdx<<2 | (chanCfg>>2)&1),
		byte((chanCfg&3)<<6 | (frameLen>>11)&3),
		byte(frameLen >> 3),
		byte((frameLen&7)<<5 | 0x1F),
		0xFC,
	}
}
func main() {
	fmt.Println(os.Args)

	if len(os.Args) < 2 {
		fmt.Println("Source file is required")
		fmt.Println("Usage :  Program_name  [source.avi]")
		return
	}

	filepath := os.Args[1]
	arr := strings.Split(filepath, ".")
	ext := arr[len(arr)-1]

	if ext != "avi" {
		fmt.Printf("Invalid source type required .avi got .%v\n", ext)
		return
	}
	vf, _ := os.Create("out.h264")
	af, _ := os.Create("out.aac")

	content, err := os.ReadFile(filepath)

	if err != nil {
		panic("cannot read file")
	}

	riff := chopFourBytes(&content)

	if riff != RIFF {
		panic("not a fiff file")
	}

	filesize := chopFourBytes(&content)

	fmt.Printf("filesize = %d\n", filesize)

	filetype := chopFourBytes(&content)

	if filetype != FILE_TYPE {
		panic("invalid filetype")
	}
	var fccType uint32
	videos := 0
	audios := 0
	empty := 0
	emptyVideo := 0
	lastTick := -1
	tick := 0

	gaps := map[int]int{}
	var startCode = []byte{0, 0, 0, 1}

	for len(content) >= 8 {
		id := chopFourBytes(&content)
		size := chopFourBytes(&content)

		if id == LIST {

			chopFourBytes(&content)
			continue

		}

		body := content[:size]
		content = content[size:]

		if size%2 != 0 && len(content) > 0 {
			content = content[1:]
		}

		switch id {
		case avih:
			dumpHeader[_avimainheader](body)
		case STRF:

			if fccType == vids {

				dumpHeader[BITMAPINFOHEADER](body[:40])
				extra := body[40:]

				if extra[4]&3 != 3 {
					panic("not 4-byte NAL lengths")
				}
				p := 6
				for i := 0; i < int(extra[5]&0x1f); i++ { // SPS
					n := int(binary.BigEndian.Uint16(extra[p:]))
					p += 2
					vf.Write(startCode)
					vf.Write(extra[p : p+n])
					p += n
				}
				numPPS := int(extra[p])
				p++
				for i := 0; i < numPPS; i++ { // PPS
					n := int(binary.BigEndian.Uint16(extra[p:]))
					p += 2
					vf.Write(startCode)
					vf.Write(extra[p : p+n])
					p += n
				}

			}
			if fccType == auds {

				dumpHeader[WAVEFORMATEX](body)
				asc := body[18:20]
				objType = int(asc[0] >> 3)
				freqIdx = int(asc[0]&7)<<1 | int(asc[1]>>7)
				chanCfg = int(asc[1]>>3) & 0xF

				fmt.Printf("asc = %x\n", asc)

			}
		case STRH:

			s := dumpHeader[AVIStreamHeader](body).(AVIStreamHeader)
			fccType = s.FccType

		case dc:

			videos++

			if len(body) == 0 {
				emptyVideo++
			} else {

				if lastTick >= 0 {
					gaps[tick-lastTick]++
				}
				lastTick = tick

			}

			tick++
			b := body
			for len(b) >= 4 {
				n := int(binary.BigEndian.Uint32(b)) // BigEndian, not Little!
				b = b[4:]
				if n > len(b) {
					panic("bad NAL length")
				}
				vf.Write(startCode)
				vf.Write(b[:n])
				b = b[n:]
			}
		case wb:

			// in the case:
			if len(body) == 0 {
				empty++
				break
			}
			// after the loop:

			if audios == 0 {

				fmt.Println(len(body), size)

			}
			af.Write(adtsHeader(objType, freqIdx, chanCfg, len(body)))
			af.Write(body)
			audios++
		default:
			printTostring(id)
			printToHex(id, "id")
			// panic("unknown id")

		}

	}

	fmt.Printf("videos = %v\n audios = %v\n", videos, audios)
	fmt.Println("empty audio chunks:", empty)
	fmt.Println("empty video chunks:", emptyVideo)
	fmt.Println(gaps)

	b, _ := os.ReadFile("out.aac")
	off, n := 0, 0
	for off+7 <= len(b) {
		if b[off] != 0xFF || b[off+1]&0xF0 != 0xF0 {
			fmt.Println("lost sync at frame", n, "offset", off)
			break
		}
		l := int(b[off+3]&3)<<11 | int(b[off+4])<<3 | int(b[off+5])>>5

		if l <= 7 {
			fmt.Println("empty frame at", n, "offset", off)
		}
		if l < 7 || off+l > len(b) {
			fmt.Println("bad/truncated frame", n, "len", l, "offset", off, "file", len(b))
			break
		}
		off += l
		n++
	}
	fmt.Println("frames:", n, "leftover bytes:", len(b)-off)

}
