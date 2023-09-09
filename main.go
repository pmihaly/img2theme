package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"runtime"
	"sync"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

type ColorfulColor struct {
	colorful.Color
}

type Settings struct {
	Palette         []ColorfulColor `yaml:"palette"`
	PaletteAffinity float64         `yaml:"palette-affinity"`
}

func main() {
	app := &cli.App{
		Name:  "ImageMapper",
		Usage: "Map colors in an image to a specified palette.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "input",
				Value: "input.jpg",
				Usage: "Input image file path",
			},
			&cli.StringFlag{
				Name:  "output",
				Value: "output.jpg",
				Usage: "Output image file path",
			},
			&cli.StringFlag{
				Name:  "settings",
				Value: "settings.yaml",
				Usage: "Settings YAML file path",
			},
		},
		Action: mainAction,
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		cli.Exit(err, 1)
	}
}

func mainAction(c *cli.Context) error {
	inputImagePath := c.String("input")
	outputImagePath := c.String("output")
	settingsFilePath := c.String("settings")

	settings, err := loadSettings(settingsFilePath)
	if err != nil {
		return err
	}

	loadedImage, err := loadImage(inputImagePath)
	if err != nil {
		return err
	}

	mappedImage := image.NewRGBA(loadedImage.Bounds())

	mappedColorByColor := sync.Map{}
	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup
	rowCh := make(chan int, numCPU)

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mapImageRows(rowCh, loadedImage, settings, mappedImage, &mappedColorByColor)
		}()
	}

	for y := loadedImage.Bounds().Min.Y; y < loadedImage.Bounds().Max.Y; y++ {
		rowCh <- y
	}
	close(rowCh)
	wg.Wait()

	err = saveMappedImage(outputImagePath, mappedImage)

	if err != nil {
		return err
	}

	fmt.Println("Image mapped and saved at: " + outputImagePath)

	return nil
}

func (c *ColorfulColor) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var hex string
	if err := unmarshal(&hex); err != nil {
		return err
	}
	color, err := colorful.Hex(hex)
	if err != nil {
		return err
	}
	*c = ColorfulColor{color}
	return nil
}

func loadSettings(settingsFilePath string) (Settings, error) {
	rawSettings, err := os.ReadFile(settingsFilePath)
	if err != nil {
		return Settings{}, err
	}

	var settings Settings
	err = yaml.Unmarshal(rawSettings, &settings)
	if err != nil {
		return Settings{}, err
	}

	return settings, nil
}

func loadImage(inputImagePath string) (image.Image, error) {
	inputFile, err := os.Open(inputImagePath)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	img, _, err := image.Decode(inputFile)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func parsePalette(palette []string) []colorful.Color {
	var parsedPalette []colorful.Color
	for _, hex := range palette {
		c, _ := colorful.Hex(hex)
		parsedPalette = append(parsedPalette, c)
	}
	return parsedPalette
}

func mapImageRows(rowCh chan int, originalImage image.Image, settings Settings, mappedImage *image.RGBA, mappedColorByColor *sync.Map) {
	for row := range rowCh {
		for x := originalImage.Bounds().Min.X; x < originalImage.Bounds().Max.X; x++ {
			pixelColor := originalImage.At(x, row)
			mapAndSetPixelColor(pixelColor, originalImage, settings, x, row, mappedImage, mappedColorByColor)
		}
	}
}

func mapAndSetPixelColor(pixelColor color.Color, originalImage image.Image, settings Settings, x, y int, mappedImage *image.RGBA, mappedColorByColor *sync.Map) {
	if mappedColor, ok := mappedColorByColor.Load(pixelColor); ok {
		mappedImage.Set(x, y, mappedColor.(color.Color))
		return
	}

	targetLab, _ := colorful.MakeColor(pixelColor)
	minDistance := math.Inf(1)
	var mappedColor colorful.Color

	for _, c := range settings.Palette {
		distance := targetLab.DistanceLab(c.Color)
		if distance < minDistance {
			minDistance = distance
			mappedColor = c.Color
		}
	}

	adjustedColor := colorful.Color{
		R: targetLab.R + (mappedColor.R-targetLab.R)*settings.PaletteAffinity,
		G: targetLab.G + (mappedColor.G-targetLab.G)*settings.PaletteAffinity,
		B: targetLab.B + (mappedColor.B-targetLab.B)*settings.PaletteAffinity,
	}

	mappedImage.Set(x, y, adjustedColor)
	mappedColorByColor.Store(pixelColor, adjustedColor)
}

func saveMappedImage(outputImagePath string, mappedImage image.Image) error {
	outputImageFile, err := os.Create(outputImagePath)
	if err != nil {
		return err
	}
	defer outputImageFile.Close()

	jpeg.Encode(outputImageFile, mappedImage, nil)

	return nil
}
