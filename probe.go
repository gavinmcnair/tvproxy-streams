package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Channels     int    `json:"channels"`
	ChannelLayout string `json:"channel_layout"`
	Profile      string `json:"profile"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

func ffprobeFile(path string) *ProbeInfo {
	cmd := exec.Command("ffprobe",
		"-hide_banner",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var probe ffprobeOutput
	if json.Unmarshal(out, &probe) != nil {
		return nil
	}

	info := &ProbeInfo{
		ProbedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, s := range probe.Streams {
		if s.CodecType == "video" && info.VideoCodec == "" {
			info.VideoCodec = normalizeVideoCodec(s.CodecName)
			info.Width = s.Width
			info.Height = s.Height
			info.Resolution = classifyResolution(s.Width, s.Height)
		}
		if s.CodecType == "audio" && info.AudioCodec == "" {
			info.AudioCodec = normalizeAudioCodec(s.CodecName)
			info.AudioLayout = classifyAudioLayout(s.Channels, s.ChannelLayout, s.CodecName, s.Profile)
		}
	}

	if probe.Format.Duration != "" {
		if d, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			info.Duration = d
		}
	}

	return info
}

func normalizeVideoCodec(codec string) string {
	switch strings.ToLower(codec) {
	case "h264", "avc":
		return "H264"
	case "hevc", "h265":
		return "HEVC"
	case "av1":
		return "AV1"
	case "vp9":
		return "VP9"
	case "mpeg2video":
		return "MPEG2"
	case "mpeg4":
		return "MPEG4"
	default:
		return strings.ToUpper(codec)
	}
}

func normalizeAudioCodec(codec string) string {
	switch strings.ToLower(codec) {
	case "aac", "aac_latm":
		return "AAC"
	case "ac3":
		return "AC3"
	case "eac3":
		return "EAC3"
	case "truehd":
		return "TrueHD"
	case "dts":
		return "DTS"
	case "flac":
		return "FLAC"
	case "mp2":
		return "MP2"
	case "mp3":
		return "MP3"
	case "opus":
		return "Opus"
	case "vorbis":
		return "Vorbis"
	default:
		return strings.ToUpper(codec)
	}
}

func classifyResolution(w, h int) string {
	if h >= 2160 || w >= 3840 {
		return "4K"
	}
	if h >= 1080 || w >= 1920 {
		return "1080p"
	}
	if h >= 720 || w >= 1280 {
		return "720p"
	}
	if h >= 480 || w >= 720 {
		return "SD"
	}
	if h > 0 {
		return fmt.Sprintf("%dp", h)
	}
	return ""
}

func classifyAudioLayout(channels int, layout, codec, profile string) string {
	codecLower := strings.ToLower(codec)
	profileLower := strings.ToLower(profile)

	if codecLower == "truehd" {
		if strings.Contains(profileLower, "atmos") || strings.Contains(layout, "7.1") {
			return "Atmos"
		}
		return "TrueHD"
	}
	if codecLower == "eac3" {
		if strings.Contains(profileLower, "atmos") {
			return "Atmos"
		}
		return "DD+"
	}
	if codecLower == "ac3" {
		if channels >= 6 {
			return "DD 5.1"
		}
		return "DD"
	}
	if codecLower == "dts" {
		if strings.Contains(profileLower, "hd ma") || strings.Contains(profileLower, "hd-ma") {
			return "DTS-HD MA"
		}
		if strings.Contains(profileLower, "hd") {
			return "DTS-HD"
		}
		return "DTS"
	}

	switch channels {
	case 8:
		return "7.1"
	case 6:
		return "5.1"
	case 2:
		return "Stereo"
	case 1:
		return "Mono"
	default:
		if channels > 0 {
			return fmt.Sprintf("%dch", channels)
		}
		return ""
	}
}
