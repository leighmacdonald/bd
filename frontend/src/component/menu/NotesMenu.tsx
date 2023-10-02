import { IconMenuItem } from 'mui-nested-menu';
import NoteAltOutlinedIcon from '@mui/icons-material/NoteAltOutlined';
import NiceModal from '@ebay/nice-modal-react';
import { ModalNotes } from '../../App';
import React from 'react';
import { SteamIDProps } from './common';
import { logError } from '../../util';

interface NotesMenuProps {
    notes: string;
    onClose: () => void;
}

export const NotesMenu = ({
    notes,
    onClose,
    steam_id
}: NotesMenuProps & SteamIDProps) => {
    return (
        <IconMenuItem
            leftIcon={<NoteAltOutlinedIcon color={'primary'} />}
            label={'Edit Notes'}
            onClick={() => {
                NiceModal.show(ModalNotes, {
                    steamId: steam_id,
                    notes: notes
                }).then((value) => {
                    logError(value);
                });
                onClose();
            }}
        />
    );
};
