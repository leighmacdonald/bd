import { Link, steamIdFormat, UserSettings } from '../../api.ts';
import {
    ChangeEvent,
    Dispatch,
    SetStateAction,
    useCallback,
    useEffect,
    useState
} from 'react';
import Dialog from '@mui/material/Dialog';
import {
    Checkbox,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    FormControlLabel,
    FormGroup,
    InputLabel,
    Select,
    SelectChangeEvent,
    TextField
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { inputValidator } from './SettingsEditorModal.tsx';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';
import { Trans, useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { logError } from '../../util.ts';
import { CancelButton, ResetButton, SaveButton } from '../Buttons.tsx';

interface SettingsLinkProps {
    link: Link;
    rowIndex: number;
    validator?: inputValidator;
    setNewSettings: Dispatch<SetStateAction<UserSettings>>;
}

export const SettingsLinkEditorModal = NiceModal.create<SettingsLinkProps>(
    ({ link, rowIndex, setNewSettings }) => {
        const modal = useModal();
        const { t } = useTranslation();

        const [newLink, setNewLink] = useState<Link>({ ...link });

        const handleReset = useCallback(() => {
            setNewLink({ ...link });
        }, [link]);

        useEffect(() => {
            handleReset();
        }, [handleReset, link]);

        const onEnabledChanged = (
            _: ChangeEvent<HTMLInputElement>,
            enabled: boolean
        ) => {
            setNewLink({ ...newLink, enabled });
        };

        const onNameChanged = useCallback(
            (event: ChangeEvent<HTMLInputElement>) => {
                setNewLink({ ...newLink, name: event.target.value });
            },
            [newLink]
        );

        const handleSave = useCallback(async () => {
            try {
                setNewSettings((prevState) => {
                    prevState.links[rowIndex] = newLink;
                    return prevState;
                });
            } catch (e) {
                logError(e);
            } finally {
                await modal.hide();
            }
        }, [modal, newLink, rowIndex, setNewSettings]);

        const onUrlChanged = (event: ChangeEvent<HTMLInputElement>) => {
            setNewLink({ ...newLink, url: event.target.value });
        };

        const onFormatChanged = (event: SelectChangeEvent) => {
            setNewLink({
                ...newLink,
                id_format: event.target.value as steamIdFormat
            });
        };

        return (
            <Dialog fullWidth {...muiDialog(modal)}>
                <DialogTitle component={Typography} variant={'h1'}>
                    {link.url == ''
                        ? t('settings.link_editor.create_title')
                        : `${t('settings.link_editor.edit_title')} ${
                              link.name
                          }`}
                </DialogTitle>
                <DialogContent dividers>
                    <Stack spacing={2}>
                        <FormGroup>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={newLink.enabled}
                                        onChange={onEnabledChanged}
                                    />
                                }
                                label={t('settings.link_editor.enabled_label')}
                            />
                        </FormGroup>

                        <TextField
                            value={newLink.name}
                            onChange={onNameChanged}
                        />

                        <FormControl fullWidth>
                            <InputLabel id="steam_id_format-select-label">
                                <Trans
                                    i18nKey={
                                        'settings.link_editor.steam_id_format'
                                    }
                                />
                            </InputLabel>
                            <Select<steamIdFormat>
                                labelId="steam_id_format-select-label"
                                id="steam_id_format-select"
                                value={newLink.id_format}
                                onChange={onFormatChanged}
                            >
                                {(
                                    [
                                        'steam64',
                                        'steam3',
                                        'steam32',
                                        'steam'
                                    ] as steamIdFormat[]
                                ).map((s) => (
                                    <MenuItem value={s} key={`steam-fmt-${s}`}>
                                        {s}
                                    </MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                        <TextField
                            fullWidth
                            value={newLink.url}
                            onChange={onUrlChanged}
                        />
                    </Stack>
                </DialogContent>

                <DialogActions>
                    <CancelButton onClick={modal.hide} />
                    <ResetButton onClick={handleReset} />
                    <SaveButton onClick={handleSave} />
                </DialogActions>
            </Dialog>
        );
    }
);

export default SettingsLinkEditorModal;
