import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

const resources = {
    en: {
        translation: {
            'app-name': 'Bot Detector',
            settings: {
                name: 'Settings',
                general: {
                    label: 'General',
                    description: 'Kicker, Tags, Chat Warnings'
                }
            }
        }
    },
    pl: {
        translation: {
            'app-name': 'BotZy Detcotzi'
        }
    }
};

i18n.use(initReactI18next).init({
    resources,
    fallbackLng: 'en',
    lng: 'en'
});

export default i18n;
