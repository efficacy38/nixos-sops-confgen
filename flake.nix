{
  description = "A Nix-flake-based Go development environment";

  inputs.nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.1"; # unstable Nixpkgs

  outputs =
    { self, ... }@inputs:

    let
      goVersion = 27; # Change this to update the whole stack

      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forEachSupportedSystem =
        f:
        inputs.nixpkgs.lib.genAttrs supportedSystems (
          system:
          f {
            inherit system;
            pkgs = import inputs.nixpkgs {
              inherit system;
              overlays = [ inputs.self.overlays.default ];
            };
          }
        );
    in
    {
      overlays.default = final: prev: {
        go = final."go_1_${toString goVersion}";
      };

      packages = forEachSupportedSystem (
        { pkgs, ... }:
        let
          nixos-sops-confgen = (pkgs.buildGoModule.override { go = pkgs.go; }) {
            pname = "nixos-sops-confgen";
            version = "0.1.0";

            src = self;

            # Hash of the Go module dependencies in go.sum.
            vendorHash = "sha256-WAh4av02nStKRqunXxSyovhlV9DRlspaaVD9MP3Rgik=";

            meta = {
              description = "Generate .sops.yaml files that are easier to manage in a NixOS workflow";
              mainProgram = "nixos-sops-confgen";
              platforms = pkgs.lib.platforms.unix;
            };
          };
        in
        {
          inherit nixos-sops-confgen;
          default = nixos-sops-confgen;
        }
      );

      devShells = forEachSupportedSystem (
        { pkgs, system }:
        {
          default = pkgs.mkShellNoCC {
            packages = with pkgs; [
              # go (version is specified by overlay)
              go

              # goimports, godoc, etc.
              gotools
              cobra-cli

              # https://github.com/golangci/golangci-lint
              golangci-lint

              self.formatter.${system}
            ];
          };
        }
      );

      formatter = forEachSupportedSystem ({ pkgs, ... }: pkgs.nixfmt);
    };
}
