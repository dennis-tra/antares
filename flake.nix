{
  description = "A gateway and pinning service probing tool.";

  inputs.flake-compat = {
    url = "github:edolstra/flake-compat";
    flake = false;
  };

  inputs.devenv.url = "github:cachix/devenv";

  outputs = { self, nixpkgs, devenv, ... }@inputs:
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

      devShells = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in {
          devenv = devenv.lib.mkShell {
            inherit inputs pkgs;
            modules = [
              {
                packages = with pkgs; [ go-migrate gnumake ];
                languages.go.enable = true;
                pre-commit.hooks.actionlint.enable = true;
                pre-commit.hooks.nixfmt.enable = true;
                pre-commit.hooks.govet.enable = true;
              }
              {
                services.postgres.enable = true;
                services.postgres.listen_addresses = "127.0.0.1";
                services.postgres.initdbArgs = [ "-U antares" ];
                services.postgres.initialDatabases = [{ name = "antares"; }];
              }
              ({ config, ... }: {
                env.ANTARES_DATABASE_HOST = config.env.PGHOST;
                env.ANTARES_DATABASE_POST = config.env.PGPORT;
                env.ANTARES_DATABASE_NAME = "antares";
                env.ANTARES_DATABASE_USER = "antares";

                process.implementation = "hivemind";
              })
            ];
          };
          default = self.devShells.${system}.devenv;
        });

      legacyPackages = forAllSystems (system: nixpkgsFor.${system});

      checks = forAllSystems (system: {
        devenv = self.devShells.${system}.devenv;
        antares = self.packages.${system}.antares;
      });
    };

  nixConfig = {
    extra-substituters = [ "https://cachix.cachix.org" ];
    extra-trusted-public-keys =
      [ "antares.cachix.org-1:K+ejStePHqJSv5zxrXunGTJLiR6WrkXKe3+dHiNw+6I=" ];
  };
}
