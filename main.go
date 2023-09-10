package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
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

func (im *ImageMapper) SaveAsJpeg(outputJpegPath string) error {
	outputJpeg, err := os.Create(outputJpegPath)
	if err != nil {
		return err
	}
	defer outputJpeg.Close()

	jpeg.Encode(outputJpeg, im.MappedImage, nil)

	return nil
}

func loadImage(filePath string) (image.Image, error) {
	inputFile, err := os.Open(filePath)
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

func mainAction(c *cli.Context) error {
	inputImagePath := c.String("input")
	outputImagePath := c.String("output")
	settingsFilePath := c.String("settings")

	settings, err := LoadSettingsFromYaml(settingsFilePath)
	if err != nil {
		return err
	}

	loadedImage, err := loadImage(inputImagePath)
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

	err = mapper.SaveAsJpeg(outputImagePath)

	if err != nil {
		return err
	}

	fmt.Println("Image mapped and saved at: " + outputImagePath)

	return nil
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
