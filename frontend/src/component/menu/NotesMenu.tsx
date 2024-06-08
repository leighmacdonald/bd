import { IconMenuItem } from 'mui-nested-menu';
import NoteAltOutlinedIcon from '@mui/icons-material/NoteAltOutlined';
import NiceModal from '@ebay/nice-modal-react';
import { SteamIDProps } from './common';
import { logError } from '../../util';
import { ModalNotes } from '../modal';

interface NotesMenuProps {
    notes: string;
    onClose: () => void;
}

export const NotesMenu = ({
    notes,
    onClose,
    steamId
}: NotesMenuProps & SteamIDProps) => {
    return (
        <IconMenuItem
            leftIcon={<NoteAltOutlinedIcon color={'primary'} />}
            label={'Edit Notes'}
            onClick={() => {
                NiceModal.show(ModalNotes, {
                    steamId: steamId,
                    notes: notes
                }).then((value) => {
                    logError(value);
                });
                onClose();
            }}
        />
    );
};

export default NotesMenu;
