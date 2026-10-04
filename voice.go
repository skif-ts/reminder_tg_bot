package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func downloadFile(url string, filename string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func testWhisper() {
	cmd := exec.Command(
		`c:\cpp\whisper.cpp\build\bin\Release\whisper-cli.exe`,
		"-m",
		`c:\cpp\whisper.cpp\ggml-base.bin`,
		"-f",
		`c:\cpp\whisper.cpp\voice.wav`,
		"-l",
		"ru",
	)

	output, err := cmd.Output()
	if err != nil {
		log.Println(err)
		return
	}

	log.Println(string(output))
}

func cleanWhisperOutput(output string) string {
	lines := strings.Split(output, "\n")

	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Убираем временной диапазон:
		// [00:00:00.000 --> 00:00:05.120]
		if strings.HasPrefix(line, "[") {
			if end := strings.Index(line, "]"); end != -1 {
				line = strings.TrimSpace(line[end+1:])
			}
		}

		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, " ")
}

func convertToWav(input, output string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", input,
		"-ar", "16000",
		"-ac", "1",
		output,
	)

	return cmd.Run()
}

func transcribeVoice(wavFile string) (string, error) {
	cmd := exec.Command(
		`C:\cpp\whisper.cpp\build\bin\Release\whisper-cli.exe`,
		"-m",
		`C:\cpp\whisper.cpp\ggml-base.bin`,
		"-f",
		wavFile,
		"-l",
		"ru",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	text := cleanWhisperOutput(string(output))

	return text, nil
}

func processVoice(inputOga, outputWav string) (string, error) {
	err := convertToWav(inputOga, outputWav)
	if err != nil {
		return "", fmt.Errorf("ошибка FFmpeg: %w", err)
	}

	text, err := transcribeVoice(outputWav)
	if err != nil {
		return "", fmt.Errorf("ошибка Whisper: %w", err)
	}

	return text, nil
}
