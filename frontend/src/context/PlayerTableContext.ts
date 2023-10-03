import { createContext, Dispatch, SetStateAction } from 'react';
import { Order, validColumns } from '../component/PlayerTable';
import { Player } from '../api';
import noop from 'lodash/noop';

export const defaultPlayerTableConfig: PlayerTableConfigContextProps = {
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

interface PlayerTableConfigContextProps {
    order: Order;
    setOrder: Dispatch<SetStateAction<Order>>;
    orderBy: keyof Player;
    setOrderBy: Dispatch<SetStateAction<keyof Player>>;
    matchesOnly: boolean;
    setMatchesOnly: Dispatch<SetStateAction<boolean>>;
    enabledColumns: validColumns[];
    saveSelectedColumns: (columns: validColumns[]) => void;
    saveSortColumn: (property: keyof Player) => void;
}

export const PlayerTableContext = createContext<PlayerTableConfigContextProps>(
    defaultPlayerTableConfig
);
