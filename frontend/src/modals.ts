import NiceModal from '@ebay/nice-modal-react';
import { NoteEditor } from './component/NoteEditor';
import { SettingsLinkEditor } from './component/SettingsLinkEditor';
import { SettingsListEditor } from './component/SettingsListEditor';
import { MarkNewTagEditor } from './component/MarkNewTagEditor';
import { SettingsKickTagEditor } from './component/SettingsKickTagEditor';

export const ModalNotes = 'modal-notes';
export const ModalMarkNewTag = 'modal-mark-new-tag';
export const ModalSettings = 'modal-settings';
export const ModalSettingsAddKickTag = 'modal-settings-add-kick-tag';
export const ModalSettingsLinks = 'modal-settings-links';
export const ModalSettingsList = 'modal-settings-list';

NiceModal.register(ModalNotes, NoteEditor);
NiceModal.register(ModalMarkNewTag, MarkNewTagEditor);
NiceModal.register(ModalSettingsAddKickTag, SettingsKickTagEditor);
NiceModal.register(ModalSettingsLinks, SettingsLinkEditor);
NiceModal.register(ModalSettingsList, SettingsListEditor);
