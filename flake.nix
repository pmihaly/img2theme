{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        deps = with pkgs; [
          go
        ];
      in
      {
	packages.default = pkgs.buildGoModule {
          name = "img2theme";
          src = ./.;
          vendorHash = null;
	};
        devShell = pkgs.mkShell {
          buildInputs = deps;
        };
      });
}
