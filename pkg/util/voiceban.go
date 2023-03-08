package util

import (
	"encoding/binary"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io"
	"log"
)

const banMgrVersion = 1
const idSize = 32

func ReadVoiceBans(reader io.Reader) (steamid.Collection, error) {
	var version int32
	errVersion := binary.Read(reader, binary.LittleEndian, &version)
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
		log.Println(fmt.Sprintf("%s", sid))
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
