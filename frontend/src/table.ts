import { validColumns } from './component/PlayerTable.tsx';

export const getDefaultColumns = (): validColumns[] => {
    const defaultCols: validColumns[] = [
        'user_id',
        'name',
        'score',
        'kills',
        'kpm',
        'deaths',
        'health',
        'connected',
        'ping',
        'alive'
    ];

    const val = localStorage.getItem('enabledColumns');
    if (!val) {
        return defaultCols;
    }
    try {
        const cols = JSON.parse(val);
        if (!cols) {
            return defaultCols;
        }
        return cols;
    } catch (_) {
        return defaultCols;
    }
};
