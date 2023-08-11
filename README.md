# Bot Detector

Automatically detect and kick bots & cheaters in TF2. 

## Warning

This is very early in development, expect bugs & non-working functionality.

## What about [TF2 Bot Detector](https://github.com/PazerOP/tf2_bot_detector)?

If it works for you, feel free to keep using it, active development however has stopped. bd supports 
importing and exporting TF2BD player and rule lists to help ease adoption to this new tool. His tool is
quite difficult to hack on, so one of the goals of this project was to simplify that to encourage more
outside contributions.

## Current & Planned Features

- [x] Automatically download updated remote TF2BD lists
  - [x] Rules
  - [x] Players
- [ ] Cool logo
- [x] Custom 3rd party links
- [x] Discord rich presence
- [x] Fetch profile summary and ban info from steam web api
- [ ] Detection Methods
  - [x] Steam ID
  - [x] Name Pattern
  - [x] Avatar Pattern
  - [ ] Multi match
- [x] Translations
  - [x] English
  - [x] Russian
- [ ] WebGUI / Widget 
  - [x] Player status display list
  - [x] Current game chat dialogue 
    - [x] Send in-game chat messages
  - [-] Player profile panel
    - [ ] Show the highest level of UGC/ETF2L/RGL league history achieved
    - [x] Show sourcebans bans history
    - [ ] Logs.tf count
  - [x] Player all-time chat history dialogue
  - [x] Player all-time name history dialogue
  - [x] Track all-time k:d against players
  - [x] External link configuration dialogue
  - [x] List configuration dialogue
  - [x] Settings dialogue
  - [ ] Rule creator & tester
  - [x] Auto start TF2 on launch & auto quit on game close.

## Installation

There is currently no pre-built binaries for this branch yet, when it's ready for user testing they will be made available. 
You can follow the [development](docs/DEVEL.md) instructions to create a build if you want to
see the current state.
