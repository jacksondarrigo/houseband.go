package stream

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"os/exec"
	"strconv"

	"layeh.com/gopus"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

type Stream struct {
	*gopus.Encoder
	url   string
	Audio chan []byte
}

func New(url string) (*Stream, error) {
	opusEncoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		log.Println("[gopus] NewEncoder Error: ", err)
		return nil, err
	}
	return &Stream{opusEncoder, url, make(chan []byte, 2)}, nil
}

func (stream *Stream) Get() {
	ffmpeg := exec.Command("ffmpeg", "-reconnect", "1", "-nostdin", "-i", stream.url, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	ffmpegStdOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Println("[ffmpeg] StdoutPipe Error: ", err)
		return
	}
	ffmpegBuffer := bufio.NewReaderSize(ffmpegStdOut, 16384)
	err = ffmpeg.Start()
	if err != nil {
		log.Println("[ffmpeg] ExecStart Error: ", err)
		return
	}
	for {
		pcmBytes := make([]int16, frameSize*channels)
		err := binary.Read(ffmpegBuffer, binary.LittleEndian, &pcmBytes)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			log.Println("[ffmpeg] Error: Cannot read from stdout: ", err)
			break
		}
		opusBytes, err := stream.Encode(pcmBytes, frameSize, maxBytes)
		if err != nil {
			log.Println("[gopus] Encoding Error: ", err)
			break
		}
		stream.Audio <- opusBytes
	}
	ffmpeg.Process.Kill()
	close(stream.Audio)
}
