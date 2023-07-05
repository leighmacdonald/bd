package voiceban

import (
	"encoding/binary"
	"io"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

const (
	banMgrVersion = 1
	idSize        = 32
)

func Read(reader io.Reader) (steamid.Collection, error) {
	var (
		version int32
		ids     steamid.Collection
	)

	errVersion := binary.Read(reader, binary.BigEndian, &version)
	if errVersion != nil {
		return nil, errors.Wrap(errVersion, "Failed to read binary version")
	}

	if version != banMgrVersion {
		return nil, errors.New("Invalid version")
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

		parsedSid := steamid.New(trimID)
		if !parsedSid.Valid() {
			return nil, errors.New("Malformed steamid")
		}

		ids = append(ids, parsedSid)
	}

	return ids, nil
}

func Write(output io.Writer, steamIds steamid.Collection) error {
	var version int32 = banMgrVersion
	if errWrite := binary.Write(output, binary.BigEndian, version); errWrite != nil {
		return errors.Wrap(errWrite, "Failed to write binary version data")
	}

	for _, sid := range steamIds {
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
			return errors.Wrap(errWrite, "Failed to write binary steamid data")
		}
	}

	return nil
}
