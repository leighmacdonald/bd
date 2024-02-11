package client

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"errors"
	"github.com/leighmacdonald/bd/discord/ipc"
)

var (
	ErrMarshalHandshake = errors.New("failed to marshal login")
	ErrOpenIPC          = errors.New("failed to open ipc socket")
	ErrSendIPC          = errors.New("failed to send login to ipc socket")
	ErrCloseIPC         = errors.New("failed to close ipc socket")
	ErrMarshalActivity  = errors.New("failed to marshal discord activity")
	ErrCreateNonce      = errors.New("failed to read rand for nonce")
)

type Client struct {
	ipcOpened atomic.Bool
	ipc       *ipc.DiscordIPC
}

func New() *Client {
	client := &Client{ipc: ipc.New()}
	client.ipcOpened.Store(false)

	return client
}

func (d *Client) Login(clientID string) error {
	if !d.ipcOpened.Load() {
		payload, errMarshal := json.Marshal(Handshake{"1", clientID})
		if errMarshal != nil {
			return errors.Join(errMarshal, ErrMarshalHandshake)
		}

		errOpen := d.ipc.OpenSocket()
		if errOpen != nil {
			return errors.Join(errOpen, ErrOpenIPC)
		}

		if _, errSend := d.ipc.Send(0, string(payload)); errSend != nil {
			return errors.Join(errSend, ErrSendIPC)
		}
	}

	d.ipcOpened.Store(true)

	return nil
}

func (d *Client) Logout() error {
	d.ipcOpened.Store(false)

	if errClose := d.ipc.Close(); errClose != nil {
		return errors.Join(errClose, ErrCloseIPC)
	}

	return nil
}

func (d *Client) SetActivity(activity Activity) error {
	if !d.ipcOpened.Load() {
		return nil
	}

	nonce, errNonce := getNonce()
	if errNonce != nil {
		return errNonce
	}

	payload, errMarshal := json.Marshal(Frame{"SET_ACTIVITY", Args{os.Getpid(), mapActivity(&activity)}, nonce})
	if errMarshal != nil {
		return errors.Join(errMarshal, ErrMarshalActivity)
	}

	resp, errSend := d.ipc.Send(1, string(payload))
	if errSend != nil {
		return errors.New(resp)
	}

	return nil
}

func getNonce() (string, error) {
	buf := make([]byte, 16)

	_, err := rand.Read(buf)
	if err != nil {
		return "", ErrCreateNonce
	}

	buf[6] = (buf[6] & 0x0f) | 0x40

	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:]), nil
}

// Activity holds the data for discord rich presence.
type Activity struct {
	// What the player is currently doing
	Details string
	// The user's current party status
	State string
	// The id for a large asset of the activity, usually a snowflake
	LargeImage string
	// Text displayed when hovering over the large image of the activity
	LargeText string
	// The id for a small asset of the activity, usually a snowflake
	SmallImage string
	// Text displayed when hovering over the small image of the activity
	SmallText string
	// Information for the current party of the player
	Party *Party
	// Unix timestamps for start and/or end of the game
	Timestamps *Timestamps
	// Secrets for Rich Presence joining and spectating
	Secrets *Secrets
	// Clickable buttons that open a URL in the browser
	Buttons []*Button
}

// Button holds a label and the corresponding URL that is opened on press.
type Button struct {
	// The label of the button
	Label string
	// The URL of the button
	URL string
}

// Party holds information for the current party of the player.
type Party struct {
	// The ID of the party
	ID string
	// Used to show the party's current size
	Players int
	// Used to show the party's maximum size
	MaxPlayers int
}

// Timestamps holds unix timestamps for start and/or end of the game.
type Timestamps struct {
	// unix time (in milliseconds) of when the activity started
	Start *time.Time
	// unix time (in milliseconds) of when the activity ends
	End *time.Time
}

// Secrets holds secrets for Rich Presence joining and spectating.
type Secrets struct {
	// The secret for a specific instanced match
	Match string
	// The secret for joining a party
	Join string
	// The secret for spectating a game
	Spectate string
}

func mapActivity(activity *Activity) *PayloadActivity {
	final := &PayloadActivity{
		Details: activity.Details,
		State:   activity.State,
		Assets: PayloadAssets{
			LargeImage: activity.LargeImage,
			LargeText:  activity.LargeText,
			SmallImage: activity.SmallImage,
			SmallText:  activity.SmallText,
		},
	}

	if activity.Timestamps != nil && activity.Timestamps.Start != nil {
		start := uint64(activity.Timestamps.Start.UnixNano() / 1e6)
		final.Timestamps = &PayloadTimestamps{
			Start: &start,
		}

		if activity.Timestamps.End != nil {
			end := uint64(activity.Timestamps.End.UnixNano() / 1e6)
			final.Timestamps.End = &end
		}
	}

	if activity.Party != nil {
		final.Party = &PayloadParty{
			ID:   activity.Party.ID,
			Size: [2]int{activity.Party.Players, activity.Party.MaxPlayers},
		}
	}

	if activity.Secrets != nil {
		final.Secrets = &PayloadSecrets{
			Join:     activity.Secrets.Join,
			Match:    activity.Secrets.Match,
			Spectate: activity.Secrets.Spectate,
		}
	}

	if len(activity.Buttons) > 0 {
		for _, btn := range activity.Buttons {
			final.Buttons = append(final.Buttons, &PayloadButton{
				Label: btn.Label,
				URL:   btn.URL,
			})
		}
	}

	return final
}

type Handshake struct {
	V        string `json:"v"`
	ClientID string `json:"client_id"`
}

type Frame struct {
	Cmd   string `json:"cmd"`
	Args  Args   `json:"args"`
	Nonce string `json:"nonce"`
}

type Args struct {
	Pid      int              `json:"pid"`
	Activity *PayloadActivity `json:"activity"`
}

type PayloadActivity struct {
	Details    string             `json:"details,omitempty"`
	State      string             `json:"state,omitempty"`
	Assets     PayloadAssets      `json:"assets,omitempty"`
	Party      *PayloadParty      `json:"party,omitempty"`
	Timestamps *PayloadTimestamps `json:"timestamps,omitempty"`
	Secrets    *PayloadSecrets    `json:"secrets,omitempty"`
	Buttons    []*PayloadButton   `json:"buttons,omitempty"`
}

type PayloadAssets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

type PayloadParty struct {
	ID   string `json:"id,omitempty"`
	Size [2]int `json:"size,omitempty"`
}

type PayloadTimestamps struct {
	Start *uint64 `json:"start,omitempty"`
	End   *uint64 `json:"end,omitempty"`
}

type PayloadSecrets struct {
	Match    string `json:"match,omitempty"`
	Join     string `json:"join,omitempty"`
	Spectate string `json:"spectate,omitempty"`
}

type PayloadButton struct {
	Label string `json:"label,omitempty"`
	URL   string `json:"url,omitempty"`
}
