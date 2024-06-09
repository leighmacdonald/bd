import Dialog from '@mui/material/Dialog';
import { DialogActions, DialogContent, DialogTitle } from '@mui/material';
import { Trans, useTranslation } from 'react-i18next';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { Buttons } from '../fields/Buttons.tsx';
import { useForm } from '@tanstack/react-form';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { TextFieldSimple } from '../fields/TextFieldSimple.tsx';
import Grid from '@mui/material/Unstable_Grid2';

export const MarkNewTagEditorModal = NiceModal.create(() => {
    const { t } = useTranslation();
    const modal = useModal();

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            modal.resolve(value.tag);
            await modal.hide();
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            tag: ''
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
                    <Trans i18nKey={'mark_new_tag.title'} />
                </DialogTitle>
                <DialogContent>
                    <Grid container>
                        <Grid xs={12}>
                            <Field
                                name={'tag'}
                                validators={{
                                    onChange: z.string().min(1).regex(/\S/)
                                }}
                                children={(props) => {
                                    return (
                                        <TextFieldSimple
                                            {...props}
                                            label={t('mark_new_tag.tag')}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
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
});
