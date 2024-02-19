package rules

import (
	"encoding/binary"
	"errors"
	"io"
	"log/slog"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

const (
	banMgrVersion = 1
	idSize        = 32
)

var (
	ErrVoiceBanVersion      = errors.New("invalid version")
	ErrVoiceBanReadVersion  = errors.New("failed to read binary version")
	ErrVoiceBanWriteVersion = errors.New("failed to write binary version")
	ErrVoiceBanWriteSteamID = errors.New("failed to write binary steamid data")
	ErrVoiceBanOpen         = errors.New("failed to open voice_ban.dt")
	ErrVoiceBanWrite        = errors.New("failed to write voice_ban.dt")
)

func VoiceBanRead(reader io.Reader) (steamid.Collection, error) {
	var (
		vbVersion int32
		ids       steamid.Collection
	)

	errVersion := binary.Read(reader, binary.BigEndian, &vbVersion)
	if errVersion != nil {
		return nil, errors.Join(errVersion, ErrVoiceBanReadVersion)
	}

	if vbVersion != banMgrVersion {
		return nil, ErrVoiceBanVersion
	}

	for {
		var (
			sid    [idSize]byte
			trimID []byte
		)

		errRead := binary.Read(reader, binary.BigEndian, &sid)
		if errors.Is(errRead, io.EOF) {
			break
		}

		for _, r := range sid {
			if r == 0 {
				break
			}

			trimID = append(trimID, r)
		}

		parsedSid := steamid.New(string(trimID))
		if !parsedSid.Valid() {
			slog.Warn("Found invalid steam id", slog.String("steam_id", string(trimID)))
			continue
		}

		ids = append(ids, parsedSid)
	}

	return ids, nil
}

func VoiceBanWrite(output io.Writer, steamIDs steamid.Collection) error {
	var vbVersion int32 = banMgrVersion
	if errWrite := binary.Write(output, binary.BigEndian, vbVersion); errWrite != nil {
		return errors.Join(errWrite, ErrVoiceBanWriteVersion)
	}

	for _, sid := range steamIDs {
		var (
			raw      = []byte(steamid.SID64ToSID3(sid))
			sidBytes []byte
		)

		sidBytes = append(sidBytes, raw...)

		// pad output
		for len(sidBytes) < idSize {
			sidBytes = append(sidBytes, 0)
		}

		if errWrite := binary.Write(output, binary.BigEndian, sidBytes); errWrite != nil {
			return errors.Join(errWrite, ErrVoiceBanWriteSteamID)
		}
	}

	return nil
}
