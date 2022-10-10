package utils

import (
	"os"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type Taskset func(progressbar.ProgressBar)

type Progress struct {
	Description string
	Count       int
	Style       progressbar.Theme
	ShowBytes   bool
	ColorCodes  bool
	Width       int
}

func (p Progress) init() {
	if p.Show
}

func (p Progress) NewBar() *progressbar.ProgressBar {

	writer := ansi.NewAnsiStdout()
	if p.ColorCodes == true {
		writer = os.Stdout
	}

	//for i := 0; i < len(p.Counts); i++ {
	bar := progressbar.NewOptions((p.Count),
		progressbar.OptionSetWriter(writer),
		progressbar.OptionEnableColorCodes(p.ColorCodes),
		progressbar.OptionShowBytes(p.ShowBytes),
		progressbar.OptionSetWidth(p.Width),
		progressbar.OptionSetDescription(p.Description),
		progressbar.OptionSetTheme(p.Style))

	return bar
}

func Example() {

	bar := Progress{
		"[cyan][1/3][reset] This is a test task",
		1000,
		progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		},
		true,
		true,
		50}.NewBar()

	for i := 0; i < 1000; i++ {
		bar.Add(1)
		time.Sleep(5 * time.Millisecond)
	}
}
