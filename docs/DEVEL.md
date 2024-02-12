# Development

To build, you'll need to install the prerequisite libraries first.
Node/yarn are both required on all platforms.

## Go Version

The *minimum* supported version is go 1.22.

## Install OS Dependencies

- Linux (debian/ubuntu)
    - `sudo apt-get install gcc libgtk-3-dev libayatana-appindicator3-dev make nodejs yarnpkg git`
  
Note that some distros may require the `libxapp-dev` package to be installed as well. If you do not have a 
supported systray, or none at all, you will need to open the url manually.

- Windows (via [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/))
  - `winget install -e --id OpenJS.NodeJS`
  - `winget install -e --id Yarn.Yarn`
  - `winget install -e --id Git.Git`
  - `winget install -e --id GnuWin32.Make`

## Checkout source

    # New checkout
    git clone git://github.com/leighmacdonald/bd.git && cd bd

If you are just planning on running the tool in its current (incomplete) state as a user you can skip to the [Building](#Building) step.

## Linters and static analysers 

The `build_deps` command only needs to be run once, or occasionally to update tools.
It installs the cli tools that do the checks (golangci-lint && staticcheck)

    make build_deps

## Run tests

    make test

## Building 

### Snapshot

Build a snapshot from the current working tree using goreleaser. The binaries built with this method embed all 
frontend assets into the binary using `embed.FS`. 

    make snapshot

Binaries are output to `build/bd_${platform}_amd64_v1/`.

This method is not convenient for development, so you should use the [Development](#Development) steps for most things.

### Development
    
Build frontend in development mode using `webpack watch` command for auto-recompilation.

    make watch

You can use the `TEST_CONSOLE_LOG` to replay logs for testing so you dont need to connect to live servers. An example
is included in the `testdata` that you can use. Otherwise you can just remove it to test live.

    TEST_CONSOLE_LOG=testdata/console.log go run main.go

### Release

Production releases are handled by github actions, but if you are going to release yourself, using goreleaser, you 
will need to set the following env vars:
    
    # This github token needs the repo scope enabled. 
    GITHUB_TOKEN=$YOUR_GITHUB_PAT GPG_FINGERPRINT=$YOUR_GPG_FINGERPRINT goreleaser release --clean

If you are not using goreleaser, you must ensure that you use the `release` build tag like: `go build -tags release`. This
tag tells the compiler to embed the assets into the binary for simple, single file deployment.

## Startup & Environment Info

Running the binary should automatically open your browser. But if it doesnt, you can open the default of http://localhost:8900 in any 
browser (eg: steam in-game overlay).

If you use steam in-game overlay, its generally easiest to just change your steam homepage to the
above link. This lets you use the home button as a fake application link.

### Data & config locations

Linux: `$HOME/.config/bd/`
  
Windows `%AppData%\bd\`

    avatars/                        Local avatar cache
    bd.log                          Current session log 
    bd.sqlite                       Database in sqlite format (contains: players, names, messages)
    bd.yaml                         Config file (Dont edit with app open)
    lists/                          Local and 3rd party user lists
    lists/playerlist.local.json     The users personal playerlist
