package main

import (
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/urfave/cli/v2"
)

func loadImageFromFile(inputFile *os.File) (image.Image, error) {
	img, _, err := image.Decode(inputFile)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func mainAction(c *cli.Context) error {
	settingsFilePath := c.Args().First()

	settings, err := loadSettingsFromYaml(settingsFilePath)
	if err != nil {
		return err
	}

	loadedImage, err := loadImageFromFile(os.Stdin)
	if err != nil {
		return err
	}

	mapper, err := NewImageMapper(settings, loadedImage)
	if err != nil {
		return err
	}

	numCPU := settings.Cpus
	if numCPU == 0 {
		numCPU = runtime.NumCPU()
	}

	var wg sync.WaitGroup
	rowCh := make(chan int, numCPU)

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mapper.QuantizeColorsToPalette(rowCh)
		}()
	}

	for y := mapper.LoadedImage.Bounds().Min.Y; y < mapper.LoadedImage.Bounds().Max.Y; y++ {
		rowCh <- y
	}
	close(rowCh)
	wg.Wait()

	jpeg.Encode(os.Stdout, mapper.MappedImage, nil)

	log.Println("Image mapped and written to stdout")

	return nil
}

func main() {
	app := &cli.App{
		Name:      "img2theme",
		Usage:     "Map colors in an image to a specified palette.\nExample usage: img2theme settings.yaml <input.jpg >output.jpg",
		ArgsUsage: "<settings.yaml>",
		Action:    mainAction,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
