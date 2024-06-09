import { List, ListType } from '../../api.ts';
import Dialog from '@mui/material/Dialog';
import { DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { useForm } from '@tanstack/react-form';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { Buttons } from '../fields/Buttons.tsx';
import { z } from 'zod';
import { TextFieldSimple } from '../fields/TextFieldSimple.tsx';
import { CheckboxSimple } from '../fields/CheckboxSimple.tsx';

interface SettingsListProps {
    list: List;
}

export const SettingsListEditorModal = NiceModal.create<SettingsListProps>(
    ({ list }) => {
        const modal = useModal();
        const { t } = useTranslation();

        const { Field, Subscribe, handleSubmit } = useForm({
            onSubmit: async ({ value }) => {
                modal.resolve(value);
                await modal.hide();
            },
            validatorAdapter: zodValidator,
            defaultValues: {
                list_type: list?.list_type ?? ListType.TF2BDRules,
                name: list?.name ?? '',
                enabled: list?.enabled ?? true,
                url: list?.url ?? ''
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
                        {list?.url == ''
                            ? t('settings.list_editor.create_title')
                            : `${t('settings.list_editor.edit_title')} ${
                                  list.name
                              }`}
                    </DialogTitle>
                    <DialogContent dividers>
                        <Stack>
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
                                                'settings.list_editor.enabled_label'
                                            )}
                                        />
                                    );
                                }}
                            />

                            <Field
                                name={'name'}
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

                            <Field
                                name={'url'}
                                validators={{
                                    onChange: z.string().url()
                                }}
                                children={(props) => {
                                    return (
                                        <TextFieldSimple
                                            {...props}
                                            label={'Update URL'}
                                        />
                                    );
                                }}
                            />
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
                                        reset={() => {}}
                                        isSubmitting={isSubmitting}
                                        canSubmit={canSubmit}
                                        showReset={true}
                                        onClose={async () => {
                                            await modal.hide();
                                        }}
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
