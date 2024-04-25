export type validColumns =
    | 'user_id'
    | 'name'
    | 'score'
    | 'kills'
    | 'deaths'
    | 'kpm'
    | 'connected'
    | 'map_time'
    | 'ping'
    | 'health'
    | 'alive';

export const defaultColumns: validColumns[] = [
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

export const getMatchesOnly = async () => {
    const storedValue = localStorage.getItem('matchesOnly');
    if (!storedValue) {
        return false;
    }

    try {
        return Boolean(JSON.parse(storedValue));
    } catch (e) {
        return false;
    }
};

export type Order = 'asc' | 'desc';

export const defaultOrderBy = 'kills';
export const defaultOrder: Order = 'desc';
export const defaultMatchesOnly = false;

export const loadOrder = async (): Promise<Order> => {
    let order = localStorage.getItem('order');
    if (order == null) {
        return defaultOrder;
    }
    try {
        order = JSON.parse(order);
        return (order ?? defaultOrder) as Order;
    } catch (e) {
        return defaultOrder;
    }
};

export const loadOrderBy = async () => {
    let columns = localStorage.getItem('orderBy');
    if (columns == null) {
        return defaultOrderBy;
    }
    try {
        columns = JSON.parse(columns);
        return columns
            ? columns.length > 0
                ? columns
                : defaultOrderBy
            : defaultOrderBy;
    } catch (e) {
        return defaultOrderBy;
    }
};

export const loadEnabledColumns = async () => {
    let columns = localStorage.getItem('enabledColumns');
    if (columns == null) {
        return defaultColumns;
    }
    try {
        columns = JSON.parse(columns);
        return columns
            ? columns.length > 0
                ? columns
                : defaultColumns
            : defaultColumns;
    } catch (e) {
        return defaultColumns;
    }
};

export const saveSortColumn = async (column: validColumns) => {
    localStorage.setItem('sortColumn', column);
    return column;
};
