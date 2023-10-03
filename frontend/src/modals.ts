import NiceModal from '@ebay/nice-modal-react';
import { NoteEditor } from './component/NoteEditor';
import { SettingsEditor } from './component/SettingsEditor';
import { SettingsLinkEditor } from './component/SettingsLinkEditor';
import { SettingsListEditor } from './component/SettingsListEditor';

export const ModalNotes = 'modal-notes';
export const ModalSettings = 'modal-settings';
export const ModalSettingsLinks = 'modal-settings-links';
export const ModalSettingsList = 'modal-settings-list';

NiceModal.register(ModalNotes, NoteEditor);
NiceModal.register(ModalSettings, SettingsEditor);
NiceModal.register(ModalSettingsLinks, SettingsLinkEditor);
NiceModal.register(ModalSettingsList, SettingsListEditor);
