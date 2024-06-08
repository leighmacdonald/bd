import NiceModal from '@ebay/nice-modal-react';
import { NoteEditorModal } from './NoteEditorModal.tsx';
import { MarkNewTagEditorModal } from './MarkNewTagEditorModal.tsx';
import { SettingsKickTagEditorModal } from './SettingsKickTagEditorModal.tsx';
import { SettingsLinkEditorModal } from './SettingsLinkEditorModal.tsx';
import { SettingsListEditorModal } from './SettingsListEditorModal.tsx';
import { SettingsEditorModal } from './SettingsEditorModal.tsx';

export const ModalNotes = 'modal-notes';
export const ModalMarkNewTag = 'modal-mark-new-tag';
export const ModalSettings = 'modal-settings';
export const ModalSettingsAddKickTag = 'modal-settings-add-kick-tag';
export const ModalSettingsLinks = 'modal-settings-links';
export const ModalSettingsList = 'modal-settings-list';

[
    [ModalNotes, NoteEditorModal],
    [ModalMarkNewTag, MarkNewTagEditorModal],
    [ModalSettingsAddKickTag, SettingsKickTagEditorModal],
    [ModalSettingsLinks, SettingsLinkEditorModal],
    [ModalSettingsList, SettingsListEditorModal],
    [ModalSettings, SettingsEditorModal]
].map((value) => {
    NiceModal.register(value[0] as never, value[1] as never);
});
