package main

import (
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"log"
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

type Settings struct {
	Palette         []ColorfulColor `yaml:"palette"`
	PaletteAffinity float64         `yaml:"palette-affinity"`
	Cpus            int             `yaml:"cpus"`
}

func LoadSettingsFromYaml(filePath string) (Settings, error) {
	rawSettings, err := os.ReadFile(filePath)
	if err != nil {
		return Settings{}, err
	}

	settings := Settings{}
	err = yaml.Unmarshal(rawSettings, &settings)
	if err != nil {
		return Settings{}, err
	}

	return settings, nil
}

type ImageMapper struct {
	Settings           Settings
	LoadedImage        image.Image
	MappedImage        *image.RGBA
	MappedColorByColor *sync.Map
}

func NewImageMapper(settings Settings, loadedImage image.Image) (*ImageMapper, error) {
	mapper := &ImageMapper{
		Settings:           settings,
		MappedColorByColor: &sync.Map{},
		LoadedImage:        loadedImage,
		MappedImage:        image.NewRGBA(loadedImage.Bounds()),
	}

	return mapper, nil
}

func (im *ImageMapper) MapPixelToPalette(x, y int) {
	currentPixelColor := im.LoadedImage.At(x, y)

	if mappedColor, ok := im.MappedColorByColor.Load(currentPixelColor); ok {
		im.MappedImage.Set(x, y, mappedColor.(color.Color))
		return
	}

	targetLab, _ := colorful.MakeColor(currentPixelColor)
	minDistance := math.Inf(1)
	var mappedColor colorful.Color

	for _, c := range im.Settings.Palette {
		distance := targetLab.DistanceLab(c.Color)
		if distance < minDistance {
			minDistance = distance
			mappedColor = c.Color
		}
	}

	adjustedColor := colorful.Color{
		R: targetLab.R + (mappedColor.R-targetLab.R)*im.Settings.PaletteAffinity,
		G: targetLab.G + (mappedColor.G-targetLab.G)*im.Settings.PaletteAffinity,
		B: targetLab.B + (mappedColor.B-targetLab.B)*im.Settings.PaletteAffinity,
	}

	im.MappedImage.Set(x, y, adjustedColor)
	im.MappedColorByColor.Store(currentPixelColor, adjustedColor)
}

func (im *ImageMapper) mapImageRows(rowCh chan int) {
	for row := range rowCh {
		for x := im.LoadedImage.Bounds().Min.X; x < im.LoadedImage.Bounds().Max.X; x++ {
			im.MapPixelToPalette(x, row)
		}
	}
}

func loadImageFromFile(inputFile *os.File) (image.Image, error) {
	img, _, err := image.Decode(inputFile)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func mainAction(c *cli.Context) error {
	settingsFilePath := c.Args().First()

	settings, err := LoadSettingsFromYaml(settingsFilePath)
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
			mapper.mapImageRows(rowCh)
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
