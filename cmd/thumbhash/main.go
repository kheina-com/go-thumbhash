package main

import (
	"encoding/base64"
	"fmt"
	"image"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/galdor/go-program"
	thumbhash "github.com/kheina-com/go-thumbhash"

	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	_ "golang.org/x/image/webp"
)

func main() {
	var c *program.Command

	p := program.NewProgram("thumbhash",
		"utilities for the go-thumbhash image placeholder generation library")

	c = p.AddCommand("image-to-raw-data", "convert an image to a raw data file",
		cmdImageToRawData)
	c.AddArgument("path", "the path of the image to decode")
	c.AddOption("o", "output", "path", "", "the path to write decoded data to")

	c = p.AddCommand("encode-image", "compute the base64-encoded hash of an image file",
		cmdEncodeImage)
	c.AddOptionalArgument("path", "the path of the image to encode")

	c = p.AddCommand("decode-image", "decode an image from a hash",
		cmdDecodeImage)
	c.AddArgument("path", "the path of the image to encode")
	c.AddArgument("hash", "the base64-encoded hash")
	c.AddOption("s", "size", "pixels", "", "the base size of the decode image")

	p.ParseCommandLine()
	p.Run()
}

func readFilePath(p *program.Program) (filePath string) {
	filePath = p.ArgumentValue("path")
	if filePath == "" {
		p.Fatal("missing required argument: path")
	}
	return
}

func cmdImageToRawData(p *program.Program) {
	filePath := readFilePath(p)

	var outputPath string
	if output := p.OptionValue("output"); output != "" {
		outputPath = output
	} else {
		ext := filepath.Ext(filePath)
		outputPath = filePath[:len(filePath)-len(ext)] + ".data"
	}

	img, err := readImage(filePath)
	if err != nil {
		p.Fatal("cannot read image from %q: %v", filePath, err)
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	data := rgba.Pix

	if err := ioutil.WriteFile(outputPath, data, 0644); err != nil {
		p.Fatal("cannot write %q: %v", outputPath, err)
	}
}

func cmdEncodeImage(p *program.Program) {
	var img image.Image
	var err error

	if stat, _ := os.Stdin.Stat(); stat.Mode()&os.ModeCharDevice == 0 {
		if img, err = readFile(os.Stdin); err != nil {
			p.Fatal("cannot read image from pipe: %v", err)
		}
	} else {
		filePath := readFilePath(p)
		if img, err = readImage(filePath); err != nil {
			p.Fatal("cannot read image from %q: %v", filePath, err)
		}
	}

	hash := thumbhash.EncodeImage(img)
	fmt.Println(base64.StdEncoding.EncodeToString(hash))
}

func cmdDecodeImage(p *program.Program) {
	filePath := readFilePath(p)
	hashString := p.ArgumentValue("hash")

	hash, err := base64.StdEncoding.DecodeString(hashString)
	if err != nil {
		p.Fatal("cannot decode base64-encoded hash: %v", err)
	}

	var cfg thumbhash.DecodingCfg

	if p.IsOptionSet("size") {
		sizeString := p.OptionValue("size")

		i64, err := strconv.ParseInt(sizeString, 10, 64)
		if err != nil || i64 < 1 || i64 > math.MaxInt32 {
			p.Fatal("invalid image size %q", sizeString)
		}

		cfg.BaseSize = int(i64)
	}

	img, err := thumbhash.DecodeImageWithCfg(hash, cfg)
	if err != nil {
		p.Fatal("cannot decode image: %v", err)
	}

	if err := writeImage(img, filePath); err != nil {
		p.Fatal("cannot encode image: %v", err)
	}
}

func readFile(file *os.File) (image.Image, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("cannot decode file: %w", err)
	}

	return img, nil
}

func readImage(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()
	return readFile(file)
}

func writeImage(img image.Image, filePath string) error {
	flags := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	file, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("cannot encode file: %w", err)
	}

	return nil
}
