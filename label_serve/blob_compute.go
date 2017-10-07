package main

import (
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
  "encoding/hex"
	"errors"
	"github.com/tensorflow/tensorflow/tensorflow/go/op"
	"github.ibm.com/Blue-Horizon/aural2/libaural2"
	"github.ibm.com/Blue-Horizon/aural2/tfutils"
	"io/ioutil"
	"strconv"
)

func getAudioClipFromFS(id libaural2.ClipID) (audioClip *libaural2.AudioClip, err error) {
	rawBytes, err := ioutil.ReadFile("audio/" + id.FSsafeString() + ".raw")
	//rawBytes, err := ioutil.ReadFile("/tmp/raw_cmd_10.raw")
	if len(rawBytes) != len(audioClip) {
		err = errors.New("Got " + strconv.Itoa(len(rawBytes)) + " bytes, expected" + strconv.Itoa(len(audioClip)))
		return
	}
	audioClip = &libaural2.AudioClip{}
	copy(audioClip[:], rawBytes) // convert the slice of bytes to an array of bytes.
	return
}


func makeAddRIFF()(addRIFF func(*libaural2.AudioClip)([]byte), err error){
  headerString := "5249464644b3040057415645666d74201000000001000100803e0000007d0000020010006461746120b30400"
  header, err := hex.DecodeString(headerString)
  if err != nil {
		panic(err)
    return
  }
  addRIFF = func(audioClip *libaural2.AudioClip)[]byte{
    return append(header, audioClip[:]...)
  }
  return
}

func makeComputeSpectrogram() (computeSpectrogram func(*libaural2.AudioClip) ([]byte, error), err error) {
	s := op.NewScope()
	bytesPH, pcm := tfutils.ParseRawBytesToPCM(s)
	specgramOP := tfutils.ComputeSpectrogram(s.SubScope("spectrogram"), pcm, 0, 0)
	specgramJpegBytesOP := tfutils.RenderImage(s.SubScope("jpeg_bytes"), specgramOP)
	feeds := map[tf.Output]*tf.Tensor{}
	renderImage, err := tfutils.BytesToBytes(s, bytesPH, specgramJpegBytesOP, feeds)
	if err != nil {
		return
	}

	computeSpectrogram = func(raw *libaural2.AudioClip) (imageBytes []byte, err error) {
		if raw == nil {
			err = errors.New("raw is nil")
			return
		}
		imageBytes, err = renderImage(raw[:])
		if err != nil {
			logger.Println(err)
			return
		}
		return
	}
	return
}

func makeComputeMFCC() (computeMFCC func(*libaural2.AudioClip) ([]byte, error), err error) {
	s := op.NewScope()
	bytesPH, pcm := tfutils.ParseRawBytesToPCM(s)
	mfccOP, sampleRatePH := tfutils.ComputeMFCC(s.SubScope("spectrogram"), pcm)
	jpegBytesOP := tfutils.RenderImage(s.SubScope("jpeg_bytes"), mfccOP)
	sampleRateTensor, err := tf.NewTensor(int32(libaural2.SampleRate))
	if err != nil {
		return
	}
	feeds := map[tf.Output]*tf.Tensor{sampleRatePH: sampleRateTensor}
	renderImage, err := tfutils.BytesToBytes(s, bytesPH, jpegBytesOP, feeds)
	if err != nil {
		return
	}
	computeMFCC = func(raw *libaural2.AudioClip) (imageBytes []byte, err error) {
		if raw == nil {
			err = errors.New("raw is nil")
			return
		}
		imageBytes, err = renderImage(raw[:])
		if err != nil {
			logger.Println(err)
			return
		}
		return
	}
	return
}
