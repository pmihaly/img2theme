package main

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/lucasb-eyer/go-colorful"
)

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

func (im *ImageMapper) QuantizePixelToPalette(x, y int) {
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

func (im *ImageMapper) QuantizeColorsToPalette(rowCh chan int) *ImageMapper {
	for row := range rowCh {
		for x := im.LoadedImage.Bounds().Min.X; x < im.LoadedImage.Bounds().Max.X; x++ {
			im.QuantizePixelToPalette(x, row)
		}
	}

	return im
}
