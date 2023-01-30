package main

import (
	"context"
	"fmt"
	"github.com/gorcon/rcon"
	"github.com/gorcon/rcon/rcontest"
	_ "github.com/leighmacdonald/bd/translations"
	"github.com/leighmacdonald/bd/ui"
	"net"
)

func genStatus() string {
	return `[default] hostname: Uncletopia | Atlanta | 1 | All Maps
version : 7757534/24 7757534 secure
udp/ip  : 74.91.112.148:27015
steamid : [G:1:4430558] (85568392924469982)
account : not logged in  (No account specified)
map     : pl_pier at: 0 x, 0 y, 0 z
tags    : nocrits,nodmgspread,payload,uncletopia
sourcetv:  74.91.112.148:27015, delay 0.0s  (local: 74.91.112.148:27016)
players : 24 humans, 1 bots (33 max)
edicts  : 1210 used of 2048 max
# userid name                uniqueid            connected ping loss state  adr
#      2 "Uncletopia | Atlanta | 1 | All " BOT                       active
#    292 "2,000,000 merged Chihuahuas" [U:1:170506368] 27:06   39    0 active 1.2.3.206:27005
#    297 "Just PYE"          [U:1:1479357239]    21:52       69    0 active 1.2.3.158:27005
#    251 "nemo"              [U:1:359724467]      1:34:14    76    0 active 1.2.3.34:27005
#    299 "tyg"               [U:1:95981139]      15:56       62    0 active 1.2.3.5:27005
#    298 "Onyx"              [U:1:315083912]     18:30       96    0 active 1.2.3.136:27005
#    276 "zavala = smurf"    [U:1:288940928]     58:30       76    0 active 1.2.3.220:27005
#    277 "Towns"             [U:1:1024397135]    54:09       61    0 active 1.2.3.67:27005
#    301 "They used"         [U:1:837570963]     14:12       61    0 active 1.2.3.78:27005
#    307 "lil hog"           [U:1:1270664108]    00:30       82    0 active 1.2.3.194:27005
#    285 "Pourpel"           [U:1:98859742]      46:24       57    0 active 1.2.3.41:27005
#    288 "lu"                [U:1:120466144]     40:20       70    0 active 1.2.3.39:27005
#    290 "BLU1| Inky (Coolest)" [U:1:1008071294] 36:31       54    0 active 1.2.3.14:27005
#    300 "Messy_"            [U:1:1135314936]    14:40      120    0 active 1.2.3.202:12607
#    273 "ATLS1 | P4KA"      [U:1:355025589]      1:08:36    73    0 active 1.2.3.190:27005
#    305 "Crescent"          [U:1:134858905]     04:19       71    0 active 1.2.3.131:53114
#    281 "Oxyclean"          [U:1:82689085]      53:14       85    0 active 1.2.3.85:27005
#    294 "Rio<33333"         [U:1:338910237]     24:29       56    0 active 1.2.3.47:27005
#    272 "Dr.Zarann"         [U:1:203687230]      1:09:10    60    0 active 1.2.3.201:55842
#    280 "ATL1 | on my hunting trip" [U:1:355075918] 53:36   68    0 active 1.2.3.134:27005
#    302 "can my wife use your toilet" [U:1:160707434] 11:30   82    0 active 1.2.3.184:27005
#    284 "Decolour"          [U:1:1085342644]    47:58       84    0 active 1.2.3.18:27005
#    306 "✿"               [U:1:178179057]     02:02       74    0 active 1.2.2.3:27005
#    233 "DuckMann"          [U:1:236792910]      1:48:54    48    0 active 1.2.3.49:27005
#    304 "damiankiller1243"  [U:1:908619362]     06:15       70    0 active 1.2.3.57:27005

`
}

func newLocalListener(addr string) net.Listener {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("rcontest: failed to listen on a port: %v", err))
	}
	return l
}

func main() {
	ctx := context.Background()
	rc := newRconConfig(true)

	server := rcontest.NewUnstartedServer()

	server.Settings.Password = rc.Password()
	server.SetAuthHandler(func(c *rcontest.Context) {
		if c.Request().Body() == c.Server().Settings.Password {
			rcon.NewPacket(rcon.SERVERDATA_AUTH_RESPONSE, c.Request().ID, "").WriteTo(c.Conn())
		} else {
			rcon.NewPacket(rcon.SERVERDATA_AUTH_RESPONSE, -1, string([]byte{0x00})).WriteTo(c.Conn())
		}
	})
	server.SetCommandHandler(func(c *rcontest.Context) {
		rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, genStatus()).WriteTo(c.Conn())
	})
	server.Start()
	defer server.Close()

	gui := ui.New(ctx)
	botDetector := New(ctx, rc)
	botDetector.AttachGui(gui)
	go botDetector.start()
	gui.Start()
}
