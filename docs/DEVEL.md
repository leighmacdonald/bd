# Development

This tool is built using the following stack:

- Backend
  - [golang](https://go.dev/)
  - [sqlite](https://gitlab.com/cznic/sqlite)
  - [golang-migrate](https://github.com/golang-migrate/migrate)

- Frontend
  - [TypeScript](https://www.typescriptlang.org/) 
  - [nodejs](https://nodejs.org/en)
  - [pnpm](https://pnpm.io/)
  - [vite](https://vitejs.dev/) 
  - [swc](https://swc.rs/)
  - [react](https://react.dev/)
  - [material-ui](https://mui.com/material-ui/)
  - [nice-modal-react](https://github.com/eBay/nice-modal-react)

## Go Version

The *minimum* supported version is go 1.22 do to the use of the new `http.ServeMux` stdlib features.

## Install OS Dependencies

- Linux (debian/ubuntu)
    - `sudo apt-get install gcc libgtk-3-dev libayatana-appindicator3-dev make nodejs git`
  
Note that some distros may require the `libxapp-dev` package to be installed as well. If you do not have a 
supported systray, or none at all, you will need to open the url manually or otherwise make sure to enable
the auto open option.

- Windows (via [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/))
  - `winget install -e --id OpenJS.NodeJS`
  - `winget install -e --id Git.Git`
  - `winget install -e --id GnuWin32.Make`

## Checkout source

    # New checkout
    $ git clone git://github.com/leighmacdonald/bd.git && cd bd

## Update existing sources

    $ git pull

If you are just planning on running the tool in its current (incomplete) state as a user you can skip to the [Building](#Building) step.

## Developer Commands

The `build_deps` command installs or updates the current go linter/static analysis tools.

    make build_deps

## Formatters

You can automatically format the source via the `gci`, `gofumpt` and `prettier` tools. This will apply the required 
formatting rules for both the golang and typescript files. Failure to do this will generally cause builds to fail 
when pushing, so be sure to do this before committing.

    make fmt

## Run tests

    make test

## Building

### Snapshot

Build a snapshot from the current working tree using goreleaser. The binaries built with this method embed all 
frontend assets into the binary using `embed.FS`. 

    make snapshot

Binaries are output to `build/bd_${platform}_${arch}/`.

This method is not convenient for development, so you should use the [Development](#Development) steps for most things.

### Development
    
Build frontend in development mode using `webpack watch` command for auto-recompilation.

    make watch

You can use the `TEST_CONSOLE_LOG` to replay logs for testing so you have to connect to live servers. An example
is included in the `testdata` that you can use. Otherwise you can just remove it to test live.

    TEST_CONSOLE_LOG=testdata/console.log go run main.go

### Release

Production release builds are handled by GitHub actions, but if you are going to release yourself, using goreleaser, you 
will need to set the following env vars:
    
    # This github token needs the repo scope enabled. 
    GITHUB_TOKEN=$YOUR_GITHUB_PAT GPG_FINGERPRINT=$YOUR_GPG_FINGERPRINT goreleaser release --clean

If you are not using goreleaser, you must ensure that you use the `release` build tag like: `go build -tags release`. This
tag tells the compiler to embed the assets into the binary for simple, single file deployment.

## Database Changes

The database schemas are automatically generated using sqlc from the store/queries.sql file. If you add new query
you will need o follow the format used there and/or read the sqlc [docs](https://docs.sqlc.dev/en/stable/tutorials/getting-started-postgresql.html).

Once you have changes read you can uses code generation to regenerate the models via:

    make generate

## Startup & Environment Info

Running the binary should automatically open your browser. But if it doesn't, you can open the corresponding default url
shown below in any browser (eg: steam in-game overlay).

- release : http://localhost:8900
- debug   : http://localhost:8901

If you use steam in-game overlay, its generally easiest to just change your steam homepage to the
above link. This lets you use the home button as a fake application link to avoid typing it all the time.

### Data & config locations

Linux: `$HOME/.config/bd/`
  
Windows `%AppData%\bd\`

    avatars/                        Local avatar cache
    bd.log                          Current session log 
    bd.sqlite                       Database in sqlite format (contains: players, names, messages)
    bd.yaml                         Config file (Dont edit with app open)
    lists/                          Local and 3rd party user lists
    lists/playerlist.local.json     The users personal playerlist
