# Development

To build, you'll need to install the prerequisite libraries first.
Node/yarn are both required on all platforms.

## Install OS Dependencies

- Linux (debian/ubuntu)
    - `sudo apt-get install gcc libgtk-3-dev libayatana-appindicator3-dev make nodejs yarnpkg git`
  
Note that some distros may require the `libxapp-dev` package to be installed as well. If you do not have a 
supported systray, or none at all, you will need to open the url manually.

- Windows (via [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/))
  - `winget install -e --id OpenJS.NodeJS`
  - `winget install -e --id Yarn.Yarn`
  - `winget install -e --id Git.Git`

## Checkout source

    # New checkout
    git clone git://github.com/leighmacdonald/bd.git && cd bd

## Install js & go dependencies

    make deps

If you are just planning on running the tool in its current (incomplete) state as a user you can skip to the [Build](#Build) step.

## Linters and static analysers 

The `check_deps` command only needs to be run once, or occasionally to update tools.
It installs the cli tools that do the checks (golangci-lint && staticcheck)

    make check_deps 
    make check

## Run tests

    make test

## Build

    make local

## Run

    ./bd

Then open http://localhost:8900 in any browser (eg: steam in-game overlay).

If you use steam in-game overlay, its generally easiest to just change your steam homepage to the
above link. This lets you use the home button as a fake application link.

## Data & config locations

Linux: `$HOME/.config/bd/`
  
Windows `%AppData%\bd\`

    avatars/                        Local avatar cache
    bd.log                          Current session log 
    bd.sqlite                       Database in sqlite format (contains: players, names, messages)
    bd.yaml                         Config file (Dont edit with app open)
    lists/                          Local and 3rd party user lists
    lists/playerlist.local.json     The users personal playerlist
    