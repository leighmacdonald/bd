import loadable from '@loadable/component';
import NiceModal from '@ebay/nice-modal-react';

const NoteEditorModal = loadable(() => import('./NoteEditorModal'));
const SettingsModal = loadable(() => import('./SettingsEditorModal'));
const SettingsKickTagsModal = loadable(
    () => import('./SettingsKickTagEditorModal')
);
const SettingsLinkEditorModal = loadable(
    () => import('./SettingsLinkEditorModal')
);
const SettingsListEditorModal = loadable(
    () => import('./SettingsListEditorModal')
);
const MarkNewTagEditorModal = loadable(() => import('./MarkNewTagEditorModal'));

export const ModalNotes = 'modal-notes';
export const ModalMarkNewTag = 'modal-mark-new-tag';
export const ModalSettings = 'modal-settings';
export const ModalSettingsAddKickTag = 'modal-settings-add-kick-tag';
export const ModalSettingsLinks = 'modal-settings-links';
export const ModalSettingsList = 'modal-settings-list';

[
    [ModalNotes, NoteEditorModal],
    [ModalMarkNewTag, MarkNewTagEditorModal],
    [ModalSettingsAddKickTag, SettingsKickTagsModal],
    [ModalSettingsLinks, SettingsLinkEditorModal],
    [ModalSettingsList, SettingsListEditorModal],
    [ModalSettings, SettingsModal]
].map((value) => {
    NiceModal.register(value[0] as never, value[1] as never);
});
