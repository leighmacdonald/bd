import { Link, steamIdFormat } from '../api';
import React, { ChangeEvent, useState } from 'react';
import Dialog from '@mui/material/Dialog';
import {
    Button,
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
import CloseIcon from '@mui/icons-material/Close';
import CheckIcon from '@mui/icons-material/Check';
import { inputValidator } from './SettingsEditor';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';

interface SettingsLinkProps {
    value: Link;
    setValue: (value: Link) => void;
    validator?: inputValidator;
    open: boolean;
    setOpen: (open: boolean) => void;
    isNew: boolean;
}

export const SettingsLinkEditor = ({
    value,
    setValue,
    open,
    setOpen,
    isNew
}: SettingsLinkProps) => {
    const [list, setList] = useState<Link>({ ...value });

    const onEnabledChanged = (
        _: ChangeEvent<HTMLInputElement>,
        enabled: boolean
    ) => {
        setList({ ...list, enabled });
    };

    const onNameChanged = (event: ChangeEvent<HTMLInputElement>) => {
        setList({ ...list, name: event.target.value });
    };

    const handleSave = () => {
        setValue(list);
        handleClose();
    };

    const onUrlChanged = (event: ChangeEvent<HTMLInputElement>) => {
        setList({ ...list, url: event.target.value });
    };

    const onFormatChanged = (event: SelectChangeEvent) => {
        setList({ ...list, id_format: event.target.value as steamIdFormat });
    };

    const handleClose = () => {
        setOpen(false);
    };

    return (
        <Dialog open={open} fullWidth>
            <DialogTitle component={Typography} variant={'h1'}>
                {isNew ? `Create New Link` : `Edit Link: ${list.name}`}
            </DialogTitle>
            <DialogContent dividers>
                <Stack spacing={2}>
                    <FormGroup>
                        <FormControlLabel
                            control={
                                <Checkbox
                                    checked={list.enabled}
                                    onChange={onEnabledChanged}
                                />
                            }
                            label="Enabled"
                        />
                    </FormGroup>

                    <TextField value={list.name} onChange={onNameChanged} />
                    <FormControl fullWidth>
                        <InputLabel id="demo-simple-select-label">
                            Steam ID Format
                        </InputLabel>
                        <Select<steamIdFormat>
                            labelId="demo-simple-select-label"
                            id="demo-simple-select"
                            value={list.id_format}
                            label="Age"
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
                        value={list.url}
                        onChange={onUrlChanged}
                    />
                </Stack>
            </DialogContent>

            <DialogActions>
                <Button
                    onClick={handleClose}
                    startIcon={<CloseIcon />}
                    color={'error'}
                    variant={'contained'}
                >
                    Cancel
                </Button>
                <Button
                    onClick={handleSave}
                    startIcon={<CheckIcon />}
                    color={'success'}
                    variant={'contained'}
                >
                    Accept
                </Button>
            </DialogActions>
        </Dialog>
    );
};
