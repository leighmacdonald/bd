import 'i18next';

// resources.ts file is generated with `npm run toc`
import resources from './resources.ts';

declare module 'i18next' {
    interface CustomTypeOptions {
        defaultNS: 'common';
        resources: typeof resources;
    }
}
