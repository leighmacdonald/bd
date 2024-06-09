import { Link, steamIdFormat } from '../../api.ts';
import { useCallback, useEffect, useState } from 'react';
import Dialog from '@mui/material/Dialog';
import {
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    InputLabel,
    Select,
    SelectChangeEvent
} from '@mui/material';
import Stack from '@mui/material/Stack';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';
import { Trans, useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { useForm } from '@tanstack/react-form';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { Buttons } from '../fields/Buttons.tsx';
import { z } from 'zod';
import { CheckboxSimple } from '../fields/CheckboxSimple.tsx';
import { TextFieldSimple } from '../fields/TextFieldSimple.tsx';
import Grid from '@mui/material/Unstable_Grid2';
import { SelectFieldSimple } from '../fields/SelectFieldSimple.tsx';

interface SettingsLinkProps {
    link: Link;
}

export const SettingsLinkEditorModal = NiceModal.create<SettingsLinkProps>(
    ({ link }) => {
        const modal = useModal();
        const { t } = useTranslation();

        const [newLink, setNewLink] = useState<Link>({ ...link });

        const handleReset = useCallback(() => {
            setNewLink({ ...link });
        }, [link]);

        useEffect(() => {
            handleReset();
        }, [handleReset, link]);

        const onFormatChanged = (event: SelectChangeEvent) => {
            setNewLink({
                ...newLink,
                id_format: event.target.value as steamIdFormat
            });
        };

        const { Field, Subscribe, handleSubmit, reset } = useForm({
            onSubmit: async ({ value }) => {
                console.log(value);
            },
            validatorAdapter: zodValidator,

            defaultValues: {
                enabled: link?.enabled ?? true,
                url: link?.url ?? '',
                id_format: link?.id_format ?? 'steam64',
                name: link?.name ?? ''
            }
        });

        return (
            <Dialog fullWidth {...muiDialog(modal)}>
                <form
                    onSubmit={async (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        await handleSubmit();
                    }}
                >
                    <DialogTitle component={Typography} variant={'h1'}>
                        {link.url == ''
                            ? t('settings.link_editor.create_title')
                            : `${t('settings.link_editor.edit_title')} ${
                                  link.name
                              }`}
                    </DialogTitle>
                    <DialogContent dividers>
                        <Grid container>
                            <Grid xs={12}>
                                <Field
                                    name={'enabled'}
                                    validators={{
                                        onSubmit: z.boolean()
                                    }}
                                    children={(props) => {
                                        return (
                                            <CheckboxSimple
                                                {...props}
                                                label={t(
                                                    'settings.link_editor.enabled_label'
                                                )}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={12}>
                                <Field
                                    name={'name'}
                                    validators={{
                                        onChange: z.string().min(1)
                                    }}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                label={'Name'}
                                            />
                                        );
                                    }}
                                />
                                <Field
                                    name={'id_format'}
                                    validators={{
                                        onChange: z.enum([
                                            'steam64',
                                            'steam3',
                                            'steam32',
                                            'steam'
                                        ])
                                    }}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'SteamID Format'}
                                                items={[
                                                    'steam64',
                                                    'steam3',
                                                    'steam32',
                                                    'steam'
                                                ]}
                                                renderMenu={(item) => {
                                                    return (
                                                        <MenuItem
                                                            value={item}
                                                            key={`format-${item}`}
                                                        >
                                                            {item}
                                                        </MenuItem>
                                                    );
                                                }}
                                            />
                                        );
                                    }}
                                />
                                <Field
                                    name={'url'}
                                    validators={{
                                        onChange: z.string().min(2)
                                    }}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                label={'Name'}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                        </Grid>
                        <Stack spacing={2}>
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
                                        <MenuItem
                                            value={s}
                                            key={`steam-fmt-${s}`}
                                        >
                                            {s}
                                        </MenuItem>
                                    ))}
                                </Select>
                            </FormControl>
                        </Stack>
                    </DialogContent>

                    <DialogActions>
                        <Subscribe
                            selector={(state) => [
                                state.canSubmit,
                                state.isSubmitting
                            ]}
                            children={([canSubmit, isSubmitting]) => {
                                return (
                                    <Buttons
                                        reset={reset}
                                        canSubmit={canSubmit}
                                        isSubmitting={isSubmitting}
                                    />
                                );
                            }}
                        />
                    </DialogActions>
                </form>
            </Dialog>
        );
    }
);
