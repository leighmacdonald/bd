import { UserSettings } from '../api.ts';
import {
    Dispatch,
    SetStateAction,
    SyntheticEvent,
    useCallback,
    useMemo
} from 'react';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import { ModalSettingsAddKickTag } from './modal';
import { logError, uniqCI } from '../util.ts';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import { Autocomplete, FormControl, TextField } from '@mui/material';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import AddIcon from '@mui/icons-material/Add';

interface SettingsMultiSelectProps {
    label: string;
    newSettings: UserSettings;
    setNewSettings: Dispatch<SetStateAction<UserSettings>>;
    tooltip: string;
}

const SettingsMultiSelect = ({
    newSettings,
    setNewSettings,
    label,
    tooltip
}: SettingsMultiSelectProps) => {
    const modal = useModal();

    const onAddKickTag = useCallback(async () => {
        try {
            await NiceModal.show(ModalSettingsAddKickTag, { setNewSettings });
        } catch (e) {
            logError(e);
        } finally {
            await modal.hide();
        }
    }, [modal, setNewSettings]);

    const handleChange = (
        _: SyntheticEvent<Element, Event>,
        value: string | string[]
    ) => {
        setNewSettings((prevState) => {
            const tags = uniqCI([
                ...(typeof value === 'string' ? value.split(',') : value)
            ]).sort();
            return {
                ...prevState,
                kick_tags: tags
            };
        });
    };

    const validTags = useMemo(() => {
        return uniqCI([
            ...newSettings.unique_tags,
            ...newSettings.kick_tags
        ]).sort();
    }, [newSettings.unique_tags, newSettings.kick_tags]);

    return (
        <Stack direction={'row'} spacing={1}>
            <Tooltip title={tooltip} placement="top">
                <FormControl fullWidth>
                    <Autocomplete
                        multiple
                        id="kick_tags-select"
                        value={newSettings.kick_tags}
                        onChange={handleChange}
                        //getOptionLabel={(option) => option.title}
                        renderInput={(params) => (
                            <TextField
                                {...params}
                                variant={'outlined'}
                                label={label}
                                placeholder="Tags"
                            />
                        )}
                        options={validTags}
                    />
                </FormControl>
            </Tooltip>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <IconButton color={'success'} onClick={onAddKickTag}>
                    <AddIcon />
                </IconButton>
            </Box>
        </Stack>
    );
};

export default SettingsMultiSelect;
