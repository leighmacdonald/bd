import Dialog from '@mui/material/Dialog';
import { DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import { Trans, useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { Buttons } from '../fields/Buttons.tsx';
import { useForm } from '@tanstack/react-form';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { TextFieldSimple } from '../fields/TextFieldSimple.tsx';
import { z } from 'zod';
import { noop } from '../../util.ts';

interface KickTagEditorProps {
    originalTag?: string;
}

export const SettingsKickTagEditorModal = NiceModal.create<KickTagEditorProps>(
    ({ originalTag }) => {
        const { t } = useTranslation();
        const modal = useModal();

        const { Field, Subscribe, handleSubmit } = useForm({
            onSubmit: async ({ value }) => {
                modal.resolve(value.tag);
                await modal.hide();
            },
            validatorAdapter: zodValidator,
            defaultValues: {
                tag: originalTag ?? ''
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
                    <DialogTitle>
                        <Trans i18nKey={'new_kick_tag.title'} />
                    </DialogTitle>
                    <DialogContent>
                        <Stack spacing={1} padding={0}>
                            <Field
                                name={'tag'}
                                validators={{
                                    onChange: z
                                        .string()
                                        .min(2)
                                        .regex(
                                            /\s/,
                                            'Must not container spaces'
                                        )
                                }}
                                children={(props) => {
                                    return (
                                        <TextFieldSimple
                                            {...props}
                                            label={t('new_kick_tag.tag')}
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
                                        reset={noop}
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
