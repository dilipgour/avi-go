package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"unsafe"
)

type FOURCC = uint32
type WORD = uint16
type DWORD = uint32
type SHORT = int16

type RECT struct {
	Left   SHORT `json:"left"`
	Top    SHORT `json:"top"`
	Right  SHORT `json:"right"`
	Bottom SHORT `json:"bottom"`
}

type _avimainheader struct {
	// fcc                   FOURCC
	// cb                    WORD
	DwMicroSecPerFrame    uint32    `json:"dwMicroSecPerFrame"`
	DwMaxBytesPerSec      uint32    `json:"dwMaxBytesPerSec"`
	DwPaddingGranularity  uint32    `json:"dwPaddingGranularity"`
	DwFlags               uint32    `json:"dwFlags"`
	DwTotalFrame          uint32    `json:"dwTotalFrame"`
	DwInitialFrames       uint32    `json:"dwInitialFrames"`
	DwStreams             uint32    `json:"dwStreams"`
	DwSuggestedBufferSize uint32    `json:"dwSuggestedBufferSize"`
	DwWidth               uint32    `json:"dwWidth"`
	DwHeight              uint32    `json:"dwHeight"`
	DwReserved            [4]uint32 `json:"dwReserved"`
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
	DwSize         uint32 `json:"biSize"`
	LWidth         int32  `json:"biWidth"`
	LHeight        int32  `json:"biHeight"`
	WPlanes        WORD   `json:"biPlanes"`
	WBitCount      WORD   `json:"biBitCount"`
	DwCompression  DWORD  `json:"biCompression"`
	DwSizeImage    DWORD  `json:"biSizeImage"`
	LXPelsPerMeter int32  `json:"biXPelsPerMeter"`
	LYPelsPerMeter int32  `json:"biYPelsPerMeter"`
	DwClrUsed      DWORD  `json:"biClrUsed"`
	DwClrImportant DWORD  `json:"biClrImportant"`
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
	fmt.Println(string(d))

}

func parse[T any](buf []byte) *T {
	// Equivalent of: mem::transmute::<*const u8, &'a T>(chunk.content.as_ptr())
	return (*T)(unsafe.Pointer(&buf[0]))
}

func unused(t any) {
	//fuck you
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
func main() {

	vf, _ := os.Create("out.h264")
	af, _ := os.Create("out.aac")

	content, err := os.ReadFile("./output.avi")

	if err != nil {
		panic("cannot read file")
	}

	riff := chopFourBytes(&content)

	if riff != RIFF {
		panic("not a fiff file")
	}

	filesize := chopFourBytes(&content)
	unused(filesize)
	fmt.Printf("filesize = %d\n", filesize)

	filetype := chopFourBytes(&content)

	if filetype != FILE_TYPE {
		panic("invalid filetype")
	}
	var fccType uint32
	videos := 0
	audios := 0
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

			}
		case STRH:

			s := dumpHeader[AVIStreamHeader](body).(AVIStreamHeader)

			fccType = s.FccType
			pretteyPrintStruct(s)
		case dc:

			videos++
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
			af.Write(body)
			audios++
		default:
			printTostring(id)
			printToHex(id, "id")
			// panic("unknown id")

		}

	}

	fmt.Printf("videos = %v\n audios = %v\n", videos, audios)

}
