import { createContext, Dispatch, SetStateAction } from 'react';
import { Order, validColumns } from '../component/PlayerTable';
import { Player } from '../api';
import { noop } from '../util';

export const defaultPlayerTableConfig: PlayerTableConfigProps = {
    order: 'desc',
    orderBy: 'kills',
    matchesOnly: false,
    enabledColumns: [],
    setMatchesOnly: noop,
    setOrder: noop,
    setOrderBy: noop,
    saveSelectedColumns: noop,
    saveSortColumn: noop
};

interface PlayerTableConfigProps {
    order: Order;
    setOrder: Dispatch<SetStateAction<Order>>;
    orderBy: keyof Player;
    setOrderBy: Dispatch<SetStateAction<keyof Player>>;
    matchesOnly: boolean;
    setMatchesOnly: Dispatch<SetStateAction<boolean>>;
    enabledColumns: validColumns[];
    saveSelectedColumns: (columns: validColumns[]) => void;
    saveSortColumn: (property: validColumns) => void;
}

export const PlayerTableContext = createContext<PlayerTableConfigProps>(
    defaultPlayerTableConfig
);
