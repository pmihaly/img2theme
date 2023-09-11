<div align="center">

# `img2theme`

[![built with nix](https://builtwithnix.org/badge.svg)](https://builtwithnix.org)

`img2theme` converts your **images to a color palette**, inspired by [ImageGoNord](https://github.com/Schrodinger-Hat/ImageGoNord) and [ImageTheming](https://github.com/daniel-seiler/ImageTheming).


[Try it out](#try-it-out) •
[Installation](#installation) •
[Roadmap](#roadmap)

</div>

[![Demo](./demo/demo.gif)](./demo/demo.gif)

## Try it out


If you already have Nix setup with flake support, you can try out img2theme like so:

```sh

cat <<EOF > nord.yaml
palette:
  - "#2e3440"
  - "#3b4252"
  - "#434c5e"
  - "#4c566a"
  - "#d8dee9"
  - "#e5e9f0"
  - "#eceff4"
  - "#8fbcbb"
  - "#88c0d0"
  - "#81a1c1"
  - "#5e81ac"
  - "#bf616a"
  - "#d08770"
  - "#ebcb8b"
  - "#a3be8c"
  - "#b48ead"
palette-affinity: 0.6  # 1.0 -> colors strictly from palette, 0.0 -> colors from the image
cpus: 0  # 0 -> use all available cpu cores
EOF

# img2theme accepts an image from the stdin and it spits out an image to stdout
# it also accepts the settings file path as an argument
nix run github:pmihaly/img2theme nord.yaml <input.jpg >output.jpg

```

## Installation

`img2theme` can be installed in 2 ways:

### Nix/OS Flakes

```nix

{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    img2theme.url = "github:pmihaly/img2theme";
  };

  outputs = { self, nixpkgs, img2theme }: {
    nixosConfigurations."<hostname>" = nixpkgs.lib.nixosSystem {
      modules = [
        ({ config, pkgs, ... }: { environment.systemPackages = [ img2theme.packages."${pkgs.system}".default ]; })
      ];
    };
  };
}


```
### Building manually

```sh

go build

cp img2theme ~/.local/bin
```


### Roadmap

- [x] Quantization - pixel-by-pixel mapping each colors to the closest color in the palette
- [x] Usable CLI interface
- [x] Pretty readme
- [x] Split up `main.go`
- [ ] Adjustable dithering to smooth out gradients
- [ ] Add webp format
- [ ] Output format should be the same as the input format
- [ ] [Solid color filter](https://github.com/lucasb-eyer/go-colorful#blending-colors) with adjustable alpha (`0` - no filter)
- [ ] Map over the same image with multiple settings
  - use user defined anchors for invariants
  - this would not work with stdout

```yaml
nord-theme: &nord-theme
  - "#2e3440"
  - "#3b4252"
  - "#434c5e"
  - "#4c566a"
  - "#d8dee9"
  - "#e5e9f0"
  - "#eceff4"
  - "#8fbcbb"
  - "#88c0d0"
  - "#81a1c1"
  - "#5e81ac"
  - "#bf616a"
  - "#d08770"
  - "#ebcb8b"
  - "#a3be8c"
  - "#b48ead"

base-mapping: &base-mapping
  palette: *nord-theme
  output-file-name: output-{{paletteAffinity}}.jpg

mappings:
  - <<: *base-mapping
    palette-affinity: 0.2
  - <<: *base-mapping
    palette-affinity: 0.4
  - <<: *base-mapping
    palette-affinity: 0.6
  - <<: *base-mapping
    palette-affinity: 0.8
cpus: 0
```
