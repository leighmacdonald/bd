import Dialog from '@mui/material/Dialog';
import { DialogActions, DialogContent, DialogTitle } from '@mui/material';
import { Trans, useTranslation } from 'react-i18next';
import { saveUserNoteMutation } from '../../api.ts';
import NiceModal, { muiDialog, useModal } from '@ebay/nice-modal-react';
import { logError } from '../../util.ts';
import { useMutation } from '@tanstack/react-query';
import { Buttons } from '../fields/Buttons.tsx';
import Grid from '@mui/material/Unstable_Grid2';
import { z } from 'zod';
import { TextFieldSimple } from '../fields/TextFieldSimple.tsx';
import { useForm } from '@tanstack/react-form';
import { zodValidator } from '@tanstack/zod-form-adapter';

interface NoteEditorProps {
    notes: string;
    steamId: string;
}

export const NoteEditorModal = NiceModal.create<NoteEditorProps>(
    ({ steamId, notes }) => {
        const { t } = useTranslation();
        const modal = useModal();

        const mutation = useMutation({
            ...saveUserNoteMutation(steamId),
            onSuccess: async () => {
                console.log(`Note updated: ${steamId}`);
                await modal.hide();
            },
            onError: (error) => {
                logError(`Error updating note: ${error}`);
            }
        });

        const { Field, Subscribe, handleSubmit, reset } = useForm({
            onSubmit: async ({ value }) => {
                mutation.mutate(value);
            },
            validatorAdapter: zodValidator,
            defaultValues: {
                notes: notes ?? ''
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
                        <Trans i18nKey={'player_table.notes.title'} />
                    </DialogTitle>
                    <DialogContent>
                        <Grid container>
                            <Grid xs={12}>
                                <Field
                                    name={'notes'}
                                    validators={{
                                        onChange: z.string().min(2)
                                    }}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                rows={10}
                                                label={t(
                                                    'player_table.notes.note_label'
                                                )}
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
    }
);
