import { createContext, Dispatch, SetStateAction } from 'react';
import { Order, validColumns } from '../component/PlayerTable';
import { noop } from '../util';
import { GameState } from '../api.ts';

export const defaultPlayerTableConfig: PlayerTableConfigProps = {
    order: 'desc',
    orderBy: 'kills',
    matchesOnly: false,
    enabledColumns: [],
    setMatchesOnly: noop,
    setOrder: noop,
    setOrderBy: noop,
    saveSelectedColumns: noop,
    saveSortColumn: noop,
    setState: noop,
    state: {
        players: [],
        server: {
            server_name: '',
            tags: [],
            current_map: '',
            last_update: ''
        },
        game_running: false
    }
};

interface PlayerTableConfigProps {
    order: Order;
    setOrder: Dispatch<SetStateAction<Order>>;
    orderBy: validColumns;
    setOrderBy: Dispatch<SetStateAction<validColumns>>;
    matchesOnly: boolean;
    setMatchesOnly: Dispatch<SetStateAction<boolean>>;
    enabledColumns: validColumns[];
    saveSelectedColumns: (columns: validColumns[]) => void;
    saveSortColumn: (property: validColumns) => void;
    state: GameState;
    setState: (state: GameState) => void;
}

export const PlayerTableContext = createContext<PlayerTableConfigProps>(
    defaultPlayerTableConfig
);
