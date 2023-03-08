package util

import (
	"encoding/binary"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io"
)

const banMgrVersion = 1
const idSize = 32

func VoiceBansRead(reader io.Reader) (steamid.Collection, error) {
	var version int32
	errVersion := binary.Read(reader, binary.BigEndian, &version)
	if errVersion != nil {
		return nil, errVersion
	}
	if version != banMgrVersion {
		return nil, errors.New("Invalid version")
	}
	var ids steamid.Collection
	for {
		var sid [idSize]byte
		errRead := binary.Read(reader, binary.BigEndian, &sid)
		if errRead == io.EOF {
			break
		}
		var trimId []byte
		for _, r := range sid {
			if r == 0 {
				break
			}
			trimId = append(trimId, r)
		}
		parsedSid := steamid.SID3ToSID64(steamid.SID3(trimId))
		if !parsedSid.Valid() {
			return nil, errors.New("Malformed steamid")
		}
		ids = append(ids, parsedSid)
	}
	return ids, nil
}

func VoiceBansWrite(output io.Writer, steamIds steamid.Collection) error {
	var version int32 = banMgrVersion
	if errWrite := binary.Write(output, binary.BigEndian, version); errWrite != nil {
		return errWrite
	}
	for _, sid := range steamIds {
		raw := []byte(steamid.SID64ToSID3(sid))
		var sidBytes []byte
		sidBytes = append(sidBytes, raw...)
		// pad output
		for len(sidBytes) < idSize {
			sidBytes = append(sidBytes, 0)
		}
		if errWrite := binary.Write(output, binary.BigEndian, sidBytes); errWrite != nil {
			return errWrite
		}
	}
	return nil
}
