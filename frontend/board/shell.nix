{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = with pkgs; [
    bun
    chromium
  ];

  shellHook = ''
    CHROMIUM_BIN="${pkgs.chromium}/bin/chromium"

    wrap_binary() {
      local dir="$1"
      local bin="$2"
      if [ -d "$dir" ] && [ ! -f "$dir/.nix-wrapped" ]; then
        [ -f "$dir/$bin" ] && mv "$dir/$bin" "$dir/$bin.orig"
        printf '#!/bin/sh\nexec "%s" "$@"\n' "$CHROMIUM_BIN" > "$dir/$bin"
        chmod +x "$dir/$bin"
        touch "$dir/.nix-wrapped"
      fi
    }

    wrap_binary \
      "$HOME/.cache/ms-playwright/chromium_headless_shell-1217/chrome-headless-shell-linux64" \
      "chrome-headless-shell"

    wrap_binary \
      "$HOME/.cache/ms-playwright/chromium-1217/chrome-linux64" \
      "chrome"
  '';
}
