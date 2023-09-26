
## Releasing

Releasing with cgo + windows is a bit annoying, so we just use wsl for now. Feel free to improve via pr.

    (wsl) $ goreleaser release --clean --split
    (win) $ goreleaser release --clean --split
    (wsl) $ cp -rv /mnt/c/projects/bd/dist/windows dist/
    (wsl) $ goreleaser continue --merge
