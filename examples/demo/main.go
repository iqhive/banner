// Command demo exercises banner without depending on any consuming project.
package main

import (
	"os"
	"time"

	"github.com/iqhive/banner"
)

func build(seed uint64) (banner.Animation, error) {
	frames, card, err := banner.Opening(banner.Card{
		Word: "banner", Tagline: "Reusable terminal banners for Go.",
		Supporting: "shared opening · themes · SVG and terminal playback",
		Install:    "go get github.com/iqhive/banner", Wordmark: banner.WordmarkOptions{Seed: seed},
		ReadHold: 2500 * time.Millisecond,
	})
	if err != nil {
		return banner.Animation{}, err
	}
	card.Hold = 4 * time.Second
	frames = append(frames, card)
	static := len(frames) - 1
	logo := banner.Frame{Hold: time.Second, HiveLogo: true}
	logo.Center(14, "IQ Hive", banner.Bright)
	logo.Center(16, "iqhive.com", banner.Muted)
	frames = append(frames, logo)
	return banner.Animation{Frames: frames, StaticFrame: static, Title: "banner", Description: "The shared opening reveals the banner wordmark, then a readable project card and the IQ Hive logo. Reduced motion shows the project card.", Command: "go run ./examples/demo"}, nil
}
func main() { os.Exit(banner.RunCLI(os.Args[1:], os.Stdout, os.Stderr, build)) }
