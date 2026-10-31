package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	outBuffer := bytes.Buffer{}
	errBuffer := bytes.Buffer{}
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer
	err := cmd.Run()
	if err != nil {
		log.Fatalf("error running command: %v; stderr: %v", err, cmd.Stderr)
		return "", err
	}
	type Output struct {
		Streams []struct {
			DisplayAspectRatio string `json:"display_aspect_ratio"`
		} `json:"streams"`
	}
	var data Output
	err = json.Unmarshal(outBuffer.Bytes(), &data)
	if err != nil {
		log.Fatalf("error reading buffer data: %v", err)
		return "", err
	}
	if data.Streams[0].DisplayAspectRatio != "9:16" && data.Streams[0].DisplayAspectRatio != "16:9" {
		return "other", nil
	}
	return data.Streams[0].DisplayAspectRatio, nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	outBuffer := bytes.Buffer{}
	errBuffer := bytes.Buffer{}
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer
	err := cmd.Run()
	if err != nil {
		log.Fatalf("error running command: %v; stderr: %v", err, cmd.Stderr)
		return "", err
	}
	return outputPath, nil
}
