{
  description = "A gateway and pinning service probing tool.";

  inputs.flake-compat = {
    url = "github:edolstra/flake-compat";
    flake = false;
  };

  outputs = { self, nixpkgs, ... }:
    let
      version = "${nixpkgs.lib.substring 0 8 self.lastModifiedDate}-${
          self.shortRev or "dirty"
        }";

      systems =
        [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];

      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);

      # Memoize nixpkgs for different platforms for efficiency.
      nixpkgsFor = forAllSystems (system:
        import nixpkgs {
          inherit system;
          overlays = [ self.overlays.default ];
        });

    in {
      overlays.antares = final: prev: {
        antares = final.callPackage ({ lib, stdenv, go, buildGoModule }:
          buildGoModule {
            pname = "antares";
            version = version;
            description = "A gateway and pinning service probing tool.";

            src = self;

            vendorHash = "sha256-I2+Gd1IfXUV5yDKUGB2C4HimTR3VU1qP6xvXPYe86yE=";

            postInstall = ''
              mkdir -p $out/share/antares/
              cp -r migrations $out/share/antares/

            '';

            # https://github.com/dennis-tra/antares/issues/3
            doCheck = false;
          }) { };

        antares-dev-shell = final.callPackage
          ({ lib, stdenv, mkShell, go, go-migrate, gnumake }:
            mkShell {
              pname = "antares-dev-shell";
              version = version;
              buildInputs = [ go go-migrate gnumake ];
            }) { };
      };
      overlays.default = self.overlays.antares;

      packages = forAllSystems (system: {
        inherit (nixpkgsFor.${system}) go-migrate antares;
        default = self.packages.${system}.antares;
      });

      apps = forAllSystems (system: {
        antares = {
          type = "app";
          program = "${self.packages.${system}.antares}/bin/antares";
        };
        # TODO: db migrate app
        default = self.apps.${system}.antares;
      });

      devShells = forAllSystems (system: {
        inherit (nixpkgsFor.${system}) antares-dev-shell;
        default = self.devShells.${system}.antares-dev-shell;
      });

      legacyPackages = forAllSystems (system: nixpkgsFor.${system});

    };
}
